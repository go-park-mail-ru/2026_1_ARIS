package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

var durationBuckets = []float64{
	0.005,
	0.01,
	0.025,
	0.05,
	0.1,
	0.25,
	0.5,
	1,
	2.5,
	5,
	10,
}

type requestLabels struct {
	Service string
	Method  string
	Route   string
	Status  string
}

type durationValue struct {
	Buckets []uint64
	Count   uint64
	Sum     float64
}

var registry = struct {
	sync.Mutex
	requests  map[requestLabels]uint64
	errors    map[requestLabels]uint64
	durations map[requestLabels]*durationValue
}{
	requests:  make(map[requestLabels]uint64),
	errors:    make(map[requestLabels]uint64),
	durations: make(map[requestLabels]*durationValue),
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (w *statusResponseWriter) Flush() {
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func Middleware(service string) func(http.Handler) http.Handler {
	service = strings.TrimSpace(service)
	if service == "" {
		service = "unknown"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := &statusResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(ww, r)

			labels := requestLabels{
				Service: service,
				Method:  r.Method,
				Route:   routePattern(r),
				Status:  fmt.Sprintf("%d", ww.status),
			}
			observe(labels, time.Since(start).Seconds())
		})
	}
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		writeMetrics(w)
	})
}

func observe(labels requestLabels, seconds float64) {
	registry.Lock()
	defer registry.Unlock()

	registry.requests[labels]++
	if statusCodeIsError(labels.Status) {
		registry.errors[labels]++
	}

	value := registry.durations[labels]
	if value == nil {
		value = &durationValue{Buckets: make([]uint64, len(durationBuckets))}
		registry.durations[labels] = value
	}
	for i, bucket := range durationBuckets {
		if seconds <= bucket {
			value.Buckets[i]++
		}
	}
	value.Count++
	value.Sum += seconds
}

func writeMetrics(w http.ResponseWriter) {
	registry.Lock()
	requests := copyCounters(registry.requests)
	errors := copyCounters(registry.errors)
	durations := copyDurations(registry.durations)
	registry.Unlock()

	fmt.Fprintln(w, "# HELP aris_http_requests_total Total HTTP requests handled by service, method, route and status.")
	fmt.Fprintln(w, "# TYPE aris_http_requests_total counter")
	for _, labels := range sortedLabels(requests) {
		fmt.Fprintf(w, "aris_http_requests_total{%s} %d\n", labels.String(), requests[labels])
	}

	fmt.Fprintln(w, "# HELP aris_http_errors_total Total HTTP requests with 4xx or 5xx status codes.")
	fmt.Fprintln(w, "# TYPE aris_http_errors_total counter")
	for _, labels := range sortedLabels(errors) {
		fmt.Fprintf(w, "aris_http_errors_total{%s} %d\n", labels.String(), errors[labels])
	}

	fmt.Fprintln(w, "# HELP aris_http_request_duration_seconds HTTP request duration histogram.")
	fmt.Fprintln(w, "# TYPE aris_http_request_duration_seconds histogram")
	for _, labels := range sortedDurationLabels(durations) {
		value := durations[labels]
		for i, bucket := range durationBuckets {
			fmt.Fprintf(w, "aris_http_request_duration_seconds_bucket{%s,le=%q} %d\n", labels.String(), formatBucket(bucket), value.Buckets[i])
		}
		fmt.Fprintf(w, "aris_http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels.String(), value.Count)
		fmt.Fprintf(w, "aris_http_request_duration_seconds_sum{%s} %s\n", labels.String(), formatFloat(value.Sum))
		fmt.Fprintf(w, "aris_http_request_duration_seconds_count{%s} %d\n", labels.String(), value.Count)
	}
}

func routePattern(r *http.Request) string {
	if ctx := chi.RouteContext(r.Context()); ctx != nil {
		if pattern := ctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	if r.URL.Path != "" {
		return r.URL.Path
	}
	return "unknown"
}

func statusCodeIsError(status string) bool {
	return len(status) == 3 && (status[0] == '4' || status[0] == '5')
}

func copyCounters(src map[requestLabels]uint64) map[requestLabels]uint64 {
	dst := make(map[requestLabels]uint64, len(src))
	for labels, value := range src {
		dst[labels] = value
	}
	return dst
}

func copyDurations(src map[requestLabels]*durationValue) map[requestLabels]durationValue {
	dst := make(map[requestLabels]durationValue, len(src))
	for labels, value := range src {
		buckets := make([]uint64, len(value.Buckets))
		copy(buckets, value.Buckets)
		dst[labels] = durationValue{
			Buckets: buckets,
			Count:   value.Count,
			Sum:     value.Sum,
		}
	}
	return dst
}

func sortedLabels(values map[requestLabels]uint64) []requestLabels {
	labels := make([]requestLabels, 0, len(values))
	for label := range values {
		labels = append(labels, label)
	}
	sortLabels(labels)
	return labels
}

func sortedDurationLabels(values map[requestLabels]durationValue) []requestLabels {
	labels := make([]requestLabels, 0, len(values))
	for label := range values {
		labels = append(labels, label)
	}
	sortLabels(labels)
	return labels
}

func sortLabels(labels []requestLabels) {
	sort.Slice(labels, func(i, j int) bool {
		if labels[i].Service != labels[j].Service {
			return labels[i].Service < labels[j].Service
		}
		if labels[i].Route != labels[j].Route {
			return labels[i].Route < labels[j].Route
		}
		if labels[i].Method != labels[j].Method {
			return labels[i].Method < labels[j].Method
		}
		return labels[i].Status < labels[j].Status
	})
}

func (l requestLabels) String() string {
	return fmt.Sprintf(
		"service=\"%s\",method=\"%s\",route=\"%s\",status=\"%s\"",
		escapeLabelValue(l.Service),
		escapeLabelValue(l.Method),
		escapeLabelValue(l.Route),
		escapeLabelValue(l.Status),
	)
}

func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return value
}

func formatBucket(value float64) string {
	return formatFloat(value)
}

func formatFloat(value float64) string {
	if math.IsInf(value, 1) {
		return "+Inf"
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
}
