package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// newChatStore is a helper that builds a repository.Store from optional individual mocks.
func newChatStore(
	chats *repomocks.MockChatRepo,
	members *repomocks.MockChatMemberRepo,
	messages *repomocks.MockMessageRepo,
	media *repomocks.MockMessageMediaRepo,
	stickers *repomocks.MockStickerRepo,
	reactions *repomocks.MockReactionRepo,
) repository.Store {
	s := repository.Store{}
	if chats != nil {
		s.Chats = chats
	}
	if members != nil {
		s.ChatMembers = members
	}
	if messages != nil {
		s.Messages = messages
	}
	if media != nil {
		s.MessageMedia = media
	}
	if stickers != nil {
		s.Stickers = stickers
	}
	if reactions != nil {
		s.Reactions = reactions
	}
	return s
}

// expectProfileByAccount sets up the mock user client to return the given profileID for accountID.
func expectProfileByAccount(client *usermock.MockUserServiceClient, accountID, profileID int64) {
	client.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: accountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
}

// expectProfileByAccountErr sets up the mock user client to return an error for accountID.
func expectProfileByAccountErr(client *usermock.MockUserServiceClient, accountID int64, err error) {
	client.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: accountID}).
		Return(nil, err)
}

// ---------------------------------------------------------------------------
// Presence methods (no gRPC required)
// ---------------------------------------------------------------------------

func TestSetPresenceOnline_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	err := svc.SetPresenceOnline(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSetPresenceOnline_NilPresence(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	err := svc.SetPresenceOnline(context.Background(), 5)
	require.NoError(t, err) // nil presence → no-op
}

func TestSetPresenceOffline_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	err := svc.SetPresenceOffline(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSetPresenceOffline_NilPresence(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	err := svc.SetPresenceOffline(context.Background(), 7)
	require.NoError(t, err)
}

func TestForcePresenceOffline_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	err := svc.ForcePresenceOffline(context.Background(), -1)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestHeartbeatPresence_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	err := svc.HeartbeatPresence(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

// ---------------------------------------------------------------------------
// CheckUserInChat
// ---------------------------------------------------------------------------

func TestCheckUserInChat_InvalidInput_ZeroChatID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.CheckUserInChat(context.Background(), 0, 1)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCheckUserInChat_InvalidInput_ZeroAccountID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.CheckUserInChat(context.Background(), 1, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCheckUserInChat_UserNotInChat(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)

	const accountID int64 = 3
	const profileID int64 = 30
	const chatID int64 = 100

	expectProfileByAccount(userClient, accountID, profileID)

	members.EXPECT().
		GetByChatID(gomock.Any(), chatID).
		Return([]model.ChatMember{
			{MemberID: 999}, // different member
		}, nil)

	svc := New(newChatStore(nil, members, nil, nil, nil, nil), userClient)

	inChat, err := svc.CheckUserInChat(context.Background(), chatID, accountID)
	require.NoError(t, err)
	require.False(t, inChat)
}

func TestCheckUserInChat_UserInChat(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)

	const accountID int64 = 3
	const profileID int64 = 30
	const chatID int64 = 100

	expectProfileByAccount(userClient, accountID, profileID)

	members.EXPECT().
		GetByChatID(gomock.Any(), chatID).
		Return([]model.ChatMember{
			{MemberID: profileID},
		}, nil)

	svc := New(newChatStore(nil, members, nil, nil, nil, nil), userClient)

	inChat, err := svc.CheckUserInChat(context.Background(), chatID, accountID)
	require.NoError(t, err)
	require.True(t, inChat)
}

// ---------------------------------------------------------------------------
// GetMessages
// ---------------------------------------------------------------------------

func TestGetMessages_UserNotInChat_Forbidden(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)

	const accountID int64 = 5
	const profileID int64 = 50
	const chatID int64 = 200

	// CheckUserInChat → GetByChatID returns members without profileID
	expectProfileByAccount(userClient, accountID, profileID)
	members.EXPECT().
		GetByChatID(gomock.Any(), chatID).
		Return([]model.ChatMember{{MemberID: 999}}, nil)

	svc := New(newChatStore(nil, members, nil, nil, nil, nil), userClient)

	_, err := svc.GetMessages(context.Background(), accountID, chatID, 10, 0)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestGetMessages_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)
	messages := repomocks.NewMockMessageRepo(ctrl)
	media := repomocks.NewMockMessageMediaRepo(ctrl)
	reactions := repomocks.NewMockReactionRepo(ctrl)

	const accountID int64 = 6
	const profileID int64 = 60
	const chatID int64 = 300

	now := time.Now()

	// CheckUserInChat call
	expectProfileByAccount(userClient, accountID, profileID)
	members.EXPECT().
		GetByChatID(gomock.Any(), chatID).
		Return([]model.ChatMember{{MemberID: profileID}}, nil)

	// Second profileIDByAccount call (inside GetMessages after checking membership)
	expectProfileByAccount(userClient, accountID, profileID)

	text := "hello"
	msgs := []model.Message{
		{
			ID:        1,
			Uid:       uuid.New(),
			Text:      &text,
			ChatID:    chatID,
			AuthorID:  profileID,
			Type:      model.MessageTypeText,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	messages.EXPECT().
		GetByChatID(gomock.Any(), chatID, 10, 0).
		Return(msgs, nil)

	media.EXPECT().
		GetByMessageIDs(gomock.Any(), []int64{1}).
		Return(map[int64][]model.MessageMedia{}, nil)

	reactions.EXPECT().
		GetSummaryByMessageIDs(gomock.Any(), []int64{1}).
		Return(map[int64][]model.ReactionSummary{}, nil)

	reactions.EXPECT().
		GetUserReactionsByMessageIDs(gomock.Any(), []int64{1}, profileID).
		Return(map[int64]string{}, nil)

	// mapMessage calls displayName → profileSummary for the author
	userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{FirstName: "Ivan", LastName: "I"}, nil)

	svc := New(newChatStore(nil, members, messages, media, nil, reactions), userClient)

	result, err := svc.GetMessages(context.Background(), accountID, chatID, 10, 0)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, int64(1), result[0].ID)
}

// ---------------------------------------------------------------------------
// GetUserChats
// ---------------------------------------------------------------------------

func TestGetUserChats_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)

	// userAccountID <= 0 → profileIDByAccount returns ErrInvalidInput
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), userClient)

	_, err := svc.GetUserChats(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetUserChats_NoChats(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	memberRepo := repomocks.NewMockChatMemberRepo(ctrl)

	const accountID int64 = 8
	const profileID int64 = 80

	expectProfileByAccount(userClient, accountID, profileID)

	memberRepo.EXPECT().
		GetByUserID(gomock.Any(), profileID).
		Return([]model.ChatMember{}, nil)

	svc := New(newChatStore(nil, memberRepo, nil, nil, nil, nil), userClient)

	chats, err := svc.GetUserChats(context.Background(), accountID)
	require.NoError(t, err)
	require.Empty(t, chats)
}

// ---------------------------------------------------------------------------
// CreatePrivateChat
// ---------------------------------------------------------------------------

func TestCreatePrivateChat_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.CreatePrivateChat(context.Background(), 0, 5)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreatePrivateChat_InvalidInput_ZeroOther(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.CreatePrivateChat(context.Background(), 1, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreatePrivateChat_SelfChat(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)

	const accountID int64 = 5
	const profileID int64 = 50

	// Both calls resolve to same accountID
	expectProfileByAccount(userClient, accountID, profileID)

	// resolveTarget for other=accountID: tries profileSummary first (returns nil since no mock), then profileIDByAccount
	userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: accountID}).
		Return(nil, errors.New("not found"))

	expectProfileByAccount(userClient, accountID, profileID)

	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), userClient)

	_, err := svc.CreatePrivateChat(context.Background(), accountID, accountID)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreatePrivateChat_NewChatSuccess(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	chats := repomocks.NewMockChatRepo(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)

	const accountID int64 = 20
	const profileID int64 = 200
	const otherAccountID int64 = 30
	const otherProfileID int64 = 300
	const chatID int64 = 1000

	expectProfileByAccount(userClient, accountID, profileID)
	userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: otherProfileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: otherProfileID, UserAccountId: otherAccountID, FirstName: "Trinity", LastName: "Matrix", Username: "trinity"}, nil).
		AnyTimes()
	members.EXPECT().GetByUserID(gomock.Any(), profileID).Return(nil, nil)
	chats.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, chat *model.Chat) error {
		chat.ID = chatID
		return nil
	})
	members.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	expectProfileByAccount(userClient, accountID, profileID)
	members.EXPECT().GetByChatID(gomock.Any(), chatID).Return([]model.ChatMember{{ChatID: chatID, MemberID: profileID}, {ChatID: chatID, MemberID: otherProfileID}}, nil)

	svc := New(newChatStore(chats, members, nil, nil, nil, nil), userClient)
	result, err := svc.CreatePrivateChat(context.Background(), accountID, otherProfileID)

	require.NoError(t, err)
	require.Equal(t, chatID, result.ID)
	require.Equal(t, "Trinity Matrix", result.Title)
	require.NotNil(t, result.InterlocutorProfileID)
	require.Equal(t, otherProfileID, *result.InterlocutorProfileID)
}

// ---------------------------------------------------------------------------
// SendMessage – input validation
// ---------------------------------------------------------------------------

func TestSendMessage_InvalidInput_ZeroChatID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.SendMessage(context.Background(), 1, 0, MessageInput{Text: "hi"})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSendMessage_InvalidInput_ZeroAccountID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.SendMessage(context.Background(), 0, 1, MessageInput{Text: "hi"})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSendMessage_EmptyText_NoAttachments(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.SendMessage(context.Background(), 1, 1, MessageInput{})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSendMessage_NotInChat_Forbidden(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)

	const accountID int64 = 9
	const profileID int64 = 90
	const chatID int64 = 400

	// CheckUserInChat calls profileIDByAccount
	expectProfileByAccount(userClient, accountID, profileID)
	members.EXPECT().
		GetByChatID(gomock.Any(), chatID).
		Return([]model.ChatMember{{MemberID: 999}}, nil)

	svc := New(newChatStore(nil, members, nil, nil, nil, nil), userClient)

	_, err := svc.SendMessage(context.Background(), accountID, chatID, MessageInput{Text: "hello world"})
	require.ErrorIs(t, err, ErrForbidden)
}

func TestSendMessage_TextSuccess(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)
	messages := repomocks.NewMockMessageRepo(ctrl)
	media := repomocks.NewMockMessageMediaRepo(ctrl)
	reactions := repomocks.NewMockReactionRepo(ctrl)

	const accountID int64 = 14
	const profileID int64 = 140
	const chatID int64 = 800
	const msgID int64 = 4001

	expectProfileByAccount(userClient, accountID, profileID)
	members.EXPECT().GetByChatID(gomock.Any(), chatID).Return([]model.ChatMember{{MemberID: profileID}}, nil)
	expectProfileByAccount(userClient, accountID, profileID)
	messages.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, msg *model.Message) error {
		msg.ID = msgID
		msg.Uid = uuid.New()
		msg.CreatedAt = time.Now()
		msg.UpdatedAt = msg.CreatedAt
		return nil
	})
	media.EXPECT().GetByMessageIDs(gomock.Any(), []int64{msgID}).Return(map[int64][]model.MessageMedia{}, nil)
	reactions.EXPECT().GetSummaryByMessageIDs(gomock.Any(), []int64{msgID}).Return(map[int64][]model.ReactionSummary{}, nil)
	reactions.EXPECT().GetUserReactionsByMessageIDs(gomock.Any(), []int64{msgID}, profileID).Return(map[int64]string{}, nil)
	userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{FirstName: "Neo", LastName: "Anderson"}, nil)

	svc := New(newChatStore(nil, members, messages, media, nil, reactions), userClient)
	result, err := svc.SendMessage(context.Background(), accountID, chatID, MessageInput{Text: "  hello world  "})

	require.NoError(t, err)
	require.Equal(t, msgID, result.ID)
	require.NotNil(t, result.Text)
	require.Equal(t, "hello world", *result.Text)
}

// ---------------------------------------------------------------------------
// UpdateMessage – input validation
// ---------------------------------------------------------------------------

func TestUpdateMessage_InvalidInput_ZeroChatID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.UpdateMessage(context.Background(), 1, 0, 1, "updated text")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdateMessage_InvalidInput_ZeroMessageID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.UpdateMessage(context.Background(), 1, 1, 0, "updated text")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdateMessage_InvalidInput_EmptyText(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.UpdateMessage(context.Background(), 1, 1, 1, "   ")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdateMessage_MessageNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	msgRepo := repomocks.NewMockMessageRepo(ctrl)

	const accountID int64 = 11
	const profileID int64 = 110
	const chatID int64 = 500
	const msgID int64 = 1001

	expectProfileByAccount(userClient, accountID, profileID)

	msgRepo.EXPECT().
		GetByID(gomock.Any(), msgID).
		Return(nil, errors.New("not found"))

	svc := New(newChatStore(nil, nil, msgRepo, nil, nil, nil), userClient)

	_, err := svc.UpdateMessage(context.Background(), accountID, chatID, msgID, "new text")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateMessage_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)
	messages := repomocks.NewMockMessageRepo(ctrl)
	media := repomocks.NewMockMessageMediaRepo(ctrl)
	reactions := repomocks.NewMockReactionRepo(ctrl)

	const accountID int64 = 15
	const profileID int64 = 150
	const chatID int64 = 900
	const msgID int64 = 5001

	oldText := "old"
	msg := &model.Message{ID: msgID, Uid: uuid.New(), Text: &oldText, ChatID: chatID, AuthorID: profileID, Type: model.MessageTypeText, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	expectProfileByAccount(userClient, accountID, profileID)
	messages.EXPECT().GetByID(gomock.Any(), msgID).Return(msg, nil)
	expectProfileByAccount(userClient, accountID, profileID)
	members.EXPECT().GetByChatID(gomock.Any(), chatID).Return([]model.ChatMember{{MemberID: profileID}}, nil)
	messages.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, updated *model.Message) error {
		require.NotNil(t, updated.Text)
		require.Equal(t, "new text", *updated.Text)
		return nil
	})
	media.EXPECT().GetByMessageIDs(gomock.Any(), []int64{msgID}).Return(map[int64][]model.MessageMedia{}, nil)
	reactions.EXPECT().GetSummaryByMessageIDs(gomock.Any(), []int64{msgID}).Return(map[int64][]model.ReactionSummary{}, nil)
	reactions.EXPECT().GetUserReactionsByMessageIDs(gomock.Any(), []int64{msgID}, profileID).Return(map[int64]string{}, nil)
	userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{FirstName: "Neo", LastName: "Anderson"}, nil)

	svc := New(newChatStore(nil, members, messages, media, nil, reactions), userClient)
	result, err := svc.UpdateMessage(context.Background(), accountID, chatID, msgID, "  new text  ")

	require.NoError(t, err)
	require.NotNil(t, result.Text)
	require.Equal(t, "new text", *result.Text)
}

// ---------------------------------------------------------------------------
// SetMessageReaction – input validation
// ---------------------------------------------------------------------------

func TestSetMessageReaction_InvalidInput_ZeroAccountID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.SetMessageReaction(context.Background(), 0, 1, 1, "👍")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSetMessageReaction_InvalidInput_EmptyReaction(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	// empty/unknown reaction → normalizeInputReaction returns ""
	_, err := svc.SetMessageReaction(context.Background(), 1, 1, 1, "unknown_reaction")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSetMessageReaction_InvalidInput_ZeroChatID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.SetMessageReaction(context.Background(), 1, 0, 1, "👍")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSetMessageReaction_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)
	messages := repomocks.NewMockMessageRepo(ctrl)
	media := repomocks.NewMockMessageMediaRepo(ctrl)
	reactions := repomocks.NewMockReactionRepo(ctrl)

	const accountID int64 = 12
	const profileID int64 = 120
	const chatID int64 = 600
	const msgID int64 = 2001

	text := "hello"
	msg := &model.Message{ID: msgID, Uid: uuid.New(), Text: &text, ChatID: chatID, AuthorID: profileID, Type: model.MessageTypeText, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	expectProfileByAccount(userClient, accountID, profileID)
	messages.EXPECT().GetByID(gomock.Any(), msgID).Return(msg, nil)
	expectProfileByAccount(userClient, accountID, profileID)
	members.EXPECT().GetByChatID(gomock.Any(), chatID).Return([]model.ChatMember{{MemberID: profileID}}, nil)
	reactions.EXPECT().Upsert(gomock.Any(), msgID, profileID, "👍").Return(nil)
	media.EXPECT().GetByMessageIDs(gomock.Any(), []int64{msgID}).Return(map[int64][]model.MessageMedia{}, nil)
	reactions.EXPECT().GetSummaryByMessageIDs(gomock.Any(), []int64{msgID}).Return(map[int64][]model.ReactionSummary{msgID: {{Type: "👍", Count: 1}}}, nil)
	reactions.EXPECT().GetUserReactionsByMessageIDs(gomock.Any(), []int64{msgID}, profileID).Return(map[int64]string{msgID: "👍"}, nil)
	userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{FirstName: "Neo", LastName: "Anderson"}, nil)

	svc := New(newChatStore(nil, members, messages, media, nil, reactions), userClient)
	result, err := svc.SetMessageReaction(context.Background(), accountID, chatID, msgID, "like")

	require.NoError(t, err)
	require.Equal(t, msgID, result.ID)
	require.NotNil(t, result.MyReaction)
	require.Equal(t, "👍", *result.MyReaction)
}

// ---------------------------------------------------------------------------
// DeleteMessageReaction – input validation
// ---------------------------------------------------------------------------

func TestDeleteMessageReaction_InvalidInput_ZeroAccountID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.DeleteMessageReaction(context.Background(), 0, 1, 1)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeleteMessageReaction_InvalidInput_ZeroChatID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.DeleteMessageReaction(context.Background(), 1, 0, 1)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeleteMessageReaction_InvalidInput_ZeroMessageID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.DeleteMessageReaction(context.Background(), 1, 1, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeleteMessageReaction_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)
	messages := repomocks.NewMockMessageRepo(ctrl)
	media := repomocks.NewMockMessageMediaRepo(ctrl)
	reactions := repomocks.NewMockReactionRepo(ctrl)

	const accountID int64 = 13
	const profileID int64 = 130
	const chatID int64 = 700
	const msgID int64 = 3001

	text := "hello"
	msg := &model.Message{ID: msgID, Uid: uuid.New(), Text: &text, ChatID: chatID, AuthorID: profileID, Type: model.MessageTypeText, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	expectProfileByAccount(userClient, accountID, profileID)
	messages.EXPECT().GetByID(gomock.Any(), msgID).Return(msg, nil)
	expectProfileByAccount(userClient, accountID, profileID)
	members.EXPECT().GetByChatID(gomock.Any(), chatID).Return([]model.ChatMember{{MemberID: profileID}}, nil)
	reactions.EXPECT().Delete(gomock.Any(), msgID, profileID).Return(nil)
	media.EXPECT().GetByMessageIDs(gomock.Any(), []int64{msgID}).Return(map[int64][]model.MessageMedia{}, nil)
	reactions.EXPECT().GetSummaryByMessageIDs(gomock.Any(), []int64{msgID}).Return(map[int64][]model.ReactionSummary{}, nil)
	reactions.EXPECT().GetUserReactionsByMessageIDs(gomock.Any(), []int64{msgID}, profileID).Return(map[int64]string{}, nil)
	userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{FirstName: "Neo", LastName: "Anderson"}, nil)

	svc := New(newChatStore(nil, members, messages, media, nil, reactions), userClient)
	result, err := svc.DeleteMessageReaction(context.Background(), accountID, chatID, msgID)

	require.NoError(t, err)
	require.Equal(t, msgID, result.ID)
	require.Nil(t, result.MyReaction)
}

// ---------------------------------------------------------------------------
// GetStickerPacks – input validation
// ---------------------------------------------------------------------------

func TestGetStickerPacks_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.GetStickerPacks(context.Background(), 0, "", false, 10, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

// ---------------------------------------------------------------------------
// CreateStickerPack – input validation
// ---------------------------------------------------------------------------

func TestCreateStickerPack_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.CreateStickerPack(context.Background(), 0, StickerPackInput{Title: "cats"})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreateStickerPack_InvalidInput_EmptyTitle(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.CreateStickerPack(context.Background(), 1, StickerPackInput{Title: ""})
	require.ErrorIs(t, err, ErrInvalidInput)
}

// ---------------------------------------------------------------------------
// GetStickersByPack – input validation
// ---------------------------------------------------------------------------

func TestGetStickersByPack_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.GetStickersByPack(context.Background(), 0, 1, 10, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetStickersByPack_InvalidInput_ZeroPackID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newChatStore(nil, nil, nil, nil, nil, nil), usermock.NewMockUserServiceClient(ctrl))

	_, err := svc.GetStickersByPack(context.Background(), 1, 0, 10, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

// ---------------------------------------------------------------------------
// Normalise helper functions (pure, no mocks needed)
// ---------------------------------------------------------------------------

func TestNormalizeInputReaction_KnownValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"👍", "👍"},
		{"like", "👍"},
		{"❤️", "❤️"},
		{"love", "❤️"},
		{"😂", "😂"},
		{"laugh", "😂"},
		{"happy", "😂"},
		{"😢", "😢"},
		{"sad", "😢"},
		{"😡", "😡"},
		{"angry", "😡"},
		{"anger", "😡"},
	}
	for _, tt := range tests {
		got := normalizeInputReaction(tt.input)
		if got != tt.want {
			t.Errorf("normalizeInputReaction(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeInputReaction_Unknown(t *testing.T) {
	t.Parallel()
	if got := normalizeInputReaction("not-a-reaction"); got != "" {
		t.Errorf("expected empty string for unknown reaction, got %q", got)
	}
}

func TestNormalizeStoredReaction_LegacyMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{`\like`, "👍"},
		{"like", "👍"},
		{`\dislike`, "😢"},
		{"dislike", "😢"},
		{`\happy`, "😂"},
		{"happy", "😂"},
		{`\anger`, "😡"},
		{"anger", "😡"},
		{"👍", "👍"}, // passthrough
	}
	for _, tt := range tests {
		got := normalizeStoredReaction(tt.input)
		if got != tt.want {
			t.Errorf("normalizeStoredReaction(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeListBounds_Defaults(t *testing.T) {
	t.Parallel()
	limit, offset := normalizeListBounds(0, -5)
	if limit != 50 {
		t.Errorf("expected default limit 50, got %d", limit)
	}
	if offset != 0 {
		t.Errorf("expected offset 0, got %d", offset)
	}
}

func TestNormalizeListBounds_TooLarge(t *testing.T) {
	t.Parallel()
	limit, offset := normalizeListBounds(200, 10)
	if limit != 50 {
		t.Errorf("expected capped limit 50, got %d", limit)
	}
	if offset != 10 {
		t.Errorf("expected offset 10, got %d", offset)
	}
}

func TestNormalizeListBounds_Valid(t *testing.T) {
	t.Parallel()
	limit, offset := normalizeListBounds(25, 5)
	if limit != 25 {
		t.Errorf("expected limit 25, got %d", limit)
	}
	if offset != 5 {
		t.Errorf("expected offset 5, got %d", offset)
	}
}

func TestAppendAttachmentInputs_NoFiles(t *testing.T) {
	t.Parallel()
	media := []AttachmentInput{{MediaID: 1}, {MediaID: 2}}
	result := appendAttachmentInputs(media, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(result))
	}
}

func TestAppendAttachmentInputs_WithFiles(t *testing.T) {
	t.Parallel()
	media := []AttachmentInput{{MediaID: 1}}
	files := []AttachmentInput{{MediaID: 2}, {MediaID: 3}}
	result := appendAttachmentInputs(media, files)
	if len(result) != 3 {
		t.Errorf("expected 3 attachments, got %d", len(result))
	}
}

func TestIsAllowedVideoNoteMime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		mime string
		want bool
	}{
		{"video/webm", true},
		{"video/mp4", true},
		{"video/quicktime", true},
		{"VIDEO/MP4", true}, // case-insensitive
		{"audio/mp4", false},
		{"image/png", false},
		{"application/pdf", false},
	}
	for _, tt := range tests {
		got := isAllowedVideoNoteMime(tt.mime)
		if got != tt.want {
			t.Errorf("isAllowedVideoNoteMime(%q) = %v, want %v", tt.mime, got, tt.want)
		}
	}
}

func TestMapReactionSummaries(t *testing.T) {
	t.Parallel()
	items := []model.ReactionSummary{
		{Type: "like", Count: 3},
		{Type: `\happy`, Count: 1},
	}
	result := mapReactionSummaries(items)
	if len(result) != 2 {
		t.Fatalf("expected 2 reactions, got %d", len(result))
	}
	if result[0].Type != "👍" {
		t.Errorf("expected 👍, got %q", result[0].Type)
	}
	if result[1].Type != "😂" {
		t.Errorf("expected 😂, got %q", result[1].Type)
	}
	if result[0].Count != 3 {
		t.Errorf("expected count 3, got %d", result[0].Count)
	}
}

// ---------------------------------------------------------------------------
// ToStatus
// ---------------------------------------------------------------------------

func TestToStatus_ErrInvalidInput(t *testing.T) {
	t.Parallel()
	err := ToStatus(ErrInvalidInput)
	require.Error(t, err)
	require.Contains(t, err.Error(), "InvalidArgument")
}

func TestToStatus_ErrForbidden(t *testing.T) {
	t.Parallel()
	err := ToStatus(ErrForbidden)
	require.Error(t, err)
	require.Contains(t, err.Error(), "PermissionDenied")
}

func TestToStatus_ErrNotFound(t *testing.T) {
	t.Parallel()
	err := ToStatus(ErrNotFound)
	require.Error(t, err)
	require.Contains(t, err.Error(), "NotFound")
}

func TestToStatus_OtherError(t *testing.T) {
	t.Parallel()
	err := ToStatus(errors.New("some internal error"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Internal")
}
