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

	got := roundResultTransitionDuration(7, members, answers, &startedAt)

	if got != 7*time.Second {
		t.Fatalf("roundResultTransitionDuration() = %s, want 7s", got)
	}
}

func TestRoundResultTransitionDurationWithMissingAnswers(t *testing.T) {
	startedAt := time.Date(2026, time.May, 25, 0, 0, 0, 0, time.UTC)
	members := []model.RoomMember{
		{ProfileID: 1},
		{ProfileID: 2},
	}

	got := roundResultTransitionDuration(0, members, nil, &startedAt)

	if got != 5*time.Second {
		t.Fatalf("roundResultTransitionDuration() = %s, want 5s", got)
	}
}

func TestRoundResultTransitionDurationWithManyPlayers(t *testing.T) {
	startedAt := time.Date(2026, time.May, 25, 0, 0, 0, 0, time.UTC)
	members := make([]model.RoomMember, 20)
	answers := make([]model.Answer, 20)
	for index := range members {
		profileID := int64(index + 1)
		members[index] = model.RoomMember{ProfileID: profileID}
		answers[index] = model.Answer{
			ProfileID:  profileID,
			Answer:     float64(index),
			Distance:   float64(index),
			AnsweredAt: startedAt.Add(time.Duration(index) * 100 * time.Millisecond),
		}
	}

	got := roundResultTransitionDuration(90, members, answers, &startedAt)

	if got != 60*time.Second {
		t.Fatalf("roundResultTransitionDuration() = %s, want 60s", got)
	}
}
