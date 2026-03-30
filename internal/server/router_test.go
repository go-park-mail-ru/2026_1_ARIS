package server

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	chathandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/profile"
	userhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
)

func TestNewRouter(t *testing.T) {
	authHandler := auth.NewAuthHandler(nil, nil, nil)
	feedHandler := feed.NewFeedHandler(
		post.PostService(nil),
		media.MediaService(nil),
		userservice.UserService(nil),
	)
	userHandler := &userhandler.UserHandler{}
	profileHandler := profile.NewProfileHandler(nil, nil, nil)
	chatHandler := chathandler.NewChatHandler(nil, nil, nil, nil)

	router := NewRouter(authHandler, nil, feedHandler, userHandler, profileHandler, chatHandler)
	assert.NotNil(t, router)
	assert.IsType(t, &chi.Mux{}, router)
}
