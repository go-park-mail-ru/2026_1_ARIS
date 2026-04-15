package message

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
)

func TestMessageStorage_Save(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	storage := &messageStorage{db: mock}

	newText := "new text"
	uid := uuid.New()
	msg := &models.Message{Uid: uid, Text: &newText, ChatID: 10, AuthorID: 20}
	mock.ExpectQuery("INSERT INTO message").
		WithArgs(uid, &newText, int64(10), int64(20)).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(123)))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err = storage.Save(ctx, msg)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), msg.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageStorage_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	storage := &messageStorage{db: mock}

	mock.ExpectQuery("SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at").
		WithArgs(int64(123)).
		WillReturnError(pgxxErrNoRows())

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	msg, err := storage.GetByID(ctx, int64(123))
	assert.Nil(t, msg)
	assert.EqualError(t, err, "message not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageStorage_GetByChatID_NoMessages(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	storage := &messageStorage{db: mock}

	rows := pgxmock.NewRows([]string{
		"id", "uid", "message_text", "parent_message_id",
		"chat_id", "author_id", "sticker_id", "is_active",
		"created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at").
		WithArgs(int64(10), 20, 0).
		WillReturnRows(rows)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	messages, err := storage.GetByChatID(ctx, int64(10), 20, 0)
	assert.NoError(t, err)
	assert.Empty(t, messages)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageStorage_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	storage := &messageStorage{db: mock}

	mock.ExpectExec("UPDATE message SET is_active=false WHERE id=\\$1").
		WithArgs(int64(123)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err = storage.Delete(ctx, int64(123))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func pgxxErrNoRows() error {
	return pgx.ErrNoRows
}

func TestMessageStorage_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	storage := &messageStorage{db: mock}

	newText := "new text"
	msg := &models.Message{ID: 123, Text: &newText}
	mock.ExpectExec("UPDATE message SET message_text=\\$1, updated_at=\\$2 WHERE id=\\$3 AND is_active=true").
		WithArgs(&newText, pgxmock.AnyArg(), int64(123)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err = storage.Update(ctx, msg)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func strPtr(s string) *string { return &s }
