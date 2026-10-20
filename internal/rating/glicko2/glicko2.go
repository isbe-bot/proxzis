package glicko2

import (
	"errors"
	"math"
)

const (
	DefaultRating     = 1500.0
	DefaultRD         = 350.0
	DefaultVolatility = 0.06
	DefaultTau        = 0.5
	ScaleConstant     = 173.7178
	ConvergenceTol    = 0.000001
)

// Rating represents a player's Glicko-2 rating components.
type Rating struct {
	Rating     float64 `json:"rating"`     // r (default 1500)
	RD         float64 `json:"rd"`         // RD (default 350)
	Volatility float64 `json:"volatility"` // sigma (default 0.06)
}

// NewDefaultRating returns a fresh player rating.
func NewDefaultRating() Rating {
	return Rating{
		Rating:     DefaultRating,
		RD:         DefaultRD,
		Volatility: DefaultVolatility,
	}
}

// Result represents the match outcome from the player's perspective:
// 1.0 = Win, 0.5 = Draw, 0.0 = Loss.
type MatchResult struct {
	Opponent Rating
	Score    float64
}

// Glicko2Engine manages rating transformations with configurable tau.
type Glicko2Engine struct {
	Tau float64
}

// NewEngine creates an engine with the standard tau (0.5).
func NewEngine() *Glicko2Engine {
	return &Glicko2Engine{Tau: DefaultTau}
}

// g converts rating deviation to Glicko-2 g(phi).
func g(phi float64) float64 {
	return 1.0 / math.Sqrt(1.0+3.0*phi*phi/(math.Pi*math.Pi))
}

// E computes expected match score.
func E(mu, muJ, phiJ float64) float64 {
	return 1.0 / (1.0 + math.Exp(-g(phiJ)*(mu-muJ)))
}

// Numerical guardrails are deliberately wider than normal competitive ratings.
// They bound intermediate products and prevent corrupt persisted input poisoning a ledger.
const (
	MaxRatingMagnitude = 1e6
	MaxRD              = 1e6
	MinVolatility      = 1e-6
	MaxVolatility      = 10.0
	MinTau             = 1e-3
	MaxTau             = 10.0
	maxIterations      = 1000
)

var (
	ErrInvalidRating = errors.New("invalid Glicko-2 rating")
	ErrInvalidScore  = errors.New("Glicko-2 score must be 0, 0.5, or 1")
	ErrInvalidTau    = errors.New("invalid Glicko-2 tau")
	ErrConvergence   = errors.New("Glicko-2 numerical convergence failed")
)

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
func validRating(p Rating) bool {
	return finite(p.Rating) && math.Abs(p.Rating) <= MaxRatingMagnitude && finite(p.RD) && p.RD > 0 && p.RD <= MaxRD && finite(p.Volatility) && p.Volatility >= MinVolatility && p.Volatility <= MaxVolatility
}

// Update preserves the legacy total-function interface. Invalid batches leave a
// valid player unchanged; corrupt player fields are reset to defaults. Callers
// that persist updates should use UpdateChecked to surface validation failures.
func (e *Glicko2Engine) Update(player Rating, matches []MatchResult) Rating {
	safe := player
	if !finite(safe.Rating) || math.Abs(safe.Rating) > MaxRatingMagnitude {
		safe.Rating = DefaultRating
	}
	if !finite(safe.RD) || safe.RD <= 0 || safe.RD > MaxRD {
		safe.RD = DefaultRD
	}
	if !finite(safe.Volatility) || safe.Volatility < MinVolatility || safe.Volatility > MaxVolatility {
		safe.Volatility = DefaultVolatility
	}
	tau := DefaultTau
	if e != nil && finite(e.Tau) && e.Tau >= MinTau && e.Tau <= MaxTau {
		tau = e.Tau
	}
	engine := Glicko2Engine{Tau: tau}
	updated, err := engine.UpdateChecked(safe, matches)
	if err != nil {
		return safe
	}
	return updated
}

// UpdateChecked follows Glickman's worked algorithm with explicit validation
// and bounded root searches. On error the original player is returned unchanged.
func (e *Glicko2Engine) UpdateChecked(player Rating, matches []MatchResult) (Rating, error) {
	if !validRating(player) {
		return player, ErrInvalidRating
	}
	if e == nil || !finite(e.Tau) || e.Tau < MinTau || e.Tau > MaxTau {
		return player, ErrInvalidTau
	}
	tau := e.Tau
	for _, m := range matches {
		if !validRating(m.Opponent) {
			return player, ErrInvalidRating
		}
		if m.Score != 0 && m.Score != 0.5 && m.Score != 1 {
			return player, ErrInvalidScore
		}
	}

	// Step 2: Convert to Glicko-2 scale:
	// mu = (r - 1500) / 173.7178
	// phi = RD / 173.7178
	// sigma = volatility
	mu := (player.Rating - DefaultRating) / ScaleConstant
	phi := player.RD / ScaleConstant
	sigma := player.Volatility

	// If player did not compete during rating period:
	if len(matches) == 0 {
		newPhi := math.Hypot(phi, sigma)
		return Rating{
			Rating:     player.Rating,
			RD:         math.Min(newPhi*ScaleConstant, MaxRD),
			Volatility: sigma,
		}, nil
	}

	// Step 3: Compute variance v and delta
	var vInv float64
	var deltaSum float64

	for _, m := range matches {
		muJ := (m.Opponent.Rating - DefaultRating) / ScaleConstant
		phiJ := m.Opponent.RD / ScaleConstant
		gPhiJ := g(phiJ)
		// Avoid zero information from rounded logistic endpoints.
		eVal := math.Max(1e-15, math.Min(1-1e-15, E(mu, muJ, phiJ)))

		vInv += gPhiJ * gPhiJ * eVal * (1.0 - eVal)
		deltaSum += gPhiJ * (m.Score - eVal)
	}

	if !finite(vInv) || vInv <= 0 || !finite(deltaSum) {
		return player, ErrConvergence
	}
	v := 1.0 / vInv
	delta := v * deltaSum

	// Step 5: Determine new volatility sigma' via Illinois algorithm
	a := math.Log(sigma * sigma)
	f := func(x float64) float64 {
		expX := math.Exp(x)
		num1 := expX * (delta*delta - phi*phi - v - expX)
		den1 := 2.0 * math.Pow(phi*phi+v+expX, 2)
		return (num1 / den1) - ((x - a) / (tau * tau))
	}

	A := a
	var B float64
	if delta*delta > phi*phi+v {
		B = math.Log(delta*delta - phi*phi - v)
	} else {
		k := 1.0
		for f(a-k*tau) < 0 && k < maxIterations {
			k++
		}
		if k >= maxIterations {
			return player, ErrConvergence
		}
		B = a - k*tau
	}

	fA := f(A)
	fB := f(B)

	iterations := 0
	for math.Abs(B-A) > ConvergenceTol {
		iterations++
		if iterations > maxIterations || !finite(fA) || !finite(fB) || fA == fB {
			return player, ErrConvergence
		}
		C := A + (A-B)*fA/(fB-fA)
		fC := f(C)
		if !finite(C) || !finite(fC) {
			return player, ErrConvergence
		}
		if fC*fB <= 0 {
			A = B
			fA = fB
		} else {
			fA = fA / 2.0
		}
		B = C
		fB = fC
	}

	newSigma := math.Exp(A / 2.0)

	// Step 6: Update rating deviation phi* and phi'
	phiStar := math.Sqrt(phi*phi + newSigma*newSigma)
	newPhi := 1.0 / math.Sqrt(1.0/(phiStar*phiStar)+1.0/v)

	// Step 7: Update rating mu'
	newMu := mu + newPhi*newPhi*deltaSum

	// Step 8: Convert back to original scale
	updated := Rating{
		Rating:     DefaultRating + ScaleConstant*newMu,
		RD:         ScaleConstant * newPhi,
		Volatility: newSigma,
	}
	if !validRating(updated) {
		return player, ErrConvergence
	}
	return updated, nil
}
