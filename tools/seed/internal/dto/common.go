package dto

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models"
)

type CommonResponse struct {
	Message string `json:"message"`
}

type CommonErrorResponse struct {
	Error string `json:"error"`
}

type UpdateFullProfileRequestDTO struct {
	UserAccountID int64   `json:"-"`
	Username      *string `json:"login,omitempty" db:"username,omitempty" validate:"omitempty,omitnil,alphanumunicode"`
	Email         *string `json:"email,omitempty" db:"email,omitempty" validate:"omitempty,omitnil,email"`
	Phone         *string `json:"phone,omitempty" db:"phone,omitempty" validate:"omitempty,omitnil,e164"`

	ProfileID     int64          `json:"-"`
	UserProfileID int64          `json:"-"`
	FirstName     *string        `db:"first_name,omitempty" json:"firstName,omitempty" validate:"omitempty,omitnil,alphaunicode"`
	LastName      *string        `db:"last_name,omitempty" json:"lastName,omitempty" validate:"omitempty,omitnil,alphaunicode"`
	Bio           *string        `db:"bio,omitempty" json:"bio,omitempty"`
	BirthdayDate  *string        `db:"birthday_date,omitempty" json:"birthdayDate,omitempty" validate:"omitempty,omitnil,min=8,max=10,datetime=2006-01-02" example:"2000-11-26"`
	Gender        *models.Gender `db:"gender,omitempty" json:"gender,omitempty" validate:"omitempty,omitnil,oneof=male female"`
	AvatarID      *int64         `db:"avatar_id,omitempty" json:"avatarID,omitempty"`
	RemoveAvatar  *bool          `json:"removeAvatar,omitempty"`

	NativeTown  *string `db:"native_town,omitempty" json:"nativeTown,omitempty"`
	Town        *string `db:"town,omitempty" json:"town,omitempty"`
	Institution *string `db:"institution,omitempty" json:"institution,omitempty"`
	Group       *string `db:"study_group,omitempty" json:"group,omitempty"`
	Company     *string `db:"company,omitempty" json:"company,omitempty"`
	JobTitle    *string `db:"job_title,omitempty" json:"jobTitle,omitempty"`
	Interests   *string `db:"interests,omitempty" json:"interests,omitempty"`
	FavMusic    *string `db:"fav_music,omitempty" json:"favMusic,omitempty"`
}

type UpdateUserAccountDTO struct {
	ID       int64
	Username *string `db:"username"`
	Email    *string `db:"email"`
	Phone    *string `db:"phone"`
}

type UpdateUserProfileDTO struct {
	ID           int64
	FirstName    *string        `db:"first_name" json:"firstName"`
	LastName     *string        `db:"last_name" json:"lastName"`
	Bio          *string        `db:"bio,omitempty" json:"bio,omitempty"`
	BirthdayDate *time.Time     `db:"birthday_date,omitempty" json:"birthdayDate"`
	Gender       *models.Gender `db:"gender" json:"gender"`

	NativeTown  *string `db:"native_town,omitempty" json:"nativeTown,omitempty"`
	Town        *string `db:"town,omitempty" json:"town,omitempty"`
	Institution *string `db:"institution,omitempty" json:"institution,omitempty"`
	Group       *string `db:"study_group,omitempty" json:"group,omitempty"`
	Company     *string `db:"company,omitempty" json:"company,omitempty"`
	JobTitle    *string `db:"job_title,omitempty" json:"jobTitle,omitempty"`
	Interests   *string `db:"interests,omitempty" json:"interests,omitempty"`
	FavMusic    *string `db:"fav_music,omitempty" json:"favMusic,omitempty"`
}

func (d UpdateUserAccountDTO) HasUpdates() bool {
	return d.Email != nil || d.Phone != nil || d.Username != nil
}

func (d UpdateUserProfileDTO) HasUpdates() bool {
	return d.FirstName != nil || d.LastName != nil ||
		d.Bio != nil || d.BirthdayDate != nil || d.Gender != nil ||
		d.NativeTown != nil || d.Town != nil ||
		d.Institution != nil || d.Group != nil ||
		d.Company != nil || d.JobTitle != nil ||
		d.Interests != nil || d.FavMusic != nil
}

type MediaRequestData struct {
	MediaID  int64  `json:"mediaID"`
	MediaURL string `json:"mediaURL"`
}

type PostUpdateDTO struct {
	Text *string `json:"text"`
}

type UserSettingsUpdate struct {
	Language *models.LanguageSetting `json:"language" validate:"omitempty,omitnil,omitzero,oneofci=RU EN"`
	Theme    *models.ThemeSetting    `json:"theme" validate:"omitempty,omitnil,omitzero,oneofci=light dark"`
}

func (u *UserSettingsUpdate) IsEmpty() bool {
	return u.Language == nil && u.Theme == nil
}

type FriendDTO struct {
	AvatarID   *int64    `db:"avatar_id" json:"avatarID"`
	ProfileID  int64     `db:"id" json:"profileID"`
	FirstName  string    `db:"first_name" json:"firstName"`
	LastName   string    `db:"last_name" json:"lastName"`
	Username   string    `db:"username" json:"login"`
	Status     string    `db:"status" json:"status"`
	AvatarLink *string   `db:"link,omitempty" json:"link,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`
}
