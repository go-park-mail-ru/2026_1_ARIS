package service

import (
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
)

type userService struct {
	repo repository.UserRepo
}

type UserService interface {
	GetUserProfileAuthorByPost(post models.Post) models.UserProfile
}
