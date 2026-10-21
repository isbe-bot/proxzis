package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// sqlStore is the shared spectator query path. SQL is written with ?
// placeholders and rebound for Postgres. Records are aggregated in the query.
type sqlStore struct {
	db       *sql.DB
	postgres bool
}

func newSQLStore(db *sql.DB, postgres bool) *sqlStore {
	return &sqlStore{db: db, postgres: postgres}
}

func (s *sqlStore) q(query string) string {
	if !s.postgres {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

const factsCTE = `
WITH sides AS (
	SELECT p1_id AS bot_id,
		CASE WHEN winner_id IS NULL THEN 0 WHEN winner_id = p1_id THEN 1 ELSE -1 END AS outcome,
		stake, created_at
	FROM spectator_matches
	UNION ALL
	SELECT p2_id,
		CASE WHEN winner_id IS NULL THEN 0 WHEN winner_id = p2_id THEN 1 ELSE -1 END,
		stake, created_at
	FROM spectator_matches
),
facts AS (
	SELECT bot_id,
		SUM(CASE WHEN outcome = 1 THEN 1 ELSE 0 END) AS wins,
		SUM(CASE WHEN outcome = -1 THEN 1 ELSE 0 END) AS losses,
		SUM(CASE WHEN outcome = 0 THEN 1 ELSE 0 END) AS draws,
		COUNT(*) AS played,
		SUM(CASE WHEN outcome = 1 THEN stake WHEN outcome = -1 THEN -stake ELSE 0 END) AS earnings,
		SUM(CASE WHEN outcome = 1 THEN stake * 2 ELSE 0 END) AS marbles_won,
		MAX(created_at) AS last_played
	FROM sides
	GROUP BY bot_id
)`

func botColumns(alias, facts string) string {
	return fmt.Sprintf(`%s.slug, %s.name, %s.tag, %s.is_house, %s.rating, %s.rd, %s.color, %s.face, %s.persona, %s.quote, %s.tactic_strike, %s.tactic_feint, %s.tactic_parry, COALESCE(%s.wins,0), COALESCE(%s.losses,0), COALESCE(%s.draws,0), COALESCE(%s.played,0), COALESCE(%s.earnings,0), COALESCE(%s.marbles_won,0), COALESCE(%s.last_played,'')`,
		alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias,
		facts, facts, facts, facts, facts, facts, facts)
}

type botScan struct {
	bot                         Bot
	house                       int
	rating, rd                  float64
	strike, feint, parry        int
	wins, losses, draws, played int64
	earnings, marbles           int64
	last                        string
}

func (b *botScan) dest() []any {
	return []any{
		&b.bot.ID, &b.bot.Name, &b.bot.Tag, &b.house, &b.rating, &b.rd, &b.bot.Color, &b.bot.Face, &b.bot.Persona, &b.bot.Quote,
		&b.strike, &b.feint, &b.parry, &b.wins, &b.losses, &b.draws, &b.played, &b.earnings, &b.marbles, &b.last,
	}
}

func (b *botScan) value() Bot {
	b.bot.House = b.house != 0
	b.bot.Rating = int(b.rating)
	b.bot.RD = int(b.rd)
	b.bot.Tactics = []int{b.strike, b.feint, b.parry}
	b.bot.Facts = Facts{
		Wins: int(b.wins), Losses: int(b.losses), Draws: int(b.draws), Played: int(b.played),
		Earnings: int(b.earnings), MarblesWon: int(b.marbles), LastPlayed: b.last,
		Balance: OpeningBalance + int(b.earnings), Tier: Tier(b.bot.Rating),
		Record: fmt.Sprintf("%d-%d-%d", b.wins, b.losses, b.draws),
	}
	return b.bot
}

func (s *sqlStore) SeedIfEmpty(ctx context.Context) error {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM spectator_bots`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return s.Seed(ctx)
	}
	return nil
}

func (s *sqlStore) Seed(ctx context.Context) error {
	bots, matches, exhibition := Generate()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, bot := range bots {
		if _, err := tx.ExecContext(ctx, s.q(`
			INSERT INTO spectator_bots (id, slug, name, email, is_house, rating, rd, tag, persona, quote, color, face, tactic_strike, tactic_feint, tactic_parry)
			VALUES (?,?,?,'',?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT (slug) DO UPDATE SET
				name = excluded.name, is_house = excluded.is_house, rating = excluded.rating, rd = excluded.rd,
				tag = excluded.tag, persona = excluded.persona, quote = excluded.quote, color = excluded.color, face = excluded.face,
				tactic_strike = excluded.tactic_strike, tactic_feint = excluded.tactic_feint, tactic_parry = excluded.tactic_parry`),
			BotUUID(bot.ID).String(), bot.ID, bot.Name, bit(bot.House), bot.Rating, bot.RD,
			bot.Tag, bot.Persona, bot.Quote, bot.Color, bot.Face, bot.Tactics[0], bot.Tactics[1], bot.Tactics[2],
		); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM spectator_rounds`); err != nil {
		return err
	}
	for _, match := range matches {
		highlights, err := json.Marshal(match.Highlights)
		if err != nil {
			return err
		}
		var winner any
		switch match.Winner {
		case "a":
			winner = BotUUID(match.A.ID).String()
		case "b":
			winner = BotUUID(match.B.ID).String()
		}
		if _, err := tx.ExecContext(ctx, s.q(`
			INSERT INTO spectator_matches (id, public_id, p1_id, p2_id, stake, p1_score, p2_score, winner_id, headline, highlights, created_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT (public_id) DO UPDATE SET
				p1_id = excluded.p1_id, p2_id = excluded.p2_id, stake = excluded.stake,
				p1_score = excluded.p1_score, p2_score = excluded.p2_score, winner_id = excluded.winner_id,
				headline = excluded.headline, highlights = excluded.highlights, created_at = excluded.created_at`),
			MatchUUID(match.ID).String(), match.ID, BotUUID(match.A.ID).String(), BotUUID(match.B.ID).String(),
			match.Stake, match.Score.A, match.Score.B, winner, match.Headline, string(highlights), match.Date,
		); err != nil {
			return err
		}
		for n, round := range match.Rounds {
			if _, err := tx.ExecContext(ctx, s.q(`
				INSERT INTO spectator_rounds (match_id, round_number, p1_stance, p2_stance, p1_banter, p2_banter)
				VALUES (?,?,?,?,?,?)`),
				MatchUUID(match.ID).String(), n+1, round.A, round.B, round.QuoteA, round.QuoteB,
			); err != nil {
				return err
			}
		}
	}
	payload, err := json.Marshal(exhibition.Rounds)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, s.q(`
		INSERT INTO spectator_exhibition (id, left_slug, right_slug, stake, rounds)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET left_slug = excluded.left_slug, right_slug = excluded.right_slug, stake = excluded.stake, rounds = excluded.rounds`),
		exhibition.Left.ID, exhibition.Right.ID, exhibition.Stake, string(payload),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *sqlStore) ListBots(ctx context.Context, q BotQuery) (Page, error) {
	limit, offset := ClampPage(q.Limit, q.Offset)
	where, args := botFilter(q)
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Page{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var total int
	if err := tx.QueryRowContext(ctx, s.q(`SELECT count(*) FROM spectator_bots b `+where), args...).Scan(&total); err != nil {
		return Page{}, err
	}
	pageArgs := append(append([]any{}, args...), limit, offset)
	bots, err := queryBots(ctx, tx, s.q(factsCTE+`
		SELECT `+botColumns("b", "f")+`
		FROM spectator_bots b
		LEFT JOIN facts f ON f.bot_id = b.id `+where+`
		ORDER BY b.rating DESC, b.name ASC
		LIMIT ? OFFSET ?`), pageArgs...)
	if err != nil {
		return Page{}, err
	}
	if err := tx.Commit(); err != nil {
		return Page{}, err
	}
	return PageOf(bots, total, limit, offset), nil
}

func (s *sqlStore) GetBot(ctx context.Context, id string) (Bot, error) {
	bots, err := queryBots(ctx, s.db, s.q(factsCTE+`
		SELECT `+botColumns("b", "f")+`
		FROM spectator_bots b
		LEFT JOIN facts f ON f.bot_id = b.id
		WHERE b.slug = ?`), id)
	if err != nil {
		return Bot{}, err
	}
	if len(bots) == 0 {
		return Bot{}, ErrNotFound
	}
	return bots[0], nil
}

func (s *sqlStore) ListMatches(ctx context.Context, q MatchQuery) (Page, error) {
	limit, offset := ClampPage(q.Limit, q.Offset)
	where, args := matchFilter(q)
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Page{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var total int
	if err := tx.QueryRowContext(ctx, s.q(`SELECT count(*) FROM spectator_matches m
		JOIN spectator_bots p1 ON p1.id = m.p1_id
		JOIN spectator_bots p2 ON p2.id = m.p2_id
		LEFT JOIN spectator_bots w ON w.id = m.winner_id `+where), args...).Scan(&total); err != nil {
		return Page{}, err
	}
	pageArgs := append(append([]any{}, args...), limit, offset)
	matches, err := s.queryMatches(ctx, tx, where+` ORDER BY m.created_at DESC, m.public_id DESC LIMIT ? OFFSET ?`, pageArgs...)
	if err != nil {
		return Page{}, err
	}
	if err := tx.Commit(); err != nil {
		return Page{}, err
	}
	return PageOf(matches, total, limit, offset), nil
}

func (s *sqlStore) GetMatch(ctx context.Context, id string) (Match, error) {
	matches, err := s.queryMatches(ctx, s.db, `WHERE m.public_id = ?`, id)
	if err != nil {
		return Match{}, err
	}
	if len(matches) == 0 {
		return Match{}, ErrNotFound
	}
	return matches[0], nil
}

func (s *sqlStore) Exhibition(ctx context.Context) (Exhibition, error) {
	var left, right, raw string
	var stake int
	err := s.db.QueryRowContext(ctx, `SELECT left_slug, right_slug, stake, rounds FROM spectator_exhibition WHERE id = 1`).Scan(&left, &right, &stake, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Exhibition{}, ErrNotFound
	}
	if err != nil {
		return Exhibition{}, err
	}
	var rounds []Round
	if err := json.Unmarshal([]byte(raw), &rounds); err != nil {
		return Exhibition{}, err
	}
	bots, err := queryBots(ctx, s.db, s.q(factsCTE+`
		SELECT `+botColumns("b", "f")+`
		FROM spectator_bots b
		LEFT JOIN facts f ON f.bot_id = b.id
		WHERE b.slug IN (?, ?)`), left, right)
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

func (s *sqlStore) CreateBot(ctx context.Context, in CreateBot) (Bot, error) {
	clean, verr := ValidateCreate(in)
	if verr != nil {
		return Bot{}, verr
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Bot{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var nameTaken, emailTaken int
	if err := tx.QueryRowContext(ctx, s.q(`SELECT
		(SELECT count(*) FROM spectator_bots WHERE lower(name) = lower(?)),
		(SELECT count(*) FROM spectator_bots WHERE email <> '' AND lower(email) = lower(?))`),
		clean.Name, clean.Email).Scan(&nameTaken, &emailTaken); err != nil {
		return Bot{}, err
	}
	fields := FieldErrors{}
	if nameTaken > 0 {
		fields["name"] = msgNameTaken
	}
	if emailTaken > 0 {
		fields["email"] = msgEmailTaken
	}
	if len(fields) > 0 {
		return Bot{}, &ValidationError{Fields: fields, Conflict: true}
	}
	bot := Bot{
		ID: NewBotSlug(clean.Name), Name: clean.Name, Email: clean.Email, Tag: "NEW CONTENDER",
		Rating: 1500, RD: 350, Color: "mint", Face: "visor", Persona: clean.Persona,
		Tactics: []int{34, 33, 33}, Quote: QuoteFromPersona(clean.Persona), Facts: EmptyFacts(1500),
	}
	_, err = tx.ExecContext(ctx, s.q(`
		INSERT INTO spectator_bots (id, slug, name, email, is_house, rating, rd, tag, persona, quote, color, face, tactic_strike, tactic_feint, tactic_parry)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`),
		BotUUID(bot.ID).String(), bot.ID, bot.Name, bot.Email, 0, bot.Rating, bot.RD, bot.Tag, bot.Persona, bot.Quote, bot.Color, bot.Face, bot.Tactics[0], bot.Tactics[1], bot.Tactics[2],
	)
	if conflict := constraintConflict(err); conflict != nil {
		return Bot{}, conflict
	}
	if err != nil {
		return Bot{}, err
	}
	if err := tx.Commit(); err != nil {
		return Bot{}, err
	}
	return bot, nil
}

type rowQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func queryBots(ctx context.Context, q rowQuery, query string, args ...any) ([]Bot, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []Bot
	for rows.Next() {
		var scanned botScan
		if err := rows.Scan(scanned.dest()...); err != nil {
			return nil, err
		}
		bots = append(bots, scanned.value())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if bots == nil {
		bots = []Bot{}
	}
	return bots, nil
}

func (s *sqlStore) queryMatches(ctx context.Context, q rowQuery, where string, args ...any) ([]Match, error) {
	rows, err := q.QueryContext(ctx, s.q(factsCTE+`
		SELECT m.public_id, m.stake, m.created_at, m.p1_score, m.p2_score,
			CASE WHEN m.winner_id IS NULL THEN 'draw' WHEN m.winner_id = m.p1_id THEN 'a' ELSE 'b' END,
			m.headline, m.highlights, `+botColumns("p1", "f1")+`, `+botColumns("p2", "f2")+`
		FROM spectator_matches m
		JOIN spectator_bots p1 ON p1.id = m.p1_id
		JOIN spectator_bots p2 ON p2.id = m.p2_id
		LEFT JOIN spectator_bots w ON w.id = m.winner_id
		LEFT JOIN facts f1 ON f1.bot_id = p1.id
		LEFT JOIN facts f2 ON f2.bot_id = p2.id
		`+where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []Match
	for rows.Next() {
		var match Match
		var highlights string
		var left, right botScan
		dest := []any{&match.ID, &match.Stake, &match.Date, &match.Score.A, &match.Score.B, &match.Winner, &match.Headline, &highlights}
		dest = append(dest, left.dest()...)
		dest = append(dest, right.dest()...)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(highlights), &match.Highlights); err != nil {
			return nil, err
		}
		match.A = left.value()
		match.B = right.value()
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return []Match{}, nil
	}
	ids := make([]any, len(matches))
	marks := make([]string, len(matches))
	byID := map[string]*Match{}
	for i := range matches {
		ids[i] = matches[i].ID
		marks[i] = "?"
		byID[matches[i].ID] = &matches[i]
	}
	roundRows, err := q.QueryContext(ctx, s.q(`
		SELECT m.public_id, r.p1_stance, r.p2_stance, r.p1_banter, r.p2_banter
		FROM spectator_rounds r
		JOIN spectator_matches m ON m.id = r.match_id
		WHERE m.public_id IN (`+strings.Join(marks, ",")+`)
		ORDER BY m.public_id, r.round_number`), ids...)
	if err != nil {
		return nil, err
	}
	defer roundRows.Close()
	for roundRows.Next() {
		var id string
		var round Round
		if err := roundRows.Scan(&id, &round.A, &round.B, &round.QuoteA, &round.QuoteB); err != nil {
			return nil, err
		}
		match := byID[id]
		match.Rounds = append(match.Rounds, round)
	}
	if err := roundRows.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}

func botFilter(q BotQuery) (string, []any) {
	where := `WHERE 1 = 1`
	var args []any
	switch q.Role {
	case "house":
		where += ` AND b.is_house = 1`
	case "contender":
		where += ` AND b.is_house = 0`
	}
	if needle := strings.TrimSpace(q.Q); needle != "" {
		where += ` AND lower(b.name || ' ' || b.tag || ' ' || b.persona || ' ' || b.quote) LIKE lower(?) ESCAPE '\'`
		args = append(args, "%"+escapeLike(needle)+"%")
	}
	return where, args
}

func matchFilter(q MatchQuery) (string, []any) {
	where := `WHERE 1 = 1`
	var args []any
	if q.Stake > 0 {
		where += ` AND m.stake = ?`
		args = append(args, q.Stake)
	}
	if q.Bot != "" {
		where += ` AND (p1.slug = ? OR p2.slug = ?)`
		args = append(args, q.Bot, q.Bot)
	}
	switch q.Outcome {
	case "draw":
		where += ` AND m.winner_id IS NULL`
	case "house":
		where += ` AND w.is_house = 1`
	case "contender":
		where += ` AND m.winner_id IS NOT NULL AND w.is_house = 0`
	}
	if needle := strings.TrimSpace(q.Q); needle != "" {
		pattern := "%" + escapeLike(needle) + "%"
		where += ` AND (
			lower(m.public_id) LIKE lower(?) ESCAPE '\' OR
			lower(m.headline) LIKE lower(?) ESCAPE '\' OR
			lower(p1.name) LIKE lower(?) ESCAPE '\' OR
			lower(p2.name) LIKE lower(?) ESCAPE '\' OR
			lower(m.highlights) LIKE lower(?) ESCAPE '\' OR
			EXISTS (
				SELECT 1 FROM spectator_rounds r
				WHERE r.match_id = m.id AND (
					lower(r.p1_banter) LIKE lower(?) ESCAPE '\' OR
					lower(r.p2_banter) LIKE lower(?) ESCAPE '\' OR
					lower(r.p1_stance) LIKE lower(?) ESCAPE '\' OR
					lower(r.p2_stance) LIKE lower(?) ESCAPE '\'
				)
			)
		)`
		for range 9 {
			args = append(args, pattern)
		}
	}
	return where, args
}

func constraintConflict(err error) *ValidationError {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if strings.Contains(pgErr.ConstraintName, "email") {
			return &ValidationError{Fields: FieldErrors{"email": msgEmailTaken}, Conflict: true}
		}
		return &ValidationError{Fields: FieldErrors{"name": msgNameTaken}, Conflict: true}
	}
	message := err.Error()
	if strings.Contains(message, "UNIQUE constraint failed") {
		if strings.Contains(message, "email") {
			return &ValidationError{Fields: FieldErrors{"email": msgEmailTaken}, Conflict: true}
		}
		return &ValidationError{Fields: FieldErrors{"name": msgNameTaken}, Conflict: true}
	}
	return nil
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func bit(value bool) int {
	if value {
		return 1
	}
	return 0
}
