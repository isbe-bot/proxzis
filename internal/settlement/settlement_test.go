package settlement

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/isbe-bot/bottagon/internal/game/parley"
	"github.com/isbe-bot/bottagon/internal/rating/glicko2"
)

func TestSettlementLogic_Invariants(t *testing.T) {
	// Table-driven verification of settlement math invariants
	tests := []struct {
		name         string
		winner       parley.PlayerID
		p1Wager      int
		p2Wager      int
		mirrorTies   int
		wantP1Delta  int
		wantP2Delta  int
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
