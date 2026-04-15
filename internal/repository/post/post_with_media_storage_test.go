package post

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPostWithMediaStorageGetMediaByPostID(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewPostWithMediaStorage(mockPool)
	rows := pgxmock.NewRows([]string{"media_id"}).AddRow(int64(11)).AddRow(int64(12))
	mockPool.ExpectQuery("SELECT media_id FROM post_with_media WHERE post_id=\\$1").
		WithArgs(int64(7)).
		WillReturnRows(rows)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	got := repo.GetMediaByPostID(ctx, 7)
	require.Equal(t, []int64{11, 12}, got)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestPostWithMediaStorageSaveAndDeleteByPostID(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewPostWithMediaStorage(mockPool)
	mockPool.ExpectExec("INSERT INTO post_with_media \\(post_id, media_id, sort_order\\) VALUES \\(\\$1, \\$2, \\$3\\)").
		WithArgs(int64(1), int64(2), 3).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mockPool.ExpectExec("DELETE FROM post_with_media WHERE post_id=\\$1").
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err = repo.Save(ctx, models.PostWithMedia{PostID: 1, MediaID: 2, Order: 3})
	require.NoError(t, err)
	err = repo.DeleteByPostID(ctx, 1)
	require.NoError(t, err)
	require.NoError(t, mockPool.ExpectationsWereMet())
}
