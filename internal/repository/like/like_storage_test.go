package like

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestLikeStorageSave(t *testing.T) {
	t.Parallel()

	t.Run("returns inserted id", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewLikeStorage(mockPool)
		postID := int64(8)
		like := models.Like{PostID: &postID, AuthorID: 77}
		rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(101))
		mockPool.ExpectQuery("INSERT INTO like_record").
			WithArgs(pgxmock.AnyArg(), like.PostID, like.CommentID, like.AuthorID).
			WillReturnRows(rows)

		id, err := repo.Save(context.Background(), like)
		require.NoError(t, err)
		require.Equal(t, int64(101), id)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("queryrow scan error bubbles up", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewLikeStorage(mockPool)
		postID := int64(8)
		like := models.Like{PostID: &postID, AuthorID: 77}
		mockPool.ExpectQuery("INSERT INTO like_record").
			WithArgs(pgxmock.AnyArg(), like.PostID, like.CommentID, like.AuthorID).
			WillReturnError(errors.New("insert failed"))

		_, err = repo.Save(context.Background(), like)
		require.EqualError(t, err, "insert failed")
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestLikeStorageGetAndCount(t *testing.T) {
	t.Parallel()

	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		repo := NewLikeStorage(mockPool)
		postID := int64(5)
		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "uid", "post_id", "comment_id", "author_id", "is_active", "created_at", "updated_at"}).
			AddRow(int64(1), uuid.New(), &postID, nil, int64(2), true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM like_record WHERE id=\\$1").WithArgs(int64(1)).WillReturnRows(rows)

		got, err := repo.Get(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, int64(1), got.ID)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("GetLikeCountOnPost stub", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		repo := NewLikeStorage(mockPool)
		got := repo.GetLikeCountOnPost(context.Background(), 10)
		require.Equal(t, 0, got)
	})
}
