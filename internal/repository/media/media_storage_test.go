package media

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMediaStorageSaveGetAndGetLink(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewMediaStorage(mockPool)
	now := time.Now()

	saveRows := pgxmock.NewRows([]string{"id"}).AddRow(int64(5))
	mockPool.ExpectQuery("INSERT INTO media").
		WithArgs(uuid.MustParse("00000000-0000-0000-0000-000000000010"), "name", "png", "image/png", 100, "https://s3/link", int64(2)).
		WillReturnRows(saveRows)

	getRows := pgxmock.NewRows([]string{"id", "uid", "media_name", "author_id", "extension", "description", "mime_type", "link", "size", "is_active", "created_at", "updated_at"}).
		AddRow(int64(5), uuid.New(), "name", int64(2), "png", nil, "image/png", "https://s3/link", 100, true, now, now)
	mockPool.ExpectQuery("SELECT \\* FROM media WHERE id=\\$1").
		WithArgs(int64(5)).
		WillReturnRows(getRows)

	linkRows := pgxmock.NewRows([]string{"link"}).AddRow("https://s3/link")
	mockPool.ExpectQuery("SELECT link FROM media WHERE id=\\$1").
		WithArgs(int64(5)).
		WillReturnRows(linkRows)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	id, err := repo.Save(ctx, models.Media{
		Uid:       uuid.MustParse("00000000-0000-0000-0000-000000000010"),
		Name:      "name",
		Extension: "png",
		MimeType:  "image/png",
		Size:      100,
		Link:      "https://s3/link",
		AuthorID:  2,
	})
	require.NoError(t, err)
	require.Equal(t, int64(5), id)

	got, err := repo.Get(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, int64(5), got.ID)

	link, err := repo.GetLink(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, "https://s3/link", link)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMediaStorageGetNotFound(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewMediaStorage(mockPool)
	rows := pgxmock.NewRows([]string{"id", "uid", "media_name", "author_id", "extension", "description", "mime_type", "link", "size", "is_active", "created_at", "updated_at"})
	mockPool.ExpectQuery("SELECT \\* FROM media WHERE id=\\$1").
		WithArgs(int64(404)).
		WillReturnRows(rows)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	_, err = repo.Get(ctx, 404)
	require.ErrorIs(t, err, xerrors.MediaNotFound)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMediaStorageUpdateLink(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewMediaStorage(mockPool)
	mockPool.ExpectExec("UPDATE table media SET link=\\$1 WHERE id=\\$2").
		WithArgs("new-link", int64(8)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err = repo.UpdateLink(ctx, 8, "new-link")
	require.NoError(t, err)
	require.NoError(t, mockPool.ExpectationsWereMet())
}
