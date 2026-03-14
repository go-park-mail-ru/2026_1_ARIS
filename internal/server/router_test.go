package server

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	handlers "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
)

func TestNewRouter(t *testing.T) {
	// Создаём хендлеры с nil-сервисами (явное приведение nil к интерфейсам)
	authHandler := handlers.NewAuthHandler(nil, nil, nil)
	feedHandler := handlers.NewFeedHandler(
		service.PostService(nil),
		service.MediaService(nil),
		service.UserService(nil),
	)
	userHandler := &handlers.UserHandler{}
	router := NewRouter(authHandler, nil, feedHandler, userHandler)
	assert.NotNil(t, router)
	assert.IsType(t, &chi.Mux{}, router)
}
