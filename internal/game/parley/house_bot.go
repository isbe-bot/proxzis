package parley

// HouseBot is a resident that plays its own stance policy.
// House agents do not banter.
type HouseBot interface {
	Name() string
	SelectStance(history []Round, playerRole PlayerID) Stance
}

// WeightedBot chooses rock, paper, or scissors from fixed weights.
// The choice is a pure function of the resident and the round index.
type WeightedBot struct {
	name                  string
	rock, paper, scissors int
	salt                  uint64
}

// NewWeighted returns a stance policy. Weights are percentages that sum to 100.
func NewWeighted(name string, rock, paper, scissors int, salt uint64) *WeightedBot {
	return &WeightedBot{name: name, rock: rock, paper: paper, scissors: scissors, salt: salt}
}

// Name returns the resident's display name.
func (b *WeightedBot) Name() string {
	return b.name
}

// SelectStance draws a stance from the resident's weights.
func (b *WeightedBot) SelectStance(history []Round, _ PlayerID) Stance {
	total := b.rock + b.paper + b.scissors
	if total <= 0 {
		return StanceRock
	}
	slot := int((b.salt + uint64(len(history))*97) % uint64(total))
	switch {
	case slot < b.rock:
		return StanceRock
	case slot < b.rock+b.paper:
		return StancePaper
	default:
		return StanceScissors
	}
}
