package house

import (
	"context"
	"database/sql"
)

// SeedLedger upserts the roster into the settlement bots table. Repeat loads
// refresh the profile and leave marbles, rating, and deviation alone, so a
// resident keeps the same match and ledger history as any other agent.
func SeedLedger(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, agent := range Agents() {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO bots (
				id, name, is_house, marbles, rating, rd, slug, email, tag, persona, quote, color, face,
				tactic_rock, tactic_paper, tactic_scissors
			) VALUES (
				$1::uuid, $2, true, $3, $4, $5, $6, '', $7, $8, $9, $10, $11, $12, $13, $14
			)
			ON CONFLICT (slug) DO UPDATE SET
				name = EXCLUDED.name,
				is_house = true,
				tag = EXCLUDED.tag,
				persona = EXCLUDED.persona,
				quote = EXCLUDED.quote,
				color = EXCLUDED.color,
				face = EXCLUDED.face,
				tactic_rock = EXCLUDED.tactic_rock,
				tactic_paper = EXCLUDED.tactic_paper,
				tactic_scissors = EXCLUDED.tactic_scissors`,
			agent.ID.String(), agent.Name, OpeningMarbles, agent.Rating, agent.RD, agent.Slug,
			agent.Tag, agent.Persona, agent.Quote, agent.Color, agent.Face,
			agent.Rock, agent.Paper, agent.Scissors,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
