package dto

type FriendDTO struct {
	AvatarID   *int64  `db:"avatar_id" json:"avatarID"`
	ProfileID  int64   `db:"id" json:"profileID"`
	FirstName  string  `db:"first_name" json:"firstName"`
	LastName   string  `db:"last_name" json:"lastName"`
	Username   string  `db:"username" json:"login"`
	Status     string  `db:"status" json:"status"`
	AvatarLink *string `db:"link,omitempty" json:"link,omitempty"`
}
