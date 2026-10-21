package matchmaker

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/isbe-bot/bottagon/internal/game/parley"
	"github.com/isbe-bot/bottagon/internal/house"
)

const (
	MaxQueueWaitTime = 8 * time.Second
	RatingTolerance  = 150.0
	SessionIdleTTL   = 15 * time.Minute
	MaxQueuedBots    = 1024
	MaxActiveMatches = 8192
	// DemoPurse is synthetic: the daemon does not load persistent balances yet.
	DemoPurse = 1000
)

var (
	ErrAlreadyQueued = errors.New("bot already has a pending matchmaking request")
	ErrQueueFull     = errors.New("matchmaking capacity reached")
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
	ctx            context.Context
}

// MatchSession holds active state for a paired game.
type MatchSession struct {
	MatchID    uuid.UUID
	P1         BotInfo
	P2         BotInfo
	Engine     *parley.Match
	HouseBot   parley.HouseBot // nil if both human/client bots
	LastActive time.Time
	SequenceID int
	Mu         sync.Mutex
	TurnReady  chan struct{} // closed/replaced under Mu when a round resolves
}

// Queue manages in-memory declarative matchmaking.
type Queue struct {
	mu        sync.Mutex
	entries   map[uuid.UUID]*QueueEntry
	sessions  map[uuid.UUID]*MatchSession
	matches   map[uuid.UUID]*MatchSession
	lastSweep time.Time
}

// NewQueue initializes a fresh matchmaking queue.
func NewQueue() *Queue {
	return &Queue{
		entries:  make(map[uuid.UUID]*QueueEntry),
		sessions: make(map[uuid.UUID]*MatchSession),
		matches:  make(map[uuid.UUID]*MatchSession),
	}
}

// FindMatch enters the matchmaking queue. It long-polls up to timeout (default 8s),
// pairing with a suitable peer within ±150 Glicko-2 rating, or seating the house
// agent whose rating is closest.
func (q *Queue) FindMatch(ctx context.Context, bot BotInfo, preferredStake, maxStake int, timeout time.Duration) (*MatchSession, error) {
	if timeout <= 0 || timeout > MaxQueueWaitTime {
		timeout = MaxQueueWaitTime
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q.mu.Lock()
	q.sweepLocked(time.Now())
	// Check if already in an active session
	if session, exists := q.sessions[bot.ID]; exists {
		session.Mu.Lock()
		completed := session.Engine.Status == parley.StatusCompleted
		session.Mu.Unlock()
		if !completed {
			q.mu.Unlock()
			return session, nil
		}
		delete(q.sessions, bot.ID)
		if q.sessions[session.P1.ID] != session && q.sessions[session.P2.ID] != session {
			delete(q.matches, session.MatchID)
		}
	}
	if len(q.entries) >= MaxQueuedBots || len(q.matches)+len(q.entries) >= MaxActiveMatches {
		q.mu.Unlock()
		return nil, ErrQueueFull
	}

	if _, exists := q.entries[bot.ID]; exists {
		q.mu.Unlock()
		return nil, ErrAlreadyQueued
	}

	// Try immediate pair with an existing waiting contestant
	for waitingID, entry := range q.entries {
		if entry.ctx.Err() != nil {
			delete(q.entries, waitingID)
			continue
		}
		if waitingID == bot.ID {
			continue
		}
		// Rating check within ±150
		if math.Abs(entry.Bot.Rating-bot.Rating) <= RatingTolerance {
			delete(q.entries, waitingID)

			matchID := uuid.New()
			cfg := parley.DefaultConfig()
			cfg.P1Purse, cfg.P2Purse = DemoPurse, DemoPurse
			matchEngine := parley.NewMatch(matchID.String(), cfg)

			session := &MatchSession{
				MatchID:    matchID,
				P1:         entry.Bot,
				P2:         bot,
				Engine:     matchEngine,
				LastActive: time.Now().UTC(),
				TurnReady:  make(chan struct{}),
			}

			q.matches[matchID] = session
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
		ctx:            ctx,
	}
	q.entries[bot.ID] = entry
	q.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	// Wait for peer pairing or timeout
	select {
	case session := <-resCh:
		return session, nil
	case <-timer.C:
		// Timeout reached -> Fall back to House Regular bot
		q.mu.Lock()
		// Pairing is committed while holding q.mu. Never overwrite it when
		// the timer and the peer notification become ready together.
		if session, exists := q.sessions[bot.ID]; exists {
			q.mu.Unlock()
			return session, nil
		}
		delete(q.entries, bot.ID)
		if err := ctx.Err(); err != nil {
			q.mu.Unlock()
			return nil, err
		}

		matchID := uuid.New()
		cfg := parley.DefaultConfig()
		cfg.P1Purse, cfg.P2Purse = DemoPurse, DemoPurse
		matchEngine := parley.NewMatch(matchID.String(), cfg)

		resident := house.Pick(bot.Rating)
		houseBot := resident.Player()
		houseInfo := BotInfo{
			ID:      resident.ID,
			Name:    resident.Name,
			Rating:  float64(resident.Rating),
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
			TurnReady:  make(chan struct{}),
		}

		q.matches[matchID] = session
		q.sessions[bot.ID] = session
		q.mu.Unlock()

		return session, nil
	case <-ctx.Done():
		q.mu.Lock()
		if q.entries[bot.ID] == entry {
			delete(q.entries, bot.ID)
		}
		// A peer may have already committed the match; preserve that session.
		session, paired := q.sessions[bot.ID]
		q.mu.Unlock()
		if paired {
			return session, nil
		}
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
	session, exists := q.matches[matchID]
	if !exists {
		return
	}
	for _, id := range []uuid.UUID{session.P1.ID, session.P2.ID} {
		if q.sessions[id] == session {
			delete(q.sessions, id)
		}
	}
	delete(q.matches, matchID)
}

// sweepLocked bounds idle state without a permanent janitor goroutine. Callers
// hold q.mu; session locks are always acquired after the queue lock.
func (q *Queue) sweepLocked(now time.Time) {
	if now.Sub(q.lastSweep) < time.Minute {
		return
	}
	q.lastSweep = now
	for id, session := range q.matches {
		session.Mu.Lock()
		expired := now.Sub(session.LastActive) > SessionIdleTTL
		session.Mu.Unlock()
		if !expired {
			continue
		}
		for _, botID := range []uuid.UUID{session.P1.ID, session.P2.ID} {
			if q.sessions[botID] == session {
				delete(q.sessions, botID)
			}
		}
		delete(q.matches, id)
	}
}
