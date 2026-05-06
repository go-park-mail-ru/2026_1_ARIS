package service

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type Education struct {
	Institution *string
	Group       *string
}

type Work struct {
	Company  *string
	JobTitle *string
}

type ProfileDetails struct {
	ProfileID     int64
	UserProfileID int64
	UserAccountID int64
	Username      string
	AvatarID      *int64
	FirstName     string
	LastName      string
	Bio           *string
	ImageLink     *string
	Gender        models.Gender
	BirthdayDate  time.Time
	NativeTown    *string
	Phone         *string
	Email         *string
	Town          *string
	Education     []Education
	Work          []Work
	Interests     *string
	FavMusic      *string
}

type UserCard struct {
	ID         int64
	FirstName  string
	LastName   string
	Username   string
	AvatarLink string
}

type LatestEvent struct {
	UserCard
	Type int
}
