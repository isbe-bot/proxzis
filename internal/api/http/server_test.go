package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

func TestHTTPAPI_FullMatchLifecycleAgainstHouse(t *testing.T) {
	q := matchmaker.NewQueue()
	server := NewServer(q)
	server.SetQueueTimeout(100 * time.Millisecond)
	router := server.Routes()

	botID := uuid.New()
	authHeader := "Bearer " + botID.String()

	// 1. Find Match (house fallback uses the short test wait)
	findReqBody := `{"preferred_stake": 10, "max_stake": 50}`
	req, _ := http.NewRequest(http.MethodPost, "/match/find", bytes.NewBufferString(findReqBody))
	req.Header.Set("Authorization", authHeader)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /match/find, got %d: %s", w.Code, w.Body.String())
	}

	var findResp TurnResponse
	if err := json.Unmarshal(w.Body.Bytes(), &findResp); err != nil {
		t.Fatalf("failed to decode find response: %v", err)
	}

	if findResp.MatchID == "" {
		t.Fatalf("expected non-empty match_id")
	}
	if !findResp.Opponent.IsHouse {
		t.Fatalf("expected house bot opponent when falling back")
	}

	// 2. Fetch Current Match (Idempotent check)
	reqCurrent, _ := http.NewRequest(http.MethodGet, "/match/current", nil)
	reqCurrent.Header.Set("Authorization", authHeader)
	wCurrent := httptest.NewRecorder()

	router.ServeHTTP(wCurrent, reqCurrent)
	if wCurrent.Code != http.StatusOK {
		t.Fatalf("expected 200 on /match/current, got %d", wCurrent.Code)
	}

	var currentResp TurnResponse
	_ = json.Unmarshal(wCurrent.Body.Bytes(), &currentResp)
	if currentResp.MatchID != findResp.MatchID {
		t.Fatalf("mismatched match ID on current: got %s, want %s", currentResp.MatchID, findResp.MatchID)
	}

	// 3. Play Rounds until match completes
	for i := 0; i < 7; i++ {
		turnReq := TurnRequest{
			Stance: "strike",
			// Matchmaking does not yet hydrate a real account purse; no credit
			// may be inferred from the demonstration bot/status response.
			SideBet: 0,
			Say:     "Good luck, mate! <script>alert(1)</script>",
		}
		bodyBytes, _ := json.Marshal(turnReq)
		reqTurn, _ := http.NewRequest(http.MethodPost, "/match/turn", bytes.NewBuffer(bodyBytes))
		reqTurn.Header.Set("Authorization", authHeader)
		wTurn := httptest.NewRecorder()

		router.ServeHTTP(wTurn, reqTurn)
		if wTurn.Code != http.StatusOK {
			t.Fatalf("turn %d failed with code %d: %s", i+1, wTurn.Code, wTurn.Body.String())
		}

		var turnResp TurnResponse
		_ = json.Unmarshal(wTurn.Body.Bytes(), &turnResp)

		// Verify banter quarantine
		if turnResp.OpponentBanter != "" {
			if turnResp.OpponentBanter[:17] != "<opponent_banter " {
				t.Errorf("opponent banter not wrapped in quarantine tags: %s", turnResp.OpponentBanter)
			}
		}

		if turnResp.Status == "completed" {
			t.Logf("Match completed after %d rounds! Winner: %s", i+1, turnResp.Winner)
			break
		}
	}

	// 4. Check Bot Status
	reqStatus, _ := http.NewRequest(http.MethodGet, "/bot/status", nil)
	reqStatus.Header.Set("Authorization", authHeader)
	wStatus := httptest.NewRecorder()

	router.ServeHTTP(wStatus, reqStatus)
	if wStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 on /bot/status, got %d", wStatus.Code)
	}

	// 5. Submit Report
	reqReport, _ := http.NewRequest(http.MethodPost, "/report", bytes.NewBufferString(`{"reason":"inappropriate"}`))
	reqReport.Header.Set("Authorization", authHeader)
	wReport := httptest.NewRecorder()

	router.ServeHTTP(wReport, reqReport)
	if wReport.Code != http.StatusOK {
		t.Fatalf("expected 200 on /report, got %d", wReport.Code)
	}
}

func TestHTTPAPI_Forfeit(t *testing.T) {
	q := matchmaker.NewQueue()
	server := NewServer(q)
	server.SetQueueTimeout(100 * time.Millisecond)
	router := server.Routes()

	botID := uuid.New()
	authHeader := "Bearer " + botID.String()

	// Find Match
	findReqBody := `{"preferred_stake": 5, "max_stake": 20}`
	req, _ := http.NewRequest(http.MethodPost, "/match/find", bytes.NewBufferString(findReqBody))
	req.Header.Set("Authorization", authHeader)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Forfeit Match
	reqForfeit, _ := http.NewRequest(http.MethodPost, "/match/forfeit", nil)
	reqForfeit.Header.Set("Authorization", authHeader)
	wForfeit := httptest.NewRecorder()
	router.ServeHTTP(wForfeit, reqForfeit)

	if wForfeit.Code != http.StatusOK {
		t.Fatalf("expected 200 on /match/forfeit, got %d", wForfeit.Code)
	}

	// Active match should now be cleared
	reqCurrent, _ := http.NewRequest(http.MethodGet, "/match/current", nil)
	reqCurrent.Header.Set("Authorization", authHeader)
	wCurrent := httptest.NewRecorder()
	router.ServeHTTP(wCurrent, reqCurrent)

	if wCurrent.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on /match/current after forfeit, got %d", wCurrent.Code)
	}
}
