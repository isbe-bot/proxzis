package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	httpapi "github.com/isbe-bot/bottagon/internal/api/http"
	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

func TestDoctor_RunAllScenariosAgainstLiveRouter(t *testing.T) {
	q := matchmaker.NewQueue()
	server := httpapi.NewServer(q)
	server.SetQueueTimeout(250 * time.Millisecond)

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/llms.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(httpapi.LLMsTxtContent))
	})
	r.Mount("/v1", server.Routes())

	ts := httptest.NewServer(r)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	results, err := runDoctorScenarios(ctx, ts.URL)
	if err != nil {
		t.Fatalf("runDoctorScenarios returned error: %v", err)
	}

	for _, r := range results {
		if !r.Passed {
			t.Errorf("scenario failed: %s - %s", r.Name, r.Details)
		}
	}
}

func TestDoctorClientRejectsOversizedResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"payload":"` + strings.Repeat("x", 1<<20) + `"}`))
	}))
	defer ts.Close()
	var out any
	_, err := NewDoctorClient(ts.URL, "").doRequest(context.Background(), http.MethodGet, "/", nil, &out)
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("got %v", err)
	}
}
