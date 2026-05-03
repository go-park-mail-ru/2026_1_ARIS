package service

import "github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

type Result struct {
	Users       []UserResult
	Communities []CommunityResult
}

type UserResult struct {
	ProfileID     int64
	UserAccountID int64
	Username      string
	FirstName     string
	LastName      string
	AvatarID      *int64
	AvatarURL     *string
}

type CommunityResult struct {
	ID           int64
	ProfileID    int64
	Username     string
	Title        string
	Bio          *string
	Type         models.CommunityType
	AvatarID     *int64
	AvatarURL    *string
	CoverMediaID *int64
	CoverURL     *string
}
