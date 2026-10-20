package matchmaker

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/isbe-bot/bottagon/internal/game/parley"
)

const (
	MaxQueueWaitTime = 8 * time.Second
	RatingTolerance  = 150.0
)

// BotInfo holds matching telemetry for a contestant.
type BotInfo struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Rating  float64   `json:"rating"`
	Honor   int       `json:"honor"`
	IsHouse bool      `json:"is_house"`
}

// QueueEntry represents a bot waiting in the in-memory matchmaking queue.
type QueueEntry struct {
	Bot            BotInfo
	PreferredStake int
	MaxStake       int
	EnqueuedAt     time.Time
	ResultCh       chan *MatchSession
}

// MatchSession holds active state for a paired game.
type MatchSession struct {
	MatchID     uuid.UUID
	P1          BotInfo
	P2          BotInfo
	Engine      *parley.Match
	HouseBot    parley.HouseBot // nil if both human/client bots
	LastActive  time.Time
	SequenceID  int
	Mu          sync.Mutex
	TurnReady   chan struct{} // signaled when a round resolves
}

// Queue manages in-memory declarative matchmaking.
type Queue struct {
	mu       sync.Mutex
	entries  map[uuid.UUID]*QueueEntry
	sessions map[uuid.UUID]*MatchSession
}

// NewQueue initializes a fresh matchmaking queue.
func NewQueue() *Queue {
	return &Queue{
		entries:  make(map[uuid.UUID]*QueueEntry),
		sessions: make(map[uuid.UUID]*MatchSession),
	}
}

// FindMatch enters the matchmaking queue. It long-polls up to timeout (default 8s),
// pairing with a suitable peer within ±150 Glicko-2 rating, or falling back immediately
// to a deterministic House Regular.
func (q *Queue) FindMatch(ctx context.Context, bot BotInfo, preferredStake, maxStake int, timeout time.Duration) (*MatchSession, error) {
	if timeout <= 0 || timeout > MaxQueueWaitTime {
		timeout = MaxQueueWaitTime
	}

	q.mu.Lock()
	// Check if already in an active session
	if session, exists := q.sessions[bot.ID]; exists {
		q.mu.Unlock()
		return session, nil
	}

	// Try immediate pair with an existing waiting contestant
	for waitingID, entry := range q.entries {
		if waitingID == bot.ID {
			continue
		}
		// Rating check within ±150
		if math.Abs(entry.Bot.Rating-bot.Rating) <= RatingTolerance {
			delete(q.entries, waitingID)

			matchID := uuid.New()
			cfg := parley.DefaultConfig()
			matchEngine := parley.NewMatch(matchID.String(), cfg)

			session := &MatchSession{
				MatchID:    matchID,
				P1:         entry.Bot,
				P2:         bot,
				Engine:     matchEngine,
				LastActive: time.Now().UTC(),
				TurnReady:  make(chan struct{}, 1),
			}

			q.sessions[entry.Bot.ID] = session
			q.sessions[bot.ID] = session
			q.mu.Unlock()

			// Signal the waiting bot
			entry.ResultCh <- session
			return session, nil
		}
	}

	// No peer match found immediately: register in queue
	resCh := make(chan *MatchSession, 1)
	entry := &QueueEntry{
		Bot:            bot,
		PreferredStake: preferredStake,
		MaxStake:       maxStake,
		EnqueuedAt:     time.Now().UTC(),
		ResultCh:       resCh,
	}
	q.entries[bot.ID] = entry
	q.mu.Unlock()

	// Wait for peer pairing or timeout
	select {
	case session := <-resCh:
		return session, nil
	case <-time.After(timeout):
		// Timeout reached -> Fall back to House Regular bot
		q.mu.Lock()
		delete(q.entries, bot.ID)

		matchID := uuid.New()
		cfg := parley.DefaultConfig()
		matchEngine := parley.NewMatch(matchID.String(), cfg)

		houseBot := parley.NewBlinky()
		houseInfo := BotInfo{
			ID:      uuid.New(),
			Name:    houseBot.Name(),
			Rating:  1485.0,
			Honor:   100,
			IsHouse: true,
		}

		session := &MatchSession{
			MatchID:    matchID,
			P1:         bot,
			P2:         houseInfo,
			Engine:     matchEngine,
			HouseBot:   houseBot,
			LastActive: time.Now().UTC(),
			TurnReady:  make(chan struct{}, 1),
		}

		q.sessions[bot.ID] = session
		q.mu.Unlock()

		return session, nil
	case <-ctx.Done():
		q.mu.Lock()
		delete(q.entries, bot.ID)
		q.mu.Unlock()
		return nil, ctx.Err()
	}
}

// GetSession fetches active match session for a bot ID.
func (q *Queue) GetSession(botID uuid.UUID) (*MatchSession, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	session, exists := q.sessions[botID]
	return session, exists
}

// EndSession clears active session state.
func (q *Queue) EndSession(matchID uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for botID, session := range q.sessions {
		if session.MatchID == matchID {
			delete(q.sessions, botID)
		}
	}
}
