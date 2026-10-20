package settlement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/isbe-bot/bottagon/internal/game/parley"
	"github.com/isbe-bot/bottagon/internal/rating/glicko2"
)

var (
	ErrBotNotFound       = errors.New("bot not found")
	ErrInsufficientFunds = errors.New("insufficient marble purse")
)

type BotRecord struct {
	ID               uuid.UUID
	Name             string
	IsHouse          bool
	Marbles          int
	Rating           float64
	RD               float64
	Volatility       float64
	Honor            int
	PledgesMade      int
	PledgesKept      int
	ConductPenalties int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Service struct {
	db           *sql.DB
	glickoEngine *glicko2.Glicko2Engine
}

func NewService(db *sql.DB) *Service {
	return &Service{
		db:           db,
		glickoEngine: glicko2.NewEngine(),
	}
}

// ComputeHonor recalculates a bot's honor score:
// honor = clamp(round(100 * (kept + 10) / (made + 10)) - penalties, 0, 100)
func ComputeHonor(kept, made, penalties int) int {
	if made < 0 {
		made = 0
	}
	if kept < 0 {
		kept = 0
	}
	ratio := float64(kept+10) / float64(made+10)
	score := int(math.Round(100.0 * ratio))
	score -= penalties
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// SettleMatch atomically persists match conclusion, double-entry marble ledger updates,
// and Glicko-2 ratings under row-level SELECT FOR UPDATE locks.
func (s *Service) SettleMatch(ctx context.Context, matchID uuid.UUID, p1ID, p2ID uuid.UUID, match *parley.Match) error {
	p1Settle, p2Settle, err := match.Settle()
	if err != nil {
		return fmt.Errorf("settlement math failed: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	// Ensure consistent lock acquisition order to prevent deadlock
	firstID, secondID := p1ID, p2ID
	if firstID.String() > secondID.String() {
		firstID, secondID = p2ID, p1ID
	}

	b1, err := lockBot(ctx, tx, firstID)
	if err != nil {
		return fmt.Errorf("failed to lock bot %s: %w", firstID, err)
	}
	b2, err := lockBot(ctx, tx, secondID)
	if err != nil {
		return fmt.Errorf("failed to lock bot %s: %w", secondID, err)
	}

	var p1Bot, p2Bot BotRecord
	if b1.ID == p1ID {
		p1Bot, p2Bot = b1, b2
	} else {
		p1Bot, p2Bot = b2, b1
	}

	// 1. Apply marble deltas & insert ledger rows
	if err := applySettlement(ctx, tx, matchID, &p1Bot, p1Settle); err != nil {
		return fmt.Errorf("failed to apply settlement for p1: %w", err)
	}
	if err := applySettlement(ctx, tx, matchID, &p2Bot, p2Settle); err != nil {
		return fmt.Errorf("failed to apply settlement for p2: %w", err)
	}

	// 2. Glicko-2 Rating calculation
	p1Score := 0.5
	p2Score := 0.5
	if match.Winner == parley.Player1 {
		p1Score, p2Score = 1.0, 0.0
	} else if match.Winner == parley.Player2 {
		p1Score, p2Score = 0.0, 1.0
	}

	p1Rating := glicko2.Rating{Rating: p1Bot.Rating, RD: p1Bot.RD, Volatility: p1Bot.Volatility}
	p2Rating := glicko2.Rating{Rating: p2Bot.Rating, RD: p2Bot.RD, Volatility: p2Bot.Volatility}

	newP1 := s.glickoEngine.Update(p1Rating, []glicko2.MatchResult{{Opponent: p2Rating, Score: p1Score}})
	newP2 := s.glickoEngine.Update(p2Rating, []glicko2.MatchResult{{Opponent: p1Rating, Score: p2Score}})

	p1Bot.Rating, p1Bot.RD, p1Bot.Volatility = newP1.Rating, newP1.RD, newP1.Volatility
	p2Bot.Rating, p2Bot.RD, p2Bot.Volatility = newP2.Rating, newP2.RD, newP2.Volatility

	// 3. Update bot records
	if err := updateBot(ctx, tx, p1Bot); err != nil {
		return fmt.Errorf("failed to update bot p1: %w", err)
	}
	if err := updateBot(ctx, tx, p2Bot); err != nil {
		return fmt.Errorf("failed to update bot p2: %w", err)
	}

	// 4. Update match status
	var winnerUUID sql.NullString
	if match.Winner == parley.Player1 {
		winnerUUID = sql.NullString{String: p1ID.String(), Valid: true}
	} else if match.Winner == parley.Player2 {
		winnerUUID = sql.NullString{String: p2ID.String(), Valid: true}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE matches
		SET status = 'completed',
		    p1_score = $1,
		    p2_score = $2,
		    winner_id = $3,
		    is_sudden_death = $4,
		    mirror_ties_count = $5,
		    completed_at = clock_timestamp(),
		    updated_at = clock_timestamp()
		WHERE id = $6
	`, match.Score.P1, match.Score.P2, winnerUUID, match.IsSuddenDeath, match.MirrorTiesCount, matchID)
	if err != nil {
		return fmt.Errorf("failed to complete match record: %w", err)
	}

	return tx.Commit()
}

func lockBot(ctx context.Context, tx *sql.Tx, botID uuid.UUID) (BotRecord, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, name, is_house, marbles, rating, rd, volatility, honor, pledges_made, pledges_kept, conduct_penalties, created_at, updated_at
		FROM bots
		WHERE id = $1
		FOR UPDATE
	`, botID)

	var b BotRecord
	err := row.Scan(
		&b.ID, &b.Name, &b.IsHouse, &b.Marbles, &b.Rating, &b.RD, &b.Volatility,
		&b.Honor, &b.PledgesMade, &b.PledgesKept, &b.ConductPenalties, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return b, ErrBotNotFound
	}
	return b, err
}

func applySettlement(ctx context.Context, tx *sql.Tx, matchID uuid.UUID, bot *BotRecord, s parley.Settlement) error {
	newBalance := bot.Marbles + s.NetMarbleDelta
	if newBalance < 0 && !bot.IsHouse {
		return ErrInsufficientFunds
	}
	bot.Marbles = newBalance

	// Write immutable ledger entry
	entryType := "match_loss"
	if s.NetMarbleDelta >= 0 {
		entryType = "match_win"
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO marble_ledger_entries (id, bot_id, match_id, account_type, entry_type, delta, balance_after, created_at)
		VALUES ($1, $2, $3, 'bot', $4, $5, $6, clock_timestamp())
	`, uuid.New(), bot.ID, matchID, entryType, s.NetMarbleDelta, newBalance)
	return err
}

func updateBot(ctx context.Context, tx *sql.Tx, bot BotRecord) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE bots
		SET marbles = $1,
		    rating = $2,
		    rd = $3,
		    volatility = $4,
		    honor = $5,
		    updated_at = clock_timestamp()
		WHERE id = $6
	`, bot.Marbles, bot.Rating, bot.RD, bot.Volatility, bot.Honor, bot.ID)
	return err
}
