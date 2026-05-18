package tarantool

import "time"

type Education struct {
	Institution *string `json:"institution,omitempty"`
	Group       *string `json:"group,omitempty"`
}

type Work struct {
	Company  *string `json:"company,omitempty"`
	JobTitle *string `json:"jobTitle,omitempty"`
}

type ProfileDetails struct {
	ProfileID     int64       `json:"profileId"`
	UserProfileID int64       `json:"userProfileId"`
	UserAccountID int64       `json:"userAccountId"`
	Username      string      `json:"username"`
	AvatarID      *int64      `json:"avatarId,omitempty"`
	FirstName     string      `json:"firstName"`
	LastName      string      `json:"lastName"`
	Bio           *string     `json:"bio,omitempty"`
	ImageLink     *string     `json:"imageLink,omitempty"`
	Gender        string      `json:"gender"`
	BirthdayDate  time.Time   `json:"birthdayDate"`
	NativeTown    *string     `json:"nativeTown,omitempty"`
	Phone         *string     `json:"phone,omitempty"`
	Email         *string     `json:"email,omitempty"`
	Town          *string     `json:"town,omitempty"`
	Education     []Education `json:"education,omitempty"`
	Work          []Work      `json:"work,omitempty"`
	Interests     *string     `json:"interests,omitempty"`
	FavMusic      *string     `json:"favMusic,omitempty"`
	IsOnline      bool        `json:"isOnline"`
	LastSeenAt    *time.Time  `json:"lastSeenAt,omitempty"`
}

type AuthUser struct {
	UserAccountID int64      `json:"userAccountId"`
	UserProfileID int64      `json:"userProfileId"`
	ProfileID     int64      `json:"profileId"`
	Login         string     `json:"login"`
	Email         *string    `json:"email,omitempty"`
	FirstName     string     `json:"firstName"`
	LastName      string     `json:"lastName"`
	AvatarID      *int64     `json:"avatarId,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	IsOnline      bool       `json:"isOnline"`
	LastSeenAt    *time.Time `json:"lastSeenAt,omitempty"`
}

type ProfileSummary struct {
	ProfileID     int64      `json:"profileId"`
	UserAccountID int64      `json:"userAccountId"`
	FirstName     string     `json:"firstName"`
	LastName      string     `json:"lastName"`
	Username      string     `json:"username"`
	AvatarID      *int64     `json:"avatarId,omitempty"`
	IsOnline      bool       `json:"isOnline"`
	LastSeenAt    *time.Time `json:"lastSeenAt,omitempty"`
}

type PresenceStatus struct {
	UserAccountID int64
	IsOnline      bool
	LastSeenAt    time.Time
	UpdatedAt     time.Time
	Connections   uint64
}
