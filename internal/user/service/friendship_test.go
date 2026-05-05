package service

import (
	"context"
	"testing"

	servicedto "github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/stretchr/testify/require"
)

func TestFriendshipReads(t *testing.T) {
	ctrl, m := newUserMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	m.profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	m.friendships.EXPECT().GetFriends(ctx, int64(20), models.FriendshipAccepted).Return([]servicedto.FriendDTO{{ProfileID: 30}}, nil)
	friends, err := m.service.GetFriends(ctx, 10, models.FriendshipAccepted)
	require.NoError(t, err)
	require.Len(t, friends, 1)

	m.profiles.EXPECT().Get(ctx, int64(30)).Return(&models.Profile{ID: 30}, nil)
	m.friendships.EXPECT().GetFriends(ctx, int64(30), models.FriendshipAccepted).Return([]servicedto.FriendDTO{{ProfileID: 20}}, nil)
	friends, err = m.service.GetUsersFriends(ctx, 30)
	require.NoError(t, err)
	require.Len(t, friends, 1)

	m.profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	m.friendships.EXPECT().GetIncomingFriends(ctx, int64(20), string(models.FriendshipPending)).Return(nil, nil)
	_, err = m.service.GetIncomingFriendRequests(ctx, 10, string(models.FriendshipPending))
	require.NoError(t, err)

	m.profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	m.friendships.EXPECT().GetOutgoingFriends(ctx, int64(20), string(models.FriendshipPending)).Return(nil, nil)
	_, err = m.service.GetOutgoingFriendRequests(ctx, 10, string(models.FriendshipPending))
	require.NoError(t, err)
}

func TestFriendshipMutations(t *testing.T) {
	ctrl, m := newUserMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	m.profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	m.profiles.EXPECT().Get(ctx, int64(30)).Return(&models.Profile{ID: 30}, nil)
	m.friendships.EXPECT().GetFriendshipStatusBy(ctx, int64(30), int64(20)).Return("", xerrors.FriendshipNotFound)
	m.friendships.EXPECT().GetFriendshipStatusBy(ctx, int64(20), int64(30)).Return("", xerrors.FriendshipNotFound)
	m.friendships.EXPECT().Create(ctx, int64(20), int64(30), string(models.FriendshipPending)).Return(nil)
	require.NoError(t, m.service.RequestFriendship(ctx, 10, 30))

	m.profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	m.friendships.EXPECT().GetFriendshipStatusBy(ctx, int64(30), int64(20)).Return(string(models.FriendshipPending), nil)
	m.friendships.EXPECT().AcceptFriendship(ctx, int64(30), int64(20)).Return(nil)
	require.NoError(t, m.service.AcceptFriendRequest(ctx, 10, 30))

	m.profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	m.friendships.EXPECT().GetFriendshipStatusBy(ctx, int64(30), int64(20)).Return(string(models.FriendshipPending), nil)
	m.friendships.EXPECT().DeclineFriendship(ctx, int64(30), int64(20)).Return(nil)
	require.NoError(t, m.service.DeclineFriendRequest(ctx, 10, 30))

	m.profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	m.friendships.EXPECT().RevokeFriendRequest(ctx, int64(20), int64(30)).Return(nil)
	require.NoError(t, m.service.RevokeFriendRequest(ctx, 10, 30))

	m.profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	m.friendships.EXPECT().DeleteFriend(ctx, int64(20), int64(30)).Return(nil)
	require.NoError(t, m.service.DeleteFriend(ctx, 10, 30))
}

func TestFriendshipValidationHelpers(t *testing.T) {
	_, m := newUserMocks(t)

	_, err := m.service.GetFriends(context.Background(), 0, models.FriendshipAccepted)
	require.ErrorIs(t, err, ErrInvalidInput)

	require.True(t, validFriendshipStatus(string(models.FriendshipPending)))
	require.True(t, validFriendshipStatus(string(models.FriendshipAccepted)))
	require.False(t, validFriendshipStatus("bad"))
	require.NoError(t, normalizeFriendshipMutationError(nil))
	require.ErrorIs(t, normalizeFriendshipMutationError(xerrors.NoRowsAffected), ErrFriendshipNotExists)
	require.ErrorIs(t, normalizeFriendshipMutationError(xerrors.AllreadyExists), ErrAlreadyFriends)
}
