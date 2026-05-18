package repository

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
	Get(ctx context.Context, id int64) (*model.Question, error)
	List(ctx context.Context, gameType string, includeInactive bool, limit, offset int) ([]model.Question, error)
	Random(ctx context.Context, gameType string, limit int) ([]model.Question, error)
}

type questionStorage struct {
	db DB
}

func NewQuestionStorage(db DB) QuestionRepo {
	return &questionStorage{db: db}
}

func (s *questionStorage) Create(ctx context.Context, q *model.Question) error {
	if q.Uid == uuid.Nil {
		q.Uid = uuid.New()
	}
	return s.db.QueryRow(ctx, `
		INSERT INTO game_question (uid, game_type, slug, question_text, correct_answer, answer_unit, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, q.Uid, q.GameType, q.Slug, q.Text, q.CorrectAnswer, q.AnswerUnit, q.IsActive).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
}

func (s *questionStorage) Update(ctx context.Context, q *model.Question) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE game_question
		SET game_type=$1, slug=$2, question_text=$3, correct_answer=$4, answer_unit=$5, is_active=$6, updated_at=NOW()
		WHERE id=$7
	`, q.GameType, q.Slug, q.Text, q.CorrectAnswer, q.AnswerUnit, q.IsActive, q.ID)
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
	err := pgxscan.Get(ctx, s.db, &q, `SELECT * FROM game_question WHERE id=$1`, id)
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
		SELECT *
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
		SELECT *
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
	ListForProfile(ctx context.Context, profileID int64, limit, offset int) ([]model.Room, error)
	ExpiredActiveIDsForProfile(ctx context.Context, profileID int64) ([]int64, error)
	HistoryForProfile(ctx context.Context, profileID int64, limit, offset int) ([]model.HistoryRoom, error)
	Update(ctx context.Context, room *model.Room) error
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
		INSERT INTO game_room (uid, invite_code, game_type, status, created_by_profile_id, question_count, answer_timeout_sec)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, room.Uid, room.InviteCode, room.GameType, room.Status, room.CreatedByProfileID, room.QuestionCount, room.AnswerTimeoutSec).Scan(&room.ID, &room.CreatedAt, &room.UpdatedAt)
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
	var rooms []model.Room
	err := pgxscan.Select(ctx, s.db, &rooms, `
		SELECT r.*
		FROM game_room r
		JOIN game_room_member m ON m.room_id=r.id
		WHERE m.profile_id=$1 AND r.is_active=true
		ORDER BY r.updated_at DESC
		LIMIT $2 OFFSET $3
	`, profileID, limit, offset)
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
		  AND r.question_deadline_at IS NOT NULL
		  AND r.question_deadline_at <= NOW()
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
		    question_started_at=$5, question_deadline_at=$6, updated_at=NOW(), finished_at=$7
		WHERE id=$8 AND is_active=true
	`, room.Status, room.WinnerProfileID, room.CurrentQuestionIndex, room.CurrentQuestionID, room.QuestionStartedAt, room.QuestionDeadlineAt, room.FinishedAt, room.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type MemberRepo interface {
	Add(ctx context.Context, roomID, profileID int64) error
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
		ON CONFLICT (room_id, profile_id) DO NOTHING
	`, uuid.New(), roomID, profileID)
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

func NowPtr() *time.Time {
	now := time.Now()
	return &now
}
