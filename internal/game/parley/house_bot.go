package parley

// HouseBot defines the interface for a deterministic House Regular bot.
type HouseBot interface {
	Name() string
	SelectStance(history []Round, playerRole PlayerID) Stance
	SelectBanter(history []Round, playerRole PlayerID) string
}

// RegularBot implements a deterministic heuristic House Regular state machine.
type RegularBot struct {
	name        string
	strategy    string
	phraseBank  []string
	counterBank []string
}

// Standard House Regular Bot Names
const (
	HouseBotBlinky = "Blinky" // Counter-puncher / Bayesian reader: leans to counter opponent's most frequent stance
	HouseBotRusty  = "Rusty"  // Aggressive forward press: heavy Strike bias, occasional Feint
	HouseBotDeacon = "Deacon" // Cautious defensive monk: heavy Parry bias, cycles methodically
)

// NewBlinky creates Blinky: an adaptive counter-reader bot.
// Analyzes opponent's stance frequency and commits the hard counter.
func NewBlinky() *RegularBot {
	return &RegularBot{
		name:     HouseBotBlinky,
		strategy: "adaptive_counter",
		phraseBank: []string{
			"Your hesitation on round 1 was noted.",
			"Patterns are hard to break, aren't they?",
			"Calculating the Nash equilibrium for your next breath.",
			"A subtle cadence, but entirely predictable.",
			"The parlor remembers every twitch.",
		},
		counterBank: []string{
			"Predictability is an expensive habit.",
			"I saw that angle three cycles ago.",
			"Defense is just patience with teeth.",
		},
	}
}

// NewRusty creates Rusty: an aggressive Strike-heavy brawler.
func NewRusty() *RegularBot {
	return &RegularBot{
		name:     HouseBotRusty,
		strategy: "aggressive_strike",
		phraseBank: []string{
			"Steel doesn't negotiate.",
			"Keep your guard up, rookie.",
			"Strike first, count marbles later.",
			"Nothing fancy, just forward momentum.",
			"Let's see how much pressure your circuits take.",
		},
		counterBank: []string{
			"Right down the middle.",
			"Told you to brace.",
			"Keep swinging.",
		},
	}
}

// NewDeacon creates Deacon: a patient, defensive monk favoring Parry and cyclic play.
func NewDeacon() *RegularBot {
	return &RegularBot{
		name:     HouseBotDeacon,
		strategy: "parry_defense",
		phraseBank: []string{
			"Stillness reflects all movement.",
			"Rushing only spends your marbles faster.",
			"A parry at rest stays at rest.",
			"Every strike carries its own ruin.",
			"Observe the threshold.",
		},
		counterBank: []string{
			"The wall does not yield.",
			"Patience yields compound interest.",
			"Reflected cleanly.",
		},
	}
}

// Name returns the bot's moniker.
func (b *RegularBot) Name() string {
	return b.name
}

// SelectStance calculates the next stance deterministically based on round history and heuristic personality.
func (b *RegularBot) SelectStance(history []Round, selfRole PlayerID) Stance {
	roundIdx := len(history)

	switch b.name {
	case HouseBotRusty:
		// Aggressive sequence: Strike, Strike, Feint, Strike, Parry, Strike, Feint...
		cycle := []Stance{StanceStrike, StanceStrike, StanceFeint, StanceStrike, StanceParry, StanceStrike, StanceFeint}
		return cycle[roundIdx%len(cycle)]

	case HouseBotDeacon:
		// Defensive sequence: Parry, Parry, Feint, Parry, Strike, Parry, Feint...
		cycle := []Stance{StanceParry, StanceParry, StanceFeint, StanceParry, StanceStrike, StanceParry, StanceFeint}
		return cycle[roundIdx%len(cycle)]

	case HouseBotBlinky:
		fallthrough
	default:
		// Blinky: Adaptive counter
		if roundIdx == 0 {
			// Round 1 default: Strike to test opponent
			return StanceStrike
		}

		// Count opponent's prior moves
		strikeCount := 0
		parryCount := 0
		feintCount := 0

		for _, r := range history {
			oppStance := r.P1Stance
			if selfRole == Player1 {
				oppStance = r.P2Stance
			}
			switch oppStance {
			case StanceStrike:
				strikeCount++
			case StanceParry:
				parryCount++
			case StanceFeint:
				feintCount++
			}
		}

		// What did the opponent play last round?
		lastRound := history[roundIdx-1]
		lastOppStance := lastRound.P1Stance
		if selfRole == Player1 {
			lastOppStance = lastRound.P2Stance
		}

		// Heuristic:
		// If opponent loves Strike -> Parry beats Strike
		// If opponent loves Parry -> Feint beats Parry
		// If opponent loves Feint -> Strike beats Feint
		// If tied frequencies, counter last round stance
		if strikeCount > parryCount && strikeCount > feintCount {
			return StanceParry
		} else if parryCount > strikeCount && parryCount > feintCount {
			return StanceFeint
		} else if feintCount > strikeCount && feintCount > parryCount {
			return StanceStrike
		}

		// Default tie breaker: counter last opponent move
		switch lastOppStance {
		case StanceStrike:
			return StanceParry
		case StanceParry:
			return StanceFeint
		case StanceFeint:
			return StanceStrike
		default:
			return StanceStrike
		}
	}
}

// SelectBanter picks a deterministic canned banter phrase based on round number.
func (b *RegularBot) SelectBanter(history []Round, selfRole PlayerID) string {
	roundIdx := len(history)
	if len(b.phraseBank) == 0 {
		return ""
	}

	// If last round self scored, pick counterBank phrase if available
	if roundIdx > 0 && len(b.counterBank) > 0 {
		lastRes := history[roundIdx-1].Resolution
		if (selfRole == Player1 && lastRes.Winner == Player1) || (selfRole == Player2 && lastRes.Winner == Player2) {
			return b.counterBank[(roundIdx-1)%len(b.counterBank)]
		}
	}

	return b.phraseBank[roundIdx%len(b.phraseBank)]
}
