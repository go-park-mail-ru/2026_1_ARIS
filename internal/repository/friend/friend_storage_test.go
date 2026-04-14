package friend

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestFriendshipStorageCreate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		affected  int64
		wantError error
	}{
		{name: "ok", affected: 1, wantError: nil},
		{name: "no rows affected", affected: 0, wantError: xerrors.NoRowsAffected},
		{name: "multiple rows affected", affected: 2, wantError: xerrors.MultipleRowsAffect},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			repo := NewFriendshipStorage(mockPool)
			mockPool.ExpectExec("insert into friendship \\(requester_id, addressee_id, status\\) values \\(\\$1, \\$2, \\$3\\)").
				WithArgs(int64(1), int64(2), "pending").
				WillReturnResult(pgxmock.NewResult("INSERT", tc.affected))

			err = repo.Create(context.Background(), 1, 2, "pending")
			if tc.wantError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantError)
			}

			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestFriendshipStorageDeleteFriend(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewFriendshipStorage(mockPool)
	mockPool.ExpectExec("DELETE FROM friendship").
		WithArgs(int64(3), int64(4)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.DeleteFriend(context.Background(), 3, 4)
	require.NoError(t, err)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestFriendshipStorageAcceptFriendship(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		affected  int64
		wantError error
	}{
		{name: "ok", affected: 1, wantError: nil},
		{name: "no rows affected", affected: 0, wantError: xerrors.NoRowsAffected},
		{name: "multiple rows affected", affected: 2, wantError: xerrors.MultipleRowsAffect},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			repo := NewFriendshipStorage(mockPool)
			mockPool.ExpectExec("update friendship f set status='accepted'").
				WithArgs(int64(10), int64(11)).
				WillReturnResult(pgxmock.NewResult("UPDATE", tc.affected))

			err = repo.AcceptFriendship(context.Background(), 10, 11)
			if tc.wantError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantError)
			}

			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestFriendshipStorageDeclineAndRevoke(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewFriendshipStorage(mockPool)
	mockPool.ExpectExec("DELETE FROM friendship WHERE LEAST").
		WithArgs(int64(6), int64(7)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mockPool.ExpectExec("DELETE FROM friendship where requester_id=\\$1 AND addressee_id=\\$2 AND status='pending'").
		WithArgs(int64(6), int64(7)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.DeclineFriendship(context.Background(), 6, 7)
	require.NoError(t, err)

	err = repo.RevokeFriendRequest(context.Background(), 6, 7)
	require.NoError(t, err)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestFriendshipStorageReadMethods(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cols := []string{"avatar_id", "id", "first_name", "last_name", "username", "link", "status", "created_at", "updated_at"}

	t.Run("GetFriends", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewFriendshipStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(nil, int64(2), "A", "B", "ab", nil, "accepted", now, now)
		mockPool.ExpectQuery("select p.avatar_id").WithArgs(int64(1), "accepted").WillReturnRows(rows)
		got, err := repo.GetFriends(context.Background(), 1, models.FriendshipAccepted)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("GetFriendshipStatus", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewFriendshipStorage(mockPool)
		rows := pgxmock.NewRows([]string{"status"}).AddRow("pending")
		mockPool.ExpectQuery("select status from friendship").WithArgs(int64(1), int64(2)).WillReturnRows(rows)
		got, err := repo.GetFriendshipStatus(context.Background(), 1, 2)
		require.NoError(t, err)
		require.Equal(t, "pending", got)
	})

	t.Run("GetFriendshipStatusBy", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewFriendshipStorage(mockPool)
		rows := pgxmock.NewRows([]string{"status"}).AddRow("accepted")
		mockPool.ExpectQuery("select status from friendship where requester_id=\\$1 and addressee_id=\\$2;").WithArgs(int64(1), int64(2)).WillReturnRows(rows)
		got, err := repo.GetFriendshipStatusBy(context.Background(), 1, 2)
		require.NoError(t, err)
		require.Equal(t, "accepted", got)
	})

	t.Run("Outgoing and Incoming", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewFriendshipStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(nil, int64(2), "A", "B", "ab", nil, "pending", now, now)
		rows2 := pgxmock.NewRows(cols).AddRow(nil, int64(3), "C", "D", "cd", nil, "pending", now, now)
		mockPool.ExpectQuery("select f.addressee_id as id").WithArgs(int64(1), "pending").WillReturnRows(rows)
		mockPool.ExpectQuery("select f.requester_id as id").WithArgs(int64(1), "pending").WillReturnRows(rows2)
		out, err := repo.GetOutgoingFriends(context.Background(), 1, "pending")
		require.NoError(t, err)
		require.Len(t, out, 1)
		in, err := repo.GetIncomingFriends(context.Background(), 1, "pending")
		require.NoError(t, err)
		require.Len(t, in, 1)
	})

	t.Run("status not found", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewFriendshipStorage(mockPool)
		rows := pgxmock.NewRows([]string{"status"})
		mockPool.ExpectQuery("select status from friendship").WithArgs(int64(9), int64(10)).WillReturnRows(rows)
		_, err := repo.GetFriendshipStatus(context.Background(), 9, 10)
		require.ErrorIs(t, err, xerrors.FriendshipNotFound)
	})

}
