package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	DefaultGameType = "number_duel"

	RoomStatusWaiting  = "waiting"
	RoomStatusActive   = "active"
	RoomStatusFinished = "finished"

	QuestionStatusPending   = "pending"
	QuestionStatusActive    = "active"
	QuestionStatusCompleted = "completed"
)

type Question struct {
	ID            int64     `db:"id"`
	Uid           uuid.UUID `db:"uid"`
	GameType      string    `db:"game_type"`
	Slug          string    `db:"slug"`
	Text          string    `db:"question_text"`
	CorrectAnswer float64   `db:"correct_answer"`
	AnswerUnit    *string   `db:"answer_unit"`
	IsActive      bool      `db:"is_active"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type Room struct {
	ID                   int64      `db:"id"`
	Uid                  uuid.UUID  `db:"uid"`
	InviteCode           string     `db:"invite_code"`
	GameType             string     `db:"game_type"`
	Status               string     `db:"status"`
	CreatedByProfileID   int64      `db:"created_by_profile_id"`
	WinnerProfileID      *int64     `db:"winner_profile_id"`
	MaxPlayers           int        `db:"max_players"`
	PasswordHash         *string    `db:"password_hash"`
	PasswordValue        *string    `db:"password_value"`
	QuestionCount        int        `db:"question_count"`
	AnswerTimeoutSec     int        `db:"answer_timeout_sec"`
	CurrentQuestionIndex int        `db:"current_question_index"`
	CurrentQuestionID    *int64     `db:"current_question_id"`
	QuestionStartedAt    *time.Time `db:"question_started_at"`
	QuestionDeadlineAt   *time.Time `db:"question_deadline_at"`
	IsActive             bool       `db:"is_active"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
	FinishedAt           *time.Time `db:"finished_at"`
}

type RoomMember struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	RoomID    int64     `db:"room_id"`
	ProfileID int64     `db:"profile_id"`
	Score     int       `db:"score"`
	IsReady   bool      `db:"is_ready"`
	JoinedAt  time.Time `db:"joined_at"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type RoomQuestion struct {
	ID              int64      `db:"id"`
	Uid             uuid.UUID  `db:"uid"`
	RoomID          int64      `db:"room_id"`
	QuestionID      int64      `db:"question_id"`
	Position        int        `db:"position"`
	Status          string     `db:"status"`
	WinnerProfileID *int64     `db:"winner_profile_id"`
	StartedAt       *time.Time `db:"started_at"`
	DeadlineAt      *time.Time `db:"deadline_at"`
	CompletedAt     *time.Time `db:"completed_at"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

type Answer struct {
	ID             int64     `db:"id"`
	Uid            uuid.UUID `db:"uid"`
	RoomQuestionID int64     `db:"room_question_id"`
	ProfileID      int64     `db:"profile_id"`
	Answer         float64   `db:"answer"`
	Distance       float64   `db:"distance"`
	AnsweredAt     time.Time `db:"answered_at"`
	CreatedAt      time.Time `db:"created_at"`
}

type HistoryRoom struct {
	Room
	MyScore       int `db:"my_score"`
	OpponentScore int `db:"opponent_score"`
}

type ProfileStats struct {
	Played int `db:"played"`
	Won    int `db:"won"`
	Lost   int `db:"lost"`
	Drawn  int `db:"drawn"`
}
