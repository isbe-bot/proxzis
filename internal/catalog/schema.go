package catalog

// spectatorDDL is the archive schema for SQLite. Postgres applies the same
// tables from migrations/00006_spectator_archive.sql. These statements must
// stay aligned with that migration: text ids, integer flags, text timestamps,
// and text JSON, with no ledger columns.
var spectatorDDL = []string{
	`CREATE TABLE IF NOT EXISTS spectator_bots (
		id TEXT PRIMARY KEY,
		slug TEXT NOT NULL,
		name TEXT NOT NULL,
		email TEXT NOT NULL DEFAULT '',
		is_house INTEGER NOT NULL DEFAULT 0,
		rating REAL NOT NULL DEFAULT 1500,
		rd REAL NOT NULL DEFAULT 350,
		tag TEXT NOT NULL DEFAULT '',
		persona TEXT NOT NULL DEFAULT '',
		quote TEXT NOT NULL DEFAULT '',
		color TEXT NOT NULL DEFAULT 'mint',
		face TEXT NOT NULL DEFAULT 'visor',
		tactic_strike INTEGER NOT NULL DEFAULT 34,
		tactic_feint INTEGER NOT NULL DEFAULT 33,
		tactic_parry INTEGER NOT NULL DEFAULT 33,
		created_at TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS spectator_bots_slug_key ON spectator_bots (slug)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS spectator_bots_name_key ON spectator_bots (lower(name))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS spectator_bots_email_key ON spectator_bots (lower(email)) WHERE email <> ''`,
	`CREATE TABLE IF NOT EXISTS spectator_matches (
		id TEXT PRIMARY KEY,
		public_id TEXT NOT NULL,
		p1_id TEXT NOT NULL REFERENCES spectator_bots(id),
		p2_id TEXT NOT NULL REFERENCES spectator_bots(id),
		stake INTEGER NOT NULL DEFAULT 0,
		p1_score INTEGER NOT NULL DEFAULT 0,
		p2_score INTEGER NOT NULL DEFAULT 0,
		winner_id TEXT REFERENCES spectator_bots(id),
		headline TEXT NOT NULL DEFAULT '',
		highlights TEXT NOT NULL DEFAULT '[]',
		created_at TEXT NOT NULL
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS spectator_matches_public_id_key ON spectator_matches (public_id)`,
	`CREATE INDEX IF NOT EXISTS spectator_matches_p1_idx ON spectator_matches (p1_id)`,
	`CREATE INDEX IF NOT EXISTS spectator_matches_p2_idx ON spectator_matches (p2_id)`,
	`CREATE TABLE IF NOT EXISTS spectator_rounds (
		match_id TEXT NOT NULL REFERENCES spectator_matches(id) ON DELETE CASCADE,
		round_number INTEGER NOT NULL,
		p1_stance TEXT NOT NULL,
		p2_stance TEXT NOT NULL,
		p1_banter TEXT NOT NULL DEFAULT '',
		p2_banter TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (match_id, round_number)
	)`,
	`CREATE TABLE IF NOT EXISTS spectator_exhibition (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		left_slug TEXT NOT NULL,
		right_slug TEXT NOT NULL,
		stake INTEGER NOT NULL,
		rounds TEXT NOT NULL
	)`,
}
