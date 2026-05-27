package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	repomock "github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

type vkidClientFunc func(context.Context, VKIDCallbackInput) (*VKIDUser, error)

func (f vkidClientFunc) ExchangeCode(ctx context.Context, in VKIDCallbackInput) (*VKIDUser, error) {
	return f(ctx, in)
}

func newAuthServiceWithMocks(t *testing.T) (*Service, *repomock.MockSessionRepo, *usermock.MockUserServiceClient, *mediamock.MockMediaServiceClient) {
	t.Helper()
	ctrl := gomock.NewController(t)
	sessions := repomock.NewMockSessionRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	media := mediamock.NewMockMediaServiceClient(ctrl)
	svc := New(sessions, users, media)
	svc.now = func() time.Time { return time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC) }
	return svc, sessions, users, media
}

func authUser(accountID int64) *userpb.AuthUserResponse {
	avatarID := int64(9)
	email := "neo@example.com"
	return &userpb.AuthUserResponse{
		UserAccountId: accountID,
		UserProfileId: 20,
		ProfileId:     30,
		Login:         "neo",
		Email:         &email,
		FirstName:     "Neo",
		LastName:      "Anderson",
		AvatarId:      &avatarID,
		CreatedAt:     "2026-05-27T12:00:00Z",
	}
}

func expectAuthResult(t *testing.T, sessions *repomock.MockSessionRepo, users *usermock.MockUserServiceClient, media *mediamock.MockMediaServiceClient, accountID int64) {
	t.Helper()
	sessions.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(model.Session{})).
		DoAndReturn(func(_ context.Context, session model.Session) error {
			require.NotEmpty(t, session.SessionID)
			require.Equal(t, accountID, session.UserID)
			require.Equal(t, SessionTTL, session.ExpiredAt.Sub(session.CreatedAt))
			return nil
		})
	users.EXPECT().
		GetAuthUserByAccount(gomock.Any(), &userpb.GetAuthUserByAccountRequest{UserAccountId: accountID}).
		Return(authUser(accountID), nil)
	media.EXPECT().
		GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: 9}).
		Return(&mediapb.GetMediaURLResponse{Url: "/media/avatar.png"}, nil)
}

func TestRegisterStepOneChecksAvailability(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		svc, _, users, _ := newAuthServiceWithMocks(t)
		users.EXPECT().
			CheckUsernameAvailable(gomock.Any(), &userpb.CheckUsernameAvailableRequest{Username: "neo"}).
			Return(&userpb.CheckUsernameAvailableResponse{Available: true}, nil)

		err := svc.RegisterStepOne(context.Background(), RegisterStepOneInput{Login: " Neo "})

		require.NoError(t, err)
	})

	t.Run("taken", func(t *testing.T) {
		svc, _, users, _ := newAuthServiceWithMocks(t)
		users.EXPECT().
			CheckUsernameAvailable(gomock.Any(), &userpb.CheckUsernameAvailableRequest{Username: "neo"}).
			Return(&userpb.CheckUsernameAvailableResponse{Available: false}, nil)

		err := svc.RegisterStepOne(context.Background(), RegisterStepOneInput{Login: "neo"})

		require.ErrorIs(t, err, ErrLoginAlreadyExists)
	})
}

func TestRegisterSuccess(t *testing.T) {
	svc, sessions, users, media := newAuthServiceWithMocks(t)
	users.EXPECT().
		CheckUsernameAvailable(gomock.Any(), &userpb.CheckUsernameAvailableRequest{Username: "neo"}).
		Return(&userpb.CheckUsernameAvailableResponse{Available: true}, nil)
	users.EXPECT().
		CreateAuthUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *userpb.CreateAuthUserRequest, _ ...grpc.CallOption) (*userpb.AuthUserResponse, error) {
			require.Equal(t, "neo", req.Username)
			require.Equal(t, "Neo", req.FirstName)
			require.Equal(t, "Anderson", req.LastName)
			require.Equal(t, "2010-05-27", req.Birthday)
			require.Equal(t, userpb.Gender_GENDER_MALE, req.Gender)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(req.PasswordHash), []byte("pass1234")))
			return &userpb.AuthUserResponse{UserAccountId: 42}, nil
		})
	expectAuthResult(t, sessions, users, media, 42)

	result, err := svc.Register(context.Background(), RegisterInput{
		FirstName: "Neo",
		LastName:  "Anderson",
		Login:     "Neo",
		Password1: "pass1234",
		Password2: "pass1234",
		Birthday:  "2010-05-27",
		Gender:    model.Male,
	})

	require.NoError(t, err)
	require.Equal(t, int64(42), result.User.UserAccountID)
	require.Equal(t, "/media/avatar.png", *result.User.AvatarURL)
}

func TestRegisterValidation(t *testing.T) {
	tests := []struct {
		name  string
		input RegisterInput
		err   error
	}{
		{name: "empty", input: RegisterInput{}, err: ErrInvalidInput},
		{name: "password mismatch", input: RegisterInput{FirstName: "A", LastName: "B", Login: "neo", Password1: "one", Password2: "two"}, err: ErrInvalidInput},
		{name: "invalid birthday", input: RegisterInput{FirstName: "A", LastName: "B", Login: "neo", Password1: "pass1234", Password2: "pass1234", Birthday: "bad"}, err: ErrInvalidBirthday},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, users, _ := newAuthServiceWithMocks(t)
			if tc.input.Login != "" && tc.input.Password1 == tc.input.Password2 {
				users.EXPECT().
					CheckUsernameAvailable(gomock.Any(), gomock.Any()).
					Return(&userpb.CheckUsernameAvailableResponse{Available: true}, nil)
			}

			result, err := svc.Register(context.Background(), tc.input)

			require.Nil(t, result)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestLoginSuccessAndInvalidCredentials(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, sessions, users, media := newAuthServiceWithMocks(t)
		hash := newPasswordHash(t, "pass1234")
		users.EXPECT().
			GetCredentialsByLogin(gomock.Any(), &userpb.GetCredentialsByLoginRequest{Login: "neo"}).
			Return(&userpb.GetCredentialsByLoginResponse{UserAccountId: 42, PasswordHash: hash}, nil)
		expectAuthResult(t, sessions, users, media, 42)

		result, err := svc.Login(context.Background(), LoginInput{Login: " Neo ", Password: "pass1234"})

		require.NoError(t, err)
		require.Equal(t, int64(42), result.User.UserAccountID)
	})

	t.Run("wrong password", func(t *testing.T) {
		svc, _, users, _ := newAuthServiceWithMocks(t)
		hash := newPasswordHash(t, "pass1234")
		users.EXPECT().
			GetCredentialsByLogin(gomock.Any(), &userpb.GetCredentialsByLoginRequest{Login: "neo"}).
			Return(&userpb.GetCredentialsByLoginResponse{UserAccountId: 42, PasswordHash: hash}, nil)

		result, err := svc.Login(context.Background(), LoginInput{Login: "neo", Password: "wrong"})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidCredentials)
	})
}

func TestValidateSessionAndGetMe(t *testing.T) {
	t.Run("expired session is deleted", func(t *testing.T) {
		svc, sessions, _, _ := newAuthServiceWithMocks(t)
		session := &model.Session{SessionID: "expired", UserID: 42, ExpiredAt: svc.now().Add(-time.Second)}
		sessions.EXPECT().GetByID(gomock.Any(), model.SessionID("expired")).Return(session, nil)
		sessions.EXPECT().Delete(gomock.Any(), model.SessionID("expired")).Return(nil)

		got, err := svc.ValidateSession(context.Background(), "expired")

		require.Nil(t, got)
		require.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("get me", func(t *testing.T) {
		svc, sessions, users, media := newAuthServiceWithMocks(t)
		session := &model.Session{SessionID: "active", UserID: 42, ExpiredAt: svc.now().Add(time.Hour)}
		sessions.EXPECT().GetByID(gomock.Any(), model.SessionID("active")).Return(session, nil)
		users.EXPECT().
			GetAuthUserByAccount(gomock.Any(), &userpb.GetAuthUserByAccountRequest{UserAccountId: 42}).
			Return(authUser(42), nil)
		media.EXPECT().
			GetMediaURL(gomock.Any(), gomock.Any()).
			Return(&mediapb.GetMediaURLResponse{Url: "/media/avatar.png"}, nil)

		user, err := svc.GetMe(context.Background(), "active")

		require.NoError(t, err)
		require.Equal(t, int64(42), user.UserAccountID)
	})
}

func TestLoginWithVKID(t *testing.T) {
	t.Run("missing client", func(t *testing.T) {
		svc, _, _, _ := newAuthServiceWithMocks(t)

		result, err := svc.LoginWithVKID(context.Background(), VKIDCallbackInput{Code: "c", CodeVerifier: "v", RedirectURI: "https://app/cb"})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrOAuthUnavailable)
	})

	t.Run("success", func(t *testing.T) {
		svc, sessions, users, media := newAuthServiceWithMocks(t)
		svc.vkid = vkidClientFunc(func(context.Context, VKIDCallbackInput) (*VKIDUser, error) {
			email := "neo@example.com"
			return &VKIDUser{ID: "123", FirstName: "Neo", LastName: "Anderson", Email: &email, Gender: userpb.Gender_GENDER_MALE}, nil
		})
		users.EXPECT().
			GetOrCreateOAuthUser(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *userpb.GetOrCreateOAuthUserRequest, _ ...grpc.CallOption) (*userpb.AuthUserResponse, error) {
				require.Equal(t, "vkid", req.Provider)
				require.Equal(t, "123", req.ProviderUserId)
				require.Equal(t, "vk123", req.Username)
				return &userpb.AuthUserResponse{UserAccountId: 42}, nil
			})
		expectAuthResult(t, sessions, users, media, 42)

		result, err := svc.LoginWithVKID(context.Background(), VKIDCallbackInput{Code: "c", CodeVerifier: "v", RedirectURI: "https://app/cb"})

		require.NoError(t, err)
		require.Equal(t, int64(42), result.User.UserAccountID)
	})

	t.Run("provider error", func(t *testing.T) {
		svc, _, _, _ := newAuthServiceWithMocks(t)
		svc.vkid = vkidClientFunc(func(context.Context, VKIDCallbackInput) (*VKIDUser, error) {
			return nil, errors.New("oauth")
		})

		result, err := svc.LoginWithVKID(context.Background(), VKIDCallbackInput{Code: "c", CodeVerifier: "v", RedirectURI: "https://app/cb"})

		require.Nil(t, result)
		require.EqualError(t, err, "oauth")
	})
}
