package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorsMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handlerToTest := CorsMiddleware(nextHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handlerToTest.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, w.Code)
}
