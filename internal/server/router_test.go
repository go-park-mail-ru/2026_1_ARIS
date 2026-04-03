package server

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	mediahandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/media"
	userhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
)

func TestNewRouter(t *testing.T) {
	// Создаём хендлеры с nil-сервисами (явное приведение nil к интерфейсам)
	authHandler := auth.NewAuthHandler(nil, nil, nil)
	feedHandler := feed.NewFeedHandler(
		post.PostService(nil),
		media.MediaService(nil),
		userservice.UserService(nil),
	)
	userHandler := &userhandler.UserHandler{}
	mediaHandler := &mediahandler.MediaHandler{}
	router := NewRouter(authHandler, nil, feedHandler, userHandler, mediaHandler)
	assert.NotNil(t, router)
	assert.IsType(t, &chi.Mux{}, router)
}
