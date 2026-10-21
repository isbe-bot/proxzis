package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/isbe-bot/bottagon/internal/catalog"
	"github.com/isbe-bot/bottagon/internal/config"
	"github.com/isbe-bot/bottagon/internal/house"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, closer, err := catalog.OpenDatabase(ctx, cfg.DatabaseURL, cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer closer.Close()
	if err := store.Seed(ctx); err != nil {
		log.Fatal(err)
	}
	if catalog.PostgresURL(cfg.DatabaseURL) {
		if err := house.SeedLedger(ctx, closer.(*sql.DB)); err != nil {
			log.Fatal(err)
		}
		log.Print("spectator archive and house ledger seeded in postgres")
		return
	}
	log.Printf("spectator archive seeded in %s", cfg.DatabasePath)
}
