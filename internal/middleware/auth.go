package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
)

func AuthMiddleware(sessionService service.SessionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			fmt.Println("session")
			if err != nil {
				http.Error(w, `{"error":"неавторизован"}`, http.StatusUnauthorized)
				return
			}

			sessionID := models.SessionID(cookie.Value)
			session, err := sessionService.Get(r.Context(), sessionID)
			if err != nil {
				http.Error(w, `{"error":"сессия недействительна или истекла"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", session.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
