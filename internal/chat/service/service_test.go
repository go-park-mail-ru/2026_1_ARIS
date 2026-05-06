package service

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	chatmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat/mock"
	membermock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat_member/mock"
	messagemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/message/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type chatMocks struct {
	chats      *chatmock.MockChatRepo
	members    *membermock.MockChatMemberRepo
	messages   *messagemock.MockMessageRepo
	userClient *usermock.MockUserServiceClient
	service    *Service
}

func newChatMocks(t *testing.T) (*gomock.Controller, chatMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := chatMocks{
		chats:      chatmock.NewMockChatRepo(ctrl),
		members:    membermock.NewMockChatMemberRepo(ctrl),
		messages:   messagemock.NewMockMessageRepo(ctrl),
		userClient: usermock.NewMockUserServiceClient(ctrl),
	}
	m.service = New(repository.NewStore(m.chats, m.members, m.messages), m.userClient)
	return ctrl, m
}

func expectProfileByAccount(m chatMocks, ctx context.Context, accountID, profileID int64) {
	m.userClient.EXPECT().
		GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: accountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
}

func expectSummary(m chatMocks, ctx context.Context, profileID, accountID int64, first, last, username string) {
	m.userClient.EXPECT().
		GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{
			ProfileId: profileID, UserAccountId: accountID, FirstName: first, LastName: last, Username: username,
		}, nil)
}

func TestGetUserChatsMapsPrivateTitle(t *testing.T) {
	ctrl, m := newChatMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	viewerAccountID := int64(10)
	viewerProfileID := int64(20)
	otherProfileID := int64(30)
	chatID := int64(5)

	expectProfileByAccount(m, ctx, viewerAccountID, viewerProfileID)
	m.members.EXPECT().GetByUserID(ctx, viewerProfileID).Return([]models.ChatMember{{ChatID: chatID, MemberID: viewerProfileID}}, nil)
	m.chats.EXPECT().GetByID(ctx, chatID).Return(&models.Chat{
		ID: chatID, Uid: uuid.New(), Type: models.PrivateChat, Title: "<private>", IsActive: true,
	}, nil)
	expectProfileByAccount(m, ctx, viewerAccountID, viewerProfileID)
	m.members.EXPECT().GetByChatID(ctx, chatID).Return([]models.ChatMember{
		{ChatID: chatID, MemberID: viewerProfileID},
		{ChatID: chatID, MemberID: otherProfileID},
	}, nil)
	expectSummary(m, ctx, otherProfileID, 11, "Trinity", "", "trinity")

	chats, err := m.service.GetUserChats(ctx, viewerAccountID)

	require.NoError(t, err)
	require.Len(t, chats, 1)
	require.Equal(t, "Trinity", chats[0].Title)
	require.Equal(t, models.PrivateChat, chats[0].Type)
}

func TestCreatePrivateChatCreatesMembersAndUsesExistingWhenFound(t *testing.T) {
	t.Run("creates new chat", func(t *testing.T) {
		ctrl, m := newChatMocks(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectProfileByAccount(m, ctx, 10, 20)
		expectSummary(m, ctx, 30, 11, "Trinity", "Zion", "trinity")
		m.members.EXPECT().GetByUserID(ctx, int64(20)).Return(nil, nil)
		expectSummary(m, ctx, 30, 11, "Trinity", "Zion", "trinity")
		m.chats.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, chat *models.Chat) error {
			require.Equal(t, models.PrivateChat, chat.Type)
			require.Equal(t, "Trinity Zion", chat.Title)
			chat.ID = 5
			return nil
		})
		m.members.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		m.members.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		expectProfileByAccount(m, ctx, 10, 20)
		m.members.EXPECT().GetByChatID(ctx, int64(5)).Return([]models.ChatMember{{MemberID: 20}, {MemberID: 30}}, nil)
		expectSummary(m, ctx, 30, 11, "Trinity", "Zion", "trinity")

		chat, err := m.service.CreatePrivateChat(ctx, 10, 30)

		require.NoError(t, err)
		require.Equal(t, int64(5), chat.ID)
		require.Equal(t, "Trinity Zion", chat.Title)
	})

	t.Run("returns existing private chat", func(t *testing.T) {
		ctrl, m := newChatMocks(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectProfileByAccount(m, ctx, 10, 20)
		expectSummary(m, ctx, 30, 11, "Trinity", "Zion", "trinity")
		m.members.EXPECT().GetByUserID(ctx, int64(20)).Return([]models.ChatMember{{ChatID: 5, MemberID: 20}}, nil)
		m.chats.EXPECT().GetByID(ctx, int64(5)).Return(&models.Chat{ID: 5, Uid: uuid.New(), Type: models.PrivateChat, Title: "old"}, nil)
		m.members.EXPECT().GetByChatID(ctx, int64(5)).Return([]models.ChatMember{{MemberID: 20}, {MemberID: 30}}, nil)
		expectProfileByAccount(m, ctx, 10, 20)
		m.members.EXPECT().GetByChatID(ctx, int64(5)).Return([]models.ChatMember{{MemberID: 20}, {MemberID: 30}}, nil)
		expectSummary(m, ctx, 30, 11, "Trinity", "Zion", "trinity")

		chat, err := m.service.CreatePrivateChat(ctx, 10, 30)

		require.NoError(t, err)
		require.Equal(t, int64(5), chat.ID)
		require.Equal(t, "Trinity Zion", chat.Title)
	})
}

func TestMessages(t *testing.T) {
	ctrl, m := newChatMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	text := "<hello>"
	updated := "safe"
	msg := &models.Message{ID: 50, Uid: uuid.New(), Text: &text, ChatID: 5, AuthorID: 20, IsActive: true}

	expectProfileByAccount(m, ctx, 10, 20)
	m.members.EXPECT().GetByChatID(ctx, int64(5)).Return([]models.ChatMember{{MemberID: 20}}, nil)
	m.messages.EXPECT().GetByChatID(ctx, int64(5), 50, 0).Return([]models.Message{*msg}, nil)
	expectSummary(m, ctx, 20, 10, "Neo", "Anderson", "neo")

	messages, err := m.service.GetMessages(ctx, 10, 5, 0, -1)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "&lt;hello&gt;", *messages[0].Text)
	require.Equal(t, "Neo Anderson", messages[0].AuthorName)

	expectProfileByAccount(m, ctx, 10, 20)
	m.members.EXPECT().GetByChatID(ctx, int64(5)).Return([]models.ChatMember{{MemberID: 20}}, nil)
	expectProfileByAccount(m, ctx, 10, 20)
	m.messages.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, saved *models.Message) error {
		require.Equal(t, "sent", *saved.Text)
		require.Equal(t, int64(5), saved.ChatID)
		require.Equal(t, int64(20), saved.AuthorID)
		saved.ID = 51
		return nil
	})
	expectSummary(m, ctx, 20, 10, "Neo", "", "neo")

	sent, err := m.service.SendMessage(ctx, 10, 5, " sent ")
	require.NoError(t, err)
	require.Equal(t, int64(51), sent.ID)

	expectProfileByAccount(m, ctx, 10, 20)
	m.messages.EXPECT().GetByID(ctx, int64(50)).Return(msg, nil)
	expectProfileByAccount(m, ctx, 10, 20)
	m.members.EXPECT().GetByChatID(ctx, int64(5)).Return([]models.ChatMember{{MemberID: 20}}, nil)
	m.messages.EXPECT().Update(ctx, msg).DoAndReturn(func(_ context.Context, got *models.Message) error {
		require.Equal(t, updated, *got.Text)
		return nil
	})
	expectSummary(m, ctx, 20, 10, "Neo", "", "neo")

	got, err := m.service.UpdateMessage(ctx, 10, 5, 50, "safe")
	require.NoError(t, err)
	require.Equal(t, updated, *got.Text)
}

func TestGetMessagesAfterFallsBackToPagedMessages(t *testing.T) {
	ctrl, m := newChatMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	text := "after"
	expectProfileByAccount(m, ctx, 10, 20)
	m.members.EXPECT().GetByChatID(ctx, int64(5)).Return([]models.ChatMember{{MemberID: 20}}, nil)
	expectProfileByAccount(m, ctx, 10, 20)
	m.members.EXPECT().GetByChatID(ctx, int64(5)).Return([]models.ChatMember{{MemberID: 20}}, nil)
	m.messages.EXPECT().GetByChatID(ctx, int64(5), 50, 0).Return([]models.Message{{
		ID: 1, Uid: uuid.New(), Text: &text, ChatID: 5, AuthorID: 20,
	}}, nil)
	expectSummary(m, ctx, 20, 10, "Neo", "", "neo")

	messages, err := m.service.GetMessagesAfter(ctx, 10, 5, -1, 0)

	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, text, *messages[0].Text)
}

func TestChatServiceErrorsAndStatus(t *testing.T) {
	ctrl, m := newChatMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	_, err := m.service.CreatePrivateChat(ctx, 0, 1)
	require.ErrorIs(t, err, ErrInvalidInput)

	m.userClient.EXPECT().
		GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: 10}).
		Return(nil, status.Error(codes.NotFound, "missing"))
	ok, err := m.service.CheckUserInChat(ctx, 5, 10)
	require.False(t, ok)
	require.ErrorIs(t, err, ErrNotFound)

	_, err = m.service.SendMessage(ctx, 10, 5, " ")
	require.ErrorIs(t, err, ErrInvalidInput)

	require.Equal(t, codes.InvalidArgument, status.Code(ToStatus(ErrInvalidInput)))
	require.Equal(t, codes.PermissionDenied, status.Code(ToStatus(ErrForbidden)))
	require.Equal(t, codes.NotFound, status.Code(ToStatus(ErrNotFound)))
	require.Equal(t, codes.Internal, status.Code(ToStatus(errors.New("boom"))))
}
