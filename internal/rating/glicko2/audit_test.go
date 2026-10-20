package glicko2

import (
	"math"
	"testing"
)

func TestAuditExtremeRatingsStayFinite(t *testing.T) {
	for _, player := range []Rating{{}, {math.NaN(), 200, .06}, {1500, math.Inf(1), .06}, {1500, 200, 0}, {1e6, 1, .06}, {-1e6, 350, .06}} {
		for _, tau := range []float64{.5, math.NaN(), math.Inf(1), 0, -1, 1e-300} {
			e := NewEngine()
			e.Tau = tau
			got := e.Update(player, []MatchResult{{Opponent: Rating{-1e6, 1, .06}, Score: 0}})
			if math.IsNaN(got.Rating) || math.IsInf(got.Rating, 0) || math.IsNaN(got.RD) || math.IsInf(got.RD, 0) || got.RD <= 0 || math.IsNaN(got.Volatility) || math.IsInf(got.Volatility, 0) || got.Volatility <= 0 {
				t.Errorf("nonfinite rating: player=%+v tau=%v got=%+v", player, tau, got)
			}
		}
	}
}

func BenchmarkUpdate(b *testing.B) {
	e := NewEngine()
	p := NewDefaultRating()
	matches := []MatchResult{{Opponent: p, Score: 1}}
	b.ReportAllocs()
	for b.Loop() {
		e.Update(p, matches)
	}
}

func TestAuditCheckedRatingRejectsInvalidInputs(t *testing.T) {
	p := NewDefaultRating()
	for _, score := range []float64{-.1, .1, 1.1, math.NaN(), math.Inf(1)} {
		got, err := NewEngine().UpdateChecked(p, []MatchResult{{Opponent: p, Score: score}})
		if err != ErrInvalidScore || got != p {
			t.Fatalf("invalid score %v: %+v %v", score, got, err)
		}
	}
	if _, err := NewEngine().UpdateChecked(Rating{}, nil); err != ErrInvalidRating {
		t.Fatalf("zero rating: %v", err)
	}
	if _, err := (&Glicko2Engine{Tau: math.Inf(1)}).UpdateChecked(p, nil); err != ErrInvalidTau {
		t.Fatalf("invalid tau: %v", err)
	}
	if got, err := NewEngine().UpdateChecked(p, []MatchResult{{Opponent: Rating{}, Score: 1}}); err != ErrInvalidRating || got != p {
		t.Fatalf("invalid opponent: %+v %v", got, err)
	}
	var nilEngine *Glicko2Engine
	if _, err := nilEngine.UpdateChecked(p, nil); err != ErrInvalidTau {
		t.Fatal(err)
	}
	if got := nilEngine.Update(p, nil); !validRating(got) {
		t.Fatalf("nil legacy engine: %+v", got)
	}
}

func TestAuditRatingSymmetryAndRepeatedUpdates(t *testing.T) {
	e := NewEngine()
	p := NewDefaultRating()
	win, err := e.UpdateChecked(p, []MatchResult{{Opponent: p, Score: 1}})
	if err != nil {
		t.Fatal(err)
	}
	loss, err := e.UpdateChecked(p, []MatchResult{{Opponent: p, Score: 0}})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(win.Rating+loss.Rating-2*p.Rating) > 1e-9 {
		t.Fatal("asymmetric rating update")
	}
	for i := range 1000 {
		var err error
		p, err = e.UpdateChecked(p, []MatchResult{{Opponent: NewDefaultRating(), Score: float64(i%3) / 2}})
		if err != nil || !validRating(p) {
			t.Fatalf("period %d: %+v %v", i, p, err)
		}
	}
}

func FuzzCheckedRatingFinite(f *testing.F) {
	f.Add(1500.0, 350.0, .06, 0.5, 1600.0, 200.0, 1.0)
	f.Add(1e6, 1.0, .06, .001, -1e6, 1.0, 0.0)
	f.Fuzz(func(t *testing.T, r, rd, vol, tau, opp, oppRD, score float64) {
		p := Rating{r, rd, vol}
		e := &Glicko2Engine{Tau: tau}
		got, err := e.UpdateChecked(p, []MatchResult{{Opponent: Rating{opp, oppRD, .06}, Score: score}})
		if err == nil && !validRating(got) {
			t.Fatalf("nonfinite success: %+v", got)
		}
		if safe := e.Update(p, []MatchResult{{Opponent: Rating{opp, oppRD, .06}, Score: score}}); !validRating(safe) {
			t.Fatalf("nonfinite legacy result: %+v", safe)
		}
	})
}
