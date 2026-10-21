package parley

// PlayerID represents a player identifier in a match (e.g. "P1", "P2", or bot UUID).
type PlayerID string

const (
	Player1 PlayerID = "P1"
	Player2 PlayerID = "P2"
)

// RoundResolution contains the outcome of resolving a single round.
type RoundResolution struct {
	P1Stance        Stance   `json:"p1_stance"`
	P2Stance        Stance   `json:"p2_stance"`
	P1Points        int      `json:"p1_points"`
	P2Points        int      `json:"p2_points"`
	IsTie           bool     `json:"is_tie"`
	Winner          PlayerID `json:"winner"`            // Empty if tie
	MirrorPenP1     int      `json:"mirror_penalty_p1"` // Marbles penalized to house if MirrorTiePenalty active
	MirrorPenP2     int      `json:"mirror_penalty_p2"`
	P1HistoryReveal bool     `json:"p1_history_reveal"` // True if PaperRebound triggered for P1
	P2HistoryReveal bool     `json:"p2_history_reveal"` // True if PaperRebound triggered for P2
}

// ResolveRound evaluates two committed stances under an optional active modifier card.
// Resolution matrix:
//
//	Rock beats Scissors (+1 pt)
//	Scissors beats Paper (+1 pt)
//	Paper beats Rock (+1 pt)
//	Ties: 0 pts unless modified
//
// Modifiers:
//
//	ScissorsSurge: A round won with Scissors awards +2 pts instead of +1.
//	MirrorTiePenalty: Ties penalize both contestants 1 marble into house pot.
//	PaperRebound: A round won with Paper triggers history reveal.
func ResolveRound(p1Stance, p2Stance Stance, card ModifierCard) (RoundResolution, error) {
	if !p1Stance.IsValid() || !p2Stance.IsValid() {
		return RoundResolution{}, ErrInvalidStance
	}

	if !card.IsValid() {
		return RoundResolution{}, ErrInvalidModifier
	}

	res := RoundResolution{
		P1Stance: p1Stance,
		P2Stance: p2Stance,
	}

	if p1Stance == p2Stance {
		res.IsTie = true
		if card == ModifierMirrorTiePenalty {
			res.MirrorPenP1 = 1
			res.MirrorPenP2 = 1
		}
		return res, nil
	}

	p1Wins := false
	switch p1Stance {
	case StanceRock:
		// Rock beats Scissors; loses to Paper
		p1Wins = (p2Stance == StanceScissors)
	case StancePaper:
		// Paper beats Rock; loses to Scissors
		p1Wins = (p2Stance == StanceRock)
	case StanceScissors:
		// Scissors beats Paper; loses to Rock
		p1Wins = (p2Stance == StancePaper)
	}

	if p1Wins {
		res.Winner = Player1
		points := 1
		if p1Stance == StanceScissors && card == ModifierScissorsSurge {
			points = 2
		}
		res.P1Points = points
		if p1Stance == StancePaper && card == ModifierPaperRebound {
			res.P1HistoryReveal = true
		}
	} else {
		res.Winner = Player2
		points := 1
		if p2Stance == StanceScissors && card == ModifierScissorsSurge {
			points = 2
		}
		res.P2Points = points
		if p2Stance == StancePaper && card == ModifierPaperRebound {
			res.P2HistoryReveal = true
		}
	}

	return res, nil
}
