package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"неавторизован"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"неправильный заголовок"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"недействительный токен"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "user", token.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
