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

// newStore is a helper to build a Store from individual mock repos.
// Any field left nil is valid – the test must only set the repos it exercises.
func newTestStore(
	accounts *repomocks.MockAccountRepo,
	profiles *repomocks.MockProfileRepo,
	userProfiles *repomocks.MockUserProfileRepo,
	friendships *repomocks.MockFriendshipRepo,
	settings *repomocks.MockSettingsRepo,
) repository.Store {
	store := repository.Store{}
	if accounts != nil {
		store.Accounts = accounts
	}
	if profiles != nil {
		store.Profiles = profiles
	}
	if userProfiles != nil {
		store.UserProfiles = userProfiles
	}
	if friendships != nil {
		store.Friendships = friendships
	}
	if settings != nil {
		store.Settings = settings
	}
	return store
}

// ---------------------------------------------------------------------------
// CheckUsernameAvailable
// ---------------------------------------------------------------------------

func TestCheckUsernameAvailable_EmptyUsername(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(repomocks.NewMockAccountRepo(ctrl), nil, nil, nil, nil))

	ok, err := svc.CheckUsernameAvailable(context.Background(), "   ")
	require.ErrorIs(t, err, ErrInvalidInput)
	require.False(t, ok)
}

func TestCheckUsernameAvailable_UsernameTaken(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	accounts.EXPECT().
		GetByUsername(gomock.Any(), "takenuser").
		Return(&model.UserAccount{ID: 1, Username: "takenuser"}, nil)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	ok, err := svc.CheckUsernameAvailable(context.Background(), "takenuser")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCheckUsernameAvailable_UsernameAvailable(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	accounts.EXPECT().
		GetByUsername(gomock.Any(), "freeuser").
		Return(nil, repository.ErrUserAccountNotFound)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	ok, err := svc.CheckUsernameAvailable(context.Background(), "freeuser")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCheckUsernameAvailable_RepoError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)
	boom := errors.New("db error")

	accounts.EXPECT().
		GetByUsername(gomock.Any(), "someuser").
		Return(nil, boom)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	ok, err := svc.CheckUsernameAvailable(context.Background(), "someuser")
	require.ErrorIs(t, err, boom)
	require.False(t, ok)
}

// ---------------------------------------------------------------------------
// CreateAuthUser
// ---------------------------------------------------------------------------

func TestCreateAuthUser_InvalidInput_EmptyUsername(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(repomocks.NewMockAccountRepo(ctrl), nil, nil, nil, nil))

	_, err := svc.CreateAuthUser(context.Background(), CreateAuthUserInput{
		Username:     "",
		PasswordHash: "hash",
		FirstName:    "Ivan",
		LastName:     "Petrov",
		Birthday:     "2000-01-01",
	})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreateAuthUser_InvalidInput_EmptyPasswordHash(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(repomocks.NewMockAccountRepo(ctrl), nil, nil, nil, nil))

	_, err := svc.CreateAuthUser(context.Background(), CreateAuthUserInput{
		Username:     "newuser",
		PasswordHash: "",
		FirstName:    "Ivan",
		LastName:     "Petrov",
		Birthday:     "2000-01-01",
	})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreateAuthUser_InvalidInput_EmptyFirstName(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(repomocks.NewMockAccountRepo(ctrl), nil, nil, nil, nil))

	_, err := svc.CreateAuthUser(context.Background(), CreateAuthUserInput{
		Username:     "newuser",
		PasswordHash: "hash",
		FirstName:    "",
		LastName:     "Petrov",
		Birthday:     "2000-01-01",
	})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreateAuthUser_UsernameTaken(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	// CheckUsernameAvailable calls GetByUsername – returns no error → taken
	accounts.EXPECT().
		GetByUsername(gomock.Any(), "takenuser").
		Return(&model.UserAccount{ID: 5, Username: "takenuser"}, nil)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	_, err := svc.CreateAuthUser(context.Background(), CreateAuthUserInput{
		Username:     "takenuser",
		PasswordHash: "hash",
		FirstName:    "Ivan",
		LastName:     "Petrov",
		Birthday:     "2000-01-01",
	})
	require.ErrorIs(t, err, ErrUsernameTaken)
}

func TestCreateAuthUser_BadBirthday(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	// Username available
	accounts.EXPECT().
		GetByUsername(gomock.Any(), "newuser").
		Return(nil, repository.ErrUserAccountNotFound)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	_, err := svc.CreateAuthUser(context.Background(), CreateAuthUserInput{
		Username:     "newuser",
		PasswordHash: "hash",
		FirstName:    "Ivan",
		LastName:     "Petrov",
		Birthday:     "not-a-date",
	})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreateAuthUser_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	const accountID int64 = 10
	const profileID int64 = 20
	const userProfileID int64 = 30

	// CheckUsernameAvailable → available
	accounts.EXPECT().
		GetByUsername(gomock.Any(), "newuser").
		Return(nil, repository.ErrUserAccountNotFound)

	// Save account
	accounts.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(accountID, nil)

	// Save profile
	profiles.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(profileID, nil)

	// Save user-profile
	userProfiles.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(userProfileID, nil)

	// GetAuthUserByAccount – called at end of CreateAuthUser
	accounts.EXPECT().
		Get(gomock.Any(), accountID).
		Return(&model.UserAccount{ID: accountID, Username: "newuser"}, nil)

	userProfiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.UserProfile{
			ID:            userProfileID,
			UserAccountID: accountID,
			ProfileID:     profileID,
			FirstName:     "Ivan",
			LastName:      "Petrov",
			BirthdayDate:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Gender:        model.Male,
		}, nil)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.Profile{ID: profileID}, nil)

	svc := New(newTestStore(accounts, profiles, userProfiles, nil, nil))

	user, err := svc.CreateAuthUser(context.Background(), CreateAuthUserInput{
		Username:     "newuser",
		PasswordHash: "hash",
		FirstName:    "Ivan",
		LastName:     "Petrov",
		Birthday:     "2000-01-01",
		Gender:       model.Male,
	})
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, accountID, user.UserAccountID)
	require.Equal(t, "newuser", user.Login)
}

// ---------------------------------------------------------------------------
// GetCredentialsByLogin
// ---------------------------------------------------------------------------

func TestGetCredentialsByLogin_EmptyLogin(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(repomocks.NewMockAccountRepo(ctrl), nil, nil, nil, nil))

	_, err := svc.GetCredentialsByLogin(context.Background(), "  ")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetCredentialsByLogin_NotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	accounts.EXPECT().
		GetByUsername(gomock.Any(), "nobody").
		Return(nil, repository.ErrUserAccountNotFound)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	_, err := svc.GetCredentialsByLogin(context.Background(), "nobody")
	require.ErrorIs(t, err, ErrUserAccountNotFound)
}

func TestGetCredentialsByLogin_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	accounts.EXPECT().
		GetByUsername(gomock.Any(), "alice").
		Return(&model.UserAccount{ID: 7, Username: "alice", PasswordHash: "secret"}, nil)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	creds, err := svc.GetCredentialsByLogin(context.Background(), "alice")
	require.NoError(t, err)
	require.NotNil(t, creds)
	require.Equal(t, int64(7), creds.UserAccountID)
	require.Equal(t, "secret", creds.PasswordHash)
}

// ---------------------------------------------------------------------------
// UpdatePasswordHash
// ---------------------------------------------------------------------------

func TestUpdatePasswordHash_InvalidInput_ZeroID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(repomocks.NewMockAccountRepo(ctrl), nil, nil, nil, nil))

	err := svc.UpdatePasswordHash(context.Background(), 0, "newhash")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdatePasswordHash_InvalidInput_EmptyHash(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(repomocks.NewMockAccountRepo(ctrl), nil, nil, nil, nil))

	err := svc.UpdatePasswordHash(context.Background(), 1, "   ")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdatePasswordHash_AccountNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	accounts.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(repository.ErrUserAccountNotFound)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	err := svc.UpdatePasswordHash(context.Background(), 99, "newhash")
	require.ErrorIs(t, err, ErrUserAccountNotFound)
}

func TestUpdatePasswordHash_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	accounts.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	err := svc.UpdatePasswordHash(context.Background(), 1, "newhash")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// GetAuthUserByAccount
// ---------------------------------------------------------------------------

func TestGetAuthUserByAccount_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(repomocks.NewMockAccountRepo(ctrl), nil, nil, nil, nil))

	_, err := svc.GetAuthUserByAccount(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetAuthUserByAccount_AccountNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)

	accounts.EXPECT().
		Get(gomock.Any(), int64(42)).
		Return(nil, repository.ErrUserAccountNotFound)

	svc := New(newTestStore(accounts, nil, nil, nil, nil))

	_, err := svc.GetAuthUserByAccount(context.Background(), 42)
	require.ErrorIs(t, err, ErrUserAccountNotFound)
}

func TestGetAuthUserByAccount_UserProfileNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	const accountID int64 = 42

	accounts.EXPECT().
		Get(gomock.Any(), accountID).
		Return(&model.UserAccount{ID: accountID, Username: "bob"}, nil)

	userProfiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(nil, repository.ErrUserProfileNotFound)

	svc := New(newTestStore(accounts, nil, userProfiles, nil, nil))

	_, err := svc.GetAuthUserByAccount(context.Background(), accountID)
	require.ErrorIs(t, err, ErrUserProfileNotFound)
}

func TestGetAuthUserByAccount_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	accounts := repomocks.NewMockAccountRepo(ctrl)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	const accountID int64 = 42
	const profileID int64 = 100
	const upID int64 = 200

	accounts.EXPECT().
		Get(gomock.Any(), accountID).
		Return(&model.UserAccount{ID: accountID, Username: "bob"}, nil)

	userProfiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.UserProfile{
			ID:            upID,
			UserAccountID: accountID,
			ProfileID:     profileID,
			FirstName:     "Bob",
			LastName:      "Smith",
		}, nil)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), accountID).
		Return(&model.Profile{ID: profileID}, nil)

	svc := New(newTestStore(accounts, profiles, userProfiles, nil, nil))

	user, err := svc.GetAuthUserByAccount(context.Background(), accountID)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, accountID, user.UserAccountID)
	require.Equal(t, "bob", user.Login)
	require.Equal(t, "Bob", user.FirstName)
	require.Equal(t, "Smith", user.LastName)
	require.Equal(t, profileID, user.ProfileID)
}

// ---------------------------------------------------------------------------
// SearchProfiles
// ---------------------------------------------------------------------------

func TestSearchProfiles_EmptyQuery(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, nil, repomocks.NewMockUserProfileRepo(ctrl), nil, nil))

	_, err := svc.SearchProfiles(context.Background(), "   ", 10)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestSearchProfiles_RepoError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)
	boom := errors.New("db error")

	userProfiles.EXPECT().
		Search(gomock.Any(), "alice", 10).
		Return(nil, boom)

	svc := New(newTestStore(nil, nil, userProfiles, nil, nil))

	_, err := svc.SearchProfiles(context.Background(), "alice", 10)
	require.ErrorIs(t, err, boom)
}

func TestSearchProfiles_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	results := []repository.SearchProfileResult{
		{ProfileID: 1, UserAccountID: 10, Username: "alice", FirstName: "Alice", LastName: "Smith"},
		{ProfileID: 2, UserAccountID: 11, Username: "alicia", FirstName: "Alicia", LastName: "Jones"},
	}

	userProfiles.EXPECT().
		Search(gomock.Any(), "ali", 10).
		Return(results, nil)

	svc := New(newTestStore(nil, nil, userProfiles, nil, nil))

	found, err := svc.SearchProfiles(context.Background(), "ali", 10)
	require.NoError(t, err)
	require.Len(t, found, 2)
	require.Equal(t, int64(1), found[0].ProfileID)
	require.Equal(t, "alice", found[0].Username)
}

func TestSearchProfiles_DefaultLimit(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	// limit 0 → defaults to 10
	userProfiles.EXPECT().
		Search(gomock.Any(), "bob", 10).
		Return([]repository.SearchProfileResult{}, nil)

	svc := New(newTestStore(nil, nil, userProfiles, nil, nil))

	found, err := svc.SearchProfiles(context.Background(), "bob", 0)
	require.NoError(t, err)
	require.Empty(t, found)
}

func TestSearchProfiles_CapLimit(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userProfiles := repomocks.NewMockUserProfileRepo(ctrl)

	// limit >50 → capped to 50
	userProfiles.EXPECT().
		Search(gomock.Any(), "eve", 50).
		Return([]repository.SearchProfileResult{}, nil)

	svc := New(newTestStore(nil, nil, userProfiles, nil, nil))

	_, err := svc.SearchProfiles(context.Background(), "eve", 200)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// GetUsersFriends
// ---------------------------------------------------------------------------

func TestGetUsersFriends_InvalidInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	_, err := svc.GetUsersFriends(context.Background(), 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetUsersFriends_ProfileNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		Get(gomock.Any(), int64(5)).
		Return(nil, repository.ErrProfileNotFound)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	_, err := svc.GetUsersFriends(context.Background(), 5)
	require.ErrorIs(t, err, ErrProfileNotFound)
}

func TestGetUsersFriends_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	const profileID int64 = 5

	profiles.EXPECT().
		Get(gomock.Any(), profileID).
		Return(&model.Profile{ID: profileID}, nil)

	expectedFriends := []model.Friend{
		{ProfileID: 7, FirstName: "Carol", LastName: "White", Username: "carol"},
	}
	friendships.EXPECT().
		GetFriends(gomock.Any(), profileID, model.FriendshipAccepted).
		Return(expectedFriends, nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	friends, err := svc.GetUsersFriends(context.Background(), profileID)
	require.NoError(t, err)
	require.Len(t, friends, 1)
	require.Equal(t, int64(7), friends[0].ProfileID)
}

// ---------------------------------------------------------------------------
// RequestFriendship
// ---------------------------------------------------------------------------

func TestRequestFriendship_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	err := svc.RequestFriendship(context.Background(), 0, 5)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRequestFriendship_InvalidInput_ZeroFriend(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.RequestFriendship(context.Background(), 1, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRequestFriendship_SelfFriendship(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	// userAccountID=1 → profileID=10; friendID=10 (same as own profileID)
	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.RequestFriendship(context.Background(), 1, 10)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRequestFriendship_FriendProfileNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	profiles.EXPECT().
		Get(gomock.Any(), int64(20)).
		Return(nil, repository.ErrProfileNotFound)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.RequestFriendship(context.Background(), 1, 20)
	require.ErrorIs(t, err, ErrProfileNotFound)
}

func TestRequestFriendship_NewRequest_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	// userAccountID=1 → profileID=10; friendID=20
	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	profiles.EXPECT().
		Get(gomock.Any(), int64(20)).
		Return(&model.Profile{ID: 20}, nil)

	// checkFriendshipBy(friendID=20, profileID=10) → not found
	friendships.EXPECT().
		GetFriendshipStatusBy(gomock.Any(), int64(20), int64(10)).
		Return("", repository.ErrFriendshipNotFound)

	// checkFriendshipBy(profileID=10, friendID=20) → not found
	friendships.EXPECT().
		GetFriendshipStatusBy(gomock.Any(), int64(10), int64(20)).
		Return("", repository.ErrFriendshipNotFound)

	friendships.EXPECT().
		Create(gomock.Any(), int64(10), int64(20), string(model.FriendshipPending)).
		Return(nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.RequestFriendship(context.Background(), 1, 20)
	require.NoError(t, err)
}

func TestRequestFriendship_IncomingPending_AutoAccepts(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	// userAccountID=1 → profileID=10; friendID=20
	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	profiles.EXPECT().
		Get(gomock.Any(), int64(20)).
		Return(&model.Profile{ID: 20}, nil)

	// checkFriendshipBy(friendID=20, profileID=10) → pending (friend already sent request)
	friendships.EXPECT().
		GetFriendshipStatusBy(gomock.Any(), int64(20), int64(10)).
		Return(string(model.FriendshipPending), nil)

	// auto-accept
	friendships.EXPECT().
		AcceptFriendship(gomock.Any(), int64(20), int64(10)).
		Return(nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.RequestFriendship(context.Background(), 1, 20)
	require.NoError(t, err)
}

func TestRequestFriendship_AlreadyFriends(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	profiles.EXPECT().
		Get(gomock.Any(), int64(20)).
		Return(&model.Profile{ID: 20}, nil)

	// checkFriendshipBy(friendID=20, profileID=10) → accepted (already friends)
	friendships.EXPECT().
		GetFriendshipStatusBy(gomock.Any(), int64(20), int64(10)).
		Return(string(model.FriendshipAccepted), nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.RequestFriendship(context.Background(), 1, 20)
	require.ErrorIs(t, err, ErrAlreadyFriends)
}

// ---------------------------------------------------------------------------
// DeleteFriend
// ---------------------------------------------------------------------------

func TestDeleteFriend_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	err := svc.DeleteFriend(context.Background(), 0, 5)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeleteFriend_InvalidInput_ZeroFriend(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.DeleteFriend(context.Background(), 1, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeleteFriend_FriendshipNotExists(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	friendships.EXPECT().
		DeleteFriend(gomock.Any(), int64(10), int64(20)).
		Return(repository.ErrNoRowsAffected)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.DeleteFriend(context.Background(), 1, 20)
	require.ErrorIs(t, err, ErrFriendshipNotExists)
}

func TestDeleteFriend_Success(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	friendships.EXPECT().
		DeleteFriend(gomock.Any(), int64(10), int64(20)).
		Return(nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.DeleteFriend(context.Background(), 1, 20)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// AcceptFriendRequest
// ---------------------------------------------------------------------------

func TestAcceptFriendRequest_InvalidInput_ZeroAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := New(newTestStore(nil, repomocks.NewMockProfileRepo(ctrl), nil, nil, nil))

	err := svc.AcceptFriendRequest(context.Background(), 0, 5)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestAcceptFriendRequest_InvalidInput_ZeroRequester(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.AcceptFriendRequest(context.Background(), 1, 0)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestAcceptFriendRequest_SelfAccept(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)

	// profileID == requesterID
	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	svc := New(newTestStore(nil, profiles, nil, nil, nil))

	err := svc.AcceptFriendRequest(context.Background(), 1, 10)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestAcceptFriendRequest_NoPendingRequest(t *testing.T) {
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

	err := svc.AcceptFriendRequest(context.Background(), 1, 30)
	require.ErrorIs(t, err, ErrFriendshipNotExists)
}

func TestAcceptFriendRequest_NotPendingStatus(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	profiles := repomocks.NewMockProfileRepo(ctrl)
	friendships := repomocks.NewMockFriendshipRepo(ctrl)

	profiles.EXPECT().
		GetByUserAccountID(gomock.Any(), int64(1)).
		Return(&model.Profile{ID: 10}, nil)

	// status is "accepted", not "pending"
	friendships.EXPECT().
		GetFriendshipStatusBy(gomock.Any(), int64(30), int64(10)).
		Return(string(model.FriendshipAccepted), nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.AcceptFriendRequest(context.Background(), 1, 30)
	require.ErrorIs(t, err, ErrFriendshipNotExists)
}

func TestAcceptFriendRequest_Success(t *testing.T) {
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
		AcceptFriendship(gomock.Any(), int64(30), int64(10)).
		Return(nil)

	svc := New(newTestStore(nil, profiles, nil, friendships, nil))

	err := svc.AcceptFriendRequest(context.Background(), 1, 30)
	require.NoError(t, err)
}
