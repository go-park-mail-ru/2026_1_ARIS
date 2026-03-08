package repository

import "github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

type Repository struct {
	UserRepo    UserRepo
	PostRepo    PostRepo
	SessionRepo SessionRepo
}

func NewRepository() *Repository {
	return &Repository{
		UserRepo: &inmemoryUserRepo{},
		PostRepo: &inmemoryPostRepo{},
		SessionRepo: &inmemorySessionRepo{
			sessions: make(map[models.SessionID]models.Session),
		},
	}
}
