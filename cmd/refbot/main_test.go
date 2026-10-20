package main

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	httpapi "github.com/isbe-bot/bottagon/internal/api/http"
	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

func TestRefBot_SimulatedMatchAgainstHouse(t *testing.T) {
	q := matchmaker.NewQueue()
	apiServer := httpapi.NewServer(q)

	r := chi.NewRouter()
	r.Mount("/v1", apiServer.Routes())

	ts := httptest.NewServer(r)
	defer ts.Close()

	client := NewBotClient(ts.URL, uuid.New().String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := runRefBot(ctx, client, 1)
	if err != nil {
		t.Fatalf("runRefBot failed: %v", err)
	}
}
