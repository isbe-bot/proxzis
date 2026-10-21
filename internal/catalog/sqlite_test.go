package catalog

import (
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSQLiteListSearchPageAndCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.db")
	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.EnsureSchema(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := store.Seed(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertArchiveReads(t, store)

	match, err := store.GetMatch(t.Context(), "AR-1048")
	if err != nil || match.Date != "2026-10-20T12:12:00.000Z" || match.Rounds[0].QuoteA == "" || len(match.Rounds) != 7 {
		t.Fatalf("%+v %v", match, err)
	}
	if err := store.SeedIfEmpty(t.Context()); err != nil {
		t.Fatal(err)
	}
	page, err := store.ListMatches(t.Context(), MatchQuery{Limit: 1})
	if err != nil || page.Total != 32 {
		t.Fatalf("%+v %v", page, err)
	}
	var ledger string
	err = store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='bots'`).Scan(&ledger)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("spectator file created a ledger bots table: %v %s", err, ledger)
	}
}

func TestSQLiteAgreesWithMemory(t *testing.T) {
	mem := NewMemory()
	path := filepath.Join(t.TempDir(), "arena.db")
	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.EnsureSchema(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := store.Seed(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	for _, q := range []MatchQuery{
		{Limit: 6},
		{Q: "rock", Limit: 50},
		{Outcome: "draw", Limit: 50},
		{Q: "habit", Outcome: "contender", Stake: 20, Limit: 50},
		{Bot: "blinky", Limit: 50},
	} {
		left, err := mem.ListMatches(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		right, err := store.ListMatches(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		if left.Total != right.Total || !reflect.DeepEqual(matchIDs(left), matchIDs(right)) {
			t.Fatalf("query %+v memory %v sqlite %v", q, matchIDs(left), matchIDs(right))
		}
	}
	for _, q := range []BotQuery{{Role: "house", Limit: 10}, {Role: "contender", Limit: 10}, {Q: "mirror", Limit: 10}} {
		left, err := mem.ListBots(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		right, err := store.ListBots(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		if left.Total != right.Total || !reflect.DeepEqual(botIDs(left), botIDs(right)) {
			t.Fatalf("bots %+v memory %v sqlite %v", q, botIDs(left), botIDs(right))
		}
	}
	left, err := mem.GetBot(ctx, "sovereign")
	if err != nil {
		t.Fatal(err)
	}
	right, err := store.GetBot(ctx, "sovereign")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left.Facts, right.Facts) {
		t.Fatalf("memory %+v sqlite %+v", left.Facts, right.Facts)
	}
}

func TestSQLiteUpgradesLegacyStanceNames(t *testing.T) {
	store, err := OpenSQLite(filepath.Join(t.TempDir(), "arena.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := t.Context()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	wantMatch, err := store.GetMatch(ctx, "AR-1048")
	if err != nil {
		t.Fatal(err)
	}
	wantBot, err := store.GetBot(ctx, "blinky")
	if err != nil {
		t.Fatal(err)
	}
	wantShow, err := store.Exhibition(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Rewind the file to the pre-00007 shape, then reopen it through EnsureSchema.
	for _, statement := range []string{
		`ALTER TABLE spectator_bots RENAME COLUMN tactic_rock TO tactic_strike`,
		`ALTER TABLE spectator_bots RENAME COLUMN tactic_paper TO tactic_parry`,
		`ALTER TABLE spectator_bots RENAME COLUMN tactic_scissors TO tactic_feint`,
		`UPDATE spectator_rounds SET
			p1_stance = CASE p1_stance WHEN 'rock' THEN 'strike' WHEN 'paper' THEN 'parry' ELSE 'feint' END,
			p2_stance = CASE p2_stance WHEN 'rock' THEN 'strike' WHEN 'paper' THEN 'parry' ELSE 'feint' END`,
		`UPDATE spectator_exhibition SET rounds =
			replace(replace(replace(rounds, '"rock"', '"strike"'), '"paper"', '"parry"'), '"scissors"', '"feint"')`,
	} {
		if _, err := store.db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("upgrade is not idempotent: %v", err)
	}
	gotMatch, err := store.GetMatch(ctx, "AR-1048")
	if err != nil || !reflect.DeepEqual(gotMatch, wantMatch) {
		t.Fatalf("match after upgrade %+v %v, want %+v", gotMatch, err, wantMatch)
	}
	gotBot, err := store.GetBot(ctx, "blinky")
	if err != nil || !reflect.DeepEqual(gotBot.Tactics, wantBot.Tactics) {
		t.Fatalf("tactics after upgrade %v %v, want %v", gotBot.Tactics, err, wantBot.Tactics)
	}
	gotShow, err := store.Exhibition(ctx)
	if err != nil || !reflect.DeepEqual(gotShow.Rounds, wantShow.Rounds) {
		t.Fatalf("exhibition after upgrade %+v %v, want %+v", gotShow.Rounds, err, wantShow.Rounds)
	}
}

func TestHouseSeedKeepsRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arena.db")
	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := t.Context()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO spectator_bots (id, slug, name) VALUES ('existing', 'existing', 'Existing')`); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedIfEmpty(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE spectator_bots SET rating = 1700 WHERE slug = 'nuggs'`); err != nil {
		t.Fatal(err)
	}
	if err := store.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	resident, err := store.GetBot(ctx, "nuggs")
	if err != nil || !resident.House || resident.Rating != 1700 || resident.Tactics[0] != 50 || resident.Tactics[1] != 25 || resident.Tactics[2] != 25 {
		t.Fatalf("%+v %v", resident, err)
	}
	if _, err := store.GetBot(ctx, "existing"); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateCreate(CreateBot{Name: "Nuggs", Email: "owner@example.com", Persona: "Reads the room and counters."}); err == nil {
		t.Fatal("house name was accepted")
	}
}

func matchIDs(page Page) []string {
	items := page.Items.([]Match)
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}

func botIDs(page Page) []string {
	items := page.Items.([]Bot)
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}
