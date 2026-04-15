package chat

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSQLChatRepoSave(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewChatStorage(mockPool)
	chat := &models.Chat{
		Uid:   uuid.New(),
		Type:  models.PrivateChat,
		Title: "test",
	}

	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(12))
	mockPool.ExpectQuery("INSERT INTO chat").
		WithArgs(chat.Uid, chat.Type, chat.Title, chat.AvatarID).
		WillReturnRows(rows)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err = repo.Save(ctx, chat)
	require.NoError(t, err)
	require.Equal(t, int64(12), chat.ID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSQLChatRepoGetByID(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewChatStorage(mockPool)
		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "uid", "chat_type", "title", "avatar_id", "is_active", "created_at", "updated_at"}).
			AddRow(int64(5), uuid.New(), string(models.PrivateChat), "chat", nil, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM chat WHERE id=\\$1").
			WithArgs(int64(5)).
			WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.GetByID(ctx, 5)
		require.NoError(t, err)
		require.Equal(t, int64(5), got.ID)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewChatStorage(mockPool)
		rows := pgxmock.NewRows([]string{"id", "uid", "chat_type", "title", "avatar_id", "is_active", "created_at", "updated_at"})
		mockPool.ExpectQuery("SELECT \\* FROM chat WHERE id=\\$1").
			WithArgs(int64(404)).
			WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		_, err = repo.GetByID(ctx, 404)
		require.EqualError(t, err, "chat not found")
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestSQLChatRepoDelete(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewChatStorage(mockPool)
	mockPool.ExpectExec("UPDATE chat SET is_active=false WHERE id=\\$1").
		WithArgs(int64(7)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err = repo.Delete(ctx, 7)
	require.NoError(t, err)
	require.NoError(t, mockPool.ExpectationsWereMet())
}
