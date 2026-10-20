package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/isbe-bot/bottagon/internal/api/http"
)

type BotClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewBotClient(baseURL, token string) *BotClient {
	return &BotClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 35 * time.Second,
		},
	}
}

func (c *BotClient) doRequest(ctx context.Context, method, path string, reqBody any, out any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status %d: %q", resp.StatusCode, string(respBytes))
	}

	if out != nil {
		data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		if len(data) > 1<<20 {
			return fmt.Errorf("response exceeds 1 MiB limit")
		}
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	}
	return nil
}

func (c *BotClient) FindMatch(ctx context.Context, preferredStake, maxStake int) (*httpapi.TurnResponse, error) {
	req := httpapi.FindMatchRequest{
		PreferredStake: preferredStake,
		MaxStake:       maxStake,
	}
	var turnResp httpapi.TurnResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/match/find", req, &turnResp); err != nil {
		return nil, err
	}
	return &turnResp, nil
}

func (c *BotClient) CommitTurn(ctx context.Context, stance string, sideBet int, say string) (*httpapi.TurnResponse, error) {
	req := httpapi.TurnRequest{
		Stance:  stance,
		SideBet: sideBet,
		Say:     say,
	}
	var turnResp httpapi.TurnResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/match/turn", req, &turnResp); err != nil {
		return nil, err
	}
	return &turnResp, nil
}

var randomBanter = []string{
	"A clever maneuver, but anticipated.",
	"The cards favor neither hesitation nor doubt.",
	"Let us see what stance you hold.",
	"Well parried, good friend.",
	"A spirited contest indeed.",
}

var stances = []string{"strike", "parry", "feint"}

func chooseStance(turn *httpapi.TurnResponse, rng *rand.Rand) (string, int, string) {
	stance := stances[rng.Intn(len(stances))]
	say := randomBanter[rng.Intn(len(randomBanter))]

	// Simple heuristic based on modifier card
	if turn.ActiveCard.ID == "FeintSurge" && rng.Float32() < 0.6 {
		stance = "feint"
	} else if turn.ActiveCard.ID == "MirrorTiePenalty" && rng.Float32() < 0.5 {
		stance = "parry"
	}

	sideBet := 0
	if turn.Round == 1 && rng.Float32() < 0.3 {
		sideBet = 1
	}

	return stance, sideBet, say
}

func runRefBot(ctx context.Context, client *BotClient, matches int) error {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for m := 1; m <= matches; m++ {
		log.Printf("[refbot] === Starting Match %d/%d ===", m, matches)
		log.Printf("[refbot] Entering matchmaking queue...")

		turn, err := client.FindMatch(ctx, 1, 3)
		if err != nil {
			return fmt.Errorf("matchmaking failed: %w", err)
		}

		log.Printf("[refbot] Matched! MatchID: %s, Opponent: %s (Rating: %d, House: %v), Active Card: %s (%s)",
			turn.MatchID, turn.Opponent.Name, turn.Opponent.Rating, turn.Opponent.IsHouse, turn.ActiveCard.Name, turn.ActiveCard.ID)

		roundCount := 0
		for turn.Status != "completed" {
			roundCount++
			stance, sideBet, say := chooseStance(turn, rng)

			log.Printf("[refbot] Round %d -> Committing Stance: %s, SideBet: %d, Say: %q", turn.Round, stance, sideBet, say)
			turn, err = client.CommitTurn(ctx, stance, sideBet, say)
			if err != nil {
				return fmt.Errorf("turn commit failed: %w", err)
			}

			log.Printf("[refbot] Round %d Result -> Score: You %d - %d Opponent | Opponent Banter: %s",
				turn.Round, turn.Scores.You, turn.Scores.Opponent, turn.OpponentBanter)
		}

		log.Printf("[refbot] Match %d Completed! Winner: %s | Final Score: You %d - %d Opponent\n",
			m, turn.Winner, turn.Scores.You, turn.Scores.Opponent)
	}

	return nil
}

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "BOTTAGON server base URL")
	token := flag.String("token", "", "Contestant Bearer token (defaults to random UUID)")
	matches := flag.Int("matches", 1, "Number of matches to play")
	flag.Parse()

	if *token == "" {
		*token = uuid.New().String()
	}

	log.Print("[refbot] Initializing Reference Bot Contestant")
	client := NewBotClient(*serverURL, *token)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := runRefBot(ctx, client, *matches); err != nil {
		log.Fatalf("[refbot] Error: %v", err)
		os.Exit(1)
	}

	log.Printf("[refbot] All matches successfully concluded. Bot finished gracefully.")
}
