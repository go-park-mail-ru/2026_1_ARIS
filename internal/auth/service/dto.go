package authservice

import "time"

// request
type RegisterServiceDTO struct {
	FirstName string
	LastName  string
	Login     string
	Password  string
	Birthday  string
	Gender    int
}

type LoginServiceRequestDTO struct {
	Login    string
	Password string
}

type LoginServiceResultDTO struct {
	UserAccountID int64
	ProfileID     int64
	FirstName     string
	LastName      string
	AvatarLink    string

	SessionID string
	ExpiresAt time.Time
}
