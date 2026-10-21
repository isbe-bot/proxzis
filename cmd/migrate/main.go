package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/isbe-bot/bottagon/internal/catalog"
	"github.com/isbe-bot/bottagon/internal/config"
	"github.com/isbe-bot/bottagon/internal/obs"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Usage: migrate <up|down|status|version>")
		os.Exit(1)
	}

	command := args[0]
	cfg := config.Load()
	logger := obs.InitLogger("info", "development")

	if !catalog.PostgresURL(cfg.DatabaseURL) {
		if command != "up" && command != "status" {
			logger.Error("the sqlite archive schema is created in place; delete the file to start over", slog.String("path", cfg.DatabasePath), slog.String("command", command))
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, closer, err := catalog.OpenDatabase(ctx, "", cfg.DatabasePath)
		if err != nil {
			logger.Error("failed to prepare sqlite archive", slog.Any("error", err))
			os.Exit(1)
		}
		defer closer.Close()
		logger.Info("sqlite schema ready", slog.String("path", cfg.DatabasePath))
		return
	}
	logger.Info("connecting to database for migrations")

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to open database connection", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Error("failed to ping database", slog.Any("error", err))
		os.Exit(1)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error("failed to set goose dialect", slog.Any("error", err))
		os.Exit(1)
	}

	migrationsDir := "migrations"
	if err := goose.Run(command, db, migrationsDir, args[1:]...); err != nil {
		logger.Error("migration command failed", slog.String("command", command), slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("migration completed successfully", slog.String("command", command))
}
