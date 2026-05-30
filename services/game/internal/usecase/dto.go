package usecase

type CreateRoomInput struct {
	Title            string
	GameType         string
	MaxPlayers       int
	Password         string
	IsRanked         bool
	QuestionCount    int
	AnswerTimeoutSec int
	RoundPauseSec    int
}

type CreatePublicRoomInput struct {
	AnswerTimeoutSec int
	RoundPauseSec    int
}

type LocalizedText struct {
	RU string
	EN string
}

type QuestionInput struct {
	GameType      string
	Text          LocalizedText
	CorrectAnswer float64
	IsActive      bool
}

type Player struct {
	ProfileID            int64
	UserAccountID        int64
	Name                 string
	Username             string
	FirstName            string
	LastName             string
	AvatarID             *int64
	Score                int
	IsReady              bool
	HasAnswered          bool
	PauseUsed            bool
	ForceResumeRequested bool
	IsMe                 bool
}

type Question struct {
	ID            int64
	Text          LocalizedText
	CorrectAnswer float64
	IsActive      bool
}

type Answer struct {
	ProfileID      int64
	Answer         float64
	Distance       float64
	AnsweredAt     string
	ResponseTimeMs int64
}

type RoomMessage struct {
	ID        int64
	RoomID    int64
	ProfileID int64
	Text      string
	Author    Player
	CreatedAt string
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
	Text        LocalizedText
	StartedAt   *string
	DeadlineAt  *string
	HasAnswered bool
}

type Room struct {
	ID                      int64
	Title                   string
	InviteCode              string
	GameType                string
	Status                  string
	CreatedByProfileID      int64
	WinnerProfileID         *int64
	MaxPlayers              int
	HasPassword             bool
	Password                string
	IsRanked                bool
	IsPublicLobby           bool
	QuestionCount           int
	AnswerTimeoutSec        int
	RoundPauseSec           int
	Creator                 Player
	CurrentQuestionIndex    int
	NextQuestionAt          *string
	PausedByProfileID       *int64
	PauseStartedAt          *string
	PauseUntilAt            *string
	PauseForceVotes         int
	PauseForceVotesRequired int
	CurrentQuestion         *CurrentQuestion
	Players                 []Player
	Questions               []RoomQuestion
	RatingChanges           []RatingChange
	ProfileStats            Stats
	CreatedAt               string
	UpdatedAt               string
	FinishedAt              *string
}

type PublicJoinResult struct {
	Token string
	Room  Room
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

type RatingSeason struct {
	SeasonNumber int
	Title        string
	StartsAt     string
	EndsAt       string
}

type RatingChange struct {
	ProfileID    int64
	Score        int
	Place        int
	BeforeRating int
	AfterRating  int
	RatingDelta  int
	RatingWeight float64
	SeasonNumber int
	SeasonTitle  string
}

type LeaderboardEntry struct {
	Rank        int
	ProfileID   int64
	Player      Player
	Rating      int
	GamesPlayed int
	Wins        int
	Draws       int
}

type Leaderboard struct {
	GameType string
	Season   RatingSeason
	Entries  []LeaderboardEntry
}
