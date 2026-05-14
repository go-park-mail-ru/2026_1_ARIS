package middleware

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"go.uber.org/zap"
)

func AuthMiddleware(authClient authpb.AuthServiceClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logg := logger.FromContext(r.Context())

			cookie, err := r.Cookie("session_id")
			if err != nil {
				utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			session, err := authClient.ValidateSession(r.Context(), &authpb.ValidateSessionRequest{SessionId: cookie.Value})
			if err != nil {
				if logg != nil {
					logg.Error("invalid_session", zap.Error(err), zap.String("session_id", cookie.Value))
				}
				utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "session is invalid or expired"})
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", session.GetUserAccountId())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
