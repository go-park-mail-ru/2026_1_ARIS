package grpc

import (
	"context"
	"testing"

	chatpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/chat"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestChatGRPCCheckUserInChat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	users := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)
	svc := usecase.New(repository.Store{ChatMembers: members}, users)
	server := New(svc)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 5}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 10}, nil)
	members.EXPECT().
		GetByChatID(gomock.Any(), int64(1)).
		Return([]model.ChatMember{{ChatID: 1, MemberID: 10}}, nil)

	resp, err := server.CheckUserInChat(context.Background(), &chatpb.CheckUserInChatRequest{ChatId: 1, UserAccountId: 5})

	require.NoError(t, err)
	require.True(t, resp.GetOk())
}
