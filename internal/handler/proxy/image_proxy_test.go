package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageProxy_MissingURL(t *testing.T) {
	req := httptest.NewRequest("GET", "/image-proxy", nil)
	w := httptest.NewRecorder()

	ImageProxy(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing url")
}
