package stress

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	httpapi "github.com/isbe-bot/bottagon/internal/api/http"
	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

type BotClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func (c *BotClient) post(ctx context.Context, path string, reqBody any, out any) error {
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// TestStress_50ConcurrentBots simulates 50 concurrent contestant bots playing matches.
func TestStress_50ConcurrentBots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 50-bot stress test in short mode")
	}

	q := matchmaker.NewQueue()
	server := httpapi.NewServer(q)

	r := chi.NewRouter()
	r.Mount("/v1", server.Routes())

	ts := httptest.NewServer(r)
	defer ts.Close()

	const numBots = 50
	var wg sync.WaitGroup
	var completedMatches int64
	var protocolErrors int64

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stances := []string{"strike", "parry", "feint"}

	for i := 0; i < numBots; i++ {
		wg.Add(1)
		botIdx := i
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(botIdx*1000)))
			botToken := uuid.New().String()
			client := &BotClient{
				BaseURL: ts.URL,
				Token:   botToken,
				HTTPClient: &http.Client{
					Timeout: 35 * time.Second,
				},
			}

			// 1. Enter matchmaking queue
			var turnResp httpapi.TurnResponse
			findReq := httpapi.FindMatchRequest{PreferredStake: 1, MaxStake: 3}
			if err := client.post(ctx, "/v1/match/find", findReq, &turnResp); err != nil {
				atomic.AddInt64(&protocolErrors, 1)
				return
			}

			// 2. Play match to completion
			roundCount := 0
			for turnResp.Status != "completed" && roundCount < 20 {
				roundCount++
				stance := stances[rng.Intn(len(stances))]
				say := fmt.Sprintf("Bot-%02d round %d move", botIdx, roundCount)

				turnReq := httpapi.TurnRequest{
					Stance:  stance,
					SideBet: 0,
					Say:     say,
				}
				if err := client.post(ctx, "/v1/match/turn", turnReq, &turnResp); err != nil {
					atomic.AddInt64(&protocolErrors, 1)
					return
				}
			}

			if turnResp.Status == "completed" {
				atomic.AddInt64(&completedMatches, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("Stress test complete: %d/%d matches cleanly completed, %d protocol errors",
		completedMatches, numBots, protocolErrors)

	if protocolErrors > 0 {
		t.Fatalf("encountered %d protocol errors during 50-bot stress test", protocolErrors)
	}
	if completedMatches != int64(numBots) {
		t.Fatalf("expected all %d bots to complete matches, got %d", numBots, completedMatches)
	}
}
