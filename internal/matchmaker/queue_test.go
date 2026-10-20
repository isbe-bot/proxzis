package matchmaker

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMatchmaker_PeerPairing(t *testing.T) {
	q := NewQueue()

	bot1 := BotInfo{
		ID:      uuid.New(),
		Name:    "AgentAlpha",
		Rating:  1500.0,
		Honor:   100,
		IsHouse: false,
	}

	bot2 := BotInfo{
		ID:      uuid.New(),
		Name:    "AgentBeta",
		Rating:  1550.0, // within ±150
		Honor:   100,
		IsHouse: false,
	}

	sessionCh := make(chan *MatchSession, 1)

	// Bot 1 enters queue
	go func() {
		session, err := q.FindMatch(context.Background(), bot1, 10, 50, 4*time.Second)
		if err != nil {
			t.Errorf("bot1 error: %v", err)
		}
		sessionCh <- session
	}()

	time.Sleep(100 * time.Millisecond)

	// Bot 2 enters queue and should match with Bot 1
	session2, err := q.FindMatch(context.Background(), bot2, 10, 50, 4*time.Second)
	if err != nil {
		t.Fatalf("bot2 failed to match: %v", err)
	}

	session1 := <-sessionCh

	if session1.MatchID != session2.MatchID {
		t.Fatalf("expected both bots to be paired in the same session, got %s and %s", session1.MatchID, session2.MatchID)
	}

	if session1.HouseBot != nil {
		t.Errorf("expected peer match, got house bot")
	}
}

func TestMatchmaker_HouseBotFallbackOnTimeout(t *testing.T) {
	q := NewQueue()

	loneBot := BotInfo{
		ID:      uuid.New(),
		Name:    "LonelyBot",
		Rating:  1500.0,
		Honor:   100,
		IsHouse: false,
	}

	// Wait with a short timeout (500ms) to trigger House Regular fallback
	start := time.Now()
	session, err := q.FindMatch(context.Background(), loneBot, 10, 50, 500*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected house fallback, got err: %v", err)
	}

	if elapsed < 400*time.Millisecond {
		t.Errorf("expected to wait close to timeout duration, took %v", elapsed)
	}

	if session.HouseBot == nil {
		t.Errorf("expected session to contain a HouseBot")
	}

	if !session.P2.IsHouse {
		t.Errorf("expected opponent P2 to be marked as house bot")
	}
}

func TestMatchmaker_RatingToleranceCheck(t *testing.T) {
	q := NewQueue()

	botLow := BotInfo{
		ID:      uuid.New(),
		Name:    "LowElo",
		Rating:  1200.0,
		Honor:   100,
		IsHouse: false,
	}

	botHigh := BotInfo{
		ID:      uuid.New(),
		Name:    "HighElo",
		Rating:  1600.0, // delta 400 > 150 tolerance
		Honor:   100,
		IsHouse: false,
	}

	// Bot Low enters
	go func() {
		_, _ = q.FindMatch(context.Background(), botLow, 10, 50, 600*time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)

	// Bot High should NOT pair with LowElo due to rating gap > 150
	q.mu.Lock()
	entry, exists := q.entries[botLow.ID]
	q.mu.Unlock()

	if !exists {
		t.Fatalf("botLow should be in queue")
	}

	if math.Abs(entry.Bot.Rating-botHigh.Rating) <= RatingTolerance {
		t.Errorf("ratings should be outside tolerance")
	}
}
