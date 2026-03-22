package server

import (
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
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
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3001", "http://arisnet.ru", "https://arisnet.ru"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(mymiddleware.AuthMiddleware(sessSvc))
			r.Get("/me", authHandler.Me)
			r.Post("/logout", authHandler.Logout)
		})
	})
	// Публичная лента (без авторизации)
	r.Get("/api/public/feed", feedHandler.GetPublicFeed)
	r.Get("/api/public/popular-users", userHandler.GetPublicPopularUsers)
	r.Get("/api/public/popular-posts", feedHandler.GetPublicPopularPosts)
	r.Get("/image-proxy", proxy.ImageProxy)
	r.Group(func(r chi.Router) {
		r.Use(mymiddleware.AuthMiddleware(sessSvc))
		r.Get("/api/users/suggested", userHandler.GetSuggestedUsers)
		r.Get("/api/users/latest-events", userHandler.GetLatestEvents)
		r.Get("/api/feed", feedHandler.GetFeed)
		r.Get("/api/posts/popular", feedHandler.GetPopularPosts)
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)
	// Раздача статических файлов (изображений)

	return r
}
