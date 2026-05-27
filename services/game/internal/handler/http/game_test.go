package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/usecase"
)

func TestHandlerRequestHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=12&bad=x", nil)
	if got := queryInt(req, "limit", 50); got != 12 {
		t.Fatalf("queryInt value = %d", got)
	}
	if got := queryInt(req, "bad", 50); got != 50 {
		t.Fatalf("queryInt fallback = %d", got)
	}

	rec := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(9)))
	if userID, ok := userIDFromContext(rec, req); !ok || userID != 9 {
		t.Fatalf("userIDFromContext() = %d, %v", userID, ok)
	}

	rec = httptest.NewRecorder()
	if _, ok := userIDFromContext(rec, httptest.NewRequest(http.MethodGet, "/", nil)); ok || rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing user to be unauthorized, got ok=%v code=%d", ok, rec.Code)
	}

	rec = httptest.NewRecorder()
	if id, ok := parsePathID(rec, "15", "bad id"); !ok || id != 15 {
		t.Fatalf("parsePathID() = %d, %v", id, ok)
	}
	rec = httptest.NewRecorder()
	if _, ok := parsePathID(rec, "0", "bad id"); ok || rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad path id, got ok=%v code=%d", ok, rec.Code)
	}
}

func TestUserAndRoomID(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/games/rooms/42", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(7)))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("roomID", "42")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	userID, roomID, ok := handler.userAndRoomID(rec, req)
	if !ok || userID != 7 || roomID != 42 {
		t.Fatalf("userAndRoomID() = %d, %d, %v", userID, roomID, ok)
	}
}

func TestServiceErrorMapping(t *testing.T) {
	cases := map[error]struct {
		status  int
		message string
	}{
		usecase.ErrInvalidInput:      {http.StatusBadRequest, "неверные данные"},
		usecase.ErrAlreadyAnswered:   {http.StatusBadRequest, "ответ уже отправлен"},
		usecase.ErrAlreadyStarted:    {http.StatusBadRequest, "игра уже началась"},
		usecase.ErrRoomFull:          {http.StatusBadRequest, "комната заполнена"},
		usecase.ErrRoomTitleTaken:    {http.StatusBadRequest, "Комната с таким названием уже существует"},
		usecase.ErrGamePaused:        {http.StatusBadRequest, "игра на паузе"},
		usecase.ErrPauseAlreadyUsed:  {http.StatusBadRequest, "пауза уже использована"},
		usecase.ErrActiveCreatedRoom: {http.StatusConflict, "У вас уже есть своя созданная комната."},
		usecase.ErrForbidden:         {http.StatusForbidden, "доступ запрещён"},
		usecase.ErrNotFound:          {http.StatusNotFound, "не найдено"},
	}
	for err, want := range cases {
		if got := serviceErrorMessage(err); got != want.message {
			t.Fatalf("serviceErrorMessage(%v) = %q, want %q", err, got, want.message)
		}
		rec := httptest.NewRecorder()
		writeServiceError(rec, err)
		if rec.Code != want.status || !bytes.Contains(rec.Body.Bytes(), []byte(want.message)) {
			t.Fatalf("writeServiceError(%v) = %d %s", err, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	writeServiceError(rec, errGameUnexpected{})
	if rec.Code != http.StatusInternalServerError || !bytes.Contains(rec.Body.Bytes(), []byte("internal server error")) {
		t.Fatalf("expected internal error, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestMapRoom(t *testing.T) {
	answerUnit := "km"
	winnerID := int64(10)
	pausedBy := int64(11)
	nextAt := "2026-05-27T10:00:00Z"
	avatarID := int64(100)

	room := usecase.Room{
		ID:                      1,
		Title:                   "Room",
		InviteCode:              "ABC123",
		GameType:                "geo",
		Status:                  "playing",
		CreatedByProfileID:      10,
		WinnerProfileID:         &winnerID,
		MaxPlayers:              4,
		HasPassword:             true,
		Password:                "secret",
		IsRanked:                true,
		QuestionCount:           3,
		AnswerTimeoutSec:        30,
		Creator:                 usecase.Player{ProfileID: 10, UserAccountID: 20, Name: "Creator", AvatarID: &avatarID},
		CurrentQuestionIndex:    1,
		NextQuestionAt:          &nextAt,
		PausedByProfileID:       &pausedBy,
		PauseStartedAt:          &nextAt,
		PauseUntilAt:            &nextAt,
		PauseForceVotes:         1,
		PauseForceVotesRequired: 2,
		CurrentQuestion: &usecase.CurrentQuestion{
			Position: 2, ID: 30, Text: "Question", AnswerUnit: &answerUnit, StartedAt: &nextAt, DeadlineAt: &nextAt, HasAnswered: true,
		},
		Players: []usecase.Player{{ProfileID: 10, UserAccountID: 20, Name: "Creator", IsMe: true}},
		Questions: []usecase.RoomQuestion{{
			Position: 1, Status: "done", Question: usecase.Question{ID: 31, Text: "Q", CorrectAnswer: 42, AnswerUnit: &answerUnit, IsActive: true}, WinnerProfileID: &winnerID, StartedAt: &nextAt, DeadlineAt: &nextAt, CompletedAt: &nextAt, Answers: []usecase.Answer{{ProfileID: 10, Answer: 42, Distance: 0, AnsweredAt: nextAt, ResponseTimeMs: 100}},
		}},
		RatingChanges: []usecase.RatingChange{{ProfileID: 10, Score: 5, Place: 1, BeforeRating: 1000, AfterRating: 1010, RatingDelta: 10, RatingWeight: 1, SeasonNumber: 1, SeasonTitle: "Season"}},
		ProfileStats:  usecase.Stats{Played: 1, Won: 1},
		CreatedAt:     nextAt,
		UpdatedAt:     nextAt,
		FinishedAt:    &nextAt,
	}

	resp := mapRoom(room)
	if resp.ID != "1" || resp.CreatedByProfileID != "10" || resp.WinnerProfileID == nil || *resp.WinnerProfileID != "10" {
		t.Fatalf("unexpected room ids: %+v", resp)
	}
	if resp.CurrentQuestion == nil || resp.CurrentQuestion.ID != "30" || !resp.CurrentQuestion.HasAnswered {
		t.Fatalf("unexpected current question: %+v", resp.CurrentQuestion)
	}
	if len(resp.Players) != 1 || resp.Players[0].ProfileID != "10" || len(resp.Questions) != 1 || len(resp.Questions[0].Answers) != 1 {
		t.Fatalf("unexpected nested room data: %+v", resp)
	}
	if len(resp.RatingChanges) != 1 || resp.ProfileStats.Played != 1 || resp.ProfileStats.Won != 1 {
		t.Fatalf("unexpected rating/stats data: %+v", resp)
	}
}

func TestOtherMappers(t *testing.T) {
	active := false
	unit := "km"
	input := mapQuestionInput(questionRequest{GameType: "geo", Text: "Q", CorrectAnswer: 42, AnswerUnit: &unit, IsActive: &active})
	if input.IsActive || input.AnswerUnit == nil || *input.AnswerUnit != "km" {
		t.Fatalf("unexpected question input: %+v", input)
	}
	if mapQuestionInput(questionRequest{}).IsActive != true {
		t.Fatal("question input should default to active")
	}

	board := mapLeaderboard(usecase.Leaderboard{
		GameType: "geo",
		Season:  usecase.RatingSeason{SeasonNumber: 1, Title: "Season", StartsAt: "start", EndsAt: "end"},
		Entries: []usecase.LeaderboardEntry{{Rank: 1, ProfileID: 10, Player: usecase.Player{ProfileID: 10, Name: "Ann"}, Rating: 1000, GamesPlayed: 3, Wins: 2, Draws: 1}},
	})
	if board.GameType != "geo" || len(board.Entries) != 1 || board.Entries[0].ProfileID != "10" {
		t.Fatalf("unexpected leaderboard: %+v", board)
	}

	message := mapRoomMessage(usecase.RoomMessage{ID: 1, RoomID: 2, ProfileID: 3, Text: "hello", Author: usecase.Player{UserAccountID: 4, Name: "Ann"}, CreatedAt: "now"})
	if message.ID != "1" || message.RoomID != "2" || message.AuthorUserAccountID != "4" {
		t.Fatalf("unexpected room message: %+v", message)
	}
	if !bytes.Contains(marshalEvent(socketEvent{Type: "error", Error: "bad"}), []byte(`"error":"bad"`)) {
		t.Fatal("expected marshalled socket error")
	}
}

func TestDisconnectHelpers(t *testing.T) {
	player := disconnectingPlayer(usecase.Room{Players: []usecase.Player{{ProfileID: 1}, {ProfileID: 2, IsMe: true}}})
	if player.ProfileID != 2 {
		t.Fatalf("unexpected disconnecting player: %+v", player)
	}
	if playerDisplayName(usecase.Player{Name: "Display"}) != "Display" {
		t.Fatal("expected display name")
	}
	if playerDisplayName(usecase.Player{FirstName: "Ann", LastName: "User"}) != "Ann User" {
		t.Fatal("expected full name")
	}
	if playerDisplayName(usecase.Player{Username: "ann"}) != "ann" {
		t.Fatal("expected username")
	}
	if playerDisplayName(usecase.Player{}) != "игроком" {
		t.Fatal("expected default player display")
	}
}

func TestInt64StringHelpers(t *testing.T) {
	if int64String(0) != "" || int64String(5) != "5" {
		t.Fatal("unexpected int64String result")
	}
	if int64PtrString(nil) != nil {
		t.Fatal("expected nil int64PtrString")
	}
	value := int64(6)
	if got := int64PtrString(&value); got == nil || *got != "6" {
		t.Fatalf("unexpected int64PtrString result: %#v", got)
	}
	if *ptr("value") != "value" {
		t.Fatal("unexpected ptr result")
	}
}

type errGameUnexpected struct{}

func (errGameUnexpected) Error() string { return "unexpected" }
