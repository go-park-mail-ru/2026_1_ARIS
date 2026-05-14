package models

import "errors"

const (
	ConstraintUserEmail    = "user_accounts_email_key"
	ConstraintUserUsername = "user_accounts_username_key"
	ConstraintUserPhone    = "user_accounts_phone_key"
)

var (
	ErrEmailAlreadyTaken    = errors.New("email already taken")
	ErrUsernameAlreadyTaken = errors.New("username already taken")
	ErrPhoneAlreadyTaken    = errors.New("phone already taken")
	ErrDuplicateEntry       = errors.New("duplicate entry")
)
