package catalog

import (
	"testing"
)

func TestGenerateMatchesSpectatorArchive(t *testing.T) {
	bots, matches, exhibition := Generate()
	if len(bots) != 8 || len(matches) != 32 {
		t.Fatalf("bots=%d matches=%d", len(bots), len(matches))
	}
	first := matches[0]
	if first.ID != "AR-1048" || first.A.ID != "sovereign" || first.B.ID != "blinky" || first.Stake != 20 || first.Winner != "a" || first.Date != "2026-10-20T12:12:00.000Z" {
		t.Fatalf("first match %+v", first)
	}
	if first.Score != (Score{2, 1}) || first.Rounds[0].A != "feint" || first.Rounds[0].B != "strike" {
		t.Fatalf("first round %+v score %+v", first.Rounds[0], first.Score)
	}
	if first.Headline != "Sovereign-1 finds the edge." || first.Highlights[0] != "A 2–1 finish, built one read at a time." {
		t.Fatalf("headline %q highlights %#v", first.Headline, first.Highlights)
	}
	seen := map[string]bool{}
	draws := 0
	earnings := 0
	for _, match := range matches {
		if seen[match.ID] || len(match.Rounds) != SpectatorRounds {
			t.Fatalf("match %s rounds %d", match.ID, len(match.Rounds))
		}
		seen[match.ID] = true
		if match.Winner == "draw" {
			draws++
		}
	}
	if draws != 2 {
		t.Fatalf("draws %d", draws)
	}
	want := map[string]Facts{
		"sovereign": {Wins: 3, Losses: 4, Draws: 1, Earnings: -120, Played: 8},
		"echo":      {Wins: 2, Losses: 5, Draws: 1, Earnings: -80, Played: 8},
	}
	for _, bot := range bots {
		if sum := bot.Tactics[0] + bot.Tactics[1] + bot.Tactics[2]; sum != 100 {
			t.Fatalf("%s tactics %d", bot.ID, sum)
		}
		earnings += bot.Facts.Earnings
		if bot.Facts.Balance != OpeningBalance+bot.Facts.Earnings || bot.Facts.Record == "" || bot.Facts.Tier == "" {
			t.Fatalf("facts %+v", bot.Facts)
		}
		if expected, ok := want[bot.ID]; ok {
			if bot.Facts.Wins != expected.Wins || bot.Facts.Losses != expected.Losses || bot.Facts.Draws != expected.Draws || bot.Facts.Earnings != expected.Earnings || bot.Facts.Played != expected.Played {
				t.Fatalf("%s facts %+v", bot.ID, bot.Facts)
			}
		}
	}
	if earnings != 0 {
		t.Fatalf("earnings %d", earnings)
	}
	if exhibition.Left.ID != "sovereign" || exhibition.Right.ID != "blinky" || exhibition.Stake != 50 || len(exhibition.Rounds) != 7 {
		t.Fatalf("exhibition %+v", exhibition)
	}
}

func TestMemoryListSearchPageAndCreate(t *testing.T) {
	store := NewMemory()
	ctx := t.Context()
	page, err := store.ListMatches(ctx, MatchQuery{Limit: 6})
	if err != nil || page.Total != 32 || page.Page != 1 || len(page.Items.([]Match)) != 6 {
		t.Fatalf("page %+v %v", page, err)
	}
	next, err := store.ListMatches(ctx, MatchQuery{Limit: 6, Offset: 6})
	if err != nil || next.Page != 2 || next.Items.([]Match)[0].ID == page.Items.([]Match)[0].ID {
		t.Fatal(next)
	}
	draws, err := store.ListMatches(ctx, MatchQuery{Outcome: "draw", Limit: 50})
	if err != nil || draws.Total != 2 {
		t.Fatalf("draws %+v %v", draws, err)
	}
	found, err := store.ListMatches(ctx, MatchQuery{Q: "Sovereign-1", Stake: 20, Limit: 50})
	if err != nil || found.Total == 0 {
		t.Fatal(found, err)
	}
	bot, err := store.GetBot(ctx, "moth")
	if err != nil || !bot.House || bot.Facts.Played != 8 {
		t.Fatalf("%+v %v", bot, err)
	}
	if _, err := store.GetMatch(ctx, "missing"); err != ErrNotFound {
		t.Fatal(err)
	}
	created, err := store.CreateBot(ctx, CreateBot{Name: "New Contender", Email: "owner@example.com", Persona: "Reads the room and counters."})
	if err != nil || created.Facts.Record != "0-0-0" || created.Facts.Balance != 1000 {
		t.Fatalf("%+v %v", created, err)
	}
	if _, err := store.CreateBot(ctx, CreateBot{Name: "new contender", Email: "other@example.com", Persona: "Reads the room and counters."}); err == nil {
		t.Fatal("duplicate name accepted")
	}
	house, err := store.ListBots(ctx, BotQuery{Role: "house", Limit: 10})
	if err != nil || house.Total != 4 {
		t.Fatalf("%+v %v", house, err)
	}
	named, err := store.ListBots(ctx, BotQuery{Q: "mirror", Limit: 10})
	if err != nil || named.Total != 1 || named.Items.([]Bot)[0].ID != "moth" {
		t.Fatalf("%+v %v", named, err)
	}
}
