package server

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	chathandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/friend"
	mediahandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/media"
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
	mediaHandler := &mediahandler.MediaHandler{}
	profileHandler := profile.NewProfileHandler(nil, nil, nil)
	chatHandler := chathandler.NewChatHandler(nil, nil, nil, nil, nil) // добавили hub = nil
	friendshipHandler := &friend.FriendHandler{}
	wsHandler := chathandler.NewWebSocketHandler(nil, nil) // создаём wsHandler

	router := NewRouter(authHandler, nil, feedHandler, userHandler, mediaHandler, profileHandler, chatHandler, friendshipHandler, wsHandler)

	assert.NotNil(t, router)
	assert.IsType(t, &chi.Mux{}, router)
}
