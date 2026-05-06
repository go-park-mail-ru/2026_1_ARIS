package middleware

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"go.uber.org/zap"
)

func AuthMiddleware(sessionService session.SessionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.FromContext(r.Context())

			cookie, err := r.Cookie("session_id")
			if err != nil {
				http.Error(w, `{"error":"неавторизован"}`, http.StatusUnauthorized)
				return
			}

			sessionID := models.SessionID(cookie.Value)

			session, err := sessionService.Get(r.Context(), sessionID)
			if err != nil {
				log.Error("invalid_session", zap.Error(err), zap.String("session_id", string(sessionID)))
				http.Error(w, `{"error":"сессия недействительна или истекла"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", session.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
