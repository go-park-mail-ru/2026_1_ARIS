package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
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
	store      repository.Store
	userClient userpb.UserServiceClient
}

func New(store repository.Store, userClient userpb.UserServiceClient) *Service {
	return &Service{store: store, userClient: userClient}
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
		if err != nil || chat.Type != models.PrivateChat {
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
	chat := models.NewChat(models.PrivateChat, title, nil)
	if err := s.store.Chats.Save(ctx, chat); err != nil {
		return Chat{}, err
	}

	now := time.Now()
	for _, profileID := range []int64{userProfileID, otherProfileID} {
		if err := s.store.ChatMembers.Save(ctx, models.ChatMember{
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
	messages, err := s.store.Messages.GetByChatID(ctx, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]Message, 0, len(messages))
	for _, message := range messages {
		result = append(result, s.mapMessage(ctx, message))
	}
	return result, nil
}

func (s *Service) SendMessage(ctx context.Context, userAccountID, chatID int64, text string) (Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Message{}, ErrInvalidInput
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
	msg := models.Message{
		Uid:       uuid.New(),
		Text:      &text,
		ChatID:    chatID,
		AuthorID:  profileID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.store.Messages.Save(ctx, &msg); err != nil {
		return Message{}, err
	}
	return s.mapMessage(ctx, msg), nil
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
	msg.Text = &text
	msg.UpdatedAt = time.Now()
	if err := s.store.Messages.Update(ctx, msg); err != nil {
		return Message{}, err
	}
	return s.mapMessage(ctx, *msg), nil
}

func (s *Service) mapChat(ctx context.Context, chat models.Chat, viewerUserAccountID int64) Chat {
	title := html.EscapeString(chat.Title)
	if chat.Type == models.PrivateChat {
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

func (s *Service) mapMessage(ctx context.Context, message models.Message) Message {
	text := message.Text
	if text != nil {
		escaped := html.EscapeString(html.UnescapeString(*text))
		text = &escaped
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
		IsActive:        message.IsActive,
		CreatedAt:       message.CreatedAt,
		UpdatedAt:       message.UpdatedAt,
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
