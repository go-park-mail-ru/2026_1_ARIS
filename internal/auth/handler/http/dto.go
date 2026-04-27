package http

type registerRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Login     string `json:"login"`
	Password  string `json:"password"`
	Birthday  string `json:"birthday"`
	Gender    string `json:"gender"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type userResponse struct {
	ID            int64   `json:"id"`
	UserAccountID int64   `json:"userAccountId"`
	FirstName     string  `json:"firstName"`
	LastName      string  `json:"lastName"`
	AvatarURL     *string `json:"avatarUrl,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}

type authResponse struct {
	User userResponse `json:"user"`
}

type errorResponse struct {
	Error string `json:"error"`
}
