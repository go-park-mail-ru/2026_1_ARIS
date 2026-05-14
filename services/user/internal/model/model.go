package model

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type Gender string

const (
	Male   Gender = "male"
	Female Gender = "female"
)

type FriendshipStatus string

const (
	FriendshipPending  FriendshipStatus = "pending"
	FriendshipAccepted FriendshipStatus = "accepted"
)

type LanguageSetting string

const (
	LanguageRU LanguageSetting = "RU"
	LanguageEN LanguageSetting = "EN"
)

type ThemeSetting string

const (
	ThemeLight ThemeSetting = "light"
	ThemeDark  ThemeSetting = "dark"
)

type UserAccount struct {
	ID           int64     `db:"id"`
	Uid          uuid.UUID `db:"uid"`
	Username     string    `db:"username"`
	Email        *string   `db:"email"`
	Phone        *string   `db:"phone"`
	PasswordHash string    `db:"password_hash"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func NewUserAccount(username string, email, phone *string, passwordHash string) *UserAccount {
	now := time.Now()
	return &UserAccount{
		ID:           rand.Int64(),
		Uid:          uuid.New(),
		Username:     username,
		Email:        email,
		Phone:        phone,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

type Profile struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	AvatarID  *int64    `db:"avatar_id"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewProfile(avatarID *int64) *Profile {
	now := time.Now()
	return &Profile{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		AvatarID:  avatarID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type UserProfile struct {
	ID            int64     `db:"id"`
	Uid           uuid.UUID `db:"uid"`
	UserAccountID int64     `db:"user_account_id"`
	ProfileID     int64     `db:"profile_id"`
	FirstName     string    `db:"first_name"`
	LastName      string    `db:"last_name"`
	Bio           *string   `db:"bio"`
	BirthdayDate  time.Time `db:"birthday_date"`
	Gender        Gender    `db:"gender"`
	NativeTown    *string   `db:"native_town"`
	Town          *string   `db:"town"`
	Institution   *string   `db:"institution"`
	Group         *string   `db:"study_group"`
	Company       *string   `db:"company"`
	JobTitle      *string   `db:"job_title"`
	Interests     *string   `db:"interests"`
	FavMusic      *string   `db:"fav_music"`
	IsActive      bool      `db:"is_active"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

func NewUserProfile(userAccountID, profileID int64, firstName, lastName string, bio *string, birthday time.Time, gender Gender) *UserProfile {
	now := time.Now()
	return &UserProfile{
		ID:            rand.Int64(),
		Uid:           uuid.New(),
		UserAccountID: userAccountID,
		ProfileID:     profileID,
		FirstName:     firstName,
		LastName:      lastName,
		Bio:           bio,
		BirthdayDate:  birthday,
		Gender:        gender,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

type UserSettings struct {
	UserAccountID int64           `db:"user_account_id" json:"userAccountID"`
	Language      LanguageSetting `db:"lang" json:"language"`
	Theme         ThemeSetting    `db:"theme" json:"theme"`
}

type Friend struct {
	ProfileID int64            `db:"id" json:"profileId"`
	AvatarID  *int64           `db:"avatar_id" json:"avatarId,omitempty"`
	FirstName string           `db:"first_name" json:"firstName"`
	LastName  string           `db:"last_name" json:"lastName"`
	Username  string           `db:"username" json:"username"`
	Link      *string          `db:"link" json:"avatarLink,omitempty"`
	Status    FriendshipStatus `db:"status" json:"status"`
	CreatedAt time.Time        `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time        `db:"updated_at" json:"updatedAt"`
}
