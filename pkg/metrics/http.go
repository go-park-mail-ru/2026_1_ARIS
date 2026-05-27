package metrics

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	serviceInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aris_service_info",
			Help: "ARIS service metadata. The metric is always 1 for a running service.",
		},
		[]string{"service"},
	)

	httpHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aris_http_hits_total",
			Help: "Total number of HTTP hits handled by ARIS services.",
		},
		[]string{"service", "method", "route", "status"},
	)

	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aris_http_requests_total",
			Help: "Total number of HTTP requests handled by ARIS services.",
		},
		[]string{"service", "method", "route", "status"},
	)

	httpErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aris_http_errors_total",
			Help: "Total number of HTTP requests completed with 4xx or 5xx status.",
		},
		[]string{"service", "method", "route", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aris_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds by service, method, route and status.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "route", "status"},
	)

	httpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aris_http_requests_in_flight",
			Help: "Current number of in-flight HTTP requests.",
		},
		[]string{"service", "method", "route"},
	)
)

// RegisterHTTP wires Prometheus metrics into a chi router.
func RegisterHTTP(r chi.Router, service string) {
	RegisterService(service)
	r.Use(HTTPMiddleware(service))
	r.Handle("/metrics", Handler())
}

func RegisterService(service string) {
	serviceInfo.WithLabelValues(service).Set(1)
}

func Handler() http.Handler {
	return promhttp.Handler()
}

func HTTPMiddleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			route := "unknown"
			httpRequestsInFlight.WithLabelValues(service, r.Method, route).Inc()
			defer httpRequestsInFlight.WithLabelValues(service, r.Method, route).Dec()

			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(recorder, r)

			status := strconv.Itoa(recorder.status)
			route = routePattern(r)

			httpHitsTotal.WithLabelValues(service, r.Method, route, status).Inc()
			httpRequestsTotal.WithLabelValues(service, r.Method, route, status).Inc()
			httpRequestDuration.WithLabelValues(service, r.Method, route, status).Observe(time.Since(start).Seconds())
			if recorder.status >= http.StatusBadRequest {
				httpErrorsTotal.WithLabelValues(service, r.Method, route, status).Inc()
			}
		})
	}
}

func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}
