-- +goose Up
-- Completion can be persisted before financial settlement. Keep a separate marker.
ALTER TABLE matches ADD COLUMN settled_at TIMESTAMPTZ;

-- House accounts are bookkeeping counterparts, not playable house bots. The
-- reserve may be negative: the game has always issued rewards without prefunding.
CREATE TABLE settlement_house_accounts (
    account_type VARCHAR(32) PRIMARY KEY CHECK (account_type IN ('house_sink', 'house_reserve')),
    balance BIGINT NOT NULL DEFAULT 0
);
INSERT INTO settlement_house_accounts (account_type) VALUES ('house_reserve'), ('house_sink');

ALTER TABLE marble_ledger_entries ALTER COLUMN bot_id DROP NOT NULL;
ALTER TABLE marble_ledger_entries ALTER COLUMN balance_after TYPE BIGINT;
-- Existing historical entries are deliberately not rewritten or backfilled.
ALTER TABLE marble_ledger_entries ADD CONSTRAINT settlement_ledger_account
    CHECK ((account_type = 'bot' AND bot_id IS NOT NULL)
        OR (account_type IN ('house_sink', 'house_reserve') AND bot_id IS NULL)) NOT VALID;

-- +goose Down
-- Downgrade would lose house accounting and settlement replay protection.
-- +goose StatementBegin
DO $$ BEGIN
    RAISE EXCEPTION '00004 is irreversible: retain financial accounting and settlement markers';
END $$;
-- +goose StatementEnd
