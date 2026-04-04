package server

import (
	chathandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/friend"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/profile"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/proxy"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/user"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(
	authHandler *auth.AuthHandler,
	sessSvc session.SessionService,
	feedHandler *feed.FeedHandler,
	userHandler *user.UserHandler,
	profileHandler *profile.ProfileHandler,
	chatHandler *chathandler.ChatHandler,
	friendshipHandler *friend.FriendHandler,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3001", "http://arisnet.ru", "https://arisnet.ru"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/register/step-one", authHandler.ValidateRegisterStepOne)
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(mymiddleware.AuthMiddleware(sessSvc))
			r.Get("/me", authHandler.Me)
			r.Post("/logout", authHandler.Logout)
		})
	})

	r.Get("/api/public/feed", feedHandler.GetPublicFeed)
	r.Get("/api/public/popular-users", userHandler.GetPublicPopularUsers)
	r.Get("/api/public/popular-posts", feedHandler.GetPublicPopularPosts)
	r.Get("/image-proxy", proxy.ImageProxy)

	r.Get("/api/users/{id}/friends", friendshipHandler.GetUsersFriends)

	r.Group(func(r chi.Router) {
		r.Use(mymiddleware.AuthMiddleware(sessSvc))
		r.Get("/api/users/suggested", userHandler.GetSuggestedUsers)
		r.Get("/api/users/latest-events", userHandler.GetLatestEvents)
		r.Get("/api/feed", feedHandler.GetFeed)
		r.Get("/api/posts/popular", feedHandler.GetPopularPosts)
		r.Get("/api/profile/me", profileHandler.GetProfileMe)
		r.Get("/api/profile/{id}", profileHandler.GetProfileByID)
		r.Patch("/api/profile/me/edit", profileHandler.EditProfileMe)
		r.Get("/api/chats", chatHandler.GetChats)
		r.Post("/api/chats", chatHandler.CreateChat)
		r.Get("/api/chats/{chatID}/messages", chatHandler.GetMessages)
		r.Post("/api/chats/{chatID}/messages", chatHandler.SendMessage)

	})

	r.Route("/api/friends", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(mymiddleware.AuthMiddleware(sessSvc))

			// Заявки в друзья
			r.Post("/request", friendshipHandler.RequestFriendship)
			r.Post("/accept/{requesterID}", friendshipHandler.AcceptFriendRequest)
			r.Post("/decline/{requesterID}", friendshipHandler.DeclineFriendRequest)
			r.Delete("/request/{addresseeID}", friendshipHandler.RevokeFriendRequest)

			// Друзья
			r.Get("/{status}", friendshipHandler.GetFriends)
			r.Get("/", friendshipHandler.GetFriends)
			r.Delete("/{userID}", friendshipHandler.DeleteFriend)

			// Входящие/исходящие
			r.Get("/requests/incoming/{status}", friendshipHandler.GetIncomingFriendRequests)
			r.Get("/requests/incoming", friendshipHandler.GetIncomingFriendRequests)
			r.Get("/requests/outgoing", friendshipHandler.GetOutgoingFriendRequests)
			r.Get("/requests/outgoing/{status}", friendshipHandler.GetOutgoingFriendRequests)
		})
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	return r
}
