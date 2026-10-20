package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestClientRejectsOversizedResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"payload":"` + strings.Repeat("x", 1<<20) + `"}`))
	}))
	defer ts.Close()
	var out any
	err := NewBotClient(ts.URL, "secret").doRequest(context.Background(), http.MethodGet, "/", nil, &out)
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("got %v", err)
	}
}

func TestClientBoundsAndEscapesRemoteError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("\x1b[31m" + strings.Repeat("x", 1<<20)))
	}))
	defer ts.Close()
	err := NewBotClient(ts.URL, "secret").doRequest(context.Background(), http.MethodGet, "/", nil, nil)
	if err == nil || len(err.Error()) > 1200 || strings.ContainsRune(err.Error(), '\x1b') {
		t.Fatalf("unsafe error response (length %d)", len(fmt.Sprint(err)))
	}
}

func TestSideBetOnlyOfferedInFirstRound(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		_, bet, _ := chooseStance(&httpapi.TurnResponse{Round: 2, Scores: httpapi.ScoresResponse{You: 2, Opponent: 1}}, rng)
		if bet != 0 {
			t.Fatal("side bet after round one")
		}
	}
}
