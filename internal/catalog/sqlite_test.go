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
		{Q: "strike", Limit: 50},
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
