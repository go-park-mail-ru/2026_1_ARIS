package session

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
)

func TestSessionLifecycle(t *testing.T) {
	repo := session.NewSessionRepo()
	svc := NewSessionService(repo)
	ctx := context.Background()
	userAccountID := int64(44) //models.UserID(42)

	session, err := svc.Create(ctx, userAccountID)
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

	if savedSess.UserID != userAccountID {
		t.Errorf("Expected UserID %d, got %d", userAccountID, savedSess.UserID)
	}
}
