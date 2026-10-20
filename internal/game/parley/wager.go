package parley

import (
	"errors"
	"math"
)

// PassLineBet represents a unilateral side bet placed in Round 1 against the House.
type PassLineBet struct {
	Wager int `json:"wager"` // Number of marbles committed
}

// Settlement represents marble accounting settlement against the House.
type Settlement struct {
	BaseRewardWon     int `json:"base_reward_won"`     // +1 marble from house on win
	BasePenaltyLost   int `json:"base_penalty_lost"`   // -1 marble to house on loss
	PassLineRefund    int `json:"pass_line_refund"`    // Wager returned if won
	PassLineProfit    int `json:"pass_line_profit"`    // Wager matched 1:1 if won
	PassLineLoss      int `json:"pass_line_loss"`      // Wager lost to house sink if lost
	MirrorPenaltyPaid int `json:"mirror_penalty_paid"` // Ties penalty if MirrorTiePenalty was active
	NetMarbleDelta    int `json:"net_marble_delta"`    // Total delta on bot's ledger
}

// SettlePassLine calculates the unilateral marble settlement between a contestant and the House.
// For legacy callers, negatives clamp to zero and overflow saturates. Ledger
// writers must use SettlePassLineChecked instead.
// Rules:
// 1. Base Match:
//   - Victor: +1 marble credited from House.
//   - Defeated: -1 marble surrendered to House.
//   - Draw: 0 base delta after exhausted tied sudden-death rounds.
//
// 2. Pass Line (Side Bet, 1:1 payout):
//   - Placed in Round 1.
//   - If player won match: Wager refunded + Wager credited (Net +Wager).
//   - If player lost match: Wager forfeited to House sink (Net -Wager).
//
// 3. Mirror Tie Penalty:
//   - Each tied round under MirrorTiePenalty penalizes 1 marble to house pot.
func SettlePassLine(isWinner bool, isDraw bool, wager int, mirrorTiesCount int) Settlement {
	if wager < 0 {
		wager = 0
	}
	if mirrorTiesCount < 0 {
		mirrorTiesCount = 0
	}

	s := Settlement{
		MirrorPenaltyPaid: mirrorTiesCount,
	}

	if isDraw {
		// Exhausted tied sudden-death rounds
		s.NetMarbleDelta = -mirrorTiesCount
		return s
	}

	if isWinner {
		s.BaseRewardWon = 1
		s.PassLineRefund = wager
		s.PassLineProfit = wager
		// Subtract first; the exact result is representable except MaxInt + 1.
		delta := wager - mirrorTiesCount
		if delta == math.MaxInt {
			s.NetMarbleDelta = math.MaxInt
		} else {
			s.NetMarbleDelta = delta + 1
		}
	} else {
		s.BasePenaltyLost = 1
		s.PassLineLoss = wager
		// Saturate rather than wrapping a debit into a credit.
		if wager > math.MaxInt-mirrorTiesCount {
			s.NetMarbleDelta = math.MinInt
		} else {
			s.NetMarbleDelta = -1 - wager - mirrorTiesCount
		}
	}

	return s
}

var ErrSettlementOverflow = errors.New("marble settlement exceeds integer bounds")
var ErrInvalidSettlement = errors.New("invalid marble settlement state")

// SettlePassLineChecked rejects invalid or unrepresentable ledger arithmetic.
// The legacy SettlePassLine helper saturates extremes; ledger writers must use
// this checked variant (as Match.Settle does), never persist a saturated value.
func SettlePassLineChecked(isWinner, isDraw bool, wager, mirrorTiesCount int) (Settlement, error) {
	if wager < 0 || mirrorTiesCount < 0 {
		return Settlement{}, ErrInvalidSettlement
	}
	if !isDraw {
		if isWinner && wager == math.MaxInt && mirrorTiesCount == 0 {
			return Settlement{}, ErrSettlementOverflow
		}
		if !isWinner && wager > math.MaxInt-mirrorTiesCount {
			return Settlement{}, ErrSettlementOverflow
		}
	}
	return SettlePassLine(isWinner, isDraw, wager, mirrorTiesCount), nil
}
