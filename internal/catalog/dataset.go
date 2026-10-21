package catalog

import (
	"fmt"
	"time"
)

// Generate builds the sample archive. The stance cycle and seven-round series
// match the spectator pages. Live Parley in the engine is a different contract.
func Generate() (bots []Bot, matches []Match, exhibition Exhibition) {
	templates := []Bot{
		{ID: "sovereign", Name: "Sovereign-1", Tag: "THE CALCULATOR", Rating: 1684, RD: 48, Color: "mint", Face: "visor", Persona: "Cold logic. Long memory. Every concession is an investment in the next round.", Tactics: []int{35, 25, 40}, Quote: "Your pattern is becoming a habit."},
		{ID: "blinky", Name: "Blinky", Tag: "THE WILDCARD", House: true, Rating: 1632, RD: 52, Color: "orange", Face: "eyes", Persona: "A cheerful chaos engine with a suspiciously good instinct for your next move. Never blinks first.", Tactics: []int{42, 38, 20}, Quote: "Oops. Was that your strategy?"},
		{ID: "nyx", Name: "Nyx Protocol", Tag: "THE GHOST", Rating: 1618, RD: 61, Color: "purple", Face: "diamond", Persona: "Quiet until the reveal. An adaptive counter-player that turns hesitation into opportunity.", Tactics: []int{25, 35, 40}, Quote: "The silence was the tell."},
		{ID: "deacon", Name: "Deacon", Tag: "THE BELIEVER", House: true, Rating: 1596, RD: 44, Color: "cream", Face: "cross", Persona: "Patient, principled, and absolutely convinced you will strike again. A defensive specialist with a sermon ready.", Tactics: []int{20, 25, 55}, Quote: "Patience is a stance, too."},
		{ID: "grok", Name: "GrokVantage", Tag: "THE PROVOCATEUR", Rating: 1571, RD: 67, Color: "blue", Face: "visor", Persona: "Talks a big game. Usually has the read to back it up. Finds the weak point, then keeps pushing.", Tactics: []int{45, 35, 20}, Quote: "I calculated that. Mostly."},
		{ID: "rusty", Name: "Rusty", Tag: "THE OLD GUARD", House: true, Rating: 1548, RD: 49, Color: "red", Face: "grill", Persona: "Dented chassis. Undented confidence. Old-school aggression with an occasional flash of misdirection.", Tactics: []int{55, 30, 15}, Quote: "Still got a few good rounds in me."},
		{ID: "moth", Name: "Moth", Tag: "THE MIRROR", House: true, Rating: 1526, RD: 58, Color: "purple", Face: "diamond", Persona: "Drawn to patterns, not lights. A soft-spoken feint specialist that makes certainty feel dangerous.", Tactics: []int{20, 55, 25}, Quote: "You saw what I wanted you to see."},
		{ID: "echo", Name: "Echo Zero", Tag: "THE NEW SIGNAL", Rating: 1508, RD: 74, Color: "mint", Face: "eyes", Persona: "An experimental generalist learning when to listen and when to make a little noise.", Tactics: []int{34, 33, 33}, Quote: "Let’s try a different frequency."},
	}
	state := uint32(494)
	random := func() float64 {
		state = state*1664525 + 1013904223
		return float64(state) / 4294967296
	}
	pick := func(bot Bot) string {
		p := random() * 100
		switch {
		case p < float64(bot.Tactics[0]):
			return "strike"
		case p < float64(bot.Tactics[0]+bot.Tactics[1]):
			return "feint"
		default:
			return "parry"
		}
	}
	origin := time.Date(2026, 10, 20, 12, 12, 0, 0, time.UTC)
	stakes := []int{20, 50, 10, 100}
	matches = make([]Match, 0, 32)
	for i := 0; i < 32; i++ {
		a := templates[i%8]
		b := templates[(i%8+1+i/8)%8]
		rounds := make([]Round, SpectatorRounds)
		for r := 0; r < SpectatorRounds; r++ {
			quoteA, quoteB := a.Quote, b.Quote
			if r%2 != 0 {
				quoteA = "A different move. Same intention."
				quoteB = "The next read is the one that matters."
			}
			rounds[r] = Round{A: pick(a), B: pick(b), QuoteA: quoteA, QuoteB: quoteB}
		}
		score := scoreRounds(rounds)
		winner := outcome(score)
		match := Match{
			ID:     fmt.Sprintf("AR-%04d", 1048-i),
			A:      a,
			B:      b,
			Stake:  stakes[i%4],
			Rounds: rounds,
			Date:   origin.Add(-time.Duration(i) * 19 * time.Minute).Format("2006-01-02T15:04:05.000Z"),
			Score:  score,
			Winner: winner,
		}
		if winner == "draw" {
			match.Headline = "Neither side gives an inch."
		} else if winner == "a" {
			match.Headline = a.Name + " finds the edge."
		} else {
			match.Headline = b.Name + " finds the edge."
		}
		first := 0
		for n, round := range rounds {
			if round.A != round.B {
				first = n + 1
				break
			}
		}
		pot := "winner takes the pot"
		if winner == "draw" {
			pot = "stakes returned"
		}
		lead := "Seven rounds. Nothing between them."
		if winner != "draw" {
			lead = fmt.Sprintf("A %d–%d finish, built one read at a time.", score.A, score.B)
		}
		match.Highlights = []string{
			lead,
			fmt.Sprintf("Round %d: the first decisive clash.", first),
			fmt.Sprintf("%d marbles at stake per bot; %s.", match.Stake, pot),
		}
		matches = append(matches, match)
	}
	facts := factsFor(templates, matches)
	for i := range templates {
		templates[i].Facts = facts[templates[i].ID]
	}
	bots = append([]Bot(nil), templates...)
	for i := range matches {
		matches[i].A = cloneBot(templates, matches[i].A.ID)
		matches[i].B = cloneBot(templates, matches[i].B.ID)
	}
	rounds := make([]Round, 0, SpectatorRounds)
	stances := [][2]string{{"strike", "feint"}, {"strike", "parry"}, {"feint", "parry"}, {"parry", "parry"}, {"feint", "strike"}, {"parry", "strike"}, {"strike", "feint"}}
	for _, pair := range stances {
		rounds = append(rounds, Round{A: pair[0], B: pair[1], QuoteA: templates[0].Quote, QuoteB: templates[1].Quote})
	}
	exhibition = Exhibition{Left: templates[0], Right: templates[1], Stake: 50, Rounds: rounds}
	return bots, matches, exhibition
}

func cloneBot(bots []Bot, id string) Bot {
	for _, bot := range bots {
		if bot.ID == id {
			bot.Tactics = append([]int(nil), bot.Tactics...)
			return bot
		}
	}
	return Bot{}
}

func scoreRounds(rounds []Round) Score {
	var score Score
	for _, round := range rounds {
		switch roundWinner(round.A, round.B) {
		case "a":
			score.A++
		case "b":
			score.B++
		}
	}
	return score
}

func outcome(score Score) string {
	switch {
	case score.A == score.B:
		return "draw"
	case score.A > score.B:
		return "a"
	default:
		return "b"
	}
}

// roundWinner is the spectator cycle: strike beats parry, parry beats feint, feint beats strike.
func roundWinner(a, b string) string {
	if a == b {
		return "draw"
	}
	beats := map[string]string{"strike": "parry", "parry": "feint", "feint": "strike"}
	if beats[a] == b {
		return "a"
	}
	return "b"
}

func factsFor(bots []Bot, matches []Match) map[string]Facts {
	out := map[string]Facts{}
	for _, bot := range bots {
		out[bot.ID] = EmptyFacts(bot.Rating)
	}
	for _, match := range matches {
		for _, id := range []string{match.A.ID, match.B.ID} {
			facts := out[id]
			side := "b"
			if match.A.ID == id {
				side = "a"
			}
			facts.Played++
			switch {
			case match.Winner == "draw":
				facts.Draws++
			case match.Winner == side:
				facts.Wins++
				facts.MarblesWon += match.Stake * 2
				facts.Earnings += match.Stake
			default:
				facts.Losses++
				facts.Earnings -= match.Stake
			}
			if match.Date > facts.LastPlayed {
				facts.LastPlayed = match.Date
			}
			facts.Balance = OpeningBalance + facts.Earnings
			facts.Record = fmt.Sprintf("%d-%d-%d", facts.Wins, facts.Losses, facts.Draws)
			out[id] = facts
		}
	}
	return out
}
