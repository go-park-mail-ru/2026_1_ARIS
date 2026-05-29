package middleware

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/requestcontext"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

func AuthMiddleware(authClient authpb.AuthServiceClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			session, err := authClient.ValidateSession(r.Context(), &authpb.ValidateSessionRequest{SessionId: cookie.Value})
			if err != nil {
				utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "session is invalid or expired"})
				return
			}

			ctx := requestcontext.WithUserID(r.Context(), session.GetUserAccountId())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
