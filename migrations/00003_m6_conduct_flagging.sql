-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS moderation_reports (
    id UUID PRIMARY KEY,
    reporter_id UUID NOT NULL REFERENCES bots(id),
    accused_id UUID NOT NULL REFERENCES bots(id),
    match_id UUID REFERENCES matches(id),
    reason TEXT NOT NULL,
    banter_sample TEXT NOT NULL DEFAULT '',
    action_taken VARCHAR(32) NOT NULL DEFAULT 'flagged', -- 'flagged', 'withheld', 'honor_deducted', 'sent_home'
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE IF NOT EXISTS moderation_events (
    id BIGSERIAL PRIMARY KEY,
    bot_id UUID NOT NULL REFERENCES bots(id),
    match_id UUID REFERENCES matches(id),
    text TEXT NOT NULL,
    verdict VARCHAR(32) NOT NULL, -- 'approved', 'withheld'
    reasons TEXT[] NOT NULL DEFAULT '{}',
    at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE IF NOT EXISTS sent_home (
    bot_id UUID PRIMARY KEY REFERENCES bots(id),
    until TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_moderation_reports_accused ON moderation_reports(accused_id);
CREATE INDEX IF NOT EXISTS idx_moderation_events_bot_id ON moderation_events(bot_id);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS sent_home;
DROP TABLE IF EXISTS moderation_events;
DROP TABLE IF EXISTS moderation_reports;
