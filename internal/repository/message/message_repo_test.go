package message

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

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

	err = storage.Update(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func strPtr(s string) *string { return &s }
