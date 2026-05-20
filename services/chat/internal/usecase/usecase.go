package usecase

import (
	"context"
	"errors"
	"fmt"
	"html"
	"math/rand/v2"
	"strings"
	"time"
	"unicode/utf8"

	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
)

type Service struct {
	store       repository.Store
	userClient  userpb.UserServiceClient
	mediaClient mediapb.MediaServiceClient
}

func New(store repository.Store, userClient userpb.UserServiceClient, mediaClient ...mediapb.MediaServiceClient) *Service {
	var media mediapb.MediaServiceClient
	if len(mediaClient) > 0 {
		media = mediaClient[0]
	}
	return &Service{store: store, userClient: userClient, mediaClient: media}
}

func (s *Service) GetUserChats(ctx context.Context, userAccountID int64) ([]Chat, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	members, err := s.store.ChatMembers.GetByUserID(ctx, profileID)
	if err != nil {
		return nil, err
	}
	chats := make([]Chat, 0, len(members))
	for _, member := range members {
		chat, err := s.store.Chats.GetByID(ctx, member.ChatID)
		if err != nil {
			continue
		}
		chats = append(chats, s.mapChat(ctx, *chat, userAccountID))
	}
	return chats, nil
}

func (s *Service) CreatePrivateChat(ctx context.Context, userAccountID, otherID int64) (Chat, error) {
	if userAccountID <= 0 || otherID <= 0 {
		return Chat{}, ErrInvalidInput
	}
	userProfileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Chat{}, err
	}
	otherAccountID, otherProfileID, err := s.resolveTarget(ctx, otherID)
	if err != nil {
		return Chat{}, err
	}
	if userAccountID == otherAccountID {
		return Chat{}, ErrInvalidInput
	}

	memberships, err := s.store.ChatMembers.GetByUserID(ctx, userProfileID)
	if err != nil {
		return Chat{}, err
	}
	for _, membership := range memberships {
		chat, err := s.store.Chats.GetByID(ctx, membership.ChatID)
		if err != nil || chat.Type != model.PrivateChat {
			continue
		}
		members, err := s.store.ChatMembers.GetByChatID(ctx, chat.ID)
		if err != nil {
			continue
		}
		for _, member := range members {
			if member.MemberID == otherProfileID {
				return s.mapChat(ctx, *chat, userAccountID), nil
			}
		}
	}

	title := fmt.Sprintf("Личный чат %d", otherAccountID)
	if summary := s.profileSummary(ctx, otherProfileID); summary != nil {
		title = strings.TrimSpace(summary.GetFirstName() + " " + summary.GetLastName())
		if title == "" {
			title = summary.GetUsername()
		}
	}
	chat := model.NewChat(model.PrivateChat, title, nil)
	if err := s.store.Chats.Save(ctx, chat); err != nil {
		return Chat{}, err
	}

	now := time.Now()
	for _, profileID := range []int64{userProfileID, otherProfileID} {
		if err := s.store.ChatMembers.Save(ctx, model.ChatMember{
			ID:        rand.Int64(),
			Uid:       uuid.New(),
			ChatID:    chat.ID,
			MemberID:  profileID,
			JoinedAt:  now,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
			Role:      "member",
		}); err != nil {
			return Chat{}, err
		}
	}

	return s.mapChat(ctx, *chat, userAccountID), nil
}

func (s *Service) CheckUserInChat(ctx context.Context, chatID, userAccountID int64) (bool, error) {
	if chatID <= 0 || userAccountID <= 0 {
		return false, ErrInvalidInput
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return false, err
	}
	members, err := s.store.ChatMembers.GetByChatID(ctx, chatID)
	if err != nil {
		return false, err
	}
	for _, member := range members {
		if member.MemberID == profileID {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) GetMessages(ctx context.Context, userAccountID, chatID int64, limit, offset int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	ok, err := s.CheckUserInChat(ctx, chatID, userAccountID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	messages, err := s.store.Messages.GetByChatID(ctx, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.mapMessages(ctx, messages, profileID)
}

func (s *Service) GetMessagesAfter(ctx context.Context, userAccountID, chatID, afterID int64, limit int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if afterID < 0 {
		afterID = 0
	}
	ok, err := s.CheckUserInChat(ctx, chatID, userAccountID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}

	messages, err := s.store.Messages.GetByChatIDAfter(ctx, chatID, afterID, limit)
	if err != nil {
		return nil, err
	}
	return s.mapMessages(ctx, messages, profileID)
}

func (s *Service) SendMessage(ctx context.Context, userAccountID, chatID int64, input MessageInput) (Message, error) {
	text := strings.TrimSpace(input.Text)
	attachments := appendAttachmentInputs(input.Media, input.Files)
	if chatID <= 0 || userAccountID <= 0 {
		return Message{}, ErrInvalidInput
	}
	if input.StickerID != nil {
		if *input.StickerID <= 0 || text != "" || len(attachments) != 0 {
			return Message{}, ErrInvalidInput
		}
		if _, err := s.store.Stickers.Get(ctx, *input.StickerID); err != nil {
			return Message{}, ErrNotFound
		}
	} else if text == "" && len(attachments) == 0 {
		return Message{}, ErrInvalidInput
	}
	if input.ParentMessageID != nil {
		if *input.ParentMessageID <= 0 {
			return Message{}, ErrInvalidInput
		}
		parent, err := s.store.Messages.GetByID(ctx, *input.ParentMessageID)
		if err != nil || parent.ChatID != chatID {
			return Message{}, ErrNotFound
		}
	}
	ok, err := s.CheckUserInChat(ctx, chatID, userAccountID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrForbidden
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Message{}, err
	}
	if err := s.validateAttachments(ctx, profileID, attachments); err != nil {
		return Message{}, err
	}
	var textPtr *string
	if text != "" {
		textPtr = &text
	}
	msg := model.Message{
		Uid:             uuid.New(),
		Text:            textPtr,
		ParentMessageID: input.ParentMessageID,
		ChatID:          chatID,
		AuthorID:        profileID,
		StickerID:       input.StickerID,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := s.store.Messages.Save(ctx, &msg); err != nil {
		return Message{}, err
	}
	if err := s.attachMedia(ctx, msg.ID, attachments); err != nil {
		_ = s.store.Messages.Delete(ctx, msg.ID)
		return Message{}, err
	}
	messages, err := s.mapMessages(ctx, []model.Message{msg}, profileID)
	if err != nil {
		return Message{}, err
	}
	return messages[0], nil
}

func (s *Service) UpdateMessage(ctx context.Context, userAccountID, chatID, messageID int64, text string) (Message, error) {
	text = strings.TrimSpace(text)
	if chatID <= 0 || messageID <= 0 || text == "" {
		return Message{}, ErrInvalidInput
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.store.Messages.GetByID(ctx, messageID)
	if err != nil {
		return Message{}, ErrNotFound
	}
	if msg.ChatID != chatID {
		return Message{}, ErrNotFound
	}
	ok, err := s.CheckUserInChat(ctx, chatID, userAccountID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrForbidden
	}
	if msg.AuthorID != profileID {
		return Message{}, ErrForbidden
	}
	if msg.StickerID != nil {
		return Message{}, ErrInvalidInput
	}
	msg.Text = &text
	msg.UpdatedAt = time.Now()
	if err := s.store.Messages.Update(ctx, msg); err != nil {
		return Message{}, err
	}
	messages, err := s.mapMessages(ctx, []model.Message{*msg}, profileID)
	if err != nil {
		return Message{}, err
	}
	return messages[0], nil
}

func (s *Service) mapChat(ctx context.Context, chat model.Chat, viewerUserAccountID int64) Chat {
	title := html.EscapeString(chat.Title)
	if chat.Type == model.PrivateChat {
		if resolved := s.privateChatTitle(ctx, chat.ID, viewerUserAccountID); resolved != "" {
			title = html.EscapeString(resolved)
		}
	}
	return Chat{
		ID:        chat.ID,
		UID:       chat.Uid.String(),
		Title:     title,
		AvatarID:  chat.AvatarID,
		Type:      chat.Type,
		IsActive:  chat.IsActive,
		CreatedAt: chat.CreatedAt,
		UpdatedAt: chat.UpdatedAt,
	}
}

func (s *Service) mapMessages(ctx context.Context, messages []model.Message, viewerProfileID int64) ([]Message, error) {
	messageIDs := make([]int64, 0, len(messages))
	for _, message := range messages {
		messageIDs = append(messageIDs, message.ID)
	}
	mediaByMessage, err := s.store.MessageMedia.GetByMessageIDs(ctx, messageIDs)
	if err != nil {
		return nil, err
	}
	reactionsByMessage, err := s.store.Reactions.GetSummaryByMessageIDs(ctx, messageIDs)
	if err != nil {
		return nil, err
	}
	myReactions, err := s.store.Reactions.GetUserReactionsByMessageIDs(ctx, messageIDs, viewerProfileID)
	if err != nil {
		return nil, err
	}
	result := make([]Message, 0, len(messages))
	for _, message := range messages {
		result = append(result, s.mapMessage(ctx, message, mediaByMessage[message.ID], reactionsByMessage[message.ID], myReactions[message.ID]))
	}
	return result, nil
}

func (s *Service) mapMessage(ctx context.Context, message model.Message, attachments []model.MessageMedia, reactions []model.ReactionSummary, myReaction string) Message {
	text := message.Text
	if text != nil {
		escaped := html.EscapeString(html.UnescapeString(*text))
		text = &escaped
	}
	media, files := s.splitAttachments(ctx, attachments)
	var sticker *Sticker
	if message.StickerID != nil {
		if item, err := s.store.Stickers.Get(ctx, *message.StickerID); err == nil {
			sticker = s.mapSticker(ctx, *item)
		}
	}
	var myReactionPtr *string
	if myReaction != "" {
		normalized := normalizeStoredReaction(myReaction)
		myReactionPtr = &normalized
	}
	return Message{
		ID:              message.ID,
		UID:             message.Uid.String(),
		Text:            text,
		AuthorName:      s.displayName(ctx, message.AuthorID),
		ParentMessageID: message.ParentMessageID,
		ChatID:          message.ChatID,
		AuthorID:        message.AuthorID,
		StickerID:       message.StickerID,
		Sticker:         sticker,
		Media:           media,
		Files:           files,
		Reactions:       mapReactionSummaries(reactions),
		MyReaction:      myReactionPtr,
		IsActive:        message.IsActive,
		CreatedAt:       message.CreatedAt,
		UpdatedAt:       message.UpdatedAt,
	}
}

func (s *Service) GetStickerPacks(ctx context.Context, userAccountID int64, query string, myOnly bool, limit, offset int) ([]StickerPack, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	limit, offset = normalizeListBounds(limit, offset)
	query = strings.TrimSpace(query)
	var packs []model.StickerPack
	switch {
	case myOnly:
		packs, err = s.store.Stickers.ListPacksByAuthorID(ctx, profileID, limit, offset)
	case query != "":
		packs, err = s.store.Stickers.SearchPacks(ctx, query, limit, offset)
	default:
		packs, err = s.store.Stickers.ListPacks(ctx, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	result := make([]StickerPack, 0, len(packs))
	for _, pack := range packs {
		result = append(result, mapStickerPack(pack))
	}
	return result, nil
}

func (s *Service) CreateStickerPack(ctx context.Context, userAccountID int64, input StickerPackInput) (StickerPack, error) {
	title := strings.TrimSpace(input.Title)
	if userAccountID <= 0 || title == "" || utf8.RuneCountInString(title) > 63 {
		return StickerPack{}, ErrInvalidInput
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return StickerPack{}, err
	}
	pack := model.StickerPack{
		Uid:      uuid.New(),
		Title:    title,
		AuthorID: &profileID,
	}
	id, err := s.store.Stickers.CreatePack(ctx, pack)
	if err != nil {
		return StickerPack{}, err
	}
	saved, err := s.store.Stickers.GetPack(ctx, id)
	if err != nil {
		return StickerPack{}, ErrNotFound
	}
	return mapStickerPack(*saved), nil
}

func (s *Service) GetStickersByPack(ctx context.Context, userAccountID, packID int64, limit, offset int) ([]Sticker, error) {
	if userAccountID <= 0 || packID <= 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.profileIDByAccount(ctx, userAccountID); err != nil {
		return nil, err
	}
	limit, offset = normalizeListBounds(limit, offset)
	stickers, err := s.store.Stickers.ListByPackID(ctx, packID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]Sticker, 0, len(stickers))
	for _, sticker := range stickers {
		result = append(result, *s.mapSticker(ctx, sticker))
	}
	return result, nil
}

func (s *Service) CreateSticker(ctx context.Context, userAccountID, packID int64, input StickerInput) (Sticker, error) {
	if userAccountID <= 0 || packID <= 0 || input.MediaID <= 0 {
		return Sticker{}, ErrInvalidInput
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Sticker{}, err
	}
	pack, err := s.store.Stickers.GetPack(ctx, packID)
	if err != nil {
		return Sticker{}, ErrNotFound
	}
	if pack.AuthorID == nil || *pack.AuthorID != profileID {
		return Sticker{}, ErrForbidden
	}
	media, err := s.store.Stickers.GetMediaInfo(ctx, input.MediaID)
	if err != nil {
		return Sticker{}, ErrNotFound
	}
	if media.AuthorID != profileID {
		return Sticker{}, ErrForbidden
	}
	if !isImageMedia(media.MimeType) {
		return Sticker{}, ErrInvalidInput
	}
	order := 0
	if input.SortOrder != nil {
		order = *input.SortOrder
	} else {
		order, err = s.store.Stickers.NextStickerOrder(ctx, packID)
		if err != nil {
			return Sticker{}, err
		}
	}
	if order < 0 || order > 100 {
		return Sticker{}, ErrInvalidInput
	}
	sticker := model.Sticker{
		Uid:     uuid.New(),
		Size:    media.Size,
		Order:   order,
		PackID:  &packID,
		MediaID: &input.MediaID,
	}
	id, err := s.store.Stickers.CreateSticker(ctx, sticker)
	if err != nil {
		return Sticker{}, err
	}
	saved, err := s.store.Stickers.Get(ctx, id)
	if err != nil {
		return Sticker{}, ErrNotFound
	}
	return *s.mapSticker(ctx, *saved), nil
}

func (s *Service) SetMessageReaction(ctx context.Context, userAccountID, chatID, messageID int64, reactionType string) (Message, error) {
	reactionType = normalizeInputReaction(reactionType)
	if userAccountID <= 0 || chatID <= 0 || messageID <= 0 || reactionType == "" {
		return Message{}, ErrInvalidInput
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.store.Messages.GetByID(ctx, messageID)
	if err != nil || msg.ChatID != chatID {
		return Message{}, ErrNotFound
	}
	ok, err := s.CheckUserInChat(ctx, chatID, userAccountID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrForbidden
	}
	if err := s.store.Reactions.Upsert(ctx, messageID, profileID, reactionType); err != nil {
		return Message{}, err
	}
	messages, err := s.mapMessages(ctx, []model.Message{*msg}, profileID)
	if err != nil {
		return Message{}, err
	}
	return messages[0], nil
}

func (s *Service) DeleteMessageReaction(ctx context.Context, userAccountID, chatID, messageID int64) (Message, error) {
	if userAccountID <= 0 || chatID <= 0 || messageID <= 0 {
		return Message{}, ErrInvalidInput
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.store.Messages.GetByID(ctx, messageID)
	if err != nil || msg.ChatID != chatID {
		return Message{}, ErrNotFound
	}
	ok, err := s.CheckUserInChat(ctx, chatID, userAccountID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrForbidden
	}
	if err := s.store.Reactions.Delete(ctx, messageID, profileID); err != nil {
		return Message{}, err
	}
	messages, err := s.mapMessages(ctx, []model.Message{*msg}, profileID)
	if err != nil {
		return Message{}, err
	}
	return messages[0], nil
}

func (s *Service) validateAttachments(ctx context.Context, actorProfileID int64, attachments []AttachmentInput) error {
	if len(attachments) > 10 {
		return ErrInvalidInput
	}
	seen := make(map[int64]struct{}, len(attachments))
	for _, item := range attachments {
		if item.MediaID <= 0 {
			return ErrInvalidInput
		}
		if _, ok := seen[item.MediaID]; ok {
			return ErrInvalidInput
		}
		seen[item.MediaID] = struct{}{}
		authorID, err := s.store.MessageMedia.GetMediaAuthorID(ctx, item.MediaID)
		if err != nil {
			return ErrInvalidInput
		}
		if authorID != actorProfileID {
			return ErrForbidden
		}
	}
	return nil
}

func (s *Service) attachMedia(ctx context.Context, messageID int64, attachments []AttachmentInput) error {
	for i, item := range attachments {
		if err := s.store.MessageMedia.Save(ctx, model.MessageMedia{
			MessageID: messageID,
			MediaID:   item.MediaID,
			Order:     i,
		}); err != nil {
			return err
		}
	}
	return nil
}

func appendAttachmentInputs(media []AttachmentInput, files []AttachmentInput) []AttachmentInput {
	if len(files) == 0 {
		return media
	}
	result := make([]AttachmentInput, 0, len(media)+len(files))
	result = append(result, media...)
	result = append(result, files...)
	return result
}

func (s *Service) splitAttachments(ctx context.Context, items []model.MessageMedia) ([]Attachment, []Attachment) {
	media := make([]Attachment, 0, len(items))
	files := make([]Attachment, 0)
	for _, item := range items {
		attachment := Attachment{
			ID:       item.MediaID,
			UID:      item.MediaUID.String(),
			Name:     item.Name,
			MimeType: item.MimeType,
			URL:      s.mediaURL(ctx, item.MediaID, item.Link),
		}
		if isMessageMedia(item.MimeType) {
			media = append(media, attachment)
			continue
		}
		files = append(files, attachment)
	}
	return media, files
}

func (s *Service) mapSticker(ctx context.Context, sticker model.Sticker) *Sticker {
	result := &Sticker{
		ID:       sticker.ID,
		UID:      sticker.Uid.String(),
		PackID:   sticker.PackID,
		MediaID:  sticker.MediaID,
		MimeType: sticker.MimeType,
	}
	if sticker.MediaID != nil && sticker.Link != nil {
		url := s.mediaURL(ctx, *sticker.MediaID, *sticker.Link)
		result.URL = &url
	}
	return result
}

func mapStickerPack(pack model.StickerPack) StickerPack {
	return StickerPack{
		ID:        pack.ID,
		UID:       pack.Uid.String(),
		Title:     html.EscapeString(pack.Title),
		AuthorID:  pack.AuthorID,
		CreatedAt: pack.CreatedAt,
		UpdatedAt: pack.UpdatedAt,
	}
}

func (s *Service) mediaURL(ctx context.Context, mediaID int64, raw string) string {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") || s.mediaClient == nil {
		return raw
	}
	resp, err := s.mediaClient.GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: mediaID})
	if err != nil || resp == nil || strings.TrimSpace(resp.GetUrl()) == "" {
		return raw
	}
	return resp.GetUrl()
}

func isMessageMedia(mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	return mimeType == "image" ||
		mimeType == "video" ||
		strings.HasPrefix(mimeType, "image/") ||
		strings.HasPrefix(mimeType, "video/")
}

func isImageMedia(mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	return mimeType == "image" || strings.HasPrefix(mimeType, "image/")
}

func normalizeListBounds(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func mapReactionSummaries(items []model.ReactionSummary) []ReactionSummary {
	result := make([]ReactionSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ReactionSummary{Type: normalizeStoredReaction(item.Type), Count: item.Count})
	}
	return result
}

func normalizeInputReaction(value string) string {
	switch strings.TrimSpace(value) {
	case "👍", "like":
		return "👍"
	case "❤️", "love":
		return "❤️"
	case "😂", "laugh", "happy":
		return "😂"
	case "😢", "sad":
		return "😢"
	case "😡", "angry", "anger":
		return "😡"
	default:
		return ""
	}
}

func normalizeStoredReaction(value string) string {
	switch value {
	case `\like`, "like":
		return "👍"
	case `\dislike`, "dislike":
		return "😢"
	case `\happy`, "happy":
		return "😂"
	case `\anger`, "anger":
		return "😡"
	default:
		return value
	}
}

func (s *Service) privateChatTitle(ctx context.Context, chatID, viewerUserAccountID int64) string {
	viewerProfileID, err := s.profileIDByAccount(ctx, viewerUserAccountID)
	if err != nil {
		return ""
	}
	members, err := s.store.ChatMembers.GetByChatID(ctx, chatID)
	if err != nil {
		return ""
	}
	for _, member := range members {
		if member.MemberID != viewerProfileID {
			return s.displayName(ctx, member.MemberID)
		}
	}
	return ""
}

func (s *Service) displayName(ctx context.Context, profileID int64) string {
	summary := s.profileSummary(ctx, profileID)
	if summary == nil {
		return "Пользователь"
	}
	name := strings.TrimSpace(summary.GetFirstName() + " " + summary.GetLastName())
	if name == "" {
		name = summary.GetUsername()
	}
	if name == "" {
		return "Пользователь"
	}
	return name
}

func (s *Service) resolveTarget(ctx context.Context, inputID int64) (int64, int64, error) {
	if summary := s.profileSummary(ctx, inputID); summary != nil && summary.GetUserAccountId() > 0 {
		return summary.GetUserAccountId(), summary.GetProfileId(), nil
	}
	profileID, err := s.profileIDByAccount(ctx, inputID)
	if err != nil {
		return 0, 0, err
	}
	return inputID, profileID, nil
}

func (s *Service) profileIDByAccount(ctx context.Context, userAccountID int64) (int64, error) {
	if userAccountID <= 0 || s.userClient == nil {
		return 0, ErrInvalidInput
	}
	resp, err := s.userClient.GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if resp.GetProfileId() <= 0 {
		return 0, ErrNotFound
	}
	return resp.GetProfileId(), nil
}

func (s *Service) profileSummary(ctx context.Context, profileID int64) *userpb.GetProfileSummaryResponse {
	if profileID <= 0 || s.userClient == nil {
		return nil
	}
	resp, err := s.userClient.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID})
	if err != nil {
		return nil
	}
	return resp
}

func ToStatus(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
