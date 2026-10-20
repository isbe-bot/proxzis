package conduct

import (
	"context"
	"html"
	"math"
	"unicode"
	"unicode/utf8"

	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/isbe-bot/bottagon/internal/game/parley"
	"golang.org/x/text/unicode/norm"
)

// Verdict indicates moderation outcome.
type Verdict string

const (
	VerdictApproved Verdict = "approved"
	VerdictWithheld Verdict = "withheld"
)

// ModerationResult encapsulates the moderation check result.
type ModerationResult struct {
	Verdict   Verdict
	Reasons   []string
	CleanText string
}

// ConductEvent records an infraction or moderation event.
type ConductEvent struct {
	ID        uuid.UUID
	BotID     uuid.UUID
	MatchID   *uuid.UUID
	Text      string
	Verdict   Verdict
	Reasons   []string
	Timestamp time.Time
}

// ModerationStore defines persistence operations for moderation and conduct.
type ModerationStore interface {
	RecordModerationEvent(ctx context.Context, event ConductEvent) error
	RecordReport(ctx context.Context, report ReportRecord) error
	SendHomeBot(ctx context.Context, botID uuid.UUID, duration time.Duration, reason string) error
	IsSentHome(ctx context.Context, botID uuid.UUID) (bool, time.Time, error)
	ApplyHonorDecay(ctx context.Context, botID uuid.UUID, deltaHonor int) (int, error)
}

// ReportRecord stores a contestant complaint/report.
type ReportRecord struct {
	ID           uuid.UUID
	ReporterID   uuid.UUID
	AccusedID    uuid.UUID
	MatchID      *uuid.UUID
	Reason       string
	BanterSample string
	ActionTaken  string
	CreatedAt    time.Time
}

// Moderator provides rule-based and phrase-based content moderation for banter.
type Moderator struct {
	mu           sync.RWMutex
	sentHome     map[uuid.UUID]time.Time
	botInfracts  map[uuid.UUID]int
	bannedTokens []*regexp.Regexp
	promptInject []*regexp.Regexp
}

// NewModerator initializes a Moderator with prompt injection and conduct rules.
func NewModerator() *Moderator {
	return &Moderator{
		sentHome:     make(map[uuid.UUID]time.Time),
		botInfracts:  make(map[uuid.UUID]int),
		bannedTokens: bannedRules,
		promptInject: injectionRules,
	}
}

// CheckBanter verifies table banter against conduct standards.
func (m *Moderator) CheckBanter(botID uuid.UUID, rawText string) ModerationResult {
	var reasons []string
	var cleanText string
	if len(rawText) > parley.MaxBanterInputBytes || !utf8.ValidString(rawText) {
		reasons = append(reasons, "invalid_banter_input")
	} else {
		// Match decoded/compatibility-normalized text before markup is removed, then
		// plain text too: both role tags and split-token obfuscation must be caught.
		normalized := normalizeBanter(rawText)
		cleanText = parley.SanitizeBanter(normalized)
		for _, re := range m.promptInject {
			if re.MatchString(normalized) {
				reasons = append(reasons, "prompt_injection_pattern")
				break
			}
		}
		for _, re := range m.bannedTokens {
			if re.MatchString(normalized) || (cleanText != normalized && re.MatchString(cleanText)) {
				reasons = append(reasons, "disallowed_content")
				break
			}
		}
	}

	if len(reasons) > 0 {
		m.mu.Lock()
		defer m.mu.Unlock()
		now := time.Now().UTC()
		if until, ok := m.sentHome[botID]; ok && !now.Before(until) {
			delete(m.sentHome, botID)
			delete(m.botInfracts, botID)
		}
		if m.botInfracts[botID] < math.MaxInt {
			m.botInfracts[botID]++
		}
		// If bot commits 3+ infractions, send home for 24h
		if m.botInfracts[botID] >= 3 {
			until := now.Add(24 * time.Hour)
			if until.After(m.sentHome[botID]) {
				m.sentHome[botID] = until
			}
		}

		return ModerationResult{
			Verdict:   VerdictWithheld,
			Reasons:   reasons,
			CleanText: "[Remark withheld for parlor conduct]",
		}
	}

	return ModerationResult{
		Verdict:   VerdictApproved,
		Reasons:   nil,
		CleanText: cleanText,
	}
}

// IsBotSentHome checks if bot has been temporarily barred.
func (m *Moderator) IsBotSentHome(botID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	until, exists := m.sentHome[botID]
	if !exists {
		return false
	}
	return time.Now().UTC().Before(until)
}

// SendHome manually sends a bot home.
func (m *Moderator) SendHome(botID uuid.UUID, dur time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentHome[botID] = time.Now().UTC().Add(dur)
}

// GetInfractionCount returns infraction count for bot.
func (m *Moderator) GetInfractionCount(botID uuid.UUID) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.botInfracts[botID]
}

// Rules are immutable and compiled once, not once per Moderator instance.
var bannedRules = compileRules([]string{
	`(?i)\b(ignore\s+(?:(?:all|the)\s+)?(?:previous|prior|above)\s+(?:instructions?|prompts?|rules?)|disregard\s+(?:previous|prior)|system\s+prompt|jailbreak|dan\s+mode)\b`,
	`(?i)\b(you\s+are\s+now|override\s+instructions|bypass\s+security|act\s+as|admin\s+mode)\b`,
	`(?i)\b(fuck|shit|asshole|bitch|cunt|dick)\b`,
	`(?i)(https?://|www\.|discord\.gg|t\.me)`,
	`(?i)[a-z0-9.!%+_-]+@[a-z0-9.-]+\.[a-z]{2,}`,
	`(?:^|\s)@[\pL\pN_]+`,
	`(?:^|[^0-9])\+?[0-9][0-9 ()-]{7,}[0-9](?:$|[^0-9])`,
})
var injectionRules = compileRules([]string{
	`(?i)<\s*/?\s*(?:system|assistant|user|instructions|prompt|opponent_banter)(?:\s[^>]*|/?)>`,
	`(?i)\[\s*(?:system|instruction|admin)\s*\]`,
	`(?i)<\|(?:im_start|im_end|start_header_id|end_header_id|eot_id)\|>|\[/?INST\]`,
})

func compileRules(patterns []string) []*regexp.Regexp {
	rules := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		rules = append(rules, regexp.MustCompile(p))
	}
	return rules
}
func normalizeBanter(raw string) string {
	decoded := norm.NFKC.String(html.UnescapeString(raw))
	var out strings.Builder
	out.Grow(len(decoded))
	space := false
	for _, r := range decoded {
		if unicode.IsSpace(r) {
			space = out.Len() > 0
			continue
		}
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			continue
		}
		if space {
			out.WriteByte(' ')
			space = false
		}
		out.WriteRune(r)
	}
	return out.String()
}
