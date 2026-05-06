package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAccessLogMiddleware(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotNil(t, logger.FromContext(r.Context()))
		w.WriteHeader(http.StatusCreated)
	})
	handler := AccessLogMiddleware(zap.NewNop())(next)
	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	wrapped := &statusResponseWriter{ResponseWriter: rec}
	wrapped.Flush()
	_, _, err := wrapped.Hijack()
	require.EqualError(t, err, "response writer does not support hijacking")
}

func TestRequestIDMiddleware(t *testing.T) {
	t.Parallel()

	called := false
	handler := RequestIDMiddleware(zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		require.NotEmpty(t, w.Header().Get("X-Request-ID"))
		require.NotNil(t, logger.FromContext(r.Context()))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	require.True(t, called)
	require.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

func TestXSSMiddlewares(t *testing.T) {
	t.Parallel()

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	for _, mw := range XSSMiddlewares() {
		handler = mw(handler).ServeHTTP
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	require.True(t, called)
	require.Equal(t, "default-src 'self';", rec.Header().Get("Content-Security-Policy"))
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}
