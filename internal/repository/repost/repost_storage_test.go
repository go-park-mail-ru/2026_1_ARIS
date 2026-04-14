package repost

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestRepostStorageSave(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewRepostStorage(mockPool)
	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(17))
	mockPool.ExpectQuery("INSERT INTO repost").
		WithArgs(pgxmock.AnyArg(), int64(1), int64(2), int64(3)).
		WillReturnRows(rows)

	id, err := repo.Save(context.Background(), models.Repost{
		AuthorID: 1,
		ChatID:   2,
		PostID:   3,
	})
	require.NoError(t, err)
	require.Equal(t, int64(17), id)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestRepostStorageGetRepostCount(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewRepostStorage(mockPool)
		rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(4))
		mockPool.ExpectQuery("SELECT COUNT\\(\\*\\) FROM repost WHERE post_id=\\$1").
			WithArgs(int64(99)).
			WillReturnRows(rows)

		count := repo.GetRepostCount(context.Background(), 99)
		require.Equal(t, 4, count)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("scan error returns zero", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewRepostStorage(mockPool)
		rows := pgxmock.NewRows([]string{"count"}).AddRow("bad")
		mockPool.ExpectQuery("SELECT COUNT\\(\\*\\) FROM repost WHERE post_id=\\$1").
			WithArgs(int64(99)).
			WillReturnRows(rows)

		count := repo.GetRepostCount(context.Background(), 99)
		require.Equal(t, 0, count)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}
