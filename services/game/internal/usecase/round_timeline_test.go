package usecase

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
)

func TestRoundResultTransitionDuration(t *testing.T) {
	startedAt := time.Date(2026, time.May, 25, 0, 0, 0, 0, time.UTC)
	members := []model.RoomMember{
		{ProfileID: 1},
		{ProfileID: 2},
	}
	answers := []model.Answer{
		{ProfileID: 1, Answer: 2, Distance: 0, AnsweredAt: startedAt.Add(900 * time.Millisecond)},
		{ProfileID: 2, Answer: 4, Distance: 2, AnsweredAt: startedAt.Add(800 * time.Millisecond)},
	}

	got := roundResultTransitionDuration(members, answers, &startedAt)

	if got != 14*time.Second {
		t.Fatalf("roundResultTransitionDuration() = %s, want 14s", got)
	}
}

func TestRoundResultTransitionDurationWithMissingAnswers(t *testing.T) {
	startedAt := time.Date(2026, time.May, 25, 0, 0, 0, 0, time.UTC)
	members := []model.RoomMember{
		{ProfileID: 1},
		{ProfileID: 2},
	}

	got := roundResultTransitionDuration(members, nil, &startedAt)

	if got != 11450*time.Millisecond {
		t.Fatalf("roundResultTransitionDuration() = %s, want 11.45s", got)
	}
}
