package parley

import (
	"errors"
	"strings"
)

// Stance represents one of the three primary Parley stances.
type Stance string

const (
	StanceNone   Stance = ""
	StanceStrike Stance = "strike"
	StanceParry  Stance = "parry"
	StanceFeint  Stance = "feint"
)

var (
	ErrInvalidPlayer     = errors.New("unknown player")
	ErrAlreadyCommitted  = errors.New("player already committed this round")
	ErrInvalidConfig     = errors.New("invalid match configuration")
	ErrInvalidWagerRound = errors.New("side wager can only be changed in round 1")
	ErrInvalidStance     = errors.New("invalid stance")
	ErrInvalidModifier   = errors.New("invalid modifier card")
	ErrMatchOver         = errors.New("match is already concluded")
	ErrNegativeWager     = errors.New("pass-line wager cannot be negative")
	ErrExcessiveWager    = errors.New("pass-line wager exceeds purse")
	ErrRoundNotPending   = errors.New("round is not in pending state")
	ErrMissingStance     = errors.New("both players must commit a stance")
)

// ParseStance parses a string into a validated Stance.
func ParseStance(s string) (Stance, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "strike":
		return StanceStrike, nil
	case "parry":
		return StanceParry, nil
	case "feint":
		return StanceFeint, nil
	default:
		return StanceNone, ErrInvalidStance
	}
}

// IsValid checks whether a stance is recognized and valid.
func (s Stance) IsValid() bool {
	return s == StanceStrike || s == StanceParry || s == StanceFeint
}

// ModifierCard represents environmental rule modifiers (max 1 active).
type ModifierCard string

const (
	ModifierNone             ModifierCard = ""
	ModifierMirrorTiePenalty ModifierCard = "MirrorTiePenalty"
	ModifierFeintSurge       ModifierCard = "FeintSurge"
	ModifierParryRebound     ModifierCard = "ParryRebound"
)

// ParseModifierCard validates and parses a modifier card string.
func ParseModifierCard(s string) (ModifierCard, error) {
	norm := strings.TrimSpace(s)
	switch norm {
	case "", "none", "None":
		return ModifierNone, nil
	case "MirrorTiePenalty", "mirror_tie_penalty", "mirror-tie-penalty":
		return ModifierMirrorTiePenalty, nil
	case "FeintSurge", "feint_surge", "feint-surge":
		return ModifierFeintSurge, nil
	case "ParryRebound", "parry_rebound", "parry-rebound":
		return ModifierParryRebound, nil
	default:
		return ModifierNone, ErrInvalidModifier
	}
}

// ModifierCardInfo contains description and metadata for a modifier card.
type ModifierCardInfo struct {
	Card        ModifierCard `json:"card"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
}

// Info returns user/bot facing metadata for the modifier card.
func (m ModifierCard) Info() ModifierCardInfo {
	switch m {
	case ModifierMirrorTiePenalty:
		return ModifierCardInfo{
			Card:        m,
			Name:        "Mirror Tie Penalty",
			Description: "Stance ties penalize both contestants 1 marble into the house pot.",
		}
	case ModifierFeintSurge:
		return ModifierCardInfo{
			Card:        m,
			Name:        "Feint Surge",
			Description: "Successful Feint awards +2 points instead of +1.",
		}
	case ModifierParryRebound:
		return ModifierCardInfo{
			Card:        m,
			Name:        "Parry Rebound",
			Description: "Successful Parry immediately reveals opponent's prior stance history.",
		}
	default:
		return ModifierCardInfo{
			Card:        ModifierNone,
			Name:        "Standard Duel",
			Description: "Standard Parley rules with no modifier cards.",
		}
	}
}

// IsValid reports whether the card is supported, including standard play.
func (m ModifierCard) IsValid() bool {
	return m == ModifierNone || m == ModifierMirrorTiePenalty || m == ModifierFeintSurge || m == ModifierParryRebound
}
