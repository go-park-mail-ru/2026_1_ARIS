package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	friendmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend/mock"
	profilemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile/mock"
	settingsmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/settings/mock"
	accountmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account/mock"
	userprofilemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/user/repository"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userMocks struct {
	accounts     *accountmock.MockUserAccountRepo
	profiles     *profilemock.MockProfileRepo
	userProfiles *userprofilemock.MockUserProfileRepo
	settings     *settingsmock.MockUserSettingsRepository
	friendships  *friendmock.MockFriendshipRepo
	media        *mediamock.MockMediaServiceClient
	service      *Service
}

func newUserMocks(t *testing.T) (*gomock.Controller, userMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := userMocks{
		accounts:     accountmock.NewMockUserAccountRepo(ctrl),
		profiles:     profilemock.NewMockProfileRepo(ctrl),
		userProfiles: userprofilemock.NewMockUserProfileRepo(ctrl),
		settings:     settingsmock.NewMockUserSettingsRepository(ctrl),
		friendships:  friendmock.NewMockFriendshipRepo(ctrl),
		media:        mediamock.NewMockMediaServiceClient(ctrl),
	}
	m.service = New(repository.NewStore(m.accounts, m.profiles, m.userProfiles, m.settings, m.friendships), m.media)
	return ctrl, m
}

func TestGetProfileByIDBuildsDetailsWithAvatar(t *testing.T) {
	ctrl, m := newUserMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	accountID := int64(10)
	profileID := int64(20)
	userProfileID := int64(30)
	avatarID := int64(40)
	email := "neo@example.test"
	phone := "+79990000000"
	bio := "hello"
	birthday := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)

	m.userProfiles.EXPECT().GetByProfileID(ctx, profileID).Return(&models.UserProfile{
		ID: userProfileID, UserAccountID: accountID, ProfileID: profileID,
		FirstName: "Neo", LastName: "Anderson", Bio: &bio, BirthdayDate: birthday, Gender: models.Male,
	}, nil)
	m.accounts.EXPECT().Get(ctx, accountID).Return(&models.UserAccount{
		ID: accountID, Username: "neo", Email: &email, Phone: &phone,
	}, nil)
	m.profiles.EXPECT().Get(ctx, profileID).Return(&models.Profile{ID: profileID, AvatarID: &avatarID}, nil)
	m.media.EXPECT().GetMediaURL(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *mediapb.GetMediaURLRequest, _ ...grpc.CallOption) (*mediapb.GetMediaURLResponse, error) {
		require.Equal(t, avatarID, req.GetMediaId())
		return &mediapb.GetMediaURLResponse{Url: "https://cdn.test/avatar.jpg"}, nil
	})

	details, err := m.service.GetProfileByID(ctx, profileID)

	require.NoError(t, err)
	require.Equal(t, profileID, details.ProfileID)
	require.Equal(t, userProfileID, details.UserProfileID)
	require.Equal(t, accountID, details.UserAccountID)
	require.Equal(t, "neo", details.Username)
	require.Equal(t, &email, details.Email)
	require.Equal(t, &phone, details.Phone)
	require.Equal(t, "https://cdn.test/avatar.jpg", *details.ImageLink)
}

func TestSimpleProfileAccessors(t *testing.T) {
	ctrl, m := newUserMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	accountID := int64(10)
	profileID := int64(20)
	userProfile := &models.UserProfile{ID: 30, UserAccountID: accountID, ProfileID: profileID}
	profile := &models.Profile{ID: profileID}
	account := &models.UserAccount{ID: accountID, Username: "neo"}

	m.profiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(profile, nil)
	gotProfile, err := m.service.GetProfileByUserAccount(ctx, accountID)
	require.NoError(t, err)
	require.Equal(t, profile, gotProfile)

	m.profiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(profile, nil)
	gotProfile, err = m.service.GetProfileByUserAccountID(ctx, accountID)
	require.NoError(t, err)
	require.Equal(t, profile, gotProfile)

	m.userProfiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(userProfile, nil)
	gotUserProfile, err := m.service.GetUserProfileByUserAccount(ctx, accountID)
	require.NoError(t, err)
	require.Equal(t, userProfile, gotUserProfile)

	m.userProfiles.EXPECT().GetByProfileID(ctx, profileID).Return(userProfile, nil)
	gotUserProfile, err = m.service.GetUserProfileByProfileID(ctx, profileID)
	require.NoError(t, err)
	require.Equal(t, userProfile, gotUserProfile)

	m.userProfiles.EXPECT().GetByProfileID(ctx, profileID).Return(userProfile, nil)
	m.accounts.EXPECT().Get(ctx, accountID).Return(account, nil)
	gotAccount, err := m.service.GetUserAccountByProfileID(ctx, profileID)
	require.NoError(t, err)
	require.Equal(t, account, gotAccount)
}

func TestGetProfileByIDErrors(t *testing.T) {
	t.Run("invalid profile id", func(t *testing.T) {
		_, m := newUserMocks(t)

		details, err := m.service.GetProfileByID(context.Background(), 0)

		require.Nil(t, details)
		require.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("normalizes user profile not found", func(t *testing.T) {
		ctrl, m := newUserMocks(t)
		defer ctrl.Finish()

		m.userProfiles.EXPECT().GetByProfileID(gomock.Any(), int64(1)).Return(nil, xerrors.UserProfileNotFound)

		details, err := m.service.GetProfileByID(context.Background(), 1)

		require.Nil(t, details)
		require.ErrorIs(t, err, ErrUserProfileNotFound)
	})
}

func TestUpdateMeUpdatesEveryMutableArea(t *testing.T) {
	ctrl, m := newUserMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	accountID := int64(10)
	profileID := int64(20)
	userProfileID := int64(30)
	avatarID := int64(40)
	removeAvatar := true
	username := "  Neo  "
	email := "neo@example.test"
	firstName := "Thomas"
	lastName := "Anderson"
	birthday := "2000-01-02"
	gender := models.Male

	m.userProfiles.EXPECT().GetByUserAccountID(ctx, accountID).Return(&models.UserProfile{
		ID: userProfileID, UserAccountID: accountID, ProfileID: profileID,
	}, nil)
	m.accounts.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, update dto.UpdateUserAccountDTO) error {
		require.Equal(t, accountID, update.ID)
		require.Equal(t, "neo", *update.Username)
		require.Equal(t, email, *update.Email)
		return nil
	})
	m.userProfiles.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, update dto.UpdateUserProfileDTO) error {
		require.Equal(t, userProfileID, update.ID)
		require.Equal(t, firstName, *update.FirstName)
		require.Equal(t, lastName, *update.LastName)
		require.Equal(t, gender, *update.Gender)
		require.Equal(t, time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), *update.BirthdayDate)
		return nil
	})
	m.profiles.EXPECT().UpdateAvatar(ctx, profileID, nil).Return(nil)
	m.profiles.EXPECT().UpdateAvatar(ctx, profileID, &avatarID).Return(nil)

	err := m.service.UpdateMe(ctx, accountID, dto.UpdateFullProfileRequestDTO{
		Username:     &username,
		Email:        &email,
		FirstName:    &firstName,
		LastName:     &lastName,
		BirthdayDate: &birthday,
		Gender:       &gender,
		AvatarID:     &avatarID,
		RemoveAvatar: &removeAvatar,
	})

	require.NoError(t, err)
}

func TestUpdateMeValidation(t *testing.T) {
	t.Run("invalid account id", func(t *testing.T) {
		_, m := newUserMocks(t)

		require.ErrorIs(t, m.service.UpdateMe(context.Background(), 0, dto.UpdateFullProfileRequestDTO{}), ErrInvalidInput)
	})

	t.Run("nothing to update", func(t *testing.T) {
		ctrl, m := newUserMocks(t)
		defer ctrl.Finish()

		m.userProfiles.EXPECT().GetByUserAccountID(gomock.Any(), int64(10)).Return(&models.UserProfile{ID: 1, ProfileID: 2}, nil)

		require.ErrorIs(t, m.service.UpdateMe(context.Background(), 10, dto.UpdateFullProfileRequestDTO{}), ErrNothingToUpdate)
	})

	t.Run("bad birthday", func(t *testing.T) {
		ctrl, m := newUserMocks(t)
		defer ctrl.Finish()

		bad := "bad"
		m.userProfiles.EXPECT().GetByUserAccountID(gomock.Any(), int64(10)).Return(&models.UserProfile{ID: 1, ProfileID: 2}, nil)

		require.ErrorIs(t, m.service.UpdateMe(context.Background(), 10, dto.UpdateFullProfileRequestDTO{BirthdayDate: &bad}), ErrInvalidInput)
	})
}

func TestCardsAndProfileCollections(t *testing.T) {
	ctrl, m := newUserMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	currentAccountID := int64(10)
	candidateAccountID := int64(11)
	currentProfileID := int64(20)
	candidateProfileID := int64(21)
	avatarID := int64(99)

	m.userProfiles.EXPECT().GetByUserAccountID(ctx, currentAccountID).Return(&models.UserProfile{
		UserAccountID: currentAccountID, ProfileID: currentProfileID,
	}, nil)
	m.profiles.EXPECT().GetAll(ctx).Return([]models.Profile{
		{ID: currentProfileID},
		{ID: candidateProfileID, AvatarID: &avatarID},
	}, nil)
	m.userProfiles.EXPECT().GetByProfileID(ctx, candidateProfileID).Return(&models.UserProfile{
		UserAccountID: candidateAccountID, ProfileID: candidateProfileID,
	}, nil)
	m.accounts.EXPECT().Get(ctx, candidateAccountID).Return(&models.UserAccount{ID: candidateAccountID, Username: "trinity"}, nil)
	m.userProfiles.EXPECT().GetByProfileID(ctx, candidateProfileID).Return(&models.UserProfile{
		UserAccountID: candidateAccountID, ProfileID: candidateProfileID, FirstName: "Trinity", LastName: "Zion",
	}, nil)
	m.accounts.EXPECT().Get(ctx, candidateAccountID).Return(&models.UserAccount{ID: candidateAccountID, Username: "trinity"}, nil)
	m.media.EXPECT().GetMediaURL(gomock.Any(), gomock.Any()).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/a.jpg"}, nil)

	cards, err := m.service.GetSuggestedUsers(ctx, currentAccountID)

	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, candidateProfileID, cards[0].ID)
	require.Equal(t, "Trinity", cards[0].FirstName)
	require.Equal(t, "https://cdn.test/a.jpg", cards[0].AvatarLink)
}

func TestPublicProfileCollections(t *testing.T) {
	ctrl, m := newUserMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	popularProfile := models.Profile{ID: 101}
	eventProfile := models.Profile{ID: 201}
	m.profiles.EXPECT().GetAll(ctx).Return([]models.Profile{popularProfile, eventProfile}, nil).Times(2)

	m.userProfiles.EXPECT().GetByProfileID(ctx, popularProfile.ID).Return(&models.UserProfile{
		UserAccountID: 1001, ProfileID: popularProfile.ID,
	}, nil)
	m.accounts.EXPECT().Get(ctx, int64(1001)).Return(&models.UserAccount{ID: 1001, Username: "sergeyshulginenko"}, nil)
	m.userProfiles.EXPECT().GetByProfileID(ctx, eventProfile.ID).Return(&models.UserProfile{
		UserAccountID: 2001, ProfileID: eventProfile.ID,
	}, nil)
	m.accounts.EXPECT().Get(ctx, int64(2001)).Return(&models.UserAccount{ID: 2001, Username: "sofiasitnichenko"}, nil)
	m.userProfiles.EXPECT().GetByProfileID(ctx, popularProfile.ID).Return(&models.UserProfile{
		UserAccountID: 1001, ProfileID: popularProfile.ID, FirstName: "Sergey", LastName: "S",
	}, nil)
	m.accounts.EXPECT().Get(ctx, int64(1001)).Return(&models.UserAccount{ID: 1001, Username: "sergeyshulginenko"}, nil)

	popular, err := m.service.GetPublicPopularUsers(ctx)
	require.NoError(t, err)
	require.Len(t, popular, 1)
	require.Equal(t, popularProfile.ID, popular[0].ID)

	m.userProfiles.EXPECT().GetByProfileID(ctx, popularProfile.ID).Return(&models.UserProfile{
		UserAccountID: 1001, ProfileID: popularProfile.ID,
	}, nil)
	m.accounts.EXPECT().Get(ctx, int64(1001)).Return(&models.UserAccount{ID: 1001, Username: "sergeyshulginenko"}, nil)
	m.userProfiles.EXPECT().GetByProfileID(ctx, eventProfile.ID).Return(&models.UserProfile{
		UserAccountID: 2001, ProfileID: eventProfile.ID,
	}, nil)
	m.accounts.EXPECT().Get(ctx, int64(2001)).Return(&models.UserAccount{ID: 2001, Username: "sofiasitnichenko"}, nil)
	m.userProfiles.EXPECT().GetByProfileID(ctx, eventProfile.ID).Return(&models.UserProfile{
		UserAccountID: 2001, ProfileID: eventProfile.ID, FirstName: "Sofia", LastName: "S",
	}, nil)
	m.accounts.EXPECT().Get(ctx, int64(2001)).Return(&models.UserAccount{ID: 2001, Username: "sofiasitnichenko"}, nil)

	events, err := m.service.GetLatestEvents(ctx)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, 1, events[0].Type)
}

func TestSettings(t *testing.T) {
	ctx := context.Background()

	t.Run("default when settings absent", func(t *testing.T) {
		ctrl, m := newUserMocks(t)
		defer ctrl.Finish()

		m.settings.EXPECT().GetByUserID(ctx, int64(10)).Return(nil, xerrors.ErrUserSettingsNotFound)

		settings, err := m.service.GetSettings(ctx, 10)

		require.NoError(t, err)
		require.Equal(t, int64(10), settings.UserAccountID)
		require.Equal(t, models.LanguageRU, settings.Language)
		require.Equal(t, models.ThemeLight, settings.Theme)
	})

	t.Run("empty update reads settings", func(t *testing.T) {
		ctrl, m := newUserMocks(t)
		defer ctrl.Finish()

		existing := &models.UserSettings{UserAccountID: 10, Language: models.LanguageEN, Theme: models.ThemeDark}
		m.settings.EXPECT().GetByUserID(ctx, int64(10)).Return(existing, nil)

		settings, err := m.service.UpdateSettings(ctx, 10, dto.UserSettingsUpdate{})

		require.NoError(t, err)
		require.Equal(t, existing, settings)
	})

	t.Run("applies update to defaults when row absent", func(t *testing.T) {
		ctrl, m := newUserMocks(t)
		defer ctrl.Finish()

		language := models.LanguageEN
		theme := models.ThemeDark
		update := dto.UserSettingsUpdate{Language: &language, Theme: &theme}
		m.settings.EXPECT().Update(ctx, int64(10), update).Return(nil, xerrors.ErrUserSettingsNotFound)

		settings, err := m.service.UpdateSettings(ctx, 10, update)

		require.NoError(t, err)
		require.Equal(t, models.LanguageEN, settings.Language)
		require.Equal(t, models.ThemeDark, settings.Theme)
	})
}

func TestToStatus(t *testing.T) {
	require.Equal(t, codes.InvalidArgument, status.Code(ToStatus(ErrInvalidInput)))
	require.Equal(t, codes.NotFound, status.Code(ToStatus(ErrProfileNotFound)))
	require.Equal(t, codes.Internal, status.Code(ToStatus(errors.New("boom"))))
}
