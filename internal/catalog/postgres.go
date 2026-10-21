package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres serves the spectator archive from the migrated database.
type Postgres struct {
	db *sql.DB
}

func OpenPostgres(db *sql.DB) *Postgres {
	db.SetMaxOpenConns(8)
	return &Postgres{db: db}
}

func (p *Postgres) SeedIfEmpty(ctx context.Context) error {
	var n int
	if err := p.db.QueryRowContext(ctx, `SELECT count(*) FROM bots WHERE slug IS NOT NULL`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return p.Seed(ctx)
	}
	return nil
}

func (p *Postgres) Seed(ctx context.Context) error {
	bots, matches, exhibition := Generate()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, bot := range bots {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO bots (id, name, is_house, marbles, rating, rd, slug, email, tag, persona, quote, color, face, tactic_strike, tactic_feint, tactic_parry)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'',$8,$9,$10,$11,$12,$13,$14,$15)
			ON CONFLICT (slug) DO UPDATE SET
				name = EXCLUDED.name, is_house = EXCLUDED.is_house, marbles = EXCLUDED.marbles,
				rating = EXCLUDED.rating, rd = EXCLUDED.rd, tag = EXCLUDED.tag, persona = EXCLUDED.persona,
				quote = EXCLUDED.quote, color = EXCLUDED.color, face = EXCLUDED.face,
				tactic_strike = EXCLUDED.tactic_strike, tactic_feint = EXCLUDED.tactic_feint, tactic_parry = EXCLUDED.tactic_parry,
				updated_at = clock_timestamp()`,
			BotUUID(bot.ID), bot.Name, bot.House, bot.Facts.Balance, bot.Rating, bot.RD, bot.ID,
			bot.Tag, bot.Persona, bot.Quote, bot.Color, bot.Face, bot.Tactics[0], bot.Tactics[1], bot.Tactics[2],
		); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM match_rounds WHERE match_id IN (SELECT id FROM matches WHERE public_id IS NOT NULL)`); err != nil {
		return err
	}
	for _, match := range matches {
		when, err := time.Parse(time.RFC3339Nano, match.Date)
		if err != nil {
			return err
		}
		highlights, err := json.Marshal(match.Highlights)
		if err != nil {
			return err
		}
		var winner any
		switch match.Winner {
		case "a":
			winner = BotUUID(match.A.ID)
		case "b":
			winner = BotUUID(match.B.ID)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO matches (id, p1_id, p2_id, status, target_wins, max_rounds, p1_score, p2_score, winner_id, public_id, stake, headline, highlights, created_at, completed_at)
			VALUES ($1,$2,$3,'completed',4,7,$4,$5,$6,$7,$8,$9,$10,$11,$11)
			ON CONFLICT (public_id) DO UPDATE SET
				p1_id = EXCLUDED.p1_id, p2_id = EXCLUDED.p2_id, p1_score = EXCLUDED.p1_score, p2_score = EXCLUDED.p2_score,
				winner_id = EXCLUDED.winner_id, stake = EXCLUDED.stake, headline = EXCLUDED.headline, highlights = EXCLUDED.highlights,
				created_at = EXCLUDED.created_at, completed_at = EXCLUDED.completed_at, updated_at = clock_timestamp()`,
			MatchUUID(match.ID), BotUUID(match.A.ID), BotUUID(match.B.ID), match.Score.A, match.Score.B, winner,
			match.ID, match.Stake, match.Headline, highlights, when,
		); err != nil {
			return err
		}
		for n, round := range match.Rounds {
			side := roundWinner(round.A, round.B)
			var roundWinnerID any
			p1, p2 := 0, 0
			switch side {
			case "a":
				p1 = 1
				roundWinnerID = BotUUID(match.A.ID)
			case "b":
				p2 = 1
				roundWinnerID = BotUUID(match.B.ID)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO match_rounds (match_id, round_number, p1_stance, p2_stance, p1_points, p2_points, is_tie, winner_id, p1_banter, p2_banter)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
				MatchUUID(match.ID), n+1, round.A, round.B, p1, p2, side == "draw", roundWinnerID, round.QuoteA, round.QuoteB,
			); err != nil {
				return err
			}
		}
	}
	payload, err := json.Marshal(exhibition.Rounds)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO spectator_exhibition (id, left_slug, right_slug, stake, rounds)
		VALUES (1, $1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET left_slug = EXCLUDED.left_slug, right_slug = EXCLUDED.right_slug, stake = EXCLUDED.stake, rounds = EXCLUDED.rounds`,
		exhibition.Left.ID, exhibition.Right.ID, exhibition.Stake, payload,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *Postgres) ListBots(ctx context.Context, q BotQuery) (Page, error) {
	limit, offset := ClampPage(q.Limit, q.Offset)
	bots, err := p.loadBots(ctx)
	if err != nil {
		return Page{}, err
	}
	var matched []Bot
	for _, bot := range bots {
		if q.Role == "house" && !bot.House || q.Role == "contender" && bot.House {
			continue
		}
		if q.Q != "" && !containsFold(bot.Name+" "+bot.Tag+" "+bot.Persona+" "+bot.Quote, q.Q) {
			continue
		}
		matched = append(matched, bot)
	}
	return PageOf(sliceBots(matched, offset, limit), len(matched), limit, offset), nil
}

func (p *Postgres) GetBot(ctx context.Context, id string) (Bot, error) {
	bots, err := p.loadBots(ctx)
	if err != nil {
		return Bot{}, err
	}
	for _, bot := range bots {
		if bot.ID == id {
			return bot, nil
		}
	}
	return Bot{}, ErrNotFound
}

func (p *Postgres) ListMatches(ctx context.Context, q MatchQuery) (Page, error) {
	limit, offset := ClampPage(q.Limit, q.Offset)
	where, args := matchFilter(q)
	var total int
	if err := p.db.QueryRowContext(ctx, `SELECT count(*) FROM matches m
		JOIN bots p1 ON p1.id = m.p1_id JOIN bots p2 ON p2.id = m.p2_id
		LEFT JOIN bots w ON w.id = m.winner_id `+where, args...).Scan(&total); err != nil {
		return Page{}, err
	}
	args = append(args, limit, offset)
	rows, err := p.db.QueryContext(ctx, `SELECT m.public_id FROM matches m
		JOIN bots p1 ON p1.id = m.p1_id JOIN bots p2 ON p2.id = m.p2_id
		LEFT JOIN bots w ON w.id = m.winner_id `+where+fmt.Sprintf(` ORDER BY m.created_at DESC, m.public_id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return Page{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	matches, err := p.matchesByID(ctx, ids)
	if err != nil {
		return Page{}, err
	}
	return PageOf(matches, total, limit, offset), nil
}

func (p *Postgres) GetMatch(ctx context.Context, id string) (Match, error) {
	matches, err := p.matchesByID(ctx, []string{id})
	if err != nil {
		return Match{}, err
	}
	if len(matches) == 0 {
		return Match{}, ErrNotFound
	}
	return matches[0], nil
}

func (p *Postgres) Exhibition(ctx context.Context) (Exhibition, error) {
	var left, right string
	var stake int
	var raw []byte
	err := p.db.QueryRowContext(ctx, `SELECT left_slug, right_slug, stake, rounds FROM spectator_exhibition WHERE id = 1`).Scan(&left, &right, &stake, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Exhibition{}, ErrNotFound
	}
	if err != nil {
		return Exhibition{}, err
	}
	var rounds []Round
	if err := json.Unmarshal(raw, &rounds); err != nil {
		return Exhibition{}, err
	}
	bots, err := p.loadBots(ctx)
	if err != nil {
		return Exhibition{}, err
	}
	exhibition := Exhibition{Stake: stake, Rounds: rounds}
	for _, bot := range bots {
		if bot.ID == left {
			exhibition.Left = bot
		}
		if bot.ID == right {
			exhibition.Right = bot
		}
	}
	if exhibition.Left.ID == "" || exhibition.Right.ID == "" {
		return Exhibition{}, ErrNotFound
	}
	return exhibition, nil
}

func (p *Postgres) CreateBot(ctx context.Context, in CreateBot) (Bot, error) {
	existing, err := p.loadBots(ctx)
	if err != nil {
		return Bot{}, err
	}
	clean, errs := ValidateCreate(in, existing)
	if len(errs) > 0 {
		return Bot{}, errs
	}
	bot := Bot{
		ID: NewBotSlug(clean.Name), Name: clean.Name, Email: clean.Email, Tag: "NEW CONTENDER",
		Rating: 1500, RD: 350, Color: "mint", Face: "visor", Persona: clean.Persona,
		Tactics: []int{34, 33, 33}, Quote: QuoteFromPersona(clean.Persona), Facts: EmptyFacts(1500),
	}
	_, err = p.db.ExecContext(ctx, `
		INSERT INTO bots (id, name, is_house, marbles, rating, rd, slug, email, tag, persona, quote, color, face, tactic_strike, tactic_feint, tactic_parry)
		VALUES ($1,$2,false,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		BotUUID(bot.ID), bot.Name, bot.Facts.Balance, bot.Rating, bot.RD, bot.ID, bot.Email, bot.Tag, bot.Persona, bot.Quote, bot.Color, bot.Face, bot.Tactics[0], bot.Tactics[1], bot.Tactics[2],
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "email") {
				return Bot{}, FieldErrors{"email": "This email is already associated with an active bot."}
			}
			return Bot{}, FieldErrors{"name": "A bot with this name is already registered."}
		}
		return Bot{}, err
	}
	return bot, nil
}

func (p *Postgres) loadBots(ctx context.Context) ([]Bot, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT slug, name, email, tag, is_house, rating, rd, color, face, persona, quote, tactic_strike, tactic_feint, tactic_parry
		FROM bots WHERE slug IS NOT NULL ORDER BY rating DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []Bot
	for rows.Next() {
		var bot Bot
		var rating, rd float64
		var strike, feint, parry int
		if err := rows.Scan(&bot.ID, &bot.Name, &bot.Email, &bot.Tag, &bot.House, &rating, &rd, &bot.Color, &bot.Face, &bot.Persona, &bot.Quote, &strike, &feint, &parry); err != nil {
			return nil, err
		}
		bot.Rating = int(rating)
		bot.RD = int(rd)
		bot.Tactics = []int{strike, feint, parry}
		bot.Facts = EmptyFacts(bot.Rating)
		bots = append(bots, bot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	facts, err := p.loadFacts(ctx, bots)
	if err != nil {
		return nil, err
	}
	for i := range bots {
		if got, ok := facts[bots[i].ID]; ok {
			bots[i].Facts = got
		}
	}
	return bots, nil
}

func (p *Postgres) loadFacts(ctx context.Context, bots []Bot) (map[string]Facts, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT m.stake, to_char(m.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'), p1.slug, p2.slug, COALESCE(w.slug, '')
		FROM matches m
		JOIN bots p1 ON p1.id = m.p1_id
		JOIN bots p2 ON p2.id = m.p2_id
		LEFT JOIN bots w ON w.id = m.winner_id
		WHERE m.public_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []Match
	for rows.Next() {
		var match Match
		var winner string
		if err := rows.Scan(&match.Stake, &match.Date, &match.A.ID, &match.B.ID, &winner); err != nil {
			return nil, err
		}
		switch winner {
		case "":
			match.Winner = "draw"
		case match.A.ID:
			match.Winner = "a"
		default:
			match.Winner = "b"
		}
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return factsFor(bots, matches), nil
}

func (p *Postgres) matchesByID(ctx context.Context, ids []string) ([]Match, error) {
	if len(ids) == 0 {
		return []Match{}, nil
	}
	bots, err := p.loadBots(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[string]Bot{}
	for _, bot := range bots {
		byID[bot.ID] = bot
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	rows, err := p.db.QueryContext(ctx, `
		SELECT m.public_id, p1.slug, p2.slug, m.stake, to_char(m.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
			m.p1_score, m.p2_score, COALESCE(w.slug, ''), m.headline, m.highlights
		FROM matches m
		JOIN bots p1 ON p1.id = m.p1_id
		JOIN bots p2 ON p2.id = m.p2_id
		LEFT JOIN bots w ON w.id = m.winner_id
		WHERE m.public_id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := map[string]Match{}
	for rows.Next() {
		var match Match
		var winner string
		var highlights []byte
		var a, b string
		if err := rows.Scan(&match.ID, &a, &b, &match.Stake, &match.Date, &match.Score.A, &match.Score.B, &winner, &match.Headline, &highlights); err != nil {
			return nil, err
		}
		match.A = byID[a]
		match.B = byID[b]
		switch winner {
		case "":
			match.Winner = "draw"
		case a:
			match.Winner = "a"
		default:
			match.Winner = "b"
		}
		if err := json.Unmarshal(highlights, &match.Highlights); err != nil {
			return nil, err
		}
		found[match.ID] = match
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	roundRows, err := p.db.QueryContext(ctx, `
		SELECT m.public_id, r.round_number, r.p1_stance, r.p2_stance, r.p1_banter, r.p2_banter
		FROM match_rounds r JOIN matches m ON m.id = r.match_id
		WHERE m.public_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY r.round_number`, args...)
	if err != nil {
		return nil, err
	}
	defer roundRows.Close()
	for roundRows.Next() {
		var id string
		var n int
		var round Round
		if err := roundRows.Scan(&id, &n, &round.A, &round.B, &round.QuoteA, &round.QuoteB); err != nil {
			return nil, err
		}
		match := found[id]
		match.Rounds = append(match.Rounds, round)
		found[id] = match
	}
	if err := roundRows.Err(); err != nil {
		return nil, err
	}
	out := make([]Match, 0, len(ids))
	for _, id := range ids {
		match, ok := found[id]
		if ok {
			out = append(out, match)
		}
	}
	return out, nil
}

func matchFilter(q MatchQuery) (string, []any) {
	where := `WHERE m.public_id IS NOT NULL`
	args := []any{}
	next := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if q.Stake > 0 {
		where += ` AND m.stake = ` + next(q.Stake)
	}
	if q.Bot != "" {
		slot := next(q.Bot)
		where += ` AND (p1.slug = ` + slot + ` OR p2.slug = ` + slot + `)`
	}
	switch q.Outcome {
	case "draw":
		where += ` AND m.winner_id IS NULL`
	case "house":
		where += ` AND w.is_house`
	case "contender":
		where += ` AND m.winner_id IS NOT NULL AND NOT w.is_house`
	}
	if needle := strings.TrimSpace(q.Q); needle != "" {
		slot := next("%" + escapeLike(needle) + "%")
		where += ` AND (
			m.public_id ILIKE ` + slot + ` ESCAPE '\' OR
			m.headline ILIKE ` + slot + ` ESCAPE '\' OR
			p1.name ILIKE ` + slot + ` ESCAPE '\' OR
			p2.name ILIKE ` + slot + ` ESCAPE '\' OR
			m.highlights::text ILIKE ` + slot + ` ESCAPE '\' OR
			EXISTS (SELECT 1 FROM match_rounds r WHERE r.match_id = m.id AND (r.p1_banter ILIKE ` + slot + ` ESCAPE '\' OR r.p2_banter ILIKE ` + slot + ` ESCAPE '\'))
		)`
	}
	return where, args
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}
