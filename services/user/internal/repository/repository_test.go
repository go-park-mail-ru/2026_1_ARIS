package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	repository "github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoriesReturnDBErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := repomocks.NewMockDB(ctrl)
	row := repomocks.NewMockRow(ctrl)
	dbErr := errors.New("db down")

	row.EXPECT().Scan(gomock.Any()).Return(dbErr).AnyTimes()
	db.EXPECT().Begin(gomock.Any()).Return(nil, dbErr).AnyTimes()
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any(), gomock.Any()).Return(row).AnyTimes()
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, dbErr).AnyTimes()
	db.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr).AnyTimes()

	ctx := context.Background()
	store := repository.NewStore(db)
	email := "u@example.com"
	account := *model.NewUserAccount("user", &email, nil, "hash")
	profile := *model.NewProfile(nil)
	userProfile := *model.NewUserProfile(1, 2, "First", "Last", nil, time.Now(), model.Male)

	require.Error(t, store.OAuth.Save(ctx, "vk", "1", 1, &email))
	_, err := store.OAuth.GetUserAccountID(ctx, "vk", "1")
	require.Error(t, err)

	_, err = store.Accounts.Save(ctx, account)
	require.Error(t, err)
	_, err = store.Accounts.Get(ctx, 1)
	require.Error(t, err)
	_, err = store.Accounts.GetByUsername(ctx, "user")
	require.Error(t, err)
	require.NoError(t, store.Accounts.Update(ctx, repository.AccountUpdate{ID: 1}))
	username := "new-user"
	require.Error(t, store.Accounts.Update(ctx, repository.AccountUpdate{ID: 1, Username: &username}))
	require.True(t, repository.AccountUpdate{Username: &username}.HasUpdates())
	require.False(t, repository.AccountUpdate{}.HasUpdates())

	_, err = store.Profiles.Save(ctx, profile)
	require.Error(t, err)
	_, err = store.Profiles.Get(ctx, 2)
	require.Error(t, err)
	_, err = store.Profiles.GetAll(ctx)
	require.Error(t, err)
	_, err = store.Profiles.GetByUserAccountID(ctx, 1)
	require.Error(t, err)
	require.Error(t, store.Profiles.UpdateAvatar(ctx, 2, nil))

	_, err = store.UserProfiles.Save(ctx, userProfile)
	require.Error(t, err)
	_, err = store.UserProfiles.GetByProfileID(ctx, 2)
	require.Error(t, err)
	_, err = store.UserProfiles.GetByUserAccountID(ctx, 1)
	require.Error(t, err)
	require.NoError(t, store.UserProfiles.Update(ctx, repository.UserProfileUpdate{ID: 1}))
	firstName := "Changed"
	require.Error(t, store.UserProfiles.Update(ctx, repository.UserProfileUpdate{ID: 1, FirstName: &firstName}))
	_, err = store.UserProfiles.Search(ctx, "first", 10)
	require.Error(t, err)
	require.True(t, repository.UserProfileUpdate{FirstName: &firstName}.HasUpdates())
	require.False(t, repository.UserProfileUpdate{}.HasUpdates())

	_, err = store.Settings.GetByUserID(ctx, 1)
	require.Error(t, err)
	language := model.LanguageEN
	_, err = store.Settings.Update(ctx, 1, repository.SettingsUpdate{Language: &language})
	require.Error(t, err)

	_, err = store.Friendships.GetFriends(ctx, 2, model.FriendshipAccepted)
	require.Error(t, err)
	_, err = store.Friendships.GetFriendshipStatusBy(ctx, 2, 3)
	require.Error(t, err)
	_, err = store.Friendships.GetOutgoingFriends(ctx, 2, "pending")
	require.Error(t, err)
	_, err = store.Friendships.GetIncomingFriends(ctx, 2, "pending")
	require.Error(t, err)
	require.Error(t, store.Friendships.Create(ctx, 2, 3, "pending"))
	require.Error(t, store.Friendships.AcceptFriendship(ctx, 2, 3))
	require.Error(t, store.Friendships.DeclineFriendship(ctx, 2, 3))
	require.Error(t, store.Friendships.RevokeFriendRequest(ctx, 2, 3))
	require.Error(t, store.Friendships.DeleteFriend(ctx, 2, 3))
}

func TestUserRepositoriesNoRowsAffected(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := repomocks.NewMockDB(ctrl)
	tx := repomocks.NewMockTx(ctrl)
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.NewCommandTag("UPDATE 0"), nil).AnyTimes()
	db.EXPECT().Begin(gomock.Any()).Return(tx, nil).AnyTimes()
	tx.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.NewCommandTag("UPDATE 0"), nil).AnyTimes()
	tx.EXPECT().Rollback(gomock.Any()).Return(nil).AnyTimes()

	ctx := context.Background()
	store := repository.NewStore(db)
	username := "new-user"
	firstName := "Changed"

	require.ErrorIs(t, store.Accounts.Update(ctx, repository.AccountUpdate{ID: 1, Username: &username}), repository.ErrUserAccountNotFound)
	require.ErrorIs(t, store.Profiles.UpdateAvatar(ctx, 2, nil), repository.ErrProfileNotFound)
	require.ErrorIs(t, store.UserProfiles.Update(ctx, repository.UserProfileUpdate{ID: 1, FirstName: &firstName}), repository.ErrUserProfileNotFound)
	require.ErrorIs(t, store.Friendships.AcceptFriendship(ctx, 2, 3), repository.ErrNoRowsAffected)
	require.ErrorIs(t, store.Friendships.DeclineFriendship(ctx, 2, 3), repository.ErrNoRowsAffected)
	require.ErrorIs(t, store.Friendships.RevokeFriendRequest(ctx, 2, 3), repository.ErrNoRowsAffected)
	require.ErrorIs(t, store.Friendships.DeleteFriend(ctx, 2, 3), repository.ErrNoRowsAffected)
}

func stringPtr(value string) *string {
	return &value
}
