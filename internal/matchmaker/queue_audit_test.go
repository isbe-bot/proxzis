package matchmaker

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/isbe-bot/bottagon/internal/game/parley"
)

func waitQueued(t *testing.T, q *Queue, id uuid.UUID) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		q.mu.Lock()
		_, ok := q.entries[id]
		q.mu.Unlock()
		if ok {
			return
		}
		runtime.Gosched()
	}
	t.Fatal("bot never entered queue")
}

func TestDuplicateFindDoesNotOverwriteWaiter(t *testing.T) {
	q := NewQueue()
	bot := BotInfo{ID: uuid.New(), Rating: 1500}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := q.FindMatch(ctx, bot, 1, 5, time.Second); done <- err }()
	waitQueued(t, q, bot.ID)
	dupCtx, dupCancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer dupCancel()
	if _, err := q.FindMatch(dupCtx, bot, 1, 5, time.Second); !errors.Is(err, ErrAlreadyQueued) {
		t.Fatalf("duplicate error: %v", err)
	}
	q.mu.Lock()
	_, exists := q.entries[bot.ID]
	q.mu.Unlock()
	if !exists {
		t.Fatal("duplicate removed original waiter")
	}
	cancel()
	<-done
}

func TestCanceledFindCannotCreateSession(t *testing.T) {
	q := NewQueue()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bot := BotInfo{ID: uuid.New(), Rating: 1500}
	if _, err := q.FindMatch(ctx, bot, 1, 5, time.Nanosecond); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if _, ok := q.GetSession(bot.ID); ok {
		t.Fatal("canceled request created session")
	}
}

func TestPairWinsAgainstTimeout(t *testing.T) {
	for i := 0; i < 100; i++ {
		q := NewQueue()
		bot := BotInfo{ID: uuid.New(), Rating: 1500}
		result := make(chan *MatchSession, 1)
		go func() { s, _ := q.FindMatch(context.Background(), bot, 1, 5, time.Millisecond); result <- s }()
		waitQueued(t, q, bot.ID)
		peer := BotInfo{ID: uuid.New(), Rating: 1500}
		paired, err := q.FindMatch(context.Background(), peer, 1, 5, time.Nanosecond)
		if err != nil {
			t.Fatal(err)
		}
		first := <-result
		if !paired.P2.IsHouse && paired.MatchID != first.MatchID {
			t.Fatal("timeout replaced committed peer pairing")
		}
	}
}

func BenchmarkExistingSession(b *testing.B) {
	q := NewQueue()
	bot := BotInfo{ID: uuid.New(), Rating: 1500}
	if _, err := q.FindMatch(context.Background(), bot, 1, 5, time.Nanosecond); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := q.FindMatch(context.Background(), bot, 1, 5, time.Second); err != nil {
			b.Fatal(err)
		}
	}
}

func TestCompletedBotCanFindNewMatch(t *testing.T) {
	q := NewQueue()
	bot := BotInfo{ID: uuid.New(), Rating: 1500}
	old, err := q.FindMatch(context.Background(), bot, 1, 5, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	old.Mu.Lock()
	old.Engine.Status = parley.StatusCompleted
	old.Mu.Unlock()
	fresh, err := q.FindMatch(context.Background(), bot, 1, 5, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.MatchID == old.MatchID {
		t.Fatal("completed match reused forever")
	}
	q.EndSession(old.MatchID)
	if got, ok := q.GetSession(bot.ID); !ok || got != fresh {
		t.Fatal("old cleanup removed newer session")
	}
}

func TestIdleSessionExpiresOnSweep(t *testing.T) {
	q := NewQueue()
	bot := BotInfo{ID: uuid.New(), Rating: 1500}
	old, err := q.FindMatch(context.Background(), bot, 1, 5, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	old.Mu.Lock()
	old.LastActive = time.Now().Add(-2 * SessionIdleTTL)
	old.Mu.Unlock()
	q.mu.Lock()
	q.lastSweep = time.Time{}
	q.mu.Unlock()
	fresh, err := q.FindMatch(context.Background(), bot, 1, 5, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	if old == fresh {
		t.Fatal("expired session retained")
	}
}

func BenchmarkEndSession(b *testing.B) {
	for _, n := range []int{1, 1000, 8000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.StopTimer()
			q := NewQueue()
			for i := 0; i < n; i++ {
				q.sessions[uuid.New()] = &MatchSession{MatchID: uuid.New()}
			}
			bot := BotInfo{ID: uuid.New(), Rating: 1500}
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				session, err := q.FindMatch(context.Background(), bot, 1, 5, time.Nanosecond)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				q.EndSession(session.MatchID)
			}
		})
	}
}

func TestQueueEngineUsesBoundedDemoPurse(t *testing.T) {
	q := NewQueue()
	bot := BotInfo{ID: uuid.New(), Rating: 1500}
	session, err := q.FindMatch(context.Background(), bot, 1, 5, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	session.Mu.Lock()
	defer session.Mu.Unlock()
	if err := session.Engine.CommitTurn(parley.Player1, parley.StanceStrike, "", DemoPurse+1); !errors.Is(err, parley.ErrExcessiveWager) {
		t.Fatalf("unfunded wager: %v", err)
	}
	if err := session.Engine.CommitTurn(parley.Player1, parley.StanceStrike, "", 1); err != nil {
		t.Fatalf("legal wager: %v", err)
	}
}

// Hold the queue lock until the timer fires, then publish a peer pairing before
// allowing the timeout handler to proceed. This deterministically exercises the
// old branch that unconditionally replaced an already committed peer session.
func TestReadyTimeoutCannotReplacePublishedPair(t *testing.T) {
	q := NewQueue()
	bot := BotInfo{ID: uuid.New(), Rating: 1500}
	done := make(chan *MatchSession, 1)
	go func() {
		session, _ := q.FindMatch(context.Background(), bot, 1, 5, 10*time.Millisecond)
		done <- session
	}()
	waitQueued(t, q, bot.ID)
	q.mu.Lock()
	entry := q.entries[bot.ID]
	time.Sleep(50 * time.Millisecond)
	paired := &MatchSession{MatchID: uuid.New(), P1: bot, P2: BotInfo{ID: uuid.New()}, Engine: parley.NewMatch("paired", parley.DefaultConfig()), LastActive: time.Now()}
	delete(q.entries, bot.ID)
	q.sessions[bot.ID] = paired
	q.sessions[paired.P2.ID] = paired
	q.matches[paired.MatchID] = paired
	entry.ResultCh <- paired
	q.mu.Unlock()
	if got := <-done; got != paired {
		t.Fatal("timeout replaced published peer session")
	}
}
