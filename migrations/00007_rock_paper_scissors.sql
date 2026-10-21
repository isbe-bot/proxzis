-- +goose Up
-- Parley's moves are Rock, Paper and Scissors. The earlier stance names map one
-- to one (strike = rock, feint = scissors, parry = paper), so every stored
-- result, score and settlement stays exactly as it was.
ALTER TABLE spectator_bots RENAME COLUMN tactic_strike TO tactic_rock;
ALTER TABLE spectator_bots RENAME COLUMN tactic_parry TO tactic_paper;
ALTER TABLE spectator_bots RENAME COLUMN tactic_feint TO tactic_scissors;
ALTER TABLE bots RENAME COLUMN tactic_strike TO tactic_rock;
ALTER TABLE bots RENAME COLUMN tactic_parry TO tactic_paper;
ALTER TABLE bots RENAME COLUMN tactic_feint TO tactic_scissors;

UPDATE spectator_rounds SET
    p1_stance = CASE p1_stance WHEN 'strike' THEN 'rock' WHEN 'parry' THEN 'paper' WHEN 'feint' THEN 'scissors' ELSE p1_stance END,
    p2_stance = CASE p2_stance WHEN 'strike' THEN 'rock' WHEN 'parry' THEN 'paper' WHEN 'feint' THEN 'scissors' ELSE p2_stance END;
UPDATE match_rounds SET
    p1_stance = CASE p1_stance WHEN 'strike' THEN 'rock' WHEN 'parry' THEN 'paper' WHEN 'feint' THEN 'scissors' ELSE p1_stance END,
    p2_stance = CASE p2_stance WHEN 'strike' THEN 'rock' WHEN 'parry' THEN 'paper' WHEN 'feint' THEN 'scissors' ELSE p2_stance END;
UPDATE matches SET modifier_card = CASE modifier_card
    WHEN 'FeintSurge' THEN 'ScissorsSurge' WHEN 'ParryRebound' THEN 'PaperRebound' ELSE modifier_card END;
UPDATE spectator_exhibition SET rounds =
    replace(replace(replace(rounds, '"strike"', '"rock"'), '"parry"', '"paper"'), '"feint"', '"scissors"');

-- +goose Down
UPDATE spectator_exhibition SET rounds =
    replace(replace(replace(rounds, '"rock"', '"strike"'), '"paper"', '"parry"'), '"scissors"', '"feint"');
UPDATE matches SET modifier_card = CASE modifier_card
    WHEN 'ScissorsSurge' THEN 'FeintSurge' WHEN 'PaperRebound' THEN 'ParryRebound' ELSE modifier_card END;
UPDATE match_rounds SET
    p1_stance = CASE p1_stance WHEN 'rock' THEN 'strike' WHEN 'paper' THEN 'parry' WHEN 'scissors' THEN 'feint' ELSE p1_stance END,
    p2_stance = CASE p2_stance WHEN 'rock' THEN 'strike' WHEN 'paper' THEN 'parry' WHEN 'scissors' THEN 'feint' ELSE p2_stance END;
UPDATE spectator_rounds SET
    p1_stance = CASE p1_stance WHEN 'rock' THEN 'strike' WHEN 'paper' THEN 'parry' WHEN 'scissors' THEN 'feint' ELSE p1_stance END,
    p2_stance = CASE p2_stance WHEN 'rock' THEN 'strike' WHEN 'paper' THEN 'parry' WHEN 'scissors' THEN 'feint' ELSE p2_stance END;
ALTER TABLE bots RENAME COLUMN tactic_scissors TO tactic_feint;
ALTER TABLE bots RENAME COLUMN tactic_paper TO tactic_parry;
ALTER TABLE bots RENAME COLUMN tactic_rock TO tactic_strike;
ALTER TABLE spectator_bots RENAME COLUMN tactic_scissors TO tactic_feint;
ALTER TABLE spectator_bots RENAME COLUMN tactic_paper TO tactic_parry;
ALTER TABLE spectator_bots RENAME COLUMN tactic_rock TO tactic_strike;
