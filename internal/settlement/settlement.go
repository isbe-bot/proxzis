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
	ErrBotNotFound        = errors.New("bot not found")
	ErrMatchNotFound      = errors.New("match not found")
	ErrInvalidSettlement  = errors.New("invalid settlement")
	ErrSettlementConflict = errors.New("settlement conflicts with persisted match")
	ErrBalanceBounds      = errors.New("marble balance outside storage bounds")
	ErrInsufficientFunds  = errors.New("insufficient marble purse")
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
	if err := validateMatch(matchID, p1ID, p2ID, match); err != nil {
		return err
	}
	p1Settle, p2Settle, err := match.Settle()
	if err != nil {
		return fmt.Errorf("settlement math failed: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	// Serialize retries before touching either contestant. completed status alone
	// is not a settlement marker: callers may persist the game outcome first.
	var storedP1, storedP2 uuid.UUID
	var status string
	var settledAt sql.NullTime
	var score1, score2, wager1, wager2, mirrors int
	var winner sql.NullString
	var sudden bool
	err = tx.QueryRowContext(ctx, `
		SELECT p1_id, p2_id, status, settled_at, p1_score, p2_score,
		p1_side_bet, p2_side_bet, mirror_ties_count, winner_id, is_sudden_death
		FROM matches WHERE id = $1 FOR UPDATE
	`, matchID).Scan(&storedP1, &storedP2, &status, &settledAt, &score1, &score2,
		&wager1, &wager2, &mirrors, &winner, &sudden)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMatchNotFound
	}
	if err != nil {
		return fmt.Errorf("lock match: %w", err)
	}
	if storedP1 != p1ID || storedP2 != p2ID {
		return ErrSettlementConflict
	}
	if settledAt.Valid {
		if status != "completed" || score1 != match.Score.P1 || score2 != match.Score.P2 ||
			wager1 != match.Config.P1SideBet || wager2 != match.Config.P2SideBet ||
			mirrors != match.MirrorTiesCount || sudden != match.IsSuddenDeath ||
			winner != winnerID(match.Winner, p1ID, p2ID) {
			return ErrSettlementConflict
		}
		return nil
	}
	if status != "pending" && status != "in_progress" && status != "completed" {
		return ErrSettlementConflict
	}
	// A legacy settlement has no marker and may already have moved funds. Fail
	// closed for reconciliation rather than paying it again or rewriting history.
	var hasLedger bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
		SELECT 1 FROM marble_ledger_entries WHERE match_id = $1
		)
	`, matchID).Scan(&hasLedger); err != nil {
		return fmt.Errorf("check existing ledger: %w", err)
	}
	if hasLedger {
		return ErrSettlementConflict
	}

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

	// Counterparts use separately locked system accounts, never an arbitrary
	// playable is_house bot. All settlements lock reserve then sink.
	if err := applyHouseCounterparts(ctx, tx, matchID, p1Settle.NetMarbleDelta, p2Settle.NetMarbleDelta); err != nil {
		return fmt.Errorf("house accounting: %w", err)
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

	newP1, err := s.glickoEngine.UpdateChecked(p1Rating, []glicko2.MatchResult{{Opponent: p2Rating, Score: p1Score}})
	if err != nil {
		return fmt.Errorf("rating p1: %w", err)
	}
	newP2, err := s.glickoEngine.UpdateChecked(p2Rating, []glicko2.MatchResult{{Opponent: p1Rating, Score: p2Score}})
	if err != nil {
		return fmt.Errorf("rating p2: %w", err)
	}

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
	winnerUUID := winnerID(match.Winner, p1ID, p2ID)

	_, err = tx.ExecContext(ctx, `
		UPDATE matches
		SET status = 'completed',
		p1_score = $1,
		p2_score = $2,
		winner_id = $3,
		is_sudden_death = $4,
		mirror_ties_count = $5,
		completed_at = COALESCE(completed_at, clock_timestamp()),
		updated_at = clock_timestamp(),
		settled_at = clock_timestamp(),
		p1_side_bet = $7,
		p2_side_bet = $8
		WHERE id = $6
	`, match.Score.P1, match.Score.P2, winnerUUID, match.IsSuddenDeath, match.MirrorTiesCount, matchID, match.Config.P1SideBet, match.Config.P2SideBet)
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
	// bots.marbles and ledger.delta are PostgreSQL INTEGER, not Go int.
	// Validate before addition; no machine-sized signed wrap is possible.
	if s.NetMarbleDelta < -math.MaxInt32 || s.NetMarbleDelta > math.MaxInt32 {
		return ErrBalanceBounds
	}
	newBalance := int64(bot.Marbles) + int64(s.NetMarbleDelta)
	if newBalance < 0 && !bot.IsHouse {
		return ErrInsufficientFunds
	}
	if newBalance < math.MinInt32 || newBalance > math.MaxInt32 {
		return ErrBalanceBounds
	}
	bot.Marbles = int(newBalance)

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

// validateMatch also bounds arithmetic before invoking the domain settlement.
// Wagers are not escrowed: a winning wager contributes +wager, not +2*wager.
func validateMatch(id, p1, p2 uuid.UUID, m *parley.Match) error {
	if m == nil || id == uuid.Nil || p1 == uuid.Nil || p2 == uuid.Nil || p1 == p2 {
		return ErrInvalidSettlement
	}
	engineID, err := uuid.Parse(m.ID)
	if err != nil || engineID != id || m.Status != parley.StatusCompleted ||
		(m.Winner != "" && m.Winner != parley.Player1 && m.Winner != parley.Player2) {
		return ErrInvalidSettlement
	}
	for _, n := range []int{m.Config.P1SideBet, m.Config.P2SideBet, m.MirrorTiesCount, m.Score.P1, m.Score.P2} {
		if n < 0 || int64(n) > math.MaxInt32 {
			return ErrInvalidSettlement
		}
	}
	for _, player := range []parley.PlayerID{parley.Player1, parley.Player2} {
		wager := m.Config.P1SideBet
		if player == parley.Player2 {
			wager = m.Config.P2SideBet
		}
		delta := -int64(m.MirrorTiesCount)
		if m.Winner == player {
			delta += 1 + int64(wager)
		} else if m.Winner != "" {
			delta -= 1 + int64(wager)
		}
		// Exclude MinInt32: its opposite would not fit ledger.delta.
		if delta < -math.MaxInt32 || delta > math.MaxInt32 {
			return ErrBalanceBounds
		}
	}
	return nil
}

func winnerID(winner parley.PlayerID, p1, p2 uuid.UUID) sql.NullString {
	if winner == parley.Player1 {
		return sql.NullString{String: p1.String(), Valid: true}
	}
	if winner == parley.Player2 {
		return sql.NullString{String: p2.String(), Valid: true}
	}
	return sql.NullString{}
}

func applyHouseCounterparts(ctx context.Context, tx *sql.Tx, matchID uuid.UUID, deltas ...int) error {
	for _, account := range []string{"house_reserve", "house_sink"} {
		var balance int64
		if err := tx.QueryRowContext(ctx, `
		SELECT balance FROM settlement_house_accounts
		WHERE account_type = $1 FOR UPDATE
	`, account).Scan(&balance); err != nil {
			return err
		}
		for _, delta := range deltas {
			if (delta >= 0) != (account == "house_reserve") {
				continue
			}
			opposite := -int64(delta)
			if (opposite > 0 && balance > math.MaxInt64-opposite) ||
				(opposite < 0 && balance < math.MinInt64-opposite) {
				return ErrBalanceBounds
			}
			balance += opposite
			if _, err := tx.ExecContext(ctx, `
		INSERT INTO marble_ledger_entries
		(id, bot_id, match_id, account_type, entry_type, delta, balance_after)
		VALUES ($1, NULL, $2, $3, 'match_counterpart', $4, $5)
	`,
				uuid.New(), matchID, account, opposite, balance); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `
		UPDATE settlement_house_accounts SET balance = $1
		WHERE account_type = $2
	`, balance, account); err != nil {
			return err
		}
	}
	return nil
}
