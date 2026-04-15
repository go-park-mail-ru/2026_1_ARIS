package friend

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	servicedto "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/dto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestFriendshipServiceDelegates(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomocks.NewMockFriendshipRepo(ctrl)
	svc := NewFriendshipService(repo)

	want := []servicedto.FriendDTO{{ProfileID: 1}}
	repo.EXPECT().GetFriends(gomock.Any(), int64(2), models.FriendshipAccepted).Return(want, nil)
	got, err := svc.GetFriends(context.Background(), 2, models.FriendshipAccepted)
	require.NoError(t, err)
	require.Equal(t, want, got)

	repo.EXPECT().GetOutgoingFriends(gomock.Any(), int64(3), "pending").Return(want, nil)
	out, err := svc.GetOutgoingFriends(context.Background(), 3, "pending")
	require.NoError(t, err)
	require.Equal(t, want, out)

	repo.EXPECT().GetIncomingFriends(gomock.Any(), int64(4), "pending").Return(want, nil)
	in, err := svc.GetIncomingFriends(context.Background(), 4, "pending")
	require.NoError(t, err)
	require.Equal(t, want, in)

	repo.EXPECT().DeleteFriend(gomock.Any(), int64(1), int64(2)).Return(nil)
	require.NoError(t, svc.DeleteFriend(context.Background(), 1, 2))

	repo.EXPECT().RevokeFriendRequest(gomock.Any(), int64(1), int64(2)).Return(nil)
	require.NoError(t, svc.RevokeFriendRequest(context.Background(), 1, 2))
}

func TestFriendshipServiceCheckFriendship(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatus(gomock.Any(), int64(1), int64(2)).Return("", xerrors.FriendshipNotFound)
		ok, st, err := svc.CheckFriendship(context.Background(), 1, 2)
		require.NoError(t, err)
		require.False(t, ok)
		require.Empty(t, st)
	})

	t.Run("other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatus(gomock.Any(), int64(1), int64(2)).Return("", errors.New("db"))
		_, _, err := svc.CheckFriendship(context.Background(), 1, 2)
		require.EqualError(t, err, "db")
	})

	t.Run("accepted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatus(gomock.Any(), int64(1), int64(2)).Return("accepted", nil)
		ok, st, err := svc.CheckFriendship(context.Background(), 1, 2)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, models.FriendshipAccepted, st)
	})
}

func TestFriendshipServiceCheckFriendshipBy(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomocks.NewMockFriendshipRepo(ctrl)
	svc := NewFriendshipService(repo)
	repo.EXPECT().GetFriendshipStatusBy(gomock.Any(), int64(1), int64(2)).Return("pending", nil)
	ok, st, err := svc.CheckFriendshipBy(context.Background(), 1, 2)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, models.FriendshipPending, st)
}

func TestFriendshipServiceMakeFriends(t *testing.T) {
	t.Parallel()

	t.Run("creates pending when no friendship", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatus(gomock.Any(), int64(10), int64(20)).Return("", xerrors.FriendshipNotFound)
		repo.EXPECT().Create(gomock.Any(), int64(10), int64(20), string(models.FriendshipPending)).Return(nil)
		require.NoError(t, svc.MakeFriends(context.Background(), 10, 20))
	})

	t.Run("accepts incoming request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatus(gomock.Any(), int64(1), int64(2)).Return("pending", nil)
		repo.EXPECT().GetFriendshipStatusBy(gomock.Any(), int64(2), int64(1)).Return(string(models.FriendshipPending), nil)
		repo.EXPECT().GetFriendshipStatusBy(gomock.Any(), int64(1), int64(2)).Return(string(models.FriendshipPending), nil)
		repo.EXPECT().AcceptFriendship(gomock.Any(), int64(1), int64(2)).Return(nil)
		require.NoError(t, svc.MakeFriends(context.Background(), 1, 2))
	})

	t.Run("already sent returns already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatus(gomock.Any(), int64(1), int64(2)).Return("pending", nil)
		repo.EXPECT().GetFriendshipStatusBy(gomock.Any(), int64(2), int64(1)).Return("", nil)
		repo.EXPECT().GetFriendshipStatusBy(gomock.Any(), int64(1), int64(2)).Return(string(models.FriendshipPending), nil)
		err := svc.MakeFriends(context.Background(), 1, 2)
		require.ErrorIs(t, err, xerrors.AllreadyExists)
	})
}

func TestFriendshipServiceAcceptDecline(t *testing.T) {
	t.Parallel()

	t.Run("accept when pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatusBy(gomock.Any(), int64(1), int64(2)).Return(string(models.FriendshipPending), nil)
		repo.EXPECT().AcceptFriendship(gomock.Any(), int64(1), int64(2)).Return(nil)
		require.NoError(t, svc.AcceptFriendship(context.Background(), 1, 2))
	})

	t.Run("decline success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatusBy(gomock.Any(), int64(1), int64(2)).Return(string(models.FriendshipPending), nil)
		repo.EXPECT().DeclineFriendship(gomock.Any(), int64(1), int64(2)).Return(nil)
		require.NoError(t, svc.DeclineFriendship(context.Background(), 1, 2))
	})

	t.Run("decline not pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockFriendshipRepo(ctrl)
		svc := NewFriendshipService(repo)
		repo.EXPECT().GetFriendshipStatusBy(gomock.Any(), int64(1), int64(2)).Return("accepted", nil)
		err := svc.DeclineFriendship(context.Background(), 1, 2)
		require.ErrorIs(t, err, xerrors.FriendshipNotFound)
	})
}
