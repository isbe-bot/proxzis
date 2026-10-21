package main

import (
	"context"
	"log"
	"time"

	"github.com/isbe-bot/bottagon/internal/catalog"
	"github.com/isbe-bot/bottagon/internal/config"
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
		log.Print("spectator archive seeded in postgres")
		return
	}
	log.Printf("spectator archive seeded in %s", cfg.DatabasePath)
}
