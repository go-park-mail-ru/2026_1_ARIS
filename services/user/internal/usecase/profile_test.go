package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GetProfileMe
// ---------------------------------------------------------------------------

func TestGetProfileMe_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	_, err := svc.GetProfileMe(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetProfileMe_NegativeInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	_, err := svc.GetProfileMe(context.Background(), -5)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetProfileMe_ProfileByAccountNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(nil, repository.ErrProfileNotFound)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	_, err := svc.GetProfileMe(context.Background(), 1)
	require.ErrorIs(t, err, ErrProfileNotFound)
}

func TestGetProfileMe_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	const accountID int64 = 7
	const profileID int64 = 42
	const upID int64 = 99

	// GetProfileMe calls Profiles.GetByUserAccountID first to get profileID
	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.Profile{ID: profileID}, nil)

	// Then calls GetProfileByID internally:
	// 1. UserProfiles.GetByProfileID
	userProfiles.EXPECT().
		GetByProfileID(gomock.Any(), profileID).
		Return(&model.UserProfile{
			ID:            upID,
			UserAccountID: accountID,
			ProfileID:     profileID,
			FirstName:     "Anna",
			LastName:      "Petrov",
			BirthdayDate:  time.Date(1995, 6, 15, 0, 0, 0, 0, time.UTC),
			Gender:        model.Female,
		}, nil)

	// 2. Accounts.Get
	accounts.EXPECT().
		Get(gomock.Any(), accountID).
		Return(&model.UserAccount{ID: accountID, Username: "annap"}, nil)

	// 3. Profiles.Get (by profileID, not by accountID)
	profiles.EXPECT().
		Get(gomock.Any(), profileID).
		Return(&model.Profile{ID: profileID}, nil)

	svc := New(newTestStore(accounts, profiles, userProfiles, nil, nil))

	details, err := svc.GetProfileMe(context.Background(), accountID)
	require.NoError(t, err)
	require.NotNil(t, details)
	require.Equal(t, profileID, details.ProfileID)
	require.Equal(t, "annap", details.Username)
	require.Equal(t, "Anna", details.FirstName)
}

// ---------------------------------------------------------------------------
// GetProfileByID
// ---------------------------------------------------------------------------

func TestGetProfileByID_InvalidInput_Zero(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, nil, repomocks.NewMockUserProfileRepo(ctrl), nil, nil))

	_, err := svc.GetProfileByID(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetProfileByID_InvalidInput_Negative(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, nil, repomocks.NewMockUserProfileRepo(ctrl), nil, nil))

	_, err := svc.GetProfileByID(context.Background(), -1)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetProfileByID_UserProfileNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	userProfiles.EXPECT().
		GetByProfileID(gomock.Any(), int64(100)).
		Return(nil, repository.ErrUserProfileNotFound)

	svc := New(newTestStore(nil, nil, userProfiles, nil, nil))

	_, err := svc.GetProfileByID(context.Background(), 100)
	require.ErrorIs(t, err, ErrUserProfileNotFound)
}

func TestGetProfileByID_AccountNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	const profileID int64 = 100
	const accountID int64 = 5

	userProfiles.EXPECT().
		GetByProfileID(gomock.Any(), profileID).
		Return(&model.UserProfile{
			ID:            1,
			UserAccountID: accountID,
			ProfileID:     profileID,
			FirstName:     "Bob",
			LastName:      "Green",
		}, nil)

	accounts.EXPECT().
		Get(gomock.Any(), accountID).
		Return(nil, repository.ErrUserAccountNotFound)

	svc := New(newTestStore(accounts, nil, userProfiles, nil, nil))

	_, err := svc.GetProfileByID(context.Background(), profileID)
	require.ErrorIs(t, err, ErrUserAccountNotFound)
}

func TestGetProfileByID_ProfileNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	const profileID int64 = 100
	const accountID int64 = 5

	userProfiles.EXPECT().
		GetByProfileID(gomock.Any(), profileID).
		Return(&model.UserProfile{
			ID:            1,
			UserAccountID: accountID,
			ProfileID:     profileID,
			FirstName:     "Bob",
			LastName:      "Green",
		}, nil)

	accounts.EXPECT().
		Get(gomock.Any(), accountID).
		Return(&model.UserAccount{ID: accountID, Username: "bobg"}, nil)

	profiles.EXPECT().
		Get(gomock.Any(), profileID).
		Return(nil, repository.ErrProfileNotFound)

	svc := New(newTestStore(accounts, profiles, userProfiles, nil, nil))

	_, err := svc.GetProfileByID(context.Background(), profileID)
	require.ErrorIs(t, err, ErrProfileNotFound)
}

func TestGetProfileByID_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	const profileID int64 = 77
	const accountID int64 = 33
	const upID int64 = 55

	userProfiles.EXPECT().
		GetByProfileID(gomock.Any(), profileID).
		Return(&model.UserProfile{
			ID:            upID,
			UserAccountID: accountID,
			ProfileID:     profileID,
			FirstName:     "Ivan",
			LastName:      "Ivanov",
			BirthdayDate:  time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			Gender:        model.Male,
		}, nil)

	accounts.EXPECT().
		Get(gomock.Any(), accountID).
		Return(&model.UserAccount{ID: accountID, Username: "ivanivanov"}, nil)

	profiles.EXPECT().
		Get(gomock.Any(), profileID).
		Return(&model.Profile{ID: profileID}, nil)

	svc := New(newTestStore(accounts, profiles, userProfiles, nil, nil))

	details, err := svc.GetProfileByID(context.Background(), profileID)
	require.NoError(t, err)
	require.NotNil(t, details)
	require.Equal(t, profileID, details.ProfileID)
	require.Equal(t, "ivanivanov", details.Username)
	require.Equal(t, "Ivan", details.FirstName)
	require.Equal(t, "Ivanov", details.LastName)
	require.Equal(t, accountID, details.UserAccountID)
}

// ---------------------------------------------------------------------------
// GetSettings
// ---------------------------------------------------------------------------

func TestGetSettings_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, nil, nil, nil, repomocks.NewMockSettingsRepo(ctrl)))

	_, err := svc.GetSettings(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetSettings_NotFound_ReturnsDefaults(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settings := repomocks.NewMockSettingsRepo(ctrl)

	const accountID int64 = 15

	settings.EXPECT().
		GetByUserID(gomock.Any(), accountID).
		Return(nil, repository.ErrSettingsNotFound)

	svc := New(newTestStore(nil, nil, nil, nil, settings))

	result, err := svc.GetSettings(context.Background(), accountID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, accountID, result.UserAccountID)
	require.Equal(t, model.LanguageRU, result.Language)
	require.Equal(t, model.ThemeLight, result.Theme)
}

func TestGetSettings_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settings := repomocks.NewMockSettingsRepo(ctrl)

	const accountID int64 = 15

	expected := &model.UserSettings{
		UserAccountID: accountID,
		Language:      model.LanguageEN,
		Theme:         model.ThemeDark,
	}

	settings.EXPECT().
		GetByUserID(gomock.Any(), accountID).
		Return(expected, nil)

	svc := New(newTestStore(nil, nil, nil, nil, settings))

	result, err := svc.GetSettings(context.Background(), accountID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, model.LanguageEN, result.Language)
	require.Equal(t, model.ThemeDark, result.Theme)
}

func TestGetSettings_RepoError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settings := repomocks.NewMockSettingsRepo(ctrl)
	boom := errors.New("db error")

	settings.EXPECT().
		GetByUserID(gomock.Any(), int64(10)).
		Return(nil, boom)

	svc := New(newTestStore(nil, nil, nil, nil, settings))

	_, err := svc.GetSettings(context.Background(), 10)
	require.ErrorIs(t, err, boom)
}

// ---------------------------------------------------------------------------
// UpdateSettings
// ---------------------------------------------------------------------------

func TestUpdateSettings_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, nil, nil, nil, repomocks.NewMockSettingsRepo(ctrl)))

	_, err := svc.UpdateSettings(context.Background(), 0, repository.SettingsUpdate{})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdateSettings_EmptyUpdate_ReturnsCurrentSettings(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settings := repomocks.NewMockSettingsRepo(ctrl)

	const accountID int64 = 8

	// Empty update → falls back to GetSettings
	settings.EXPECT().
		GetByUserID(gomock.Any(), accountID).
		Return(&model.UserSettings{
			UserAccountID: accountID,
			Language:      model.LanguageRU,
			Theme:         model.ThemeLight,
		}, nil)

	svc := New(newTestStore(nil, nil, nil, nil, settings))

	result, err := svc.UpdateSettings(context.Background(), accountID, repository.SettingsUpdate{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, model.LanguageRU, result.Language)
}

func TestUpdateSettings_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settings := repomocks.NewMockSettingsRepo(ctrl)

	const accountID int64 = 8
	lang := model.LanguageEN
	update := repository.SettingsUpdate{Language: &lang}

	expected := &model.UserSettings{
		UserAccountID: accountID,
		Language:      model.LanguageEN,
		Theme:         model.ThemeLight,
	}

	settings.EXPECT().
		Update(gomock.Any(), accountID, update).
		Return(expected, nil)

	svc := New(newTestStore(nil, nil, nil, nil, settings))

	result, err := svc.UpdateSettings(context.Background(), accountID, update)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, model.LanguageEN, result.Language)
}

func TestUpdateSettings_NotFound_ReturnsDefaultWithUpdate(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settings := repomocks.NewMockSettingsRepo(ctrl)

	const accountID int64 = 8
	theme := model.ThemeDark
	update := repository.SettingsUpdate{Theme: &theme}

	settings.EXPECT().
		Update(gomock.Any(), accountID, update).
		Return(nil, repository.ErrSettingsNotFound)

	svc := New(newTestStore(nil, nil, nil, nil, settings))

	result, err := svc.UpdateSettings(context.Background(), accountID, update)
	require.NoError(t, err)
	require.NotNil(t, result)
	// default settings with theme applied
	require.Equal(t, model.ThemeDark, result.Theme)
	require.Equal(t, model.LanguageRU, result.Language)
}

// ---------------------------------------------------------------------------
// GetFriends
// ---------------------------------------------------------------------------

func TestGetFriends_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	_, err := svc.GetFriends(context.Background(), 0, model.FriendshipAccepted)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetFriends_InvalidStatus(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	_, err := svc.GetFriends(context.Background(), 1, model.FriendshipStatus("unknown"))
	require.ErrorIs(t, err, ErrInvalidStatus)
}

func TestGetFriends_Success_Accepted(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	const accountID int64 = 1
	const profileID int64 = 10

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.Profile{ID: profileID}, nil)

	expectedFriends := []model.Friend{
		{ProfileID: 20, FirstName: "Alice", LastName: "W", Username: "alice", Status: model.FriendshipAccepted},
		{ProfileID: 21, FirstName: "Bob", LastName: "K", Username: "bobk", Status: model.FriendshipAccepted},
	}
	friendships.EXPECT().
		GetFriends(gomock.Any(), profileID, model.FriendshipAccepted).
		Return(expectedFriends, nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	friends, err := svc.GetFriends(context.Background(), accountID, model.FriendshipAccepted)
	require.NoError(t, err)
	require.Len(t, friends, 2)
	require.Equal(t, int64(20), friends[0].ProfileID)
}

func TestGetFriends_Success_Pending(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	const accountID int64 = 3
	const profileID int64 = 30

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.Profile{ID: profileID}, nil)

	friendships.EXPECT().
		GetFriends(gomock.Any(), profileID, model.FriendshipPending).
		Return([]model.Friend{{ProfileID: 50, FirstName: "C", LastName: "D", Username: "cd", Status: model.FriendshipPending}}, nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	friends, err := svc.GetFriends(context.Background(), accountID, model.FriendshipPending)
	require.NoError(t, err)
	require.Len(t, friends, 1)
}

// ---------------------------------------------------------------------------
// GetIncomingFriendRequests
// ---------------------------------------------------------------------------

func TestGetIncomingFriendRequests_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	_, err := svc.GetIncomingFriendRequests(context.Background(), 0, string(model.FriendshipPending))
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetIncomingFriendRequests_InvalidStatus(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	_, err := svc.GetIncomingFriendRequests(context.Background(), 1, "invalid")
	require.ErrorIs(t, err, ErrInvalidStatus)
}

func TestGetIncomingFriendRequests_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	const accountID int64 = 2
	const profileID int64 = 20

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.Profile{ID: profileID}, nil)

	incoming := []model.Friend{
		{ProfileID: 5, FirstName: "Eve", LastName: "Z", Username: "evez", Status: model.FriendshipPending},
	}
	friendships.EXPECT().
		GetIncomingFriends(gomock.Any(), profileID, string(model.FriendshipPending)).
		Return(incoming, nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	result, err := svc.GetIncomingFriendRequests(context.Background(), accountID, string(model.FriendshipPending))
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, int64(5), result[0].ProfileID)
}

// ---------------------------------------------------------------------------
// GetOutgoingFriendRequests
// ---------------------------------------------------------------------------

func TestGetOutgoingFriendRequests_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	_, err := svc.GetOutgoingFriendRequests(context.Background(), 0, string(model.FriendshipPending))
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetOutgoingFriendRequests_InvalidStatus(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	_, err := svc.GetOutgoingFriendRequests(context.Background(), 1, "bogus")
	require.ErrorIs(t, err, ErrInvalidStatus)
}

func TestGetOutgoingFriendRequests_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	const accountID int64 = 4
	const profileID int64 = 40

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.Profile{ID: profileID}, nil)

	outgoing := []model.Friend{
		{ProfileID: 99, FirstName: "Max", LastName: "N", Username: "maxn", Status: model.FriendshipPending},
	}
	friendships.EXPECT().
		GetOutgoingFriends(gomock.Any(), profileID, string(model.FriendshipPending)).
		Return(outgoing, nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	result, err := svc.GetOutgoingFriendRequests(context.Background(), accountID, string(model.FriendshipPending))
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, int64(99), result[0].ProfileID)
}

// ---------------------------------------------------------------------------
// RevokeFriendRequest
// ---------------------------------------------------------------------------

func TestRevokeFriendRequest_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	err := svc.RevokeFriendRequest(context.Background(), 0, 5)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRevokeFriendRequest_InvalidInput_ZeroAddressee(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.RevokeFriendRequest(context.Background(), 1, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRevokeFriendRequest_SelfRevoke(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.RevokeFriendRequest(context.Background(), 1, 10)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRevokeFriendRequest_NotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	friendships.EXPECT().
		RevokeFriendRequest(gomock.Any(), int64(10), int64(20)).
		Return(repository.ErrNoRowsAffected)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.RevokeFriendRequest(context.Background(), 1, 20)
	require.ErrorIs(t, err, ErrFriendshipNotExists)
}

func TestRevokeFriendRequest_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	friendships.EXPECT().
		RevokeFriendRequest(gomock.Any(), int64(10), int64(20)).
		Return(nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.RevokeFriendRequest(context.Background(), 1, 20)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// DeclineFriendRequest
// ---------------------------------------------------------------------------

func TestDeclineFriendRequest_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	err := svc.DeclineFriendRequest(context.Background(), 0, 5)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeclineFriendRequest_InvalidInput_ZeroRequester(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.DeclineFriendRequest(context.Background(), 1, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeclineFriendRequest_SelfDecline(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.DeclineFriendRequest(context.Background(), 1, 10)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeclineFriendRequest_NoPendingRequest(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	// checkFriendshipBy(requesterID=30, profileID=10) → not found
	friendships.EXPECT().
		GetFriendshipStatusBy(gomock.Any(), int64(30), int64(10)).
		Return("", repository.ErrFriendshipNotFound)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.DeclineFriendRequest(context.Background(), 1, 30)
	require.ErrorIs(t, err, ErrFriendshipNotExists)
}

func TestDeclineFriendRequest_NotPendingStatus(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	// status is accepted, not pending
	friendships.EXPECT().
		GetFriendshipStatusBy(gomock.Any(), int64(30), int64(10)).
		Return(string(model.FriendshipAccepted), nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.DeclineFriendRequest(context.Background(), 1, 30)
	require.ErrorIs(t, err, ErrFriendshipNotExists)
}

func TestDeclineFriendRequest_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	friendships.EXPECT().
		GetFriendshipStatusBy(gomock.Any(), int64(30), int64(10)).
		Return(string(model.FriendshipPending), nil)

	friendships.EXPECT().
		DeclineFriendship(gomock.Any(), int64(30), int64(10)).
		Return(nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.DeclineFriendRequest(context.Background(), 1, 30)
	require.NoError(t, err)
}
