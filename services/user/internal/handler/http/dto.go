package http

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
)

type educationResponse struct {
	Institution *string `json:"institution,omitempty"`
	Group       *string `json:"grade,omitempty"`
}

type workResponse struct {
	Company  *string `json:"company,omitempty"`
	JobTitle *string `json:"jobTitle,omitempty"`
}

type profileResponse struct {
	ProfileID     int64               `json:"profileId"`
	UserAccountID int64               `json:"userAccountId"`
	Username      string              `json:"username,omitempty"`
	FirstName     string              `json:"firstName,omitempty"`
	LastName      string              `json:"lastName,omitempty"`
	Bio           *string             `json:"bio,omitempty"`
	ImageLink     *string             `json:"imageLink,omitempty"`
	Gender        model.Gender        `json:"gender,omitempty"`
	BirthdayDate  string              `json:"birthday,omitempty"`
	NativeTown    *string             `json:"nativeTown,omitempty"`
	Phone         *string             `json:"phone,omitempty"`
	Email         *string             `json:"email,omitempty"`
	Town          *string             `json:"town,omitempty"`
	Education     []educationResponse `json:"education,omitempty"`
	Work          []workResponse      `json:"work,omitempty"`
	Interests     *string             `json:"interests,omitempty"`
	FavMusic      *string             `json:"favMusic,omitempty"`
	IsOnline      bool                `json:"isOnline"`
	LastSeenAt    *string             `json:"lastSeenAt,omitempty"`
}

type updateProfileRequest struct {
	Username     *string       `json:"login,omitempty" validate:"omitempty,omitnil,alphanumunicode"`
	Email        *string       `json:"email,omitempty" validate:"omitempty,omitnil,email"`
	Phone        *string       `json:"phone,omitempty" validate:"omitempty,omitnil,e164"`
	FirstName    *string       `json:"firstName,omitempty" validate:"omitempty,omitnil,alphaunicode"`
	LastName     *string       `json:"lastName,omitempty" validate:"omitempty,omitnil,alphaunicode"`
	Bio          *string       `json:"bio,omitempty"`
	BirthdayDate *string       `json:"birthdayDate,omitempty" validate:"omitempty,omitnil,min=8,max=10,datetime=2006-01-02"`
	Gender       *model.Gender `json:"gender,omitempty" validate:"omitempty,omitnil,oneof=male female"`
	AvatarID     *int64        `json:"avatarID,omitempty"`
	RemoveAvatar *bool         `json:"removeAvatar,omitempty"`
	NativeTown   *string       `json:"nativeTown,omitempty"`
	Town         *string       `json:"town,omitempty"`
	Institution  *string       `json:"institution,omitempty"`
	Group        *string       `json:"group,omitempty"`
	Company      *string       `json:"company,omitempty"`
	JobTitle     *string       `json:"jobTitle,omitempty"`
	Interests    *string       `json:"interests,omitempty"`
	FavMusic     *string       `json:"favMusic,omitempty"`
}

type settingsUpdateRequest struct {
	Language *model.LanguageSetting `json:"language" validate:"omitempty,omitnil,omitzero,oneofci=RU EN"`
	Theme    *model.ThemeSetting    `json:"theme" validate:"omitempty,omitnil,omitzero,oneofci=light dark"`
}

type latestEventDTO struct {
	ID         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Username   string `json:"username"`
	AvatarLink string `json:"avatarLink"`
	Type       int    `json:"type"`
}

type latestEventsResponse struct {
	Items []latestEventDTO `json:"items"`
}

type userCardDTO struct {
	ID         string  `json:"id"`
	FirstName  string  `json:"firstName"`
	LastName   string  `json:"lastName"`
	Username   string  `json:"username"`
	AvatarLink string  `json:"avatarLink"`
	IsOnline   bool    `json:"isOnline"`
	LastSeenAt *string `json:"lastSeenAt,omitempty"`
}

type userCardsResponse struct {
	Items []userCardDTO `json:"items"`
}

type friendDTO struct {
	AvatarID   *int64                 `json:"avatarID,omitempty"`
	ProfileID  int64                  `json:"profileID"`
	FirstName  string                 `json:"firstName"`
	LastName   string                 `json:"lastName"`
	Username   string                 `json:"login"`
	Status     model.FriendshipStatus `json:"status"`
	AvatarLink *string                `json:"link,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

type friendsResponse struct {
	Friends []friendDTO `json:"friends"`
}

type friendRequest struct {
	FriendID int64 `json:"friendID"`
}
