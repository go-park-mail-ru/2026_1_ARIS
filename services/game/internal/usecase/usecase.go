package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrRoomFull        = errors.New("room is full")
	ErrAlreadyStarted  = errors.New("game already started")
	ErrAlreadyAnswered = errors.New("answer already submitted")
)

type Notifier func(context.Context, int64)

type Service struct {
	store      repository.Store
	userClient userpb.UserServiceClient
	notify     Notifier
	timers     sync.Map
}

func New(store repository.Store, userClient userpb.UserServiceClient) *Service {
	return &Service{store: store, userClient: userClient}
}

func (s *Service) SetNotifier(notifier Notifier) {
	s.notify = notifier
}

func (s *Service) CreateRoom(ctx context.Context, userAccountID int64, in CreateRoomInput) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	normalizeCreateInput(&in)
	var room model.Room
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		var createErr error
		for i := 0; i < 5; i++ {
			room = model.Room{
				Uid:                uuid.New(),
				InviteCode:         inviteCode(),
				GameType:           in.GameType,
				Status:             model.RoomStatusWaiting,
				CreatedByProfileID: profileID,
				QuestionCount:      in.QuestionCount,
				AnswerTimeoutSec:   in.AnswerTimeoutSec,
			}
			createErr = tx.Rooms.Create(ctx, &room)
			if createErr == nil {
				break
			}
			if !isUniqueViolation(createErr) {
				return createErr
			}
		}
		if createErr != nil {
			return createErr
		}
		return tx.Members.Add(ctx, room.ID, profileID)
	})
	if err != nil {
		return Room{}, err
	}
	return s.GetRoom(ctx, userAccountID, room.ID)
}

func (s *Service) JoinRoom(ctx context.Context, userAccountID int64, inviteCode string) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	code := strings.ToUpper(strings.TrimSpace(inviteCode))
	if code == "" {
		return Room{}, ErrInvalidInput
	}
	var roomID int64
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetByInviteCode(ctx, code)
		if err != nil {
			return mapRepoErr(err)
		}
		roomID = room.ID
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		members, err := tx.Members.List(ctx, room.ID)
		if err != nil {
			return err
		}
		for _, member := range members {
			if member.ProfileID == profileID {
				return nil
			}
		}
		if len(members) >= 2 {
			return ErrRoomFull
		}
		return tx.Members.Add(ctx, room.ID, profileID)
	})
	if err != nil {
		return Room{}, err
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoom(ctx, userAccountID, roomID)
}

func (s *Service) StartRoom(ctx context.Context, userAccountID, roomID int64) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	var deadline *time.Time
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.CreatedByProfileID != profileID {
			return ErrForbidden
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		members, err := tx.Members.List(ctx, room.ID)
		if err != nil {
			return err
		}
		if len(members) != 2 {
			return ErrInvalidInput
		}
		questions, err := tx.Questions.Random(ctx, room.GameType, room.QuestionCount)
		if err != nil {
			return err
		}
		if len(questions) < room.QuestionCount {
			return ErrInvalidInput
		}
		for i, question := range questions {
			if err := tx.RoomQs.Add(ctx, room.ID, question.ID, i+1); err != nil {
				return err
			}
		}
		deadline, err = startQuestion(ctx, tx, room, 1)
		return err
	})
	if err != nil {
		return Room{}, err
	}
	if deadline != nil {
		s.scheduleDeadline(roomID, *deadline)
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoom(ctx, userAccountID, roomID)
}

func (s *Service) SubmitAnswer(ctx context.Context, userAccountID, roomID int64, value float64) (Room, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return Room{}, ErrInvalidInput
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	var deadline *time.Time
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusActive {
			return ErrInvalidInput
		}
		if room.QuestionDeadlineAt != nil && time.Now().After(*room.QuestionDeadlineAt) {
			if err := completeActiveQuestion(ctx, tx, room); err != nil {
				return err
			}
			if room.Status == model.RoomStatusActive && room.QuestionDeadlineAt != nil {
				deadline = room.QuestionDeadlineAt
			}
			return nil
		}
		ok, err := tx.Members.IsMember(ctx, roomID, profileID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		active, err := tx.RoomQs.GetActive(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		question, err := tx.Questions.Get(ctx, active.QuestionID)
		if err != nil {
			return mapRepoErr(err)
		}
		answer := model.Answer{
			RoomQuestionID: active.ID,
			ProfileID:      profileID,
			Answer:         value,
			Distance:       math.Abs(value - question.CorrectAnswer),
		}
		if err := tx.Answers.Add(ctx, &answer); err != nil {
			if isUniqueViolation(err) {
				return ErrAlreadyAnswered
			}
			return err
		}
		answerCount, err := tx.Answers.Count(ctx, active.ID)
		if err != nil {
			return err
		}
		members, err := tx.Members.List(ctx, roomID)
		if err != nil {
			return err
		}
		if answerCount >= len(members) {
			if err := completeActiveQuestion(ctx, tx, room); err != nil {
				return err
			}
			if room.Status == model.RoomStatusActive && room.QuestionDeadlineAt != nil {
				deadline = room.QuestionDeadlineAt
			}
		}
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	if deadline != nil {
		s.scheduleDeadline(roomID, *deadline)
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoom(ctx, userAccountID, roomID)
}

func (s *Service) FinalizeExpired(ctx context.Context, roomID int64) error {
	var nextDeadline *time.Time
	changed := false
	err := s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusActive || room.QuestionDeadlineAt == nil || time.Now().Before(*room.QuestionDeadlineAt) {
			return nil
		}
		if err := completeActiveQuestion(ctx, tx, room); err != nil {
			return err
		}
		changed = true
		if room.Status == model.RoomStatusActive && room.QuestionDeadlineAt != nil {
			nextDeadline = room.QuestionDeadlineAt
		}
		return nil
	})
	if err != nil {
		return err
	}
	if nextDeadline != nil {
		s.scheduleDeadline(roomID, *nextDeadline)
	}
	if changed {
		s.notifyRoom(ctx, roomID)
	}
	return nil
}

func (s *Service) GetRoom(ctx context.Context, userAccountID, roomID int64) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	room, err := s.store.Rooms.Get(ctx, roomID)
	if err != nil {
		return Room{}, mapRepoErr(err)
	}
	ok, err := s.store.Members.IsMember(ctx, roomID, profileID)
	if err != nil {
		return Room{}, err
	}
	if !ok {
		return Room{}, ErrForbidden
	}
	if room.Status == model.RoomStatusActive && room.QuestionDeadlineAt != nil && time.Now().After(*room.QuestionDeadlineAt) {
		_ = s.FinalizeExpired(ctx, roomID)
		room, err = s.store.Rooms.Get(ctx, roomID)
		if err != nil {
			return Room{}, mapRepoErr(err)
		}
	}
	return s.buildRoom(ctx, *room, profileID)
}

func (s *Service) ListRooms(ctx context.Context, userAccountID int64, limit, offset int) ([]Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if err := s.finalizeExpiredForProfile(ctx, profileID); err != nil {
		return nil, err
	}
	rooms, err := s.store.Rooms.ListForProfile(ctx, profileID, normalizeLimit(limit, 50, 100), normalizeOffset(offset))
	if err != nil {
		return nil, err
	}
	result := make([]Room, 0, len(rooms))
	for _, room := range rooms {
		view, err := s.buildRoom(ctx, room, profileID)
		if err == nil {
			result = append(result, view)
		}
	}
	return result, nil
}

func (s *Service) History(ctx context.Context, userAccountID int64, limit, offset int) ([]HistoryItem, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if err := s.finalizeExpiredForProfile(ctx, profileID); err != nil {
		return nil, err
	}
	rooms, err := s.store.Rooms.HistoryForProfile(ctx, profileID, normalizeLimit(limit, 50, 100), normalizeOffset(offset))
	if err != nil {
		return nil, err
	}
	result := make([]HistoryItem, 0, len(rooms))
	for _, item := range rooms {
		view, err := s.buildRoom(ctx, item.Room, profileID)
		if err == nil {
			result = append(result, HistoryItem{Room: view, MyScore: item.MyScore, OpponentScore: item.OpponentScore})
		}
	}
	return result, nil
}

func (s *Service) Stats(ctx context.Context, userAccountID int64) (Stats, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Stats{}, err
	}
	if err := s.finalizeExpiredForProfile(ctx, profileID); err != nil {
		return Stats{}, err
	}
	stats, err := s.store.Members.Stats(ctx, profileID)
	if err != nil {
		return Stats{}, err
	}
	return Stats{Played: stats.Played, Won: stats.Won, Lost: stats.Lost, Drawn: stats.Drawn}, nil
}

func (s *Service) finalizeExpiredForProfile(ctx context.Context, profileID int64) error {
	roomIDs, err := s.store.Rooms.ExpiredActiveIDsForProfile(ctx, profileID)
	if err != nil {
		return err
	}
	for _, roomID := range roomIDs {
		if err := s.FinalizeExpired(ctx, roomID); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

func (s *Service) ListQuestions(ctx context.Context, gameType string, includeInactive bool, limit, offset int) ([]Question, error) {
	gameType = normalizeGameType(gameType)
	items, err := s.store.Questions.List(ctx, gameType, includeInactive, normalizeLimit(limit, 100, 200), normalizeOffset(offset))
	if err != nil {
		return nil, err
	}
	return mapQuestions(items), nil
}

func (s *Service) CreateQuestion(ctx context.Context, in QuestionInput) (Question, error) {
	q, err := mapQuestionInput(in)
	if err != nil {
		return Question{}, err
	}
	if err := s.store.Questions.Create(ctx, &q); err != nil {
		return Question{}, err
	}
	return mapQuestion(q), nil
}

func (s *Service) UpdateQuestion(ctx context.Context, id int64, in QuestionInput) (Question, error) {
	if id <= 0 {
		return Question{}, ErrInvalidInput
	}
	q, err := mapQuestionInput(in)
	if err != nil {
		return Question{}, err
	}
	q.ID = id
	if err := s.store.Questions.Update(ctx, &q); err != nil {
		return Question{}, mapRepoErr(err)
	}
	saved, err := s.store.Questions.Get(ctx, id)
	if err != nil {
		return Question{}, mapRepoErr(err)
	}
	return mapQuestion(*saved), nil
}

func (s *Service) profileIDByAccount(ctx context.Context, userAccountID int64) (int64, error) {
	if userAccountID <= 0 {
		return 0, ErrInvalidInput
	}
	resp, err := s.userClient.GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID})
	if err != nil {
		return 0, mapGRPCErr(err)
	}
	return resp.GetProfileId(), nil
}

func (s *Service) buildRoom(ctx context.Context, room model.Room, meProfileID int64) (Room, error) {
	members, err := s.store.Members.List(ctx, room.ID)
	if err != nil {
		return Room{}, err
	}
	stats, err := s.store.Members.Stats(ctx, meProfileID)
	if err != nil {
		return Room{}, err
	}
	answeredProfiles := make(map[int64]bool)
	if room.CurrentQuestionID != nil {
		if active, err := s.store.RoomQs.GetActive(ctx, room.ID); err == nil {
			if answers, err := s.store.Answers.List(ctx, active.ID); err == nil {
				for _, answer := range answers {
					answeredProfiles[answer.ProfileID] = true
				}
			}
		}
	}
	players := make([]Player, 0, len(members))
	for _, member := range members {
		players = append(players, s.player(ctx, member, meProfileID, answeredProfiles[member.ProfileID]))
	}
	roomQuestions, err := s.store.RoomQs.List(ctx, room.ID)
	if err != nil {
		return Room{}, err
	}
	questions := make([]RoomQuestion, 0, len(roomQuestions))
	var current *CurrentQuestion
	for _, rq := range roomQuestions {
		q, err := s.store.Questions.Get(ctx, rq.QuestionID)
		if err != nil {
			continue
		}
		answers := []Answer{}
		if rq.Status == model.QuestionStatusCompleted {
			storedAnswers, _ := s.store.Answers.List(ctx, rq.ID)
			answers = mapAnswers(storedAnswers)
		}
		view := RoomQuestion{
			Position:        rq.Position,
			Status:          rq.Status,
			Question:        mapQuestionForRoom(*q, rq.Status == model.QuestionStatusCompleted),
			WinnerProfileID: rq.WinnerProfileID,
			StartedAt:       timePtrString(rq.StartedAt),
			DeadlineAt:      timePtrString(rq.DeadlineAt),
			CompletedAt:     timePtrString(rq.CompletedAt),
			Answers:         answers,
		}
		questions = append(questions, view)
		if room.CurrentQuestionID != nil && *room.CurrentQuestionID == q.ID && rq.Status == model.QuestionStatusActive {
			current = &CurrentQuestion{
				Position:   rq.Position,
				ID:         q.ID,
				Text:       q.Text,
				AnswerUnit: q.AnswerUnit,
				StartedAt:  timePtrString(rq.StartedAt),
				DeadlineAt: timePtrString(rq.DeadlineAt),
			}
			current.HasAnswered = answeredProfiles[meProfileID]
		}
	}
	return Room{
		ID:                   room.ID,
		InviteCode:           room.InviteCode,
		GameType:             room.GameType,
		Status:               room.Status,
		CreatedByProfileID:   room.CreatedByProfileID,
		WinnerProfileID:      room.WinnerProfileID,
		QuestionCount:        room.QuestionCount,
		AnswerTimeoutSec:     room.AnswerTimeoutSec,
		CurrentQuestionIndex: room.CurrentQuestionIndex,
		CurrentQuestion:      current,
		Players:              players,
		Questions:            questions,
		ProfileStats:         Stats{Played: stats.Played, Won: stats.Won, Lost: stats.Lost, Drawn: stats.Drawn},
		CreatedAt:            room.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:            room.UpdatedAt.Format(time.RFC3339Nano),
		FinishedAt:           timePtrString(room.FinishedAt),
	}, nil
}

func (s *Service) player(ctx context.Context, member model.RoomMember, meProfileID int64, hasAnswered bool) Player {
	player := Player{ProfileID: member.ProfileID, Score: member.Score, HasAnswered: hasAnswered, IsMe: member.ProfileID == meProfileID}
	summary, err := s.userClient.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: member.ProfileID})
	if err == nil {
		player.UserAccountID = summary.GetUserAccountId()
		player.Username = summary.GetUsername()
		player.FirstName = summary.GetFirstName()
		player.LastName = summary.GetLastName()
		player.Name = strings.TrimSpace(summary.GetFirstName() + " " + summary.GetLastName())
		if player.Name == "" {
			player.Name = summary.GetUsername()
		}
		player.AvatarID = summary.AvatarId
	}
	return player
}

func startQuestion(ctx context.Context, tx repository.Store, room *model.Room, position int) (*time.Time, error) {
	rq, err := tx.RoomQs.GetByPosition(ctx, room.ID, position)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	now := time.Now()
	deadline := now.Add(time.Duration(room.AnswerTimeoutSec) * time.Second)
	rq.Status = model.QuestionStatusActive
	rq.StartedAt = &now
	rq.DeadlineAt = &deadline
	if err := tx.RoomQs.Update(ctx, rq); err != nil {
		return nil, err
	}
	room.Status = model.RoomStatusActive
	room.CurrentQuestionIndex = position
	room.CurrentQuestionID = &rq.QuestionID
	room.QuestionStartedAt = &now
	room.QuestionDeadlineAt = &deadline
	room.FinishedAt = nil
	if err := tx.Rooms.Update(ctx, room); err != nil {
		return nil, err
	}
	return &deadline, nil
}

func completeActiveQuestion(ctx context.Context, tx repository.Store, room *model.Room) error {
	active, err := tx.RoomQs.GetActive(ctx, room.ID)
	if err != nil {
		return mapRepoErr(err)
	}
	answers, err := tx.Answers.List(ctx, active.ID)
	if err != nil {
		return err
	}
	winner := answerWinner(answers)
	now := time.Now()
	active.Status = model.QuestionStatusCompleted
	active.WinnerProfileID = winner
	active.CompletedAt = &now
	if err := tx.RoomQs.Update(ctx, active); err != nil {
		return err
	}
	if winner != nil {
		if err := tx.Members.IncrementScore(ctx, room.ID, *winner); err != nil {
			return err
		}
	}
	if active.Position >= room.QuestionCount {
		members, err := tx.Members.List(ctx, room.ID)
		if err != nil {
			return err
		}
		roomWinner := scoreWinner(members)
		room.Status = model.RoomStatusFinished
		room.WinnerProfileID = roomWinner
		room.CurrentQuestionID = nil
		room.QuestionStartedAt = nil
		room.QuestionDeadlineAt = nil
		room.FinishedAt = &now
		return tx.Rooms.Update(ctx, room)
	}
	_, err = startQuestion(ctx, tx, room, active.Position+1)
	return err
}

func answerWinner(answers []model.Answer) *int64 {
	if len(answers) == 0 {
		return nil
	}
	best := answers[0]
	tie := false
	for _, answer := range answers[1:] {
		switch {
		case answer.Distance < best.Distance:
			best = answer
			tie = false
		case answer.Distance == best.Distance:
			tie = true
		}
	}
	if tie {
		return nil
	}
	return &best.ProfileID
}

func scoreWinner(members []model.RoomMember) *int64 {
	if len(members) == 0 {
		return nil
	}
	best := members[0]
	tie := false
	for _, member := range members[1:] {
		switch {
		case member.Score > best.Score:
			best = member
			tie = false
		case member.Score == best.Score:
			tie = true
		}
	}
	if tie {
		return nil
	}
	return &best.ProfileID
}

func (s *Service) scheduleDeadline(roomID int64, deadline time.Time) {
	key := fmt.Sprintf("%d:%d", roomID, deadline.UnixNano())
	if _, loaded := s.timers.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	go func() {
		defer s.timers.Delete(key)
		delay := time.Until(deadline)
		if delay > 0 {
			time.Sleep(delay + 150*time.Millisecond)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.FinalizeExpired(ctx, roomID)
	}()
}

func (s *Service) notifyRoom(ctx context.Context, roomID int64) {
	if s.notify != nil {
		s.notify(ctx, roomID)
	}
}

func mapQuestionInput(in QuestionInput) (model.Question, error) {
	gameType := normalizeGameType(in.GameType)
	slug := strings.TrimSpace(in.Slug)
	text := strings.TrimSpace(in.Text)
	if slug == "" || text == "" || math.IsNaN(in.CorrectAnswer) || math.IsInf(in.CorrectAnswer, 0) {
		return model.Question{}, ErrInvalidInput
	}
	return model.Question{
		Uid:           uuid.New(),
		GameType:      gameType,
		Slug:          slug,
		Text:          text,
		CorrectAnswer: in.CorrectAnswer,
		AnswerUnit:    in.AnswerUnit,
		IsActive:      in.IsActive,
	}, nil
}

func mapQuestions(items []model.Question) []Question {
	result := make([]Question, 0, len(items))
	for _, item := range items {
		result = append(result, mapQuestion(item))
	}
	return result
}

func mapQuestion(item model.Question) Question {
	return Question{
		ID:            item.ID,
		Slug:          item.Slug,
		Text:          item.Text,
		CorrectAnswer: item.CorrectAnswer,
		AnswerUnit:    item.AnswerUnit,
		IsActive:      item.IsActive,
	}
}

func mapQuestionForRoom(item model.Question, revealAnswer bool) Question {
	q := mapQuestion(item)
	if !revealAnswer {
		q.CorrectAnswer = 0
	}
	return q
}

func mapAnswers(items []model.Answer) []Answer {
	result := make([]Answer, 0, len(items))
	for _, item := range items {
		result = append(result, Answer{
			ProfileID:  item.ProfileID,
			Answer:     item.Answer,
			Distance:   item.Distance,
			AnsweredAt: item.AnsweredAt.Format(time.RFC3339Nano),
		})
	}
	return result
}

func normalizeCreateInput(in *CreateRoomInput) {
	in.GameType = normalizeGameType(in.GameType)
	if in.QuestionCount <= 0 || in.QuestionCount > 25 {
		in.QuestionCount = 5
	}
	if in.AnswerTimeoutSec <= 0 {
		in.AnswerTimeoutSec = 10
	}
	if in.AnswerTimeoutSec < 3 {
		in.AnswerTimeoutSec = 3
	}
	if in.AnswerTimeoutSec > 120 {
		in.AnswerTimeoutSec = 120
	}
}

func normalizeGameType(gameType string) string {
	gameType = strings.TrimSpace(gameType)
	if gameType == "" {
		return model.DefaultGameType
	}
	return gameType
}

func normalizeLimit(value, fallback, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func normalizeOffset(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func timePtrString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	out := value.Format(time.RFC3339Nano)
	return &out
}

func inviteCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 6)
	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return strings.ToUpper(uuid.NewString()[:6])
	}
	for i, b := range randomBytes {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(buf)
}

func mapRepoErr(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func mapGRPCErr(err error) error {
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if status.Code(err) == codes.InvalidArgument {
		return ErrInvalidInput
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
