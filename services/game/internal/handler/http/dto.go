package http

import "encoding/json"
//go:generate go run github.com/mailru/easyjson/easyjson -all $GOFILE

type createRoomRequest struct {
	Title            string `json:"title"`
	GameType         string `json:"gameType"`
	MaxPlayers       int    `json:"maxPlayers"`
	Password         string `json:"password"`
	IsRanked         bool   `json:"isRanked"`
	QuestionCount    int    `json:"questionCount"`
	AnswerTimeoutSec int    `json:"answerTimeoutSec"`
}

type joinRoomRequest struct {
	InviteCode string `json:"inviteCode"`
	RoomID     string `json:"roomId"`
	Password   string `json:"password"`
}

type readyRequest struct {
	IsReady bool `json:"isReady"`
}

type passwordRequest struct {
	Password string `json:"password"`
}

type titleRequest struct {
	Title string `json:"title"`
}

type rankedRequest struct {
	IsRanked bool `json:"isRanked"`
}

type adminRequest struct {
	ProfileID string `json:"profileId"`
}

type submitAnswerRequest struct {
	Answer float64 `json:"answer"`
}

type roomMessageRequest struct {
	Text string `json:"text"`
}

type localizedTextPayload struct {
	RU string `json:"ru"`
	EN string `json:"en"`
}

func (p *localizedTextPayload) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		p.RU = text
		p.EN = text
		return nil
	}
	type alias localizedTextPayload
	var parsed alias
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	p.RU = parsed.RU
	p.EN = parsed.EN
	return nil
}

type questionRequest struct {
	GameType      string               `json:"gameType"`
	Text          localizedTextPayload `json:"text"`
	CorrectAnswer float64              `json:"correctAnswer"`
	IsActive      *bool                `json:"isActive,omitempty"`
}

type playerResponse struct {
	ProfileID            string `json:"profileId"`
	UserAccountID        string `json:"userAccountId"`
	Name                 string `json:"name"`
	Username             string `json:"username"`
	FirstName            string `json:"firstName"`
	LastName             string `json:"lastName"`
	AvatarID             *int64 `json:"avatarId,omitempty"`
	Score                int    `json:"score"`
	IsReady              bool   `json:"isReady"`
	HasAnswered          bool   `json:"hasAnswered"`
	PauseUsed            bool   `json:"pauseUsed"`
	ForceResumeRequested bool   `json:"forceResumeRequested"`
	IsMe                 bool   `json:"isMe"`
}

type questionResponse struct {
	ID            string                `json:"id"`
	Text          localizedTextResponse `json:"text"`
	CorrectAnswer float64               `json:"correctAnswer,omitempty"`
	IsActive      bool                  `json:"isActive"`
}

type localizedTextResponse struct {
	RU string `json:"ru"`
	EN string `json:"en"`
}

type roomQuestionPayloadResponse struct {
	ID            string  `json:"id"`
	Text          string  `json:"text"`
	CorrectAnswer float64 `json:"correctAnswer,omitempty"`
	IsActive      bool    `json:"isActive"`
}

type answerResponse struct {
	ProfileID      string  `json:"profileId"`
	Answer         float64 `json:"answer"`
	Distance       float64 `json:"distance"`
	AnsweredAt     string  `json:"answeredAt"`
	ResponseTimeMs int64   `json:"responseTimeMs"`
}

type ratingChangeResponse struct {
	ProfileID    string  `json:"profileId"`
	Score        int     `json:"score"`
	Place        int     `json:"place"`
	BeforeRating int     `json:"beforeRating"`
	AfterRating  int     `json:"afterRating"`
	RatingDelta  int     `json:"ratingDelta"`
	RatingWeight float64 `json:"ratingWeight"`
	SeasonNumber int     `json:"seasonNumber"`
	SeasonTitle  string  `json:"seasonTitle"`
}

type roomMessageResponse struct {
	ID                  string `json:"id"`
	RoomID              string `json:"roomId"`
	AuthorProfileID     string `json:"authorProfileId"`
	AuthorUserAccountID string `json:"authorUserAccountId"`
	AuthorName          string `json:"authorName"`
	AuthorFirstName     string `json:"authorFirstName"`
	AuthorLastName      string `json:"authorLastName"`
	AuthorUsername      string `json:"authorUsername"`
	AuthorAvatarID      *int64 `json:"authorAvatarId,omitempty"`
	Text                string `json:"text"`
	CreatedAt           string `json:"createdAt"`
}

type roomQuestionResponse struct {
	Position        int                         `json:"position"`
	Status          string                      `json:"status"`
	Question        roomQuestionPayloadResponse `json:"question"`
	WinnerProfileID *string                     `json:"winnerProfileId,omitempty"`
	StartedAt       *string                     `json:"startedAt,omitempty"`
	DeadlineAt      *string                     `json:"deadlineAt,omitempty"`
	CompletedAt     *string                     `json:"completedAt,omitempty"`
	Answers         []answerResponse            `json:"answers"`
}

type currentQuestionResponse struct {
	Position    int     `json:"position"`
	ID          string  `json:"id"`
	Text        string  `json:"text"`
	StartedAt   *string `json:"startedAt,omitempty"`
	DeadlineAt  *string `json:"deadlineAt,omitempty"`
	HasAnswered bool    `json:"hasAnswered"`
}

type roomResponse struct {
	ID                      string                   `json:"id"`
	Title                   string                   `json:"title"`
	InviteCode              string                   `json:"inviteCode"`
	GameType                string                   `json:"gameType"`
	Status                  string                   `json:"status"`
	CreatedByProfileID      string                   `json:"createdByProfileId"`
	WinnerProfileID         *string                  `json:"winnerProfileId,omitempty"`
	MaxPlayers              int                      `json:"maxPlayers"`
	HasPassword             bool                     `json:"hasPassword"`
	Password                string                   `json:"password,omitempty"`
	IsRanked                bool                     `json:"isRanked"`
	QuestionCount           int                      `json:"questionCount"`
	AnswerTimeoutSec        int                      `json:"answerTimeoutSec"`
	Creator                 playerResponse           `json:"creator"`
	CurrentQuestionIndex    int                      `json:"currentQuestionIndex"`
	NextQuestionAt          *string                  `json:"nextQuestionAt,omitempty"`
	PausedByProfileID       *string                  `json:"pausedByProfileId,omitempty"`
	PauseStartedAt          *string                  `json:"pauseStartedAt,omitempty"`
	PauseUntilAt            *string                  `json:"pauseUntilAt,omitempty"`
	PauseForceVotes         int                      `json:"pauseForceVotes"`
	PauseForceVotesRequired int                      `json:"pauseForceVotesRequired"`
	CurrentQuestion         *currentQuestionResponse `json:"currentQuestion,omitempty"`
	Players                 []playerResponse         `json:"players"`
	Questions               []roomQuestionResponse   `json:"questions"`
	RatingChanges           []ratingChangeResponse   `json:"ratingChanges,omitempty"`
	ProfileStats            statsResponse            `json:"profileStats"`
	CreatedAt               string                   `json:"createdAt"`
	UpdatedAt               string                   `json:"updatedAt"`
	FinishedAt              *string                  `json:"finishedAt,omitempty"`
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

type ratingSeasonResponse struct {
	SeasonNumber int    `json:"seasonNumber"`
	Title        string `json:"title"`
	StartsAt     string `json:"startsAt"`
	EndsAt       string `json:"endsAt"`
}

type leaderboardEntryResponse struct {
	Rank        int            `json:"rank"`
	ProfileID   string         `json:"profileId"`
	Player      playerResponse `json:"player"`
	Rating      int            `json:"rating"`
	GamesPlayed int            `json:"gamesPlayed"`
	Wins        int            `json:"wins"`
	Draws       int            `json:"draws"`
}

type leaderboardResponse struct {
	GameType string                     `json:"gameType"`
	Season   ratingSeasonResponse       `json:"season"`
	Entries  []leaderboardEntryResponse `json:"entries"`
}

type socketEvent struct {
	Type    string               `json:"type"`
	Room    *roomResponse        `json:"room,omitempty"`
	Message *roomMessageResponse `json:"message,omitempty"`
	Error   string               `json:"error,omitempty"`
}

type socketMessage struct {
	Type    string  `json:"type"`
	Answer  float64 `json:"answer"`
	Text    string  `json:"text"`
	IsReady bool    `json:"isReady"`
}

type errorResponse struct {
	Error string `json:"error"`
}
