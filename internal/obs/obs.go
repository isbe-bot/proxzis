package obs

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bottagon_http_requests_total",
			Help: "Total count of HTTP requests handled by status code and method.",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bottagon_http_request_duration_seconds",
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

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		d := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(d, r)
		duration := time.Since(start).Seconds()

		// Use the router template, never attacker-controlled URLs or methods.
		// Raw paths turn misses into unbounded Prometheus series and leak secrets.
		pattern := "unmatched"
		if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil && routeCtx.RoutePattern() != "" {
			pattern = routeCtx.RoutePattern()
		} else if r.Pattern != "" {
			pattern = r.Pattern
		}
		method := r.Method
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodConnect, http.MethodTrace:
		default:
			method = "OTHER"
		}
		status := d.Status()
		if status == 0 {
			status = http.StatusOK
		}

		HttpRequestsTotal.WithLabelValues(method, pattern, http.StatusText(status)).Inc()
		HttpRequestDurationSeconds.WithLabelValues(method, pattern).Observe(duration)
	})
}
