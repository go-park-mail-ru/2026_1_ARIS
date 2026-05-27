package grpc

import (
	"context"
	"testing"
	"time"

	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	repomock "github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthGRPCRegisterAndValidate(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	sessions := repomock.NewMockSessionRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	media := mediamock.NewMockMediaServiceClient(ctrl)
	server := New(usecase.New(sessions, users, media))

	users.EXPECT().
		CheckUsernameAvailable(gomock.Any(), &userpb.CheckUsernameAvailableRequest{Username: "neo"}).
		Return(&userpb.CheckUsernameAvailableResponse{Available: true}, nil)
	users.EXPECT().
		CreateAuthUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *userpb.CreateAuthUserRequest, _ ...grpc.CallOption) (*userpb.AuthUserResponse, error) {
			require.Equal(t, "neo", req.Username)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(req.PasswordHash), []byte("pass1234")))
			return &userpb.AuthUserResponse{UserAccountId: 42}, nil
		})
	sessions.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(model.Session{})).Return(nil)
	users.EXPECT().
		GetAuthUserByAccount(gomock.Any(), &userpb.GetAuthUserByAccountRequest{UserAccountId: 42}).
		Return(authGRPCUser(42), nil)
	media.EXPECT().
		GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: 9}).
		Return(&mediapb.GetMediaURLResponse{Url: "/media/avatar.png"}, nil)

	resp, err := server.Register(context.Background(), &authpb.RegisterRequest{
		FirstName: "Neo", LastName: "Anderson", Login: "Neo",
		Password1: "pass1234", Password2: "pass1234", Birthday: "2010-05-27", Gender: authpb.Gender_GENDER_MALE,
	})

	require.NoError(t, err)
	require.Equal(t, int64(42), resp.GetUser().GetUserAccountId())

	sessions.EXPECT().
		GetByID(gomock.Any(), model.SessionID("active")).
		Return(&model.Session{SessionID: "active", UserID: 42, ExpiredAt: time.Now().Add(time.Hour)}, nil)

	validateResp, err := server.ValidateSession(context.Background(), &authpb.ValidateSessionRequest{SessionId: "active"})

	require.NoError(t, err)
	require.Equal(t, int64(42), validateResp.GetUserAccountId())
}

func TestAuthGRPCStatusMapping(t *testing.T) {
	t.Parallel()

	require.Equal(t, model.Male, fromProtoGender(authpb.Gender_GENDER_MALE))
	require.Equal(t, model.Female, fromProtoGender(authpb.Gender_GENDER_FEMALE))
	require.Equal(t, codes.AlreadyExists, status.Code(toStatus(usecase.ErrLoginAlreadyExists)))
	require.Equal(t, codes.Unauthenticated, status.Code(toStatus(usecase.ErrSessionNotFound)))
	require.Equal(t, codes.InvalidArgument, status.Code(toStatus(usecase.ErrInvalidInput)))
	require.Equal(t, codes.Internal, status.Code(toStatus(context.Canceled)))
}

func authGRPCUser(accountID int64) *userpb.AuthUserResponse {
	avatarID := int64(9)
	return &userpb.AuthUserResponse{
		UserAccountId: accountID,
		UserProfileId: 20,
		ProfileId:     30,
		FirstName:     "Neo",
		LastName:      "Anderson",
		AvatarId:      &avatarID,
		CreatedAt:     "2026-05-27T12:00:00Z",
	}
}
