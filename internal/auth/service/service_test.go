package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	profilemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile/mock"
	sessionmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session/mock"
	accountmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account/mock"
	userprofilemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile/mock"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type authMocks struct {
	accounts     *accountmock.MockUserAccountRepo
	profiles     *profilemock.MockProfileRepo
	userProfiles *userprofilemock.MockUserProfileRepo
	sessions     *sessionmock.MockSessionRepo
	service      *Service
}

func newAuthMocks(t *testing.T) (*gomock.Controller, authMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := authMocks{
		accounts:     accountmock.NewMockUserAccountRepo(ctrl),
		profiles:     profilemock.NewMockProfileRepo(ctrl),
		userProfiles: userprofilemock.NewMockUserProfileRepo(ctrl),
		sessions:     sessionmock.NewMockSessionRepo(ctrl),
	}
	m.service = New(repository.NewStore(m.accounts, m.profiles, m.userProfiles, m.sessions))
	m.service.now = func() time.Time { return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC) }
	return ctrl, m
}

func TestRegisterCreatesAccountProfileUserProfileAndSession(t *testing.T) {
	ctrl, m := newAuthMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	accountID := int64(10)
	profileID := int64(20)
	birthday := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	avatarID := int64(99)
	email := "neo@example.test"
	created := time.Date(2026, 1, 1, 1, 2, 3, 0, time.UTC)

	m.accounts.EXPECT().GetByUsername(ctx, "neo").Return(nil, errors.New("not found"))
	m.accounts.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, account models.UserAccount) (int64, error) {
		require.Equal(t, "neo", account.Username)
		require.NotEmpty(t, account.PasswordHash)
		return accountID, nil
	})
	m.profiles.EXPECT().Save(ctx, gomock.Any()).Return(profileID, nil)
	m.userProfiles.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, profile models.UserProfile) (int64, error) {
		require.Equal(t, accountID, profile.UserAccountID)
		require.Equal(t, profileID, profile.ProfileID)
		require.Equal(t, "Neo", profile.FirstName)
		require.Equal(t, "Anderson", profile.LastName)
		require.Equal(t, birthday, profile.BirthdayDate)
		require.Equal(t, models.Female, profile.Gender)
		return int64(30), nil
	})
	m.sessions.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, session models.Session) error {
		require.Equal(t, accountID, session.UserID)
		require.Equal(t, m.service.now(), session.CreatedAt)
		require.Equal(t, m.service.now().Add(SessionTTL), session.ExpiredAt)
		return nil
	})
	m.accounts.EXPECT().Get(ctx, accountID).Return(&models.UserAccount{
		ID: accountID, Username: "neo", Email: &email, CreatedAt: created,
	}, nil)
	m.userProfiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(&models.UserProfile{
		ID: 30, UserAccountID: accountID, ProfileID: profileID, FirstName: "Neo", LastName: "Anderson", CreatedAt: created,
	}, nil)
	m.profiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(&models.Profile{ID: profileID, AvatarID: &avatarID}, nil)

	result, err := m.service.Register(ctx, RegisterInput{
		FirstName: "Neo",
		LastName:  "Anderson",
		Login:     "  NEO ",
		Password1: "chosen-password",
		Birthday:  "02/01/2000",
		Gender:    models.Gender("unknown"),
	})

	require.NoError(t, err)
	require.Equal(t, accountID, result.User.UserAccountID)
	require.Equal(t, profileID, result.User.ProfileID)
	require.Equal(t, "neo", result.User.Login)
	require.Nil(t, result.User.AvatarURL)
	require.Equal(t, accountID, result.Session.UserID)
}

func TestRegisterValidation(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		_, m := newAuthMocks(t)

		result, err := m.service.Register(context.Background(), RegisterInput{})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("login already exists", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()

		m.accounts.EXPECT().GetByUsername(gomock.Any(), "neo").Return(&models.UserAccount{ID: 1}, nil)

		result, err := m.service.Register(context.Background(), RegisterInput{
			FirstName: "Neo", LastName: "Anderson", Login: "neo", Password1: "secret",
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrLoginAlreadyExists)
	})

	t.Run("invalid birthday", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()

		m.accounts.EXPECT().GetByUsername(gomock.Any(), "neo").Return(nil, errors.New("not found"))

		result, err := m.service.Register(context.Background(), RegisterInput{
			FirstName: "Neo", LastName: "Anderson", Login: "neo", Password1: "secret", Birthday: "bad",
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidBirthday)
	})

	t.Run("too young", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()

		m.accounts.EXPECT().GetByUsername(gomock.Any(), "neo").Return(nil, errors.New("not found"))

		result, err := m.service.Register(context.Background(), RegisterInput{
			FirstName: "Neo", LastName: "Anderson", Login: "neo", Password1: "secret", Birthday: "01/01/2020",
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrTooYoung)
	})
}

func TestRegisterStepOne(t *testing.T) {
	t.Run("free login", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()

		m.accounts.EXPECT().GetByUsername(gomock.Any(), "trinity").Return(nil, errors.New("not found"))

		require.NoError(t, m.service.RegisterStepOne(context.Background(), RegisterStepOneInput{Login: " Trinity "}))
	})

	t.Run("taken login", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()

		m.accounts.EXPECT().GetByUsername(gomock.Any(), "trinity").Return(&models.UserAccount{ID: 1}, nil)

		require.ErrorIs(t, m.service.RegisterStepOne(context.Background(), RegisterStepOneInput{Login: "trinity"}), ErrLoginAlreadyExists)
	})
}

func TestLoginIssuesSession(t *testing.T) {
	ctrl, m := newAuthMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	accountID := int64(11)
	profileID := int64(22)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	m.accounts.EXPECT().GetByUsername(ctx, "neo").Return(&models.UserAccount{
		ID: accountID, Username: "neo", PasswordHash: string(hash),
	}, nil)
	m.sessions.EXPECT().Save(ctx, gomock.Any()).Return(nil)
	m.accounts.EXPECT().Get(ctx, accountID).Return(&models.UserAccount{ID: accountID, Username: "neo"}, nil)
	m.userProfiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(&models.UserProfile{
		UserAccountID: accountID, ProfileID: profileID, FirstName: "Neo", LastName: "Anderson",
	}, nil)
	m.profiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(&models.Profile{ID: profileID}, nil)

	result, err := m.service.Login(ctx, LoginInput{Login: " NEO ", Password: "secret"})

	require.NoError(t, err)
	require.Equal(t, accountID, result.User.UserAccountID)
	require.Equal(t, profileID, result.User.ProfileID)
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	t.Run("unknown login", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()

		m.accounts.EXPECT().GetByUsername(gomock.Any(), "neo").Return(nil, errors.New("not found"))

		result, err := m.service.Login(context.Background(), LoginInput{Login: "neo", Password: "secret"})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("wrong password", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()

		hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
		require.NoError(t, err)
		m.accounts.EXPECT().GetByUsername(gomock.Any(), "neo").Return(&models.UserAccount{PasswordHash: string(hash)}, nil)

		result, err := m.service.Login(context.Background(), LoginInput{Login: "neo", Password: "wrong"})

		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidCredentials)
	})
}

func TestSessions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	t.Run("valid session", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()
		m.service.now = func() time.Time { return now }

		session := &models.Session{SessionID: "sid", UserID: 7, ExpiredAt: now.Add(time.Hour)}
		m.sessions.EXPECT().GetByID(ctx, models.SessionID("sid")).Return(session, nil)

		got, err := m.service.ValidateSession(ctx, "sid")

		require.NoError(t, err)
		require.Equal(t, session, got)
	})

	t.Run("expired session is removed", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()
		m.service.now = func() time.Time { return now }

		m.sessions.EXPECT().GetByID(ctx, models.SessionID("sid")).Return(&models.Session{SessionID: "sid", ExpiredAt: now.Add(-time.Second)}, nil)
		m.sessions.EXPECT().Delete(ctx, models.SessionID("sid")).Return(nil)

		got, err := m.service.ValidateSession(ctx, "sid")

		require.Nil(t, got)
		require.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("blank session", func(t *testing.T) {
		_, m := newAuthMocks(t)

		got, err := m.service.ValidateSession(ctx, " ")

		require.Nil(t, got)
		require.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("logout ignores blank and deletes nonblank", func(t *testing.T) {
		ctrl, m := newAuthMocks(t)
		defer ctrl.Finish()

		require.NoError(t, m.service.Logout(ctx, " "))
		m.sessions.EXPECT().Delete(ctx, models.SessionID("sid")).Return(nil)
		require.NoError(t, m.service.Logout(ctx, "sid"))
	})
}

func TestGetMeReturnsCurrentUserWithAvatarURL(t *testing.T) {
	ctrl, m := newAuthMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	accountID := int64(10)
	profileID := int64(20)
	avatarID := int64(30)
	email := "neo@example.test"
	mediaClient := mediamock.NewMockMediaServiceClient(ctrl)
	m.service.mediaClient = mediaClient
	m.service.now = func() time.Time { return now }

	m.sessions.EXPECT().GetByID(ctx, models.SessionID("sid")).Return(&models.Session{
		SessionID: "sid",
		UserID:    accountID,
		ExpiredAt: now.Add(time.Hour),
	}, nil)
	m.accounts.EXPECT().Get(ctx, accountID).Return(&models.UserAccount{
		ID: accountID, Username: "neo", Email: &email,
	}, nil)
	m.userProfiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(&models.UserProfile{
		UserAccountID: accountID, ProfileID: profileID, FirstName: "Neo", LastName: "Anderson",
	}, nil)
	m.profiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(&models.Profile{ID: profileID, AvatarID: &avatarID}, nil)
	mediaClient.EXPECT().GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: avatarID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/a.png"}, nil)

	user, err := m.service.GetMe(ctx, "sid")

	require.NoError(t, err)
	require.Equal(t, accountID, user.UserAccountID)
	require.Equal(t, profileID, user.ProfileID)
	require.Equal(t, "https://cdn.test/a.png", *user.AvatarURL)
}
