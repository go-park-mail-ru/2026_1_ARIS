package http

type response struct {
	Users       []userResult      `json:"users"`
	Communities []communityResult `json:"communities"`
	Posts       []postResult      `json:"posts"`
}

type userResult struct {
	ProfileID     int64   `json:"profileId"`
	UserAccountID int64   `json:"userAccountId"`
	Username      string  `json:"username"`
	FirstName     string  `json:"firstName"`
	LastName      string  `json:"lastName"`
	AvatarID      *int64  `json:"avatarId,omitempty"`
	AvatarURL     *string `json:"avatarUrl,omitempty"`
}

type communityResult struct {
	ID           int64   `json:"id"`
	ProfileID    int64   `json:"profileId"`
	Username     string  `json:"username"`
	Title        string  `json:"title"`
	Bio          *string `json:"bio,omitempty"`
	Type         string  `json:"type"`
	AvatarID     *int64  `json:"avatarId,omitempty"`
	AvatarURL    *string `json:"avatarUrl,omitempty"`
	CoverMediaID *int64  `json:"coverId,omitempty"`
	CoverURL     *string `json:"coverUrl,omitempty"`
}

type postResult struct{}

type errorResponse struct {
	Error string `json:"error"`
}
