package glicko2

import (
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

// Update computes the new Rating for a player given their match results in a rating period.
// Follows Glickman's paper: "Example of the Glicko-2 system".
func (e *Glicko2Engine) Update(player Rating, matches []MatchResult) Rating {
	tau := e.Tau
	if tau <= 0 {
		tau = DefaultTau
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
		newPhi := math.Sqrt(phi*phi + sigma*sigma)
		return Rating{
			Rating:     player.Rating,
			RD:         newPhi * ScaleConstant,
			Volatility: sigma,
		}
	}

	// Step 3: Compute variance v and delta
	var vInv float64
	var deltaSum float64

	for _, m := range matches {
		muJ := (m.Opponent.Rating - DefaultRating) / ScaleConstant
		phiJ := m.Opponent.RD / ScaleConstant
		gPhiJ := g(phiJ)
		eVal := E(mu, muJ, phiJ)

		vInv += gPhiJ * gPhiJ * eVal * (1.0 - eVal)
		deltaSum += gPhiJ * (m.Score - eVal)
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
		for f(a-k*tau) < 0 {
			k++
		}
		B = a - k*tau
	}

	fA := f(A)
	fB := f(B)

	for math.Abs(B-A) > ConvergenceTol {
		C := A + (A-B)*fA/(fB-fA)
		fC := f(C)
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
	return Rating{
		Rating:     DefaultRating + ScaleConstant*newMu,
		RD:         ScaleConstant * newPhi,
		Volatility: newSigma,
	}
}
