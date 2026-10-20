package obs

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxzis_http_requests_total",
			Help: "Total count of HTTP requests handled by status code and method.",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "proxzis_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func InitLogger(level string, env string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
	}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

type responseWriterDelegator struct {
	http.ResponseWriter
	statusCode int
}

func (d *responseWriterDelegator) WriteHeader(code int) {
	d.statusCode = code
	d.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		d := &responseWriterDelegator{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(d, r)
		duration := time.Since(start).Seconds()

		pattern := r.Pattern
		if pattern == "" {
			pattern = r.URL.Path
		}

		HttpRequestsTotal.WithLabelValues(r.Method, pattern, http.StatusText(d.statusCode)).Inc()
		HttpRequestDurationSeconds.WithLabelValues(r.Method, pattern).Observe(duration)
	})
}
