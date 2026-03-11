package repository

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSessionRepo(t *testing.T) {
	repo := NewSessionRepo().(*inmemorySessionRepo)
	session := models.Session{
		SessionID: "test-id",
		UserID:    uuid.New(),
		CreatedAt: time.Now(),
		ExpiredAt: time.Now().Add(time.Hour),
	}

	repo.Save(context.Background(), session)
	got, err := repo.GetByID(context.Background(), "test-id")
	assert.NoError(t, err)
	assert.Equal(t, session.UserID, got.UserID)

	err = repo.Delete(context.Background(), "test-id")
	assert.NoError(t, err)

	_, err = repo.GetByID(context.Background(), "test-id")
	assert.Error(t, err)
}
