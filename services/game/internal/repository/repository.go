package repository

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks DB,QuestionRepo,RoomRepo,MemberRepo,RoomQuestionRepo,AnswerRepo,MessageRepo,RatingRepo
//go:generate mockgen -destination=mocks/pgx_mock.go -package=mocks github.com/jackc/pgx/v5 Row,Rows

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Store struct {
	db        DB
	pool      *pgxpool.Pool
	Questions QuestionRepo
	Rooms     RoomRepo
	Members   MemberRepo
	RoomQs    RoomQuestionRepo
	Answers   AnswerRepo
	Messages  MessageRepo
	Ratings   RatingRepo
}

func NewStore(db *pgxpool.Pool) Store {
	return newStore(db, db)
}

func newStore(db DB, pool *pgxpool.Pool) Store {
	return Store{
		db:        db,
		pool:      pool,
		Questions: NewQuestionStorage(db),
		Rooms:     NewRoomStorage(db),
		Members:   NewMemberStorage(db),
		RoomQs:    NewRoomQuestionStorage(db),
		Answers:   NewAnswerStorage(db),
		Messages:  NewMessageStorage(db),
		Ratings:   NewRatingStorage(db),
	}
}

func (s Store) InTx(ctx context.Context, fn func(Store) error) error {
	if s.pool == nil {
		return fn(s)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(newStore(tx, nil)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type QuestionRepo interface {
	Create(ctx context.Context, q *model.Question) error
	Update(ctx context.Context, q *model.Question) error
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (*model.Question, error)
	List(ctx context.Context, gameType string, includeInactive bool, limit, offset int) ([]model.Question, error)
	Random(ctx context.Context, gameType string, limit int) ([]model.Question, error)
}

type questionStorage struct {
	db DB
}

const questionColumns = `
	id, uid, game_type, question_text, correct_answer, answer_unit, is_active, created_at, updated_at
`

func NewQuestionStorage(db DB) QuestionRepo {
	return &questionStorage{db: db}
}

func (s *questionStorage) Create(ctx context.Context, q *model.Question) error {
	if q.Uid == uuid.Nil {
		q.Uid = uuid.New()
	}
	return s.db.QueryRow(ctx, `
		INSERT INTO game_question (uid, game_type, question_text, correct_answer, answer_unit, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, q.Uid, q.GameType, q.Text, q.CorrectAnswer, q.AnswerUnit, q.IsActive).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
}

func (s *questionStorage) Update(ctx context.Context, q *model.Question) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_question
		SET game_type=$1, question_text=$2, correct_answer=$3, answer_unit=$4, is_active=$5, updated_at=NOW()
		WHERE id=$6
	`, q.GameType, q.Text, q.CorrectAnswer, q.AnswerUnit, q.IsActive, q.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *questionStorage) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_question
		SET is_active=false, updated_at=NOW()
		WHERE id=$1
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *questionStorage) Get(ctx context.Context, id int64) (*model.Question, error) {
	var q model.Question
	err := pgxscan.Get(ctx, s.db, &q, `SELECT `+questionColumns+` FROM game_question WHERE id=$1`, id)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &q, nil
}

func (s *questionStorage) List(ctx context.Context, gameType string, includeInactive bool, limit, offset int) ([]model.Question, error) {
	var questions []model.Question
	err := pgxscan.Select(ctx, s.db, &questions, `
		SELECT `+questionColumns+`
		FROM game_question
		WHERE game_type=$1 AND ($2 OR is_active=true)
		ORDER BY id DESC
		LIMIT $3 OFFSET $4
	`, gameType, includeInactive, limit, offset)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return questions, nil
}

func (s *questionStorage) Random(ctx context.Context, gameType string, limit int) ([]model.Question, error) {
	var questions []model.Question
	err := pgxscan.Select(ctx, s.db, &questions, `
		SELECT `+questionColumns+`
		FROM game_question
		WHERE game_type=$1 AND is_active=true
		ORDER BY random()
		LIMIT $2
	`, gameType, limit)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return questions, nil
}

type RoomRepo interface {
	Create(ctx context.Context, room *model.Room) error
	Get(ctx context.Context, id int64) (*model.Room, error)
	GetByInviteCode(ctx context.Context, code string) (*model.Room, error)
	GetForUpdate(ctx context.Context, id int64) (*model.Room, error)
	GetWaitingCreatedByProfile(ctx context.Context, profileID int64) (*model.Room, error)
	ListForProfile(ctx context.Context, profileID int64, limit, offset int) ([]model.Room, error)
	ExpiredActiveIDsForProfile(ctx context.Context, profileID int64) ([]int64, error)
	HistoryForProfile(ctx context.Context, profileID int64, limit, offset int) ([]model.HistoryRoom, error)
	Update(ctx context.Context, room *model.Room) error
	UpdateTitle(ctx context.Context, roomID int64, title string) error
	UpdateAdmin(ctx context.Context, roomID, profileID int64) error
	Deactivate(ctx context.Context, roomID int64) error
	TouchEmptyWaiting(ctx context.Context, roomID int64) error
	DeactivateEmptyWaitingOlderThan(ctx context.Context, olderThan time.Duration) error
}

type roomStorage struct {
	db DB
}

func NewRoomStorage(db DB) RoomRepo {
	return &roomStorage{db: db}
}

func (s *roomStorage) Create(ctx context.Context, room *model.Room) error {
	if room.Uid == uuid.Nil {
		room.Uid = uuid.New()
	}
	return s.db.QueryRow(ctx, `
		INSERT INTO game_room (uid, title, invite_code, game_type, status, created_by_profile_id, max_players, password_hash, password_value, is_ranked, question_count, answer_timeout_sec)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`, room.Uid, room.Title, room.InviteCode, room.GameType, room.Status, room.CreatedByProfileID, room.MaxPlayers, room.PasswordHash, room.PasswordValue, room.IsRanked, room.QuestionCount, room.AnswerTimeoutSec).Scan(&room.ID, &room.CreatedAt, &room.UpdatedAt)
}

func (s *roomStorage) Get(ctx context.Context, id int64) (*model.Room, error) {
	return s.get(ctx, `SELECT * FROM game_room WHERE id=$1 AND is_active=true`, id)
}

func (s *roomStorage) GetByInviteCode(ctx context.Context, code string) (*model.Room, error) {
	return s.get(ctx, `SELECT * FROM game_room WHERE invite_code=$1 AND is_active=true`, code)
}

func (s *roomStorage) GetForUpdate(ctx context.Context, id int64) (*model.Room, error) {
	return s.get(ctx, `SELECT * FROM game_room WHERE id=$1 AND is_active=true FOR UPDATE`, id)
}

func (s *roomStorage) GetWaitingCreatedByProfile(ctx context.Context, profileID int64) (*model.Room, error) {
	return s.get(ctx, `
		SELECT *
		FROM game_room
		WHERE created_by_profile_id=$1
		  AND is_active=true
		  AND status='waiting'
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, profileID)
}

func (s *roomStorage) get(ctx context.Context, query string, args ...any) (*model.Room, error) {
	var room model.Room
	err := pgxscan.Get(ctx, s.db, &room, query, args...)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &room, nil
}

func (s *roomStorage) ListForProfile(ctx context.Context, profileID int64, limit, offset int) ([]model.Room, error) {
	_ = profileID
	var rooms []model.Room
	err := pgxscan.Select(ctx, s.db, &rooms, `
		SELECT r.*
		FROM game_room r
		WHERE r.is_active=true
		  AND r.status='waiting'
		ORDER BY r.updated_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return rooms, nil
}

func (s *roomStorage) ExpiredActiveIDsForProfile(ctx context.Context, profileID int64) ([]int64, error) {
	var ids []int64
	err := pgxscan.Select(ctx, s.db, &ids, `
		SELECT r.id
		FROM game_room r
		JOIN game_room_member m ON m.room_id=r.id
		WHERE m.profile_id=$1
		  AND r.status='active'
		  AND r.is_active=true
		  AND (
		    (r.paused_by_profile_id IS NOT NULL AND r.pause_until_at IS NOT NULL AND r.pause_until_at <= NOW())
		    OR (r.paused_by_profile_id IS NULL AND r.question_deadline_at IS NOT NULL AND r.question_deadline_at <= NOW())
		    OR (r.paused_by_profile_id IS NULL AND r.next_question_at IS NOT NULL AND r.next_question_at <= NOW())
		  )
	`, profileID)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return ids, nil
}

func (s *roomStorage) HistoryForProfile(ctx context.Context, profileID int64, limit, offset int) ([]model.HistoryRoom, error) {
	var rooms []model.HistoryRoom
	err := pgxscan.Select(ctx, s.db, &rooms, `
		SELECT r.*, me.score AS my_score, COALESCE(op.score, 0) AS opponent_score
		FROM game_room r
		JOIN game_room_member me ON me.room_id=r.id AND me.profile_id=$1
		LEFT JOIN game_room_member op ON op.room_id=r.id AND op.profile_id<>$1
		WHERE r.status='finished' AND r.is_active=true
		ORDER BY r.finished_at DESC NULLS LAST, r.updated_at DESC
		LIMIT $2 OFFSET $3
	`, profileID, limit, offset)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return rooms, nil
}

func (s *roomStorage) Update(ctx context.Context, room *model.Room) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room
		SET status=$1, winner_profile_id=$2, current_question_index=$3, current_question_id=$4,
		    question_started_at=$5, question_deadline_at=$6, next_question_at=$7,
		    paused_by_profile_id=$8, pause_started_at=$9, pause_until_at=$10,
		    updated_at=NOW(), finished_at=$11, password_hash=$12, password_value=$13,
		    is_ranked=$14
		WHERE id=$15 AND is_active=true
	`, room.Status, room.WinnerProfileID, room.CurrentQuestionIndex, room.CurrentQuestionID, room.QuestionStartedAt, room.QuestionDeadlineAt, room.NextQuestionAt, room.PausedByProfileID, room.PauseStartedAt, room.PauseUntilAt, room.FinishedAt, room.PasswordHash, room.PasswordValue, room.IsRanked, room.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *roomStorage) UpdateTitle(ctx context.Context, roomID int64, title string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room
		SET title=$1, updated_at=NOW()
		WHERE id=$2 AND is_active=true
	`, title, roomID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *roomStorage) UpdateAdmin(ctx context.Context, roomID, profileID int64) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room
		SET created_by_profile_id=$1, updated_at=NOW()
		WHERE id=$2 AND is_active=true
	`, profileID, roomID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *roomStorage) Deactivate(ctx context.Context, roomID int64) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room
		SET is_active=false, updated_at=NOW(), finished_at=COALESCE(finished_at, NOW())
		WHERE id=$1 AND is_active=true
	`, roomID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *roomStorage) TouchEmptyWaiting(ctx context.Context, roomID int64) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room r
		SET updated_at=NOW()
		WHERE r.id=$1
		  AND r.status='waiting'
		  AND r.is_active=true
		  AND NOT EXISTS (
		    SELECT 1
		    FROM game_room_member m
		    WHERE m.room_id=r.id AND m.is_active=true
		  )
	`, roomID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *roomStorage) DeactivateEmptyWaitingOlderThan(ctx context.Context, olderThan time.Duration) error {
	seconds := int(olderThan / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	_, err := s.db.Exec(ctx, `
		UPDATE game_room r
		SET is_active=false, updated_at=NOW(), finished_at=COALESCE(finished_at, NOW())
		WHERE r.status='waiting'
		  AND r.is_active=true
		  AND r.updated_at <= NOW() - ($1 * INTERVAL '1 second')
		  AND NOT EXISTS (
		    SELECT 1
		    FROM game_room_member m
		    WHERE m.room_id=r.id AND m.is_active=true
		  )
	`, seconds)
	return err
}

type MemberRepo interface {
	Add(ctx context.Context, roomID, profileID int64) error
	Deactivate(ctx context.Context, roomID, profileID int64) error
	DeactivateWaitingForProfile(ctx context.Context, profileID int64) ([]int64, error)
	DeactivateStaleWaiting(ctx context.Context, olderThan time.Duration) ([]int64, error)
	TouchWaiting(ctx context.Context, roomID, profileID int64) error
	SetReady(ctx context.Context, roomID, profileID int64, isReady bool) error
	ClearReady(ctx context.Context, roomID int64) error
	ResetForReplay(ctx context.Context, roomID int64) error
	SetPauseUsed(ctx context.Context, roomID, profileID int64) error
	SetForceResumeRequested(ctx context.Context, roomID, profileID int64, requested bool) error
	ClearForceResumeRequests(ctx context.Context, roomID int64) error
	List(ctx context.Context, roomID int64) ([]model.RoomMember, error)
	IsMember(ctx context.Context, roomID, profileID int64) (bool, error)
	IncrementScore(ctx context.Context, roomID, profileID int64) error
	Stats(ctx context.Context, profileID int64) (model.ProfileStats, error)
}

type memberStorage struct {
	db DB
}

func NewMemberStorage(db DB) MemberRepo {
	return &memberStorage{db: db}
}

func (s *memberStorage) Add(ctx context.Context, roomID, profileID int64) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO game_room_member (uid, room_id, profile_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (room_id, profile_id)
		DO UPDATE SET is_active=true, is_ready=false, pause_used=false, force_resume_requested=false, updated_at=NOW()
	`, uuid.New(), roomID, profileID)
	return err
}

func (s *memberStorage) Deactivate(ctx context.Context, roomID, profileID int64) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room_member
		SET is_active=false, updated_at=NOW()
		WHERE room_id=$1 AND profile_id=$2 AND is_active=true
	`, roomID, profileID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *memberStorage) DeactivateWaitingForProfile(ctx context.Context, profileID int64) ([]int64, error) {
	var roomIDs []int64
	err := pgxscan.Select(ctx, s.db, &roomIDs, `
		UPDATE game_room_member m
		SET is_active=false, updated_at=NOW()
		FROM game_room r
		WHERE m.room_id=r.id
		  AND m.profile_id=$1
		  AND m.is_active=true
		  AND r.is_active=true
		  AND r.status='waiting'
		RETURNING m.room_id
	`, profileID)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return roomIDs, nil
}

func (s *memberStorage) DeactivateStaleWaiting(ctx context.Context, olderThan time.Duration) ([]int64, error) {
	seconds := int(olderThan / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	var roomIDs []int64
	err := pgxscan.Select(ctx, s.db, &roomIDs, `
		UPDATE game_room_member m
		SET is_active=false, updated_at=NOW()
		FROM game_room r
		WHERE m.room_id=r.id
		  AND m.is_active=true
		  AND r.is_active=true
		  AND r.status='waiting'
		  AND m.updated_at <= NOW() - ($1 * INTERVAL '1 second')
		RETURNING m.room_id
	`, seconds)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return roomIDs, nil
}

func (s *memberStorage) TouchWaiting(ctx context.Context, roomID, profileID int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE game_room_member m
		SET updated_at=NOW()
		FROM game_room r
		WHERE m.room_id=r.id
		  AND m.room_id=$1
		  AND m.profile_id=$2
		  AND m.is_active=true
		  AND r.is_active=true
		  AND r.status='waiting'
	`, roomID, profileID)
	return err
}

func (s *memberStorage) SetReady(ctx context.Context, roomID, profileID int64, isReady bool) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room_member
		SET is_ready=$3, updated_at=NOW()
		WHERE room_id=$1 AND profile_id=$2 AND is_active=true
	`, roomID, profileID, isReady)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *memberStorage) ClearReady(ctx context.Context, roomID int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE game_room_member
		SET is_ready=false, updated_at=NOW()
		WHERE room_id=$1 AND is_active=true
	`, roomID)
	return err
}

func (s *memberStorage) ResetForReplay(ctx context.Context, roomID int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE game_room_member
		SET score=0,
		    is_ready=false,
		    pause_used=false,
		    force_resume_requested=false,
		    updated_at=NOW()
		WHERE room_id=$1 AND is_active=true
	`, roomID)
	return err
}

func (s *memberStorage) SetPauseUsed(ctx context.Context, roomID, profileID int64) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room_member
		SET pause_used=true, updated_at=NOW()
		WHERE room_id=$1 AND profile_id=$2 AND is_active=true
	`, roomID, profileID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *memberStorage) SetForceResumeRequested(ctx context.Context, roomID, profileID int64, requested bool) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room_member
		SET force_resume_requested=$3, updated_at=NOW()
		WHERE room_id=$1 AND profile_id=$2 AND is_active=true
	`, roomID, profileID, requested)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *memberStorage) ClearForceResumeRequests(ctx context.Context, roomID int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE game_room_member
		SET force_resume_requested=false, updated_at=NOW()
		WHERE room_id=$1 AND is_active=true
	`, roomID)
	return err
}

func (s *memberStorage) List(ctx context.Context, roomID int64) ([]model.RoomMember, error) {
	var members []model.RoomMember
	err := pgxscan.Select(ctx, s.db, &members, `
		SELECT * FROM game_room_member WHERE room_id=$1 AND is_active=true ORDER BY joined_at ASC
	`, roomID)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return members, nil
}

func (s *memberStorage) IsMember(ctx context.Context, roomID, profileID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM game_room_member WHERE room_id=$1 AND profile_id=$2 AND is_active=true)
	`, roomID, profileID).Scan(&exists)
	return exists, err
}

func (s *memberStorage) IncrementScore(ctx context.Context, roomID, profileID int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE game_room_member SET score=score+1, updated_at=NOW() WHERE room_id=$1 AND profile_id=$2
	`, roomID, profileID)
	return err
}

func (s *memberStorage) Stats(ctx context.Context, profileID int64) (model.ProfileStats, error) {
	var stats model.ProfileStats
	err := pgxscan.Get(ctx, s.db, &stats, `
		SELECT
			COUNT(*)::INT AS played,
			COUNT(*) FILTER (WHERE r.winner_profile_id=$1)::INT AS won,
			COUNT(*) FILTER (WHERE r.winner_profile_id IS NOT NULL AND r.winner_profile_id<>$1)::INT AS lost,
			COUNT(*) FILTER (WHERE r.winner_profile_id IS NULL)::INT AS drawn
		FROM game_room r
		JOIN game_room_member m ON m.room_id=r.id
		WHERE m.profile_id=$1 AND r.status='finished' AND r.is_active=true
	`, profileID)
	return stats, err
}

type RoomQuestionRepo interface {
	Add(ctx context.Context, roomID int64, questionID int64, position int) error
	Clear(ctx context.Context, roomID int64) error
	List(ctx context.Context, roomID int64) ([]model.RoomQuestion, error)
	GetByPosition(ctx context.Context, roomID int64, position int) (*model.RoomQuestion, error)
	GetActive(ctx context.Context, roomID int64) (*model.RoomQuestion, error)
	Update(ctx context.Context, rq *model.RoomQuestion) error
}

type roomQuestionStorage struct {
	db DB
}

func NewRoomQuestionStorage(db DB) RoomQuestionRepo {
	return &roomQuestionStorage{db: db}
}

func (s *roomQuestionStorage) Add(ctx context.Context, roomID int64, questionID int64, position int) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO game_room_question (uid, room_id, question_id, position)
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), roomID, questionID, position)
	return err
}

func (s *roomQuestionStorage) Clear(ctx context.Context, roomID int64) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM game_room_question WHERE room_id=$1
	`, roomID)
	return err
}

func (s *roomQuestionStorage) List(ctx context.Context, roomID int64) ([]model.RoomQuestion, error) {
	var items []model.RoomQuestion
	err := pgxscan.Select(ctx, s.db, &items, `
		SELECT * FROM game_room_question WHERE room_id=$1 ORDER BY position ASC
	`, roomID)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return items, nil
}

func (s *roomQuestionStorage) GetByPosition(ctx context.Context, roomID int64, position int) (*model.RoomQuestion, error) {
	var rq model.RoomQuestion
	err := pgxscan.Get(ctx, s.db, &rq, `
		SELECT * FROM game_room_question WHERE room_id=$1 AND position=$2
	`, roomID, position)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rq, nil
}

func (s *roomQuestionStorage) GetActive(ctx context.Context, roomID int64) (*model.RoomQuestion, error) {
	var rq model.RoomQuestion
	err := pgxscan.Get(ctx, s.db, &rq, `
		SELECT * FROM game_room_question WHERE room_id=$1 AND status='active'
	`, roomID)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rq, nil
}

func (s *roomQuestionStorage) Update(ctx context.Context, rq *model.RoomQuestion) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_room_question
		SET status=$1, winner_profile_id=$2, started_at=$3, deadline_at=$4, completed_at=$5, updated_at=NOW()
		WHERE id=$6
	`, rq.Status, rq.WinnerProfileID, rq.StartedAt, rq.DeadlineAt, rq.CompletedAt, rq.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type AnswerRepo interface {
	Add(ctx context.Context, answer *model.Answer) error
	List(ctx context.Context, roomQuestionID int64) ([]model.Answer, error)
	Count(ctx context.Context, roomQuestionID int64) (int, error)
}

type answerStorage struct {
	db DB
}

func NewAnswerStorage(db DB) AnswerRepo {
	return &answerStorage{db: db}
}

func (s *answerStorage) Add(ctx context.Context, answer *model.Answer) error {
	if answer.Uid == uuid.Nil {
		answer.Uid = uuid.New()
	}
	return s.db.QueryRow(ctx, `
		INSERT INTO game_answer (uid, room_question_id, profile_id, answer, distance)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, answered_at, created_at
	`, answer.Uid, answer.RoomQuestionID, answer.ProfileID, answer.Answer, answer.Distance).Scan(&answer.ID, &answer.AnsweredAt, &answer.CreatedAt)
}

func (s *answerStorage) List(ctx context.Context, roomQuestionID int64) ([]model.Answer, error) {
	var answers []model.Answer
	err := pgxscan.Select(ctx, s.db, &answers, `
		SELECT * FROM game_answer WHERE room_question_id=$1 ORDER BY answered_at ASC
	`, roomQuestionID)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return answers, nil
}

func (s *answerStorage) Count(ctx context.Context, roomQuestionID int64) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*)::INT FROM game_answer WHERE room_question_id=$1`, roomQuestionID).Scan(&count)
	return count, err
}

type MessageRepo interface {
	Add(ctx context.Context, message *model.RoomMessage) error
	List(ctx context.Context, roomID int64, limit, offset int) ([]model.RoomMessage, error)
}

type messageStorage struct {
	db DB
}

func NewMessageStorage(db DB) MessageRepo {
	return &messageStorage{db: db}
}

func (s *messageStorage) Add(ctx context.Context, message *model.RoomMessage) error {
	if message.Uid == uuid.Nil {
		message.Uid = uuid.New()
	}
	return s.db.QueryRow(ctx, `
		INSERT INTO game_room_message (uid, room_id, profile_id, message_text)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, message.Uid, message.RoomID, message.ProfileID, message.Text).Scan(&message.ID, &message.CreatedAt)
}

func (s *messageStorage) List(ctx context.Context, roomID int64, limit, offset int) ([]model.RoomMessage, error) {
	var messages []model.RoomMessage
	err := pgxscan.Select(ctx, s.db, &messages, `
		SELECT *
		FROM game_room_message
		WHERE room_id=$1
		ORDER BY created_at ASC, id ASC
		LIMIT $2 OFFSET $3
	`, roomID, limit, offset)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return messages, nil
}

type RatingRepo interface {
	EnsureSeason(ctx context.Context, season *model.RatingSeason) error
	EnsurePlayerRatings(ctx context.Context, seasonID int64, gameType string, profileIDs []int64) ([]model.PlayerRating, error)
	CountMatchesForGroup(ctx context.Context, gameType, groupHash string, from, to time.Time) (int, error)
	AddMatch(ctx context.Context, match *model.RatingMatch) error
	AddMatchPlayer(ctx context.Context, player *model.RatingMatchPlayer) error
	ApplyPlayerRatingChange(ctx context.Context, seasonID, profileID int64, delta int, isWin bool, isDraw bool) error
	RatingChangesForRoom(ctx context.Context, roomID int64) ([]model.RatingChange, error)
	Leaderboard(ctx context.Context, seasonID int64, limit, offset int) ([]model.LeaderboardEntry, error)
}

type ratingStorage struct {
	db DB
}

func NewRatingStorage(db DB) RatingRepo {
	return &ratingStorage{db: db}
}

func (s *ratingStorage) EnsureSeason(ctx context.Context, season *model.RatingSeason) error {
	if season.Uid == uuid.Nil {
		season.Uid = uuid.New()
	}
	return s.db.QueryRow(ctx, `
		INSERT INTO game_rating_season (uid, game_type, season_number, season_year, season_month, title, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (game_type, season_year, season_month)
		DO UPDATE SET title=EXCLUDED.title, starts_at=EXCLUDED.starts_at, ends_at=EXCLUDED.ends_at
		RETURNING id, created_at
	`, season.Uid, season.GameType, season.SeasonNumber, season.SeasonYear, season.SeasonMonth, season.Title, season.StartsAt, season.EndsAt).Scan(&season.ID, &season.CreatedAt)
}

func (s *ratingStorage) EnsurePlayerRatings(ctx context.Context, seasonID int64, gameType string, profileIDs []int64) ([]model.PlayerRating, error) {
	for _, profileID := range profileIDs {
		if profileID <= 0 {
			continue
		}
		if _, err := s.db.Exec(ctx, `
			INSERT INTO game_player_rating (uid, season_id, game_type, profile_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (season_id, profile_id) DO NOTHING
		`, uuid.New(), seasonID, gameType, profileID); err != nil {
			return nil, err
		}
	}
	var ratings []model.PlayerRating
	err := pgxscan.Select(ctx, s.db, &ratings, `
		SELECT *
		FROM game_player_rating
		WHERE season_id=$1 AND profile_id=ANY($2)
		ORDER BY profile_id ASC
		FOR UPDATE
	`, seasonID, profileIDs)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return ratings, nil
}

func (s *ratingStorage) CountMatchesForGroup(ctx context.Context, gameType, groupHash string, from, to time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)::INT
		FROM game_rating_match
		WHERE game_type=$1
		  AND group_hash=$2
		  AND played_at >= $3
		  AND played_at < $4
	`, gameType, groupHash, from, to).Scan(&count)
	return count, err
}

func (s *ratingStorage) AddMatch(ctx context.Context, match *model.RatingMatch) error {
	if match.Uid == uuid.Nil {
		match.Uid = uuid.New()
	}
	return s.db.QueryRow(ctx, `
		INSERT INTO game_rating_match (uid, room_id, season_id, game_type, group_hash, group_occurrence, rating_weight, played_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`, match.Uid, match.RoomID, match.SeasonID, match.GameType, match.GroupHash, match.GroupOccurrence, match.RatingWeight, match.PlayedAt).Scan(&match.ID, &match.CreatedAt)
}

func (s *ratingStorage) AddMatchPlayer(ctx context.Context, player *model.RatingMatchPlayer) error {
	return s.db.QueryRow(ctx, `
		INSERT INTO game_rating_match_player (match_id, profile_id, score, place, before_rating, after_rating, rating_delta)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, player.MatchID, player.ProfileID, player.Score, player.Place, player.BeforeRating, player.AfterRating, player.RatingDelta).Scan(&player.ID, &player.CreatedAt)
}

func (s *ratingStorage) ApplyPlayerRatingChange(ctx context.Context, seasonID, profileID int64, delta int, isWin bool, isDraw bool) error {
	winValue := 0
	if isWin {
		winValue = 1
	}
	drawValue := 0
	if isDraw {
		drawValue = 1
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE game_player_rating
		SET rating=GREATEST(0, rating + $3),
		    games_played=games_played + 1,
		    wins=wins + $4,
		    draws=draws + $5,
		    updated_at=NOW()
		WHERE season_id=$1 AND profile_id=$2
	`, seasonID, profileID, delta, winValue, drawValue)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ratingStorage) RatingChangesForRoom(ctx context.Context, roomID int64) ([]model.RatingChange, error) {
	var changes []model.RatingChange
	err := pgxscan.Select(ctx, s.db, &changes, `
		SELECT
			m.room_id,
			m.season_id,
			s.season_number,
			s.title AS season_title,
			m.rating_weight,
			mp.profile_id,
			mp.score,
			mp.place,
			mp.before_rating,
			mp.after_rating,
			mp.rating_delta
		FROM game_rating_match_player mp
		JOIN game_rating_match m ON m.id=mp.match_id
		JOIN game_rating_season s ON s.id=m.season_id
		WHERE m.room_id=$1
		ORDER BY mp.place ASC, mp.rating_delta DESC, mp.profile_id ASC
	`, roomID)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return changes, nil
}

func (s *ratingStorage) Leaderboard(ctx context.Context, seasonID int64, limit, offset int) ([]model.LeaderboardEntry, error) {
	var entries []model.LeaderboardEntry
	err := pgxscan.Select(ctx, s.db, &entries, `
		SELECT
			RANK() OVER (ORDER BY rating DESC, games_played DESC, wins DESC, profile_id ASC)::INT AS rank,
			season_id,
			game_type,
			profile_id,
			rating,
			games_played,
			wins,
			draws
		FROM game_player_rating
		WHERE season_id=$1 AND games_played > 0
		ORDER BY rating DESC, games_played DESC, wins DESC, profile_id ASC
		LIMIT $2 OFFSET $3
	`, seasonID, limit, offset)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return entries, nil
}

func NowPtr() *time.Time {
	now := time.Now()
	return &now
}
