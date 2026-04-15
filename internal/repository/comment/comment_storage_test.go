package comment

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestCommentStorageGetCommentCount(t *testing.T) {
	t.Parallel()

	t.Run("returns count", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewCommentStorage(mockPool)
		rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(3))
		mockPool.ExpectQuery("SELECT COUNT\\(\\*\\) FROM comment WHERE post_id=\\$1;").
			WithArgs(int64(11)).
			WillReturnRows(rows)

		count := repo.GetCommentCount(context.Background(), 11)
		require.Equal(t, 3, count)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("scan error returns zero", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewCommentStorage(mockPool)
		rows := pgxmock.NewRows([]string{"count"}).AddRow("bad")
		mockPool.ExpectQuery("SELECT COUNT\\(\\*\\) FROM comment WHERE post_id=\\$1;").
			WithArgs(int64(11)).
			WillReturnRows(rows)

		count := repo.GetCommentCount(context.Background(), 11)
		require.Equal(t, 0, count)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestCommentStorageSave(t *testing.T) {
	t.Parallel()
	text := "hello"

	t.Run("returns inserted id", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewCommentStorage(mockPool)
		comment := models.Comment{Text: &text, TargetPostID: 2, AuthorID: 9}
		rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(44))
		mockPool.ExpectQuery("INSERT INTO comment").
			WithArgs(pgxmock.AnyArg(), comment.Text, comment.TargetPostID, comment.ParentCommentID, comment.StickerID, comment.AuthorID).
			WillReturnRows(rows)

		id, err := repo.Save(context.Background(), comment)
		require.NoError(t, err)
		require.Equal(t, int64(44), id)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("empty returned rows gives bad query", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewCommentStorage(mockPool)
		comment := models.Comment{Text: &text, TargetPostID: 2, AuthorID: 9}
		rows := pgxmock.NewRows([]string{"id"})
		mockPool.ExpectQuery("INSERT INTO comment").
			WithArgs(pgxmock.AnyArg(), comment.Text, comment.TargetPostID, comment.ParentCommentID, comment.StickerID, comment.AuthorID).
			WillReturnRows(rows)

		_, err = repo.Save(context.Background(), comment)
		require.EqualError(t, err, "Bad query")
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("query error bubbles up", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewCommentStorage(mockPool)
		comment := models.Comment{Text: &text, TargetPostID: 2, AuthorID: 9}
		mockPool.ExpectQuery("INSERT INTO comment").
			WithArgs(pgxmock.AnyArg(), comment.Text, comment.TargetPostID, comment.ParentCommentID, comment.StickerID, comment.AuthorID).
			WillReturnError(errors.New("db down"))

		_, err = repo.Save(context.Background(), comment)
		require.EqualError(t, err, "db down")
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}
