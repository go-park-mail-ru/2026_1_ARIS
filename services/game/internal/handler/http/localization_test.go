package http

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/usecase"
)

func TestQuestionRequestAcceptsLocalizedPayload(t *testing.T) {
	var req questionRequest
	err := json.Unmarshal([]byte(`{
		"gameType": "number_duel",
		"text": {"ru": "Сколько клеток на шахматной доске?", "en": "How many squares are there on a chessboard?"},
		"correctAnswer": 64
	}`), &req)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if req.Text.RU != "Сколько клеток на шахматной доске?" || req.Text.EN != "How many squares are there on a chessboard?" {
		t.Fatalf("unexpected localized text: %#v", req.Text)
	}
}

func TestQuestionRequestKeepsLegacyStringPayload(t *testing.T) {
	var req questionRequest
	err := json.Unmarshal([]byte(`{
		"gameType": "number_duel",
		"text": "Сколько клеток на шахматной доске?",
		"correctAnswer": 64
	}`), &req)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if req.Text.RU != req.Text.EN || req.Text.RU == "" {
		t.Fatalf("legacy text was not mirrored: %#v", req.Text)
	}
}

func TestRequestLanguage(t *testing.T) {
	req := httptest.NewRequest(nethttp.MethodGet, "/api/games/rooms?lang=en", nil)
	req.Header.Set("Accept-Language", "ru-RU")
	if language := requestLanguage(req); language != "en" {
		t.Fatalf("expected query language to win, got %q", language)
	}

	req = httptest.NewRequest(nethttp.MethodGet, "/api/games/rooms", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,ru;q=0.5")
	if language := requestLanguage(req); language != "en" {
		t.Fatalf("expected english accept-language, got %q", language)
	}
}

func TestMapRoomKeepsZeroCorrectAnswerForCompletedQuestion(t *testing.T) {
	completedAt := "2026-05-30T10:00:00Z"
	room := usecase.Room{
		ID:         1,
		GameType:   "number_duel",
		Status:     "active",
		MaxPlayers: 2,
		Questions: []usecase.RoomQuestion{
			{
				Position:    1,
				Status:      "completed",
				CompletedAt: &completedAt,
				Question: usecase.Question{
					ID:            10,
					Text:          usecase.LocalizedText{RU: "Сколько животных каждого вида взял Моисей на ковчег?", EN: "How many animals of each kind did Moses take onto the ark?"},
					CorrectAnswer: 0,
					IsActive:      true,
				},
			},
			{
				Position: 2,
				Status:   "active",
				Question: usecase.Question{
					ID:            11,
					Text:          usecase.LocalizedText{RU: "Скрытый вопрос", EN: "Hidden question"},
					CorrectAnswer: 42,
					IsActive:      true,
				},
			},
		},
	}

	data, err := json.Marshal(mapRoom(room, "ru"))
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"correctAnswer":0`) {
		t.Fatalf("expected completed zero answer in payload: %s", body)
	}
	if strings.Contains(body, `"correctAnswer":42`) {
		t.Fatalf("active question leaked correct answer: %s", body)
	}
}

func TestMapRoomLocalizesQuestionText(t *testing.T) {
	room := usecase.Room{
		ID:         1,
		GameType:   "number_duel",
		Status:     "active",
		MaxPlayers: 2,
		CurrentQuestion: &usecase.CurrentQuestion{
			ID:       10,
			Position: 1,
			Text: usecase.LocalizedText{
				RU: "Сколько клеток на шахматной доске?",
				EN: "How many squares are there on a chessboard?",
			},
		},
		Questions: []usecase.RoomQuestion{
			{
				Position: 1,
				Status:   "completed",
				Question: usecase.Question{
					ID: 10,
					Text: usecase.LocalizedText{
						RU: "Сколько клеток на шахматной доске?",
						EN: "How many squares are there on a chessboard?",
					},
					CorrectAnswer: 64,
					IsActive:      true,
				},
			},
		},
	}

	resp := mapRoom(room, "en")
	if resp.CurrentQuestion == nil {
		t.Fatal("expected localized current question")
	}
	if resp.CurrentQuestion.Text != "How many squares are there on a chessboard?" {
		t.Fatalf("unexpected current question text: %q", resp.CurrentQuestion.Text)
	}
	if len(resp.Questions) != 1 || resp.Questions[0].Question.Text != "How many squares are there on a chessboard?" {
		t.Fatalf("unexpected archived question text: %#v", resp.Questions)
	}
}
