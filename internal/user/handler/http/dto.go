package http

import "github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

type EducationResponse struct {
	Institution *string `json:"institution,omitempty"`
	Group       *string `json:"grade,omitempty"`
}

type WorkResponse struct {
	Company  *string `json:"company,omitempty"`
	JobTitle *string `json:"jobTitle,omitempty"`
}

type GetProfileMeResponse struct {
	ProfileID     int64               `json:"profileId"`
	UserAccountID int64               `json:"userAccountId"`
	Username      string              `json:"username,omitempty"`
	FirstName     string              `json:"firstName,omitempty"`
	LastName      string              `json:"lastName,omitempty"`
	Bio           *string             `json:"bio,omitempty"`
	ImageLink     *string             `json:"imageLink,omitempty"`
	Gender        models.Gender       `json:"gender,omitempty"`
	BirthdayDate  string              `json:"birthday,omitempty"`
	NativeTown    *string             `json:"nativeTown,omitempty"`
	Phone         *string             `json:"phone,omitempty"`
	Email         *string             `json:"email,omitempty"`
	Town          *string             `json:"town,omitempty"`
	Education     []EducationResponse `json:"education,omitempty"`
	Work          []WorkResponse      `json:"work,omitempty"`
	Interests     *string             `json:"interests,omitempty"`
	FavMusic      *string             `json:"favMusic,omitempty"`
}

type latestEventDTO struct {
	Id         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Username   string `json:"username"`
	AvatarLink string `json:"avatarLink"`
	Type       int    `json:"type"`
}

type latestEventsResponse struct {
	Items []latestEventDTO `json:"items"`
}

type suggestedUserDTO struct {
	Id         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Username   string `json:"username"`
	AvatarLink string `json:"avatarLink"`
}

type suggestedUsersResponse struct {
	Items []suggestedUserDTO `json:"items"`
}

type errorResponse struct {
	Error string `json:"error"`
}
