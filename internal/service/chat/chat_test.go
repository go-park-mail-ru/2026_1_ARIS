package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	servicemocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestChatServiceGetChatByIDAndMembers(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chatRepo := repomocks.NewMockChatRepo(ctrl)
	memRepo := repomocks.NewMockChatMemberRepo(ctrl)
	svc := NewChatService(chatRepo, memRepo, nil)

	want := &models.Chat{ID: 5, Title: "t"}
	chatRepo.EXPECT().GetByID(gomock.Any(), int64(5)).Return(want, nil)
	got, err := svc.GetChatByID(context.Background(), 5)
	require.NoError(t, err)
	require.Equal(t, want, got)

	members := []models.ChatMember{{ID: 1, ChatID: 5}}
	memRepo.EXPECT().GetByChatID(gomock.Any(), int64(5)).Return(members, nil)
	gotM, err := svc.GetChatMembers(context.Background(), 5)
	require.NoError(t, err)
	require.Equal(t, members, gotM)
}

func TestChatServiceGetUserChats(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chatRepo := repomocks.NewMockChatRepo(ctrl)
	memRepo := repomocks.NewMockChatMemberRepo(ctrl)
	userSvc := servicemocks.NewMockUserService(ctrl)
	svc := NewChatService(chatRepo, memRepo, userSvc)

	userSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(7)).Return(&models.UserProfile{ProfileID: 100}, nil)
	memRepo.EXPECT().GetByUserID(gomock.Any(), int64(100)).Return([]models.ChatMember{{ChatID: 1}}, nil)
	chatRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&models.Chat{ID: 1, Title: "c"}, nil)

	chats, err := svc.GetUserChats(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, chats, 1)
	require.Equal(t, "c", chats[0].Title)
}

func TestChatServiceCheckUserInChat(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chatRepo := repomocks.NewMockChatRepo(ctrl)
	memRepo := repomocks.NewMockChatMemberRepo(ctrl)
	userSvc := servicemocks.NewMockUserService(ctrl)
	svc := NewChatService(chatRepo, memRepo, userSvc)

	userSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(9)).Return(&models.UserProfile{ProfileID: 50}, nil)
	memRepo.EXPECT().GetByChatID(gomock.Any(), int64(3)).Return([]models.ChatMember{{MemberID: 50}}, nil)
	ok, err := svc.CheckUserInChat(context.Background(), 3, 9)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestChatServiceCreatePrivateChatNew(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chatRepo := repomocks.NewMockChatRepo(ctrl)
	memRepo := repomocks.NewMockChatMemberRepo(ctrl)
	userSvc := servicemocks.NewMockUserService(ctrl)
	svc := NewChatService(chatRepo, memRepo, userSvc)

	userSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(1)).Return(&models.UserProfile{ProfileID: 10}, nil)
	up2 := &models.UserProfile{ProfileID: 20, FirstName: "B", LastName: "B"}
	userSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(2)).Return(up2, nil).Times(2)
	memRepo.EXPECT().GetByUserID(gomock.Any(), int64(10)).Return(nil, nil)

	chatRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c *models.Chat) error {
		c.ID = 99
		return nil
	})
	memRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	got, err := svc.CreatePrivateChat(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, int64(99), got.ID)
	require.Equal(t, models.PrivateChat, got.Type)
}

func TestChatServiceCreatePrivateChatReturnsExisting(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chatRepo := repomocks.NewMockChatRepo(ctrl)
	memRepo := repomocks.NewMockChatMemberRepo(ctrl)
	userSvc := servicemocks.NewMockUserService(ctrl)
	svc := NewChatService(chatRepo, memRepo, userSvc)

	userSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(1)).Return(&models.UserProfile{ProfileID: 10}, nil)
	userSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(2)).Return(&models.UserProfile{ProfileID: 20}, nil)

	existing := &models.Chat{ID: 5, Type: models.PrivateChat, Title: "old"}
	memRepo.EXPECT().GetByUserID(gomock.Any(), int64(10)).Return([]models.ChatMember{{ChatID: 5}}, nil)
	chatRepo.EXPECT().GetByID(gomock.Any(), int64(5)).Return(existing, nil)
	memRepo.EXPECT().GetByChatID(gomock.Any(), int64(5)).Return([]models.ChatMember{{MemberID: 10}, {MemberID: 20}}, nil)

	got, err := svc.CreatePrivateChat(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, existing, got)
}

func TestChatServiceErrors(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chatRepo := repomocks.NewMockChatRepo(ctrl)
	memRepo := repomocks.NewMockChatMemberRepo(ctrl)
	userSvc := servicemocks.NewMockUserService(ctrl)
	svc := NewChatService(chatRepo, memRepo, userSvc)

	userSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(1)).Return(nil, errors.New("no profile"))
	_, err := svc.GetUserChats(context.Background(), 1)
	require.Error(t, err)
}
