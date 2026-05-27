package http

import (
	"bytes"
	"context"
	json "encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	handlermocks "github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/handler/http/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/usecase"
	"github.com/golang/mock/gomock"
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
			Position: 2, ID: 30, Text: usecase.LocalizedText{RU: "Вопрос", EN: "Question"}, StartedAt: &nextAt, DeadlineAt: &nextAt, HasAnswered: true,
		},
		Players: []usecase.Player{{ProfileID: 10, UserAccountID: 20, Name: "Creator", IsMe: true}},
		Questions: []usecase.RoomQuestion{{
			Position: 1, Status: "done", Question: usecase.Question{ID: 31, Text: usecase.LocalizedText{RU: "Вопрос", EN: "Question"}, CorrectAnswer: 42, IsActive: true}, WinnerProfileID: &winnerID, StartedAt: &nextAt, DeadlineAt: &nextAt, CompletedAt: &nextAt, Answers: []usecase.Answer{{ProfileID: 10, Answer: 42, Distance: 0, AnsweredAt: nextAt, ResponseTimeMs: 100}},
		}},
		RatingChanges: []usecase.RatingChange{{ProfileID: 10, Score: 5, Place: 1, BeforeRating: 1000, AfterRating: 1010, RatingDelta: 10, RatingWeight: 1, SeasonNumber: 1, SeasonTitle: "Season"}},
		ProfileStats:  usecase.Stats{Played: 1, Won: 1},
		CreatedAt:     nextAt,
		UpdatedAt:     nextAt,
		FinishedAt:    &nextAt,
	}

	resp := mapRoom(room, "ru")
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
	input := mapQuestionInput(questionRequest{GameType: "geo", Text: localizedTextPayload{RU: "Вопрос", EN: "Question"}, CorrectAnswer: 42, IsActive: &active})
	if input.IsActive || input.Text.RU != "Вопрос" || input.Text.EN != "Question" {
		t.Fatalf("unexpected question input: %+v", input)
	}
	if mapQuestionInput(questionRequest{}).IsActive != true {
		t.Fatal("question input should default to active")
	}

	board := mapLeaderboard(usecase.Leaderboard{
		GameType: "geo",
		Season:   usecase.RatingSeason{SeasonNumber: 1, Title: "Season", StartsAt: "start", EndsAt: "end"},
		Entries:  []usecase.LeaderboardEntry{{Rank: 1, ProfileID: 10, Player: usecase.Player{ProfileID: 10, Name: "Ann"}, Rating: 1000, GamesPlayed: 3, Wins: 2, Draws: 1}},
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

// ---------------------------------------------------------------------------
// TestGameHandlerEndpoints – HTTP integration smoke tests using a real router
// ---------------------------------------------------------------------------

func newGameRouter(t *testing.T) (*chi.Mux, *Handler) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	svc := handlermocks.NewMockGameService(ctrl)
	svc.EXPECT().SetNotifier(gomock.Any()).AnyTimes()
	svc.EXPECT().CreateRoom(gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Room{}, nil).AnyTimes()
	svc.EXPECT().JoinRoom(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Room{}, nil).AnyTimes()
	svc.EXPECT().ListRooms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	svc.EXPECT().GetRoom(gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Room{}, nil).AnyTimes()
	svc.EXPECT().DisbandRoom(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().LeaveRoom(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().KickPlayer(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().SetReady(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().SetReplayReady(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Room{}, nil).AnyTimes()
	svc.EXPECT().UpdateRoomPassword(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().UpdateRoomTitle(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().UpdateRoomRanked(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().AssignAdmin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().StartRoom(gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Room{}, nil).AnyTimes()
	svc.EXPECT().SubmitAnswer(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Room{}, nil).AnyTimes()
	svc.EXPECT().PauseRoom(gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Room{}, nil).AnyTimes()
	svc.EXPECT().ForceResumeRoom(gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Room{}, nil).AnyTimes()
	svc.EXPECT().ListRoomMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	svc.EXPECT().SendRoomMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.RoomMessage{}, nil).AnyTimes()
	svc.EXPECT().History(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	svc.EXPECT().Stats(gomock.Any(), gomock.Any()).Return(usecase.Stats{}, nil).AnyTimes()
	svc.EXPECT().Leaderboard(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Leaderboard{}, nil).AnyTimes()
	svc.EXPECT().ListQuestions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	svc.EXPECT().CreateQuestion(gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Question{}, nil).AnyTimes()
	svc.EXPECT().UpdateQuestion(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(usecase.Question{}, nil).AnyTimes()
	svc.EXPECT().DeleteQuestion(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	h := New(svc, nil)
	router := chi.NewRouter()
	h.RegisterRoutes(router, nil)
	return router, h
}

func serveGame(t *testing.T, router *chi.Mux, method, path string, body any, userID int64) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestGameHandlerEndpoints(t *testing.T) {
	router, _ := newGameRouter(t)

	const uid = int64(5)

	cases := []struct {
		name   string
		method string
		path   string
		body   any
		userID int64
	}{
		{"CreateRoom", http.MethodPost, "/games/rooms", map[string]any{"title": "Room1"}, uid},
		{"ListRooms", http.MethodGet, "/games/rooms", nil, uid},
		{"GetRoom", http.MethodGet, "/games/rooms/1", nil, uid},
		{"DisbandRoom", http.MethodDelete, "/games/rooms/1", nil, uid},
		{"LeaveRoom", http.MethodDelete, "/games/rooms/1/members/me", nil, uid},
		{"KickPlayer", http.MethodDelete, "/games/rooms/1/members/3", nil, uid},
		{"SetReady", http.MethodPatch, "/games/rooms/1/ready", map[string]any{"ready": true}, uid},
		{"SetReplayReady", http.MethodPatch, "/games/rooms/1/replay", map[string]any{"ready": false}, uid},
		{"UpdateRoomPassword", http.MethodPatch, "/games/rooms/1/password", map[string]any{"password": ""}, uid},
		{"UpdateRoomTitle", http.MethodPatch, "/games/rooms/1/title", map[string]any{"title": "NewTitle"}, uid},
		{"UpdateRoomRanked", http.MethodPatch, "/games/rooms/1/ranked", map[string]any{"isRanked": false}, uid},
		{"AssignAdmin", http.MethodPatch, "/games/rooms/1/admin", map[string]any{"profileId": "3"}, uid},
		{"StartRoom", http.MethodPost, "/games/rooms/1/start", nil, uid},
		{"SubmitAnswer", http.MethodPost, "/games/rooms/1/answers", map[string]any{"value": 42}, uid},
		{"PauseRoom", http.MethodPost, "/games/rooms/1/pause", nil, uid},
		{"ForceResumeRoom", http.MethodPost, "/games/rooms/1/force-resume", nil, uid},
		{"ListRoomMessages", http.MethodGet, "/games/rooms/1/messages", nil, uid},
		{"SendRoomMessage", http.MethodPost, "/games/rooms/1/messages", map[string]any{"text": "hi"}, uid},
		{"Leaderboard", http.MethodGet, "/games/ratings/geo/leaderboard", nil, uid},
		{"History", http.MethodGet, "/games/history", nil, uid},
		{"Stats", http.MethodGet, "/games/stats", nil, uid},
		{"ListQuestions", http.MethodGet, "/games/questions", nil, uid},
		{"CreateQuestion", http.MethodPost, "/games/questions", map[string]any{"gameType": "geo", "text": "Q?", "correctAnswer": 100}, uid},
		{"UpdateQuestion", http.MethodPatch, "/games/questions/1", map[string]any{"gameType": "geo", "text": "Q?", "correctAnswer": 100}, uid},
		{"DeleteQuestion", http.MethodDelete, "/games/questions/1", nil, uid},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rr := serveGame(t, router, tc.method, tc.path, tc.body, tc.userID)
			if rr.Code == 0 {
				t.Fatalf("%s %s: response code is 0 (handler did not run)", tc.method, tc.path)
			}
		})
	}
}

func TestGameHandlerUnauthorized(t *testing.T) {
	router, _ := newGameRouter(t)

	// key endpoints without user_id should return 401
	unauthPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/games/rooms"},
		{http.MethodPost, "/games/rooms"},
		{http.MethodGet, "/games/rooms/1"},
		{http.MethodDelete, "/games/rooms/1"},
		{http.MethodGet, "/games/history"},
		{http.MethodGet, "/games/stats"},
	}

	for _, tc := range unauthPaths {
		tc := tc
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			rr := serveGame(t, router, tc.method, tc.path, nil, 0) // no userID
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 without user_id, got %d for %s %s", rr.Code, tc.method, tc.path)
			}
		})
	}
}
