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
	P1HistoryReveal bool     `json:"p1_history_reveal"` // True if ParryRebound triggered for P1
	P2HistoryReveal bool     `json:"p2_history_reveal"` // True if ParryRebound triggered for P2
}

// ResolveRound evaluates two committed stances under an optional active modifier card.
// Resolution matrix:
//
//	Strike beats Feint (+1 pt)
//	Feint beats Parry (+1 pt)
//	Parry beats Strike (+1 pt)
//	Ties: 0 pts unless modified
//
// Modifiers:
//
//	FeintSurge: Successful Feint awards +2 pts instead of +1.
//	MirrorTiePenalty: Ties penalize both contestants 1 marble into house pot.
//	ParryRebound: Successful Parry triggers history reveal.
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
	case StanceStrike:
		// Strike beats Feint; loses to Parry
		p1Wins = (p2Stance == StanceFeint)
	case StanceParry:
		// Parry beats Strike; loses to Feint
		p1Wins = (p2Stance == StanceStrike)
	case StanceFeint:
		// Feint beats Parry; loses to Strike
		p1Wins = (p2Stance == StanceParry)
	}

	if p1Wins {
		res.Winner = Player1
		points := 1
		if p1Stance == StanceFeint && card == ModifierFeintSurge {
			points = 2
		}
		res.P1Points = points
		if p1Stance == StanceParry && card == ModifierParryRebound {
			res.P1HistoryReveal = true
		}
	} else {
		res.Winner = Player2
		points := 1
		if p2Stance == StanceFeint && card == ModifierFeintSurge {
			points = 2
		}
		res.P2Points = points
		if p2Stance == StanceParry && card == ModifierParryRebound {
			res.P2HistoryReveal = true
		}
	}

	return res, nil
}
