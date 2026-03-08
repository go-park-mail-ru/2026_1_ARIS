package server

import (
	handlers "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(
	authHandler *handlers.AuthHandler,
	feedHandler *handlers.FeedHandler,
	jwtSecret []byte,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	r.Group(func(r chi.Router) {

		r.Use(mymiddleware.AuthMiddleware(jwtSecret))
		r.Get("/api/feed", feedHandler.GetFeed)
	})

	return r
}
