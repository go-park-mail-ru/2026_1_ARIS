package middleware

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestIDMiddleware(base *zap.Logger) func(http.Handler) http.Handler {
	if base == nil {
		base = zap.L()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.New().String()
			w.Header().Set("X-Request-ID", requestID)

			reqLogger := base.With(zap.String("request_id", requestID))
			ctx := logger.WithLogger(r.Context(), reqLogger)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
