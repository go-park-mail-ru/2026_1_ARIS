package post

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPostStorageDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		affected  int64
		wantError error
	}{
		{
			name:      "ok",
			affected:  1,
			wantError: nil,
		},
		{
			name:      "no rows affected",
			affected:  0,
			wantError: xerrors.PostNotFound,
		},
		{
			name:      "multiple rows affected",
			affected:  2,
			wantError: xerrors.MultipleRowsAffect,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			repo := NewPostStorage(mockPool)

			mockPool.ExpectExec("DELETE FROM like_record WHERE post_id=\\$1").
				WithArgs(int64(10)).
				WillReturnResult(pgxmock.NewResult("DELETE", 1))

			mockPool.ExpectExec("DELETE FROM post WHERE id=\\$1").
				WithArgs(int64(10)).
				WillReturnResult(pgxmock.NewResult("DELETE", tc.affected))

			mockLogger := zap.NewNop()
			ctx := logger.WithLogger(context.Background(), mockLogger)
			err = repo.Delete(ctx, 10)

			if tc.wantError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantError)
			}

			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestPostStorageUpdate(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewPostStorage(mockPool)
	postText := "updated"
	post := models.Post{
		ID:        7,
		Text:      &postText,
		UpdatedAt: time.Now(),
	}

	mockPool.ExpectExec("UPDATE post SET post_text=\\$1, updated_at=\\$2 WHERE id=\\$3").
		WithArgs(post.Text, post.UpdatedAt, post.ID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)
	err = repo.Update(ctx, post)
	require.NoError(t, err)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestPostStorageSave(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewPostStorage(mockPool)

	postText := "post text"
	post := models.Post{
		Text:          &postText,
		AuthorID:      5,
		IsPublicDemo:  true,
		AllowComments: true,
	}

	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(42))
	mockPool.ExpectQuery("INSERT INTO post").
		WithArgs(pgxmock.AnyArg(), post.Text, post.AuthorID, post.CommunityID, post.IsPublicDemo, post.AllowComments).
		WillReturnRows(rows)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	gotID, err := repo.Save(ctx, post)
	require.NoError(t, err)
	assert.Equal(t, int64(42), gotID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestPostStorageReadMethods(t *testing.T) {
	t.Parallel()

	now := time.Now()
	text := "txt"
	cols := []string{"id", "uid", "post_text", "author_id", "community_id", "is_public_demo", "allow_comments", "is_active", "created_at", "updated_at"}

	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewPostStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(int64(5), uuid.New(), &text, int64(2), nil, true, true, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM post WHERE id=\\$1").WithArgs(int64(5)).WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.Get(ctx, 5)
		require.NoError(t, err)
		require.Equal(t, int64(5), got.ID)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("Get not found", func(t *testing.T) {
		t.Parallel()
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewPostStorage(mockPool)
		rows := pgxmock.NewRows(cols)
		mockPool.ExpectQuery("SELECT \\* FROM post WHERE id=\\$1").WithArgs(int64(500)).WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		_, err := repo.Get(ctx, 500)
		require.ErrorIs(t, err, xerrors.PostNotFound)
	})

	t.Run("GetAll", func(t *testing.T) {
		t.Parallel()
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewPostStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(int64(1), uuid.New(), &text, int64(2), nil, true, true, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM post").WillReturnRows(rows)
		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewPostStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(int64(1), uuid.New(), &text, int64(2), nil, true, true, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM post ORDER BY id LIMIT \\$1 OFFSET \\$2").WithArgs(10, 0).WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.List(ctx, 0, 10)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("GetByAuthorID", func(t *testing.T) {
		t.Parallel()
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewPostStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(int64(3), uuid.New(), &text, int64(77), nil, true, true, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM post WHERE author_id=\\$1 ORDER BY created_at DESC").WithArgs(int64(77)).WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.GetByAuthorID(ctx, 77)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("GetByCommunityID", func(t *testing.T) {
		t.Parallel()
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewPostStorage(mockPool)
		communityID := int64(55)
		rows := pgxmock.NewRows(cols).AddRow(int64(4), uuid.New(), &text, int64(77), &communityID, true, true, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM post WHERE community_id=\\$1 ORDER BY created_at DESC").WithArgs(communityID).WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.GetByCommunityID(ctx, communityID)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, &communityID, got[0].CommunityID)
	})

	t.Run("List query error", func(t *testing.T) {
		t.Parallel()
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewPostStorage(mockPool)
		mockPool.ExpectQuery("SELECT \\* FROM post ORDER BY id LIMIT \\$1 OFFSET \\$2").WithArgs(10, 0).WillReturnError(errors.New("db err"))

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		_, err := repo.List(ctx, 0, 10)
		require.EqualError(t, err, "db err")
	})
}
