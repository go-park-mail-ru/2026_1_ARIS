package common

//go:generate go run github.com/mailru/easyjson/easyjson -all $GOFILE

type ErrorResponse struct {
	Error string `json:"error"`
}
