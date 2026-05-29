package logger

import (
	"context"
	"testing"
)

func TestContextLogger(t *testing.T) {
	log, err := New("debug")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if FromContext(context.Background()) != nil {
		t.Fatal("expected empty context to have no logger")
	}
	ctx := WithLogger(context.Background(), log)
	if FromContext(ctx) != log {
		t.Fatal("expected logger from context")
	}
}

func TestNewFallbackLevel(t *testing.T) {
	log, err := New("not-a-level")
	if err != nil {
		t.Fatalf("New() should fall back on invalid level, got %v", err)
	}
	if log == nil {
		t.Fatal("expected logger")
	}
}
