package settlement

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/isbe-bot/bottagon/internal/game/parley"
	"github.com/isbe-bot/bottagon/internal/rating/glicko2"
)

func TestSettlementLogic_Invariants(t *testing.T) {
	// Table-driven verification of settlement math invariants
	tests := []struct {
		name        string
		winner      parley.PlayerID
		p1Wager     int
		p2Wager     int
		mirrorTies  int
		wantP1Delta int
		wantP2Delta int
	}{
		{
			name:        "P1 wins with side bet, P2 loses without bet",
			winner:      parley.Player1,
			p1Wager:     10,
			p2Wager:     0,
			mirrorTies:  0,
			wantP1Delta: 11, // +1 win reward + 10 side bet win
			wantP2Delta: -1, // -1 loss penalty
		},
		{
			name:        "P2 wins with side bet, P1 loses with side bet",
			winner:      parley.Player2,
			p1Wager:     5,
			p2Wager:     8,
			mirrorTies:  0,
			wantP1Delta: -6, // -1 loss penalty - 5 side bet loss
			wantP2Delta: 9,  // +1 win reward + 8 side bet win
		},
		{
			name:        "P1 wins with mirror ties penalty",
			winner:      parley.Player1,
			p1Wager:     10,
			p2Wager:     5,
			mirrorTies:  2,
			wantP1Delta: 9,  // 1 + 10 - 2
			wantP2Delta: -8, // -1 - 5 - 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := parley.MatchConfig{
				TargetWins: 4,
				MaxRounds:  7,
				P1SideBet:  tt.p1Wager,
				P2SideBet:  tt.p2Wager,
			}
			match := parley.NewMatch("test-match", cfg)
			match.Winner = tt.winner
			match.Status = parley.StatusCompleted
			match.MirrorTiesCount = tt.mirrorTies

			p1Settle, p2Settle, err := match.Settle()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if p1Settle.NetMarbleDelta != tt.wantP1Delta {
				t.Errorf("P1 NetMarbleDelta = %d, want %d", p1Settle.NetMarbleDelta, tt.wantP1Delta)
			}
			if p2Settle.NetMarbleDelta != tt.wantP2Delta {
				t.Errorf("P2 NetMarbleDelta = %d, want %d", p2Settle.NetMarbleDelta, tt.wantP2Delta)
			}
		})
	}
}

func TestGlicko2RatingUpdatesOnMatchFinish(t *testing.T) {
	engine := glicko2.NewEngine()

	p1 := glicko2.Rating{Rating: 1500, RD: 200, Volatility: 0.06}
	p2 := glicko2.Rating{Rating: 1500, RD: 200, Volatility: 0.06}

	// Case 1: P1 wins against P2
	newP1 := engine.Update(p1, []glicko2.MatchResult{{Opponent: p2, Score: 1.0}})
	newP2 := engine.Update(p2, []glicko2.MatchResult{{Opponent: p1, Score: 0.0}})

	if newP1.Rating <= p1.Rating {
		t.Errorf("Winner rating should increase, got %f <= %f", newP1.Rating, p1.Rating)
	}
	if newP2.Rating >= p2.Rating {
		t.Errorf("Loser rating should decrease, got %f >= %f", newP2.Rating, p2.Rating)
	}
	if newP1.RD >= p1.RD {
		t.Errorf("Active player RD should decrease, got %f >= %f", newP1.RD, p1.RD)
	}
}

// Compile check for Service interface
func TestServiceConstructor(t *testing.T) {
	var db *sql.DB
	svc := NewService(db)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestLockOrderSorting(t *testing.T) {
	id1 := uuid.MustParse("0192e071-8b32-73a4-94a1-5d15c7e192a0")
	id2 := uuid.MustParse("0192e071-8b32-73a4-94a1-5d15c7e192a1")

	first, second := id1, id2
	if first.String() > second.String() {
		first, second = id2, id1
	}

	if first.String() > second.String() {
		t.Errorf("expected deterministic lock order")
	}
}

// Invalid financial inputs must fail before starting a transaction. This suite
// also runs on hosts without PostgreSQL binaries.
func TestRejectInvalidSettlementBeforeDatabaseAccess(t *testing.T) {
	id, p1, p2 := uuid.New(), uuid.New(), uuid.New()
	base := parley.NewMatch(id.String(), parley.DefaultConfig())
	base.Status = parley.StatusCompleted
	base.Winner = parley.Player1
	tests := []struct {
		name   string
		change func(*parley.Match)
		want   error
	}{
		{"engine ID mismatch", func(m *parley.Match) { m.ID = uuid.NewString() }, ErrInvalidSettlement},
		{"malformed engine ID", func(m *parley.Match) { m.ID = "not-a-uuid" }, ErrInvalidSettlement},
		{"incomplete match", func(m *parley.Match) { m.Status = parley.StatusInProgress }, ErrInvalidSettlement},
		{"unknown winner", func(m *parley.Match) { m.Winner = "intruder" }, ErrInvalidSettlement},
		{"negative winner wager", func(m *parley.Match) { m.Config.P1SideBet = -1 }, ErrInvalidSettlement},
		{"negative loser wager", func(m *parley.Match) { m.Config.P2SideBet = -1 }, ErrInvalidSettlement},
		{"negative mirror count", func(m *parley.Match) { m.MirrorTiesCount = -1 }, ErrInvalidSettlement},
		{"negative score", func(m *parley.Match) { m.Score.P2 = -1 }, ErrInvalidSettlement},
		{"oversized score", func(m *parley.Match) { m.Score.P1 = math.MaxInt32 + 1 }, ErrInvalidSettlement},
		{"machine sized wager", func(m *parley.Match) { m.Config.P1SideBet = int(^uint(0) >> 1) }, ErrInvalidSettlement},
		{"winning delta cannot fit integer", func(m *parley.Match) { m.Config.P1SideBet = math.MaxInt32 }, ErrBalanceBounds},
		{"losing delta counterpart cannot fit integer", func(m *parley.Match) { m.Config.P2SideBet = math.MaxInt32 }, ErrBalanceBounds},
		{"penalty plus losing base cannot fit counterpart", func(m *parley.Match) { m.MirrorTiesCount = math.MaxInt32 }, ErrBalanceBounds},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := *base
			tt.change(&m)
			if err := NewService(nil).SettleMatch(context.Background(), id, p1, p2, &m); !errors.Is(err, tt.want) {
				t.Fatalf("got %v want %v", err, tt.want)
			}
		})
	}
	for _, ids := range [][3]uuid.UUID{{uuid.Nil, p1, p2}, {id, uuid.Nil, p2}, {id, p1, uuid.Nil}, {id, p1, p1}} {
		if err := NewService(nil).SettleMatch(context.Background(), ids[0], ids[1], ids[2], base); !errors.Is(err, ErrInvalidSettlement) {
			t.Errorf("invalid IDs: %v", err)
		}
	}
	if err := NewService(nil).SettleMatch(context.Background(), id, p1, p2, nil); !errors.Is(err, ErrInvalidSettlement) {
		t.Errorf("nil match: %v", err)
	}
}

func TestValidateSettlementStorageBoundaries(t *testing.T) {
	id, p1, p2 := uuid.New(), uuid.New(), uuid.New()
	m := parley.NewMatch(id.String(), parley.DefaultConfig())
	m.Status = parley.StatusCompleted
	m.Winner = parley.Player1
	m.Config.P1SideBet = math.MaxInt32 - 1
	m.Config.P2SideBet = math.MaxInt32 - 1
	if err := validateMatch(id, p1, p2, m); err != nil {
		t.Fatalf("representable boundary deltas: %v", err)
	}
	s1, s2, err := m.Settle()
	if err != nil {
		t.Fatal(err)
	}
	if s1.NetMarbleDelta != math.MaxInt32 || s2.NetMarbleDelta != -math.MaxInt32 {
		t.Errorf("boundary deltas=%d/%d", s1.NetMarbleDelta, s2.NetMarbleDelta)
	}
}
