package http

type createRoomRequest struct {
	GameType         string `json:"gameType"`
	QuestionCount    int    `json:"questionCount"`
	AnswerTimeoutSec int    `json:"answerTimeoutSec"`
}

type joinRoomRequest struct {
	InviteCode string `json:"inviteCode"`
}

type submitAnswerRequest struct {
	Answer float64 `json:"answer"`
}

type questionRequest struct {
	GameType      string  `json:"gameType"`
	Slug          string  `json:"slug"`
	Text          string  `json:"text"`
	CorrectAnswer float64 `json:"correctAnswer"`
	AnswerUnit    *string `json:"answerUnit,omitempty"`
	IsActive      *bool   `json:"isActive,omitempty"`
}

type playerResponse struct {
	ProfileID     string `json:"profileId"`
	UserAccountID string `json:"userAccountId"`
	Name          string `json:"name"`
	Username      string `json:"username"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	AvatarID      *int64 `json:"avatarId,omitempty"`
	Score         int    `json:"score"`
	HasAnswered   bool   `json:"hasAnswered"`
	IsMe          bool   `json:"isMe"`
}

type questionResponse struct {
	ID            string  `json:"id"`
	Slug          string  `json:"slug"`
	Text          string  `json:"text"`
	CorrectAnswer float64 `json:"correctAnswer,omitempty"`
	AnswerUnit    *string `json:"answerUnit,omitempty"`
	IsActive      bool    `json:"isActive"`
}

type answerResponse struct {
	ProfileID  string  `json:"profileId"`
	Answer     float64 `json:"answer"`
	Distance   float64 `json:"distance"`
	AnsweredAt string  `json:"answeredAt"`
}

type roomQuestionResponse struct {
	Position        int              `json:"position"`
	Status          string           `json:"status"`
	Question        questionResponse `json:"question"`
	WinnerProfileID *string          `json:"winnerProfileId,omitempty"`
	StartedAt       *string          `json:"startedAt,omitempty"`
	DeadlineAt      *string          `json:"deadlineAt,omitempty"`
	CompletedAt     *string          `json:"completedAt,omitempty"`
	Answers         []answerResponse `json:"answers"`
}

type currentQuestionResponse struct {
	Position    int     `json:"position"`
	ID          string  `json:"id"`
	Text        string  `json:"text"`
	AnswerUnit  *string `json:"answerUnit,omitempty"`
	StartedAt   *string `json:"startedAt,omitempty"`
	DeadlineAt  *string `json:"deadlineAt,omitempty"`
	HasAnswered bool    `json:"hasAnswered"`
}

type roomResponse struct {
	ID                   string                   `json:"id"`
	InviteCode           string                   `json:"inviteCode"`
	GameType             string                   `json:"gameType"`
	Status               string                   `json:"status"`
	CreatedByProfileID   string                   `json:"createdByProfileId"`
	WinnerProfileID      *string                  `json:"winnerProfileId,omitempty"`
	QuestionCount        int                      `json:"questionCount"`
	AnswerTimeoutSec     int                      `json:"answerTimeoutSec"`
	CurrentQuestionIndex int                      `json:"currentQuestionIndex"`
	CurrentQuestion      *currentQuestionResponse `json:"currentQuestion,omitempty"`
	Players              []playerResponse         `json:"players"`
	Questions            []roomQuestionResponse   `json:"questions"`
	ProfileStats         statsResponse            `json:"profileStats"`
	CreatedAt            string                   `json:"createdAt"`
	UpdatedAt            string                   `json:"updatedAt"`
	FinishedAt           *string                  `json:"finishedAt,omitempty"`
}

type historyResponse struct {
	Room          roomResponse `json:"room"`
	MyScore       int          `json:"myScore"`
	OpponentScore int          `json:"opponentScore"`
}

type statsResponse struct {
	Played int `json:"played"`
	Won    int `json:"won"`
	Lost   int `json:"lost"`
	Drawn  int `json:"drawn"`
}

type socketEvent struct {
	Type  string        `json:"type"`
	Room  *roomResponse `json:"room,omitempty"`
	Error string        `json:"error,omitempty"`
}

type socketMessage struct {
	Type   string  `json:"type"`
	Answer float64 `json:"answer"`
}

type errorResponse struct {
	Error string `json:"error"`
}
