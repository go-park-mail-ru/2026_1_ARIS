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
	TextRU        string    `db:"question_text_ru"`
	TextEN        string    `db:"question_text_en"`
	CorrectAnswer float64   `db:"correct_answer"`
	IsActive      bool      `db:"is_active"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type Room struct {
	ID                   int64      `db:"id"`
	Uid                  uuid.UUID  `db:"uid"`
	Title                string     `db:"title"`
	InviteCode           string     `db:"invite_code"`
	GameType             string     `db:"game_type"`
	Status               string     `db:"status"`
	CreatedByProfileID   int64      `db:"created_by_profile_id"`
	WinnerProfileID      *int64     `db:"winner_profile_id"`
	MaxPlayers           int        `db:"max_players"`
	PasswordHash         *string    `db:"password_hash"`
	PasswordValue        *string    `db:"password_value"`
	IsRanked             bool       `db:"is_ranked"`
	IsPublicLobby        bool       `db:"is_public_lobby"`
	QuestionCount        int        `db:"question_count"`
	AnswerTimeoutSec     int        `db:"answer_timeout_sec"`
	RoundPauseSec        int        `db:"round_pause_sec"`
	CurrentQuestionIndex int        `db:"current_question_index"`
	CurrentQuestionID    *int64     `db:"current_question_id"`
	QuestionStartedAt    *time.Time `db:"question_started_at"`
	QuestionDeadlineAt   *time.Time `db:"question_deadline_at"`
	NextQuestionAt       *time.Time `db:"next_question_at"`
	PausedByProfileID    *int64     `db:"paused_by_profile_id"`
	PauseStartedAt       *time.Time `db:"pause_started_at"`
	PauseUntilAt         *time.Time `db:"pause_until_at"`
	IsActive             bool       `db:"is_active"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
	FinishedAt           *time.Time `db:"finished_at"`
}

type RoomMember struct {
	ID                   int64     `db:"id"`
	Uid                  uuid.UUID `db:"uid"`
	RoomID               int64     `db:"room_id"`
	ProfileID            int64     `db:"profile_id"`
	Score                int       `db:"score"`
	IsReady              bool      `db:"is_ready"`
	PauseUsed            bool      `db:"pause_used"`
	ForceResumeRequested bool      `db:"force_resume_requested"`
	JoinedAt             time.Time `db:"joined_at"`
	IsActive             bool      `db:"is_active"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
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

type RoomMessage struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	RoomID    int64     `db:"room_id"`
	ProfileID int64     `db:"profile_id"`
	Text      string    `db:"message_text"`
	CreatedAt time.Time `db:"created_at"`
}

type PublicParticipant struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	RoomID    int64     `db:"room_id"`
	ProfileID int64     `db:"profile_id"`
	TokenHash string    `db:"token_hash"`
	FirstName string    `db:"first_name"`
	LastName  string    `db:"last_name"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
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

type RatingSeason struct {
	ID           int64     `db:"id"`
	Uid          uuid.UUID `db:"uid"`
	GameType     string    `db:"game_type"`
	SeasonNumber int       `db:"season_number"`
	SeasonYear   int       `db:"season_year"`
	SeasonMonth  int       `db:"season_month"`
	Title        string    `db:"title"`
	StartsAt     time.Time `db:"starts_at"`
	EndsAt       time.Time `db:"ends_at"`
	CreatedAt    time.Time `db:"created_at"`
}

type PlayerRating struct {
	ID          int64     `db:"id"`
	Uid         uuid.UUID `db:"uid"`
	SeasonID    int64     `db:"season_id"`
	GameType    string    `db:"game_type"`
	ProfileID   int64     `db:"profile_id"`
	Rating      int       `db:"rating"`
	GamesPlayed int       `db:"games_played"`
	Wins        int       `db:"wins"`
	Draws       int       `db:"draws"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type RatingMatch struct {
	ID              int64     `db:"id"`
	Uid             uuid.UUID `db:"uid"`
	RoomID          int64     `db:"room_id"`
	SeasonID        int64     `db:"season_id"`
	GameType        string    `db:"game_type"`
	GroupHash       string    `db:"group_hash"`
	GroupOccurrence int       `db:"group_occurrence"`
	RatingWeight    float64   `db:"rating_weight"`
	PlayedAt        time.Time `db:"played_at"`
	CreatedAt       time.Time `db:"created_at"`
}

type RatingMatchPlayer struct {
	ID           int64     `db:"id"`
	MatchID      int64     `db:"match_id"`
	ProfileID    int64     `db:"profile_id"`
	Score        int       `db:"score"`
	Place        int       `db:"place"`
	BeforeRating int       `db:"before_rating"`
	AfterRating  int       `db:"after_rating"`
	RatingDelta  int       `db:"rating_delta"`
	CreatedAt    time.Time `db:"created_at"`
}

type RatingChange struct {
	RoomID       int64   `db:"room_id"`
	SeasonID     int64   `db:"season_id"`
	SeasonNumber int     `db:"season_number"`
	SeasonTitle  string  `db:"season_title"`
	RatingWeight float64 `db:"rating_weight"`
	ProfileID    int64   `db:"profile_id"`
	Score        int     `db:"score"`
	Place        int     `db:"place"`
	BeforeRating int     `db:"before_rating"`
	AfterRating  int     `db:"after_rating"`
	RatingDelta  int     `db:"rating_delta"`
}

type LeaderboardEntry struct {
	Rank        int    `db:"rank"`
	SeasonID    int64  `db:"season_id"`
	GameType    string `db:"game_type"`
	ProfileID   int64  `db:"profile_id"`
	Rating      int    `db:"rating"`
	GamesPlayed int    `db:"games_played"`
	Wins        int    `db:"wins"`
	Draws       int    `db:"draws"`
}
