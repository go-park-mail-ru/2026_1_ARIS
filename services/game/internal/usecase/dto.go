package usecase

type CreateRoomInput struct {
	GameType         string
	QuestionCount    int
	AnswerTimeoutSec int
}

type QuestionInput struct {
	GameType      string
	Slug          string
	Text          string
	CorrectAnswer float64
	AnswerUnit    *string
	IsActive      bool
}

type Player struct {
	ProfileID     int64
	UserAccountID int64
	Name          string
	Username      string
	FirstName     string
	LastName      string
	AvatarID      *int64
	Score         int
	HasAnswered   bool
	IsMe          bool
}

type Question struct {
	ID            int64
	Slug          string
	Text          string
	CorrectAnswer float64
	AnswerUnit    *string
	IsActive      bool
}

type Answer struct {
	ProfileID  int64
	Answer     float64
	Distance   float64
	AnsweredAt string
}

type RoomQuestion struct {
	Position        int
	Status          string
	Question        Question
	WinnerProfileID *int64
	StartedAt       *string
	DeadlineAt      *string
	CompletedAt     *string
	Answers         []Answer
}

type CurrentQuestion struct {
	Position    int
	ID          int64
	Text        string
	AnswerUnit  *string
	StartedAt   *string
	DeadlineAt  *string
	HasAnswered bool
}

type Room struct {
	ID                   int64
	InviteCode           string
	GameType             string
	Status               string
	CreatedByProfileID   int64
	WinnerProfileID      *int64
	QuestionCount        int
	AnswerTimeoutSec     int
	CurrentQuestionIndex int
	CurrentQuestion      *CurrentQuestion
	Players              []Player
	Questions            []RoomQuestion
	ProfileStats         Stats
	CreatedAt            string
	UpdatedAt            string
	FinishedAt           *string
}

type HistoryItem struct {
	Room
	MyScore       int
	OpponentScore int
}

type Stats struct {
	Played int
	Won    int
	Lost   int
	Drawn  int
}
