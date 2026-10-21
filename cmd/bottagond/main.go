package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/isbe-bot/bottagon/internal/api/http"
	"github.com/isbe-bot/bottagon/internal/catalog"
	"github.com/isbe-bot/bottagon/internal/config"
	"github.com/isbe-bot/bottagon/internal/house"
	"github.com/isbe-bot/bottagon/internal/matchmaker"
	"github.com/isbe-bot/bottagon/internal/obs"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

var Version = "0.1.0-dev"

func main() {
	cfg := config.Load()
	logger := obs.InitLogger(cfg.LogLevel, cfg.Env)

	logger.Info("starting bottagond service",
		slog.Int("port", cfg.Port),
		slog.String("env", cfg.Env),
		slog.String("version", Version),
	)

	r, _ := newRouter(0, openCatalog(cfg, logger))

	serverCtx, stopServer := context.WithCancel(context.Background())
	defer stopServer()
	server := newHTTPServer(cfg, r, serverCtx)

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("HTTP server listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdown)

	select {
	case err := <-serverErrors:
		logger.Error("server startup error", slog.Any("error", err))
		os.Exit(1)
	case sig := <-shutdown:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
		stopServer()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", slog.Any("error", err))
			_ = server.Close()
		}
		logger.Info("bottagond terminated cleanly")
	}
}

// newRouter is the daemon surface: health, metrics, discovery, REST, and MCP.
// queueTimeout of zero uses the production matchmaking wait.
func openCatalog(cfg *config.Config, logger *slog.Logger) catalog.Store {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, closer, err := catalog.OpenDatabase(ctx, cfg.DatabaseURL, cfg.DatabasePath)
	if err != nil {
		logger.Error("spectator archive disabled", slog.Any("error", err))
		return nil
	}
	if err := store.SeedIfEmpty(ctx); err != nil {
		logger.Error("spectator seed failed", slog.Any("error", err))
		_ = closer.Close()
		return nil
	}
	if catalog.PostgresURL(cfg.DatabaseURL) {
		if err := house.SeedLedger(ctx, closer.(*sql.DB)); err != nil {
			logger.Error("house ledger seed failed", slog.Any("error", err))
			_ = closer.Close()
			return nil
		}
		logger.Info("spectator archive ready", slog.String("driver", "postgres"))
	} else {
		logger.Info("spectator archive ready", slog.String("driver", "sqlite"), slog.String("path", cfg.DatabasePath))
	}
	return store
}

func newRouter(queueTimeout time.Duration, archive catalog.Store) (http.Handler, *httpapi.Server) {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(obs.MetricsMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:    "ok",
			Timestamp: time.Now().UTC(),
			Version:   Version,
		})
	})

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/llms.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(httpapi.LLMsTxtContent))
	})

	apiServer := httpapi.NewServer(matchmaker.NewQueue())
	if queueTimeout > 0 {
		apiServer.SetQueueTimeout(queueTimeout)
	}
	apiServer.SetCatalog(archive)
	r.Mount("/v1", apiServer.Routes())
	r.Mount("/mcp", httpapi.NewMCPAdapter(apiServer).Handler())
	return r, apiServer
}

func newHTTPServer(cfg *config.Config, handler http.Handler, baseCtx context.Context) *http.Server {
	return &http.Server{
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    16 << 10,
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

}
