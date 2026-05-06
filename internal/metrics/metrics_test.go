package metrics

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareObservesRequestsAndHandlerRendersMetrics(t *testing.T) {
	registry.Lock()
	registry.requests = make(map[requestLabels]uint64)
	registry.errors = make(map[requestLabels]uint64)
	registry.durations = make(map[requestLabels]*durationValue)
	registry.Unlock()

	r := chi.NewRouter()
	r.Use(Middleware(" api "))
	r.Get("/users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	r.Get("/fail", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	})
	r.Get("/metrics", Handler().ServeHTTP)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users/10", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/fail", nil))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := rec.Body.String()
	require.Contains(t, body, `aris_http_requests_total{service="api",method="GET",route="/users/{id}",status="201"} 1`)
	require.Contains(t, body, `aris_http_errors_total{service="api",method="GET",route="/fail",status="400"} 1`)
	require.Contains(t, body, "aris_http_request_duration_seconds_bucket")
	require.Equal(t, "text/plain; version=0.0.4; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestMetricsHelpers(t *testing.T) {
	labels := requestLabels{Service: "svc\n\"x\"", Method: "GET", Route: "/a\\b", Status: "500"}
	require.Contains(t, labels.String(), `service="svc\n\"x\""`)
	require.Contains(t, labels.String(), `route="/a\\b"`)
	require.True(t, statusCodeIsError("404"))
	require.True(t, statusCodeIsError("500"))
	require.False(t, statusCodeIsError("302"))
	require.Equal(t, "+Inf", formatFloat(math.Inf(1)))
	require.Equal(t, "1.25", formatBucket(1.25))

	observe(labels, 0.006)
	counters := copyCounters(map[requestLabels]uint64{labels: 2})
	require.Equal(t, uint64(2), counters[labels])
	durations := copyDurations(registry.durations)
	require.NotEmpty(t, durations)
	require.NotEmpty(t, sortedLabels(counters))
	require.NotEmpty(t, sortedDurationLabels(durations))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.URL.Path = ""
	require.Equal(t, "unknown", routePattern(req))
	req = httptest.NewRequest(http.MethodGet, "/plain", nil)
	require.Equal(t, "/plain", routePattern(req))

	rec := httptest.NewRecorder()
	w := &statusResponseWriter{ResponseWriter: rec}
	w.WriteHeader(http.StatusAccepted)
	w.Flush()
	_, _, err := w.Hijack()
	require.Error(t, err)

	metricsRec := httptest.NewRecorder()
	writeMetrics(metricsRec)
	require.Contains(t, metricsRec.Body.String(), "# HELP aris_http_requests_total")
}
