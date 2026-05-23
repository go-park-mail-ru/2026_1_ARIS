package http

type registerStepOneRequest struct {
	Login     string `json:"login" validate:"required,alphanumunicode"`
	Password1 string `json:"password1" validate:"required,min=6,max=72,printascii"`
	Password2 string `json:"password2" validate:"required,min=6,max=72,printascii"`
}

type registerRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Login     string `json:"login"`
	Password1 string `json:"password1"`
	Password2 string `json:"password2"`
	Birthday  string `json:"birthday"`
	Gender    int    `json:"gender"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	OldPassword  string `json:"oldPassword"`
	NewPassword1 string `json:"newPassword1"`
	NewPassword2 string `json:"newPassword2"`
}

type userResponse struct {
	ID            int64   `json:"id"`
	UserAccountID int64   `json:"userAccountId"`
	FirstName     string  `json:"firstName"`
	LastName      string  `json:"lastName"`
	Login         string  `json:"login"`
	Email         string  `json:"email"`
	Role          string  `json:"role"`
	AvatarURL     *string `json:"avatarUrl,omitempty"`
	AvatarLink    *string `json:"avatarLink,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}

type meResponse struct {
	ID         string  `json:"id"`
	FirstName  string  `json:"firstName"`
	LastName   string  `json:"lastName"`
	Login      string  `json:"login"`
	Email      string  `json:"email"`
	Role       string  `json:"role"`
	AvatarLink *string `json:"avatarLink,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}
