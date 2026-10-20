package glicko2

import (
	"math"
	"testing"
)

// TestGlicko2_GlickmanPaperExample validates the implementation against the worked
// example in Mark Glickman's paper:
// Player: Rating 1500, RD 200, Volatility 0.06, tau = 0.5.
// Opponents:
// 1. Rating 1400, RD 30, Score 1 (Win)
// 2. Rating 1550, RD 100, Score 0 (Loss)
// 3. Rating 1700, RD 300, Score 0 (Loss)
// Expected after update:
// Rating ≈ 1464.06, RD ≈ 151.52, Volatility ≈ 0.05999
func TestGlicko2_GlickmanPaperExample(t *testing.T) {
	player := Rating{
		Rating:     1500.0,
		RD:         200.0,
		Volatility: 0.06,
	}

	matches := []MatchResult{
		{Opponent: Rating{Rating: 1400, RD: 30, Volatility: 0.06}, Score: 1.0},
		{Opponent: Rating{Rating: 1550, RD: 100, Volatility: 0.06}, Score: 0.0},
		{Opponent: Rating{Rating: 1700, RD: 300, Volatility: 0.06}, Score: 0.0},
	}

	engine := NewEngine()
	engine.Tau = 0.5

	updated := engine.Update(player, matches)

	if math.Abs(updated.Rating-1464.06) > 0.1 {
		t.Errorf("Rating = %f, expected ~1464.06", updated.Rating)
	}
	if math.Abs(updated.RD-151.52) > 0.1 {
		t.Errorf("RD = %f, expected ~151.52", updated.RD)
	}
	if math.Abs(updated.Volatility-0.05999) > 0.0001 {
		t.Errorf("Volatility = %f, expected ~0.05999", updated.Volatility)
	}
}

func TestGlicko2_NoMatchesIncreasesRD(t *testing.T) {
	player := Rating{
		Rating:     1500.0,
		RD:         200.0,
		Volatility: 0.06,
	}

	engine := NewEngine()
	updated := engine.Update(player, nil)

	if updated.Rating != player.Rating {
		t.Errorf("Rating should not change when no matches played, got %f", updated.Rating)
	}
	if updated.RD <= player.RD {
		t.Errorf("RD should increase over inactivity period, got %f <= %f", updated.RD, player.RD)
	}
}
