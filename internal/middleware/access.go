package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"go.uber.org/zap"
)

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

func AccessLogMiddleware(base *zap.Logger) func(http.Handler) http.Handler {
	if base == nil {
		base = zap.NewNop()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := &statusResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(ww, r)

			log := logger.FromContext(r.Context())
			if log == nil {
				log = base
			}

			log.Info("access",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.status),
				zap.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		})
	}
}
