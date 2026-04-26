package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestStatusResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &statusResponseWriter{ResponseWriter: rec, status: http.StatusOK}

	w.WriteHeader(http.StatusTeapot)

	assert.Equal(t, http.StatusTeapot, w.status)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

func TestAccessLogMiddleware_WithNilLogger(t *testing.T) {
	mw := AccessLogMiddleware(nil)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestRequestIDMiddleware_SetsHeader(t *testing.T) {
	mw := RequestIDMiddleware(zap.NewNop())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotNil(t, r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

func TestXSSMiddlewares_SetHeaders(t *testing.T) {
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var handler http.Handler = final
	for i := len(XSSMiddlewares()) - 1; i >= 0; i-- {
		handler = XSSMiddlewares()[i](handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "default-src 'self';", rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}
