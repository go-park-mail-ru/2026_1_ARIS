package authhandler

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	AvatarLink string `json:"avatarLink"`
	ProfileID  int64  `json:"profileID"`
}
