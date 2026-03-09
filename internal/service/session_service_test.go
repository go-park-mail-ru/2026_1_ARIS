package service

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/google/uuid"
)

func TestSessionLifecycle(t *testing.T) {
	repo := repository.NewSessionRepo()
	svc := NewSessionService(repo)
	ctx := context.Background()
	userID := uuid.New() //models.UserID(42)

	session, err := svc.Create(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.SessionID == "" {
		t.Error("Expected non-empty SessionID (UUID)")
	}

	savedSess, err := svc.Get(ctx, session.SessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if savedSess.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, savedSess.UserID)
	}
}
