package server

import (
	handlers "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(
	authHandler *handlers.AuthHandler,
	sessSvc service.SessionService,
	feedHandler *handlers.FeedHandler,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://arisnet.ru", "https://arisnet.ru"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api/auth", func(r chi.Router) {
		// Публичные роуты
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		// Защищённые роуты
		r.Group(func(r chi.Router) {
			r.Use(mymiddleware.AuthMiddleware(sessSvc))
			r.Get("/me", authHandler.Me)
			r.Post("/logout", authHandler.Logout)
		})
	})

	// Остальные защищённые роуты вне /api/auth
	r.Group(func(r chi.Router) {
		r.Use(mymiddleware.AuthMiddleware(sessSvc))
		r.Get("/api/feed", feedHandler.GetFeed)
	})

	return r
}
