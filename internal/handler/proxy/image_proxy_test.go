package proxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---- mock transport ----

type mockRoundTripper struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

func TestImageProxy_MissingURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	w := httptest.NewRecorder()

	ImageProxy(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "missing url") {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestImageProxy_HTTPGetError(t *testing.T) {
	// подменяем транспорт
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	http.DefaultTransport = &mockRoundTripper{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/image?url=http://example.com/img.png", nil)
	w := httptest.NewRecorder()

	ImageProxy(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "failed to fetch image") {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestImageProxy_Success(t *testing.T) {
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	http.DefaultTransport = &mockRoundTripper{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"image/png"},
				},
				Body: io.NopCloser(strings.NewReader("image-data")),
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/image?url=http://example.com/img.png", nil)
	w := httptest.NewRecorder()

	ImageProxy(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	if ct := res.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("expected content-type image/png, got %s", ct)
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) != "image-data" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}
