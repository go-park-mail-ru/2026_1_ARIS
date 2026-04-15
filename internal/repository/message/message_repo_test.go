package message

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestMessageStorageSave(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewMessageStorage(mockPool)
	text := "hello"
	msg := &models.Message{
		Uid:      uuid.New(),
		Text:     &text,
		ChatID:   1,
		AuthorID: 2,
	}

	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(77))
	mockPool.ExpectQuery("INSERT INTO message").
		WithArgs(msg.Uid, msg.Text, msg.ChatID, msg.AuthorID).
		WillReturnRows(rows)

	err = repo.Save(context.Background(), msg)
	require.NoError(t, err)
	require.Equal(t, int64(77), msg.ID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageStorageGetByID(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewMessageStorage(mockPool)
	now := time.Now()
	text := "text"
	rows := pgxmock.NewRows([]string{"id", "uid", "message_text", "parent_message_id", "chat_id", "author_id", "sticker_id", "is_active", "created_at", "updated_at"}).
		AddRow(int64(9), uuid.New(), &text, nil, int64(2), int64(3), nil, true, now, now)
	mockPool.ExpectQuery("SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at").
		WithArgs(int64(9)).
		WillReturnRows(rows)

	got, err := repo.GetByID(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, int64(9), got.ID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageStorageGetByChatID(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewMessageStorage(mockPool)
	now := time.Now()
	text := "text"
	rows := pgxmock.NewRows([]string{"id", "uid", "message_text", "parent_message_id", "chat_id", "author_id", "sticker_id", "is_active", "created_at", "updated_at"}).
		AddRow(int64(1), uuid.New(), &text, nil, int64(7), int64(3), nil, true, now, now)
	mockPool.ExpectQuery("SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at").
		WithArgs(int64(7), 20, 0).
		WillReturnRows(rows)

	msgs, err := repo.GetByChatID(context.Background(), 7, 20, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageStorageUpdateDelete(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewMessageStorage(mockPool)
	text := "updated"

	mockPool.ExpectExec("UPDATE message SET message_text=\\$1, updated_at=\\$2 WHERE id=\\$3 AND is_active=true").
		WithArgs(&text, pgxmock.AnyArg(), int64(10)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mockPool.ExpectExec("UPDATE message SET is_active=false WHERE id=\\$1").
		WithArgs(int64(10)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), &models.Message{ID: 10, Text: &text})
	require.NoError(t, err)
	err = repo.Delete(context.Background(), 10)
	require.NoError(t, err)
	require.NoError(t, mockPool.ExpectationsWereMet())
}
