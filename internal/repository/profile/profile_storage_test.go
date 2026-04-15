package profile

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

func TestProfileStorageSave(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewProfileStorage(mockPool)
	avatarID := int64(123)

	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(55))
	mockPool.ExpectQuery("INSERT INTO profile").
		WithArgs(pgxmock.AnyArg(), &avatarID).
		WillReturnRows(rows)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	gotID, err := repo.Save(ctx, models.Profile{AvatarID: &avatarID})
	require.NoError(t, err)
	assert.Equal(t, int64(55), gotID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestProfileStorageUpdateAvatar(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		affected  int64
		wantError bool
	}{
		{name: "ok", affected: 1, wantError: false},
		{name: "no rows affected", affected: 0, wantError: true},
		{name: "multiple rows affected", affected: 2, wantError: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			repo := NewProfileStorage(mockPool)
			avatarID := int64(777)

			mockPool.ExpectExec("UPDATE profile SET avatar_id = \\$1, updated_at = NOW\\(\\) WHERE id = \\$2").
				WithArgs(&avatarID, int64(10)).
				WillReturnResult(pgxmock.NewResult("UPDATE", tc.affected))

			mockLogger := zap.NewNop()
			ctx := logger.WithLogger(context.Background(), mockLogger)

			err = repo.UpdateAvatar(ctx, 10, &avatarID)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestProfileStorageGet(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewProfileStorage(mockPool)
		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "uid", "avatar_id", "is_active", "created_at", "updated_at"}).
			AddRow(int64(3), uuid.New(), nil, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM profile WHERE id=\\$1;").
			WithArgs(int64(3)).
			WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.Get(ctx, 3)
		require.NoError(t, err)
		require.Equal(t, int64(3), got.ID)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewProfileStorage(mockPool)
		rows := pgxmock.NewRows([]string{"id", "uid", "avatar_id", "is_active", "created_at", "updated_at"})
		mockPool.ExpectQuery("SELECT \\* FROM profile WHERE id=\\$1;").
			WithArgs(int64(404)).
			WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		_, err = repo.Get(ctx, 404)
		require.ErrorIs(t, err, xerrors.ProfileNotFound)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestProfileStorageGetByUserAccountID(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewProfileStorage(mockPool)
		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "uid", "avatar_id", "is_active", "created_at", "updated_at"}).
			AddRow(int64(8), uuid.New(), nil, true, now, now)
		mockPool.ExpectQuery("select p.id, p.uid, p.avatar_id, p.is_active, p.created_at, p.updated_at from user_account ua join user_profile up on up.user_account_id=ua.id join profile p on up.profile_id=p.id where ua.id=\\$1;").
			WithArgs(int64(44)).
			WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.GetByUserAccountID(ctx, 44)
		require.NoError(t, err)
		require.Equal(t, int64(8), got.ID)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewProfileStorage(mockPool)
		rows := pgxmock.NewRows([]string{"id", "uid", "avatar_id", "is_active", "created_at", "updated_at"})
		mockPool.ExpectQuery("select p.id, p.uid, p.avatar_id, p.is_active, p.created_at, p.updated_at from user_account ua join user_profile up on up.user_account_id=ua.id join profile p on up.profile_id=p.id where ua.id=\\$1;").
			WithArgs(int64(404)).
			WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		_, err = repo.GetByUserAccountID(ctx, 404)
		require.ErrorIs(t, err, xerrors.ProfileNotFound)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestProfileStorageGetAll(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewProfileStorage(mockPool)
		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "uid", "avatar_id", "is_active", "created_at", "updated_at"}).
			AddRow(int64(1), uuid.New(), nil, true, now, now).
			AddRow(int64(2), uuid.New(), nil, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM profile").
			WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewProfileStorage(mockPool)
		mockPool.ExpectQuery("SELECT \\* FROM profile").
			WillReturnError(errors.New("db error"))

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		_, err = repo.GetAll(ctx)
		require.EqualError(t, err, "db error")
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}
