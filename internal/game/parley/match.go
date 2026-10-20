package parley

import (
	"errors"
	"fmt"
)

// MatchStatus represents the state of a match.
type MatchStatus string

const (
	StatusPending   MatchStatus = "pending"
	StatusInProgress MatchStatus = "in_progress"
	StatusCompleted MatchStatus = "completed"
)

// Round represents a single played or in-progress round.
type Round struct {
	RoundNumber int             `json:"round_number"`
	P1Stance    Stance          `json:"p1_stance"`
	P2Stance    Stance          `json:"p2_stance"`
	P1Banter    string          `json:"p1_banter"`
	P2Banter    string          `json:"p2_banter"`
	Resolution  RoundResolution `json:"resolution"`
	IsSuddenDeath bool          `json:"is_sudden_death"`
}

// Score tracks player scores.
type Score struct {
	P1 int `json:"p1"`
	P2 int `json:"p2"`
}

// MatchConfig sets configuration for a match.
type MatchConfig struct {
	TargetWins    int          `json:"target_wins"`     // Default 4 (Best-of-7: first to 4 wins)
	MaxRounds     int          `json:"max_rounds"`      // Default 7 (Best-of-7)
	ModifierCard  ModifierCard `json:"modifier_card"`   // 0 or 1 active modifier
	P1SideBet     int          `json:"p1_side_bet"`     // Round 1 side wager
	P2SideBet     int          `json:"p2_side_bet"`
	P1Purse       int          `json:"p1_purse"`
	P2Purse       int          `json:"p2_purse"`
}

// DefaultConfig returns the standard Best-of-7 configuration.
func DefaultConfig() MatchConfig {
	return MatchConfig{
		TargetWins: 4,
		MaxRounds:  7,
		ModifierCard: ModifierNone,
	}
}

// Match is the state machine governing a complete Parley duel.
type Match struct {
	ID          string       `json:"id"`
	Config      MatchConfig  `json:"config"`
	Status      MatchStatus  `json:"status"`
	CurrentRound int         `json:"current_round"` // 1-indexed
	Score       Score        `json:"score"`
	Rounds      []Round      `json:"rounds"`
	Winner      PlayerID     `json:"winner"`
	IsSuddenDeath bool       `json:"is_sudden_death"`
	MirrorTiesCount int      `json:"mirror_ties_count"`

	// Pending turn commitments for current round
	p1Committed bool
	p1Stance    Stance
	p1Banter    string
	p2Committed bool
	p2Stance    Stance
	p2Banter    string
}

// NewMatch initializes a new Match state machine.
func NewMatch(id string, cfg MatchConfig) *Match {
	if cfg.TargetWins <= 0 {
		cfg.TargetWins = 4
	}
	if cfg.MaxRounds <= 0 {
		cfg.MaxRounds = 7
	}
	return &Match{
		ID:           id,
		Config:       cfg,
		Status:       StatusPending,
		CurrentRound: 1,
		Score:        Score{P1: 0, P2: 0},
		Rounds:       make([]Round, 0, cfg.MaxRounds),
	}
}

// CommitTurn registers a player's move for the current round.
func (m *Match) CommitTurn(player PlayerID, stance Stance, banter string, sideBet int) error {
	if m.Status == StatusCompleted {
		return ErrMatchOver
	}
	if !stance.IsValid() {
		return ErrInvalidStance
	}

	// Validate Round 1 side wager
	if m.CurrentRound == 1 {
		if sideBet < 0 {
			return ErrNegativeWager
		}
		if player == Player1 {
			if m.Config.P1Purse > 0 && sideBet > m.Config.P1Purse {
				return ErrExcessiveWager
			}
			m.Config.P1SideBet = sideBet
		} else if player == Player2 {
			if m.Config.P2Purse > 0 && sideBet > m.Config.P2Purse {
				return ErrExcessiveWager
			}
			m.Config.P2SideBet = sideBet
		}
	}

	sanitizedBanter := SanitizeBanter(banter)

	switch player {
	case Player1:
		m.p1Stance = stance
		m.p1Banter = sanitizedBanter
		m.p1Committed = true
	case Player2:
		m.p2Stance = stance
		m.p2Banter = sanitizedBanter
		m.p2Committed = true
	default:
		return errors.New("unknown player")
	}

	if m.Status == StatusPending {
		m.Status = StatusInProgress
	}

	// If both committed, resolve immediately
	if m.p1Committed && m.p2Committed {
		return m.resolveCurrentRound()
	}

	return nil
}

// PlayRound is a convenience function to directly play a round with both players' stances.
func (m *Match) PlayRound(p1Stance, p2Stance Stance, p1Banter, p2Banter string) (RoundResolution, error) {
	if m.Status == StatusCompleted {
		return RoundResolution{}, ErrMatchOver
	}
	if err := m.CommitTurn(Player1, p1Stance, p1Banter, m.Config.P1SideBet); err != nil {
		return RoundResolution{}, err
	}
	if err := m.CommitTurn(Player2, p2Stance, p2Banter, m.Config.P2SideBet); err != nil {
		return RoundResolution{}, err
	}

	lastRound := m.Rounds[len(m.Rounds)-1]
	return lastRound.Resolution, nil
}

func (m *Match) resolveCurrentRound() error {
	res, err := ResolveRound(m.p1Stance, m.p2Stance, m.Config.ModifierCard)
	if err != nil {
		return err
	}

	if res.IsTie && m.Config.ModifierCard == ModifierMirrorTiePenalty {
		m.MirrorTiesCount++
	}

	m.Score.P1 += res.P1Points
	m.Score.P2 += res.P2Points

	roundRecord := Round{
		RoundNumber:   m.CurrentRound,
		P1Stance:      m.p1Stance,
		P2Stance:      m.p2Stance,
		P1Banter:      m.p1Banter,
		P2Banter:      m.p2Banter,
		Resolution:    res,
		IsSuddenDeath: m.IsSuddenDeath,
	}
	m.Rounds = append(m.Rounds, roundRecord)

	// Reset pending turn commitments
	m.p1Committed = false
	m.p2Committed = false
	m.p1Stance = StanceNone
	m.p2Stance = StanceNone
	m.p1Banter = ""
	m.p2Banter = ""

	// Check win condition
	m.evaluateMatchStatus()
	return nil
}

func (m *Match) evaluateMatchStatus() {
	target := m.Config.TargetWins

	// 1. Check if either player reached target wins
	if m.Score.P1 >= target || m.Score.P2 >= target {
		if m.Score.P1 > m.Score.P2 {
			m.Winner = Player1
			m.Status = StatusCompleted
			return
		} else if m.Score.P2 > m.Score.P1 {
			m.Winner = Player2
			m.Status = StatusCompleted
			return
		}
	}

	// 2. Regular max rounds check (e.g. 7 rounds)
	if !m.IsSuddenDeath && m.CurrentRound >= m.Config.MaxRounds {
		if m.Score.P1 > m.Score.P2 {
			m.Winner = Player1
			m.Status = StatusCompleted
			return
		} else if m.Score.P2 > m.Score.P1 {
			m.Winner = Player2
			m.Status = StatusCompleted
			return
		} else {
			// Tied at end of regulation -> Trigger Sudden Death
			m.IsSuddenDeath = true
		}
	}

	// 3. In sudden death: any round where one player leads ends the match immediately
	if m.IsSuddenDeath && m.Score.P1 != m.Score.P2 {
		if m.Score.P1 > m.Score.P2 {
			m.Winner = Player1
		} else {
			m.Winner = Player2
		}
		m.Status = StatusCompleted
		return
	}

	// Advance to next round if match is still running
	m.CurrentRound++
}

// Settle returns the Pass-Line settlements for both players.
func (m *Match) Settle() (Settlement, Settlement, error) {
	if m.Status != StatusCompleted {
		return Settlement{}, Settlement{}, fmt.Errorf("cannot settle incomplete match")
	}

	p1Won := m.Winner == Player1
	p2Won := m.Winner == Player2
	isDraw := m.Winner == ""

	p1Settle := SettlePassLine(p1Won, isDraw, m.Config.P1SideBet, m.MirrorTiesCount)
	p2Settle := SettlePassLine(p2Won, isDraw, m.Config.P2SideBet, m.MirrorTiesCount)

	return p1Settle, p2Settle, nil
}
