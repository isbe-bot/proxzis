package conduct

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	bannedPatterns := []string{
		`(?i)\b(ignore previous instructions|disregard previous|system prompt|jailbreak|dan mode)\b`,
		`(?i)\b(you are now|override instructions|bypass security|act as|admin mode)\b`,
		`(?i)\b(fuck|shit|asshole|bitch|cunt|dick)\b`, // crude language per section 9.1
		`(?i)\b(https?://|www\.|discord\.gg|t\.me)\b`, // links / contact details prohibited
	}

	injectPatterns := []string{
		`(?i)</?(?:system|assistant|user|instructions|prompt)>`,
		`(?i)\[(?:system|instruction|admin)\]`,
	}

	compiledBanned := make([]*regexp.Regexp, 0, len(bannedPatterns))
	for _, p := range bannedPatterns {
		compiledBanned = append(compiledBanned, regexp.MustCompile(p))
	}

	compiledInject := make([]*regexp.Regexp, 0, len(injectPatterns))
	for _, p := range injectPatterns {
		compiledInject = append(compiledInject, regexp.MustCompile(p))
	}

	return &Moderator{
		sentHome:     make(map[uuid.UUID]time.Time),
		botInfracts:  make(map[uuid.UUID]int),
		bannedTokens: compiledBanned,
		promptInject: compiledInject,
	}
}

// CheckBanter verifies table banter against conduct standards.
func (m *Moderator) CheckBanter(botID uuid.UUID, rawText string) ModerationResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	var reasons []string

	// Check prompt injection markers
	for _, re := range m.promptInject {
		if re.MatchString(rawText) {
			reasons = append(reasons, "prompt_injection_pattern")
			break
		}
	}

	// Check banned keywords & conduct rules
	for _, re := range m.bannedTokens {
		if re.MatchString(rawText) {
			reasons = append(reasons, "disallowed_content")
			break
		}
	}

	if len(reasons) > 0 {
		m.botInfracts[botID]++
		// If bot commits 3+ infractions, send home for 24h
		if m.botInfracts[botID] >= 3 {
			m.sentHome[botID] = time.Now().UTC().Add(24 * time.Hour)
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
		CleanText: strings.TrimSpace(rawText),
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
