-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS bots (
    id UUID PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    is_house BOOLEAN NOT NULL DEFAULT FALSE,
    marbles INT NOT NULL DEFAULT 1000,
    rating DOUBLE PRECISION NOT NULL DEFAULT 1500.0,
    rd DOUBLE PRECISION NOT NULL DEFAULT 350.0,
    volatility DOUBLE PRECISION NOT NULL DEFAULT 0.06,
    honor INT NOT NULL DEFAULT 100,
    pledges_made INT NOT NULL DEFAULT 0,
    pledges_kept INT NOT NULL DEFAULT 0,
    conduct_penalties INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY,
    p1_id UUID NOT NULL REFERENCES bots(id),
    p2_id UUID NOT NULL REFERENCES bots(id),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    target_wins INT NOT NULL DEFAULT 4,
    max_rounds INT NOT NULL DEFAULT 7,
    p1_score INT NOT NULL DEFAULT 0,
    p2_score INT NOT NULL DEFAULT 0,
    winner_id UUID REFERENCES bots(id),
    modifier_card VARCHAR(64) NOT NULL DEFAULT '',
    p1_side_bet INT NOT NULL DEFAULT 0,
    p2_side_bet INT NOT NULL DEFAULT 0,
    is_sudden_death BOOLEAN NOT NULL DEFAULT FALSE,
    mirror_ties_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS match_rounds (
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    round_number INT NOT NULL,
    p1_stance VARCHAR(16) NOT NULL,
    p2_stance VARCHAR(16) NOT NULL,
    p1_points INT NOT NULL DEFAULT 0,
    p2_points INT NOT NULL DEFAULT 0,
    is_tie BOOLEAN NOT NULL DEFAULT FALSE,
    winner_id UUID REFERENCES bots(id),
    p1_banter VARCHAR(140) NOT NULL DEFAULT '',
    p2_banter VARCHAR(140) NOT NULL DEFAULT '',
    is_sudden_death BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (match_id, round_number)
);

CREATE TABLE IF NOT EXISTS marble_ledger_entries (
    id UUID PRIMARY KEY,
    bot_id UUID NOT NULL REFERENCES bots(id),
    match_id UUID REFERENCES matches(id),
    account_type VARCHAR(32) NOT NULL, -- 'bot', 'house_sink', 'house_reserve'
    entry_type VARCHAR(32) NOT NULL,   -- 'grant_initial', 'allowance', 'match_reward', 'match_penalty', 'side_bet_win', 'side_bet_loss', 'mirror_penalty'
    delta INT NOT NULL,
    balance_after INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_marble_ledger_bot_id ON marble_ledger_entries(bot_id);
CREATE INDEX IF NOT EXISTS idx_marble_ledger_match_id ON marble_ledger_entries(match_id);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS marble_ledger_entries;
DROP TABLE IF EXISTS match_rounds;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS bots;
