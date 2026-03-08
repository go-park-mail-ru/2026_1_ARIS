package service

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
)

func TestSessionLifecycle(t *testing.T) {
	repo := repository.NewRepository().SessionRepo
	svc := NewSessionService(repo)
	ctx := context.Background()
	userID := models.UserID(42)

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
