package usecase

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
)

type RegisterStepOneInput struct {
	Login     string
	Password1 string
	Password2 string
}

type RegisterInput struct {
	FirstName string
	LastName  string
	Login     string
	Password1 string
	Password2 string
	Birthday  string
	Gender    model.Gender
}

type LoginInput struct {
	Login    string
	Password string
}

type User struct {
	UserAccountID int64
	UserProfileID int64
	ProfileID     int64
	Login         string
	Email         *string
	FirstName     string
	LastName      string
	AvatarURL     *string
	CreatedAt     time.Time
}

type AuthResult struct {
	User    User
	Session model.Session
}
