package catalog

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// Memory is the in-process archive used by tests. The daemon uses SQLite or Postgres.
type Memory struct {
	mu         sync.Mutex
	bots       []Bot
	matches    []Match
	exhibition Exhibition
	seeded     bool
}

func NewMemory() *Memory {
	bots, matches, exhibition := Generate()
	return &Memory{bots: bots, matches: matches, exhibition: exhibition, seeded: true}
}

func (m *Memory) Seed(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	bots, matches, exhibition := Generate()
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []Bot
	for _, bot := range m.bots {
		if strings.HasPrefix(bot.ID, "c-") {
			kept = append(kept, bot)
		}
	}
	m.bots = append(bots, kept...)
	m.matches = matches
	m.exhibition = exhibition
	m.seeded = true
	return nil
}

func (m *Memory) SeedIfEmpty(ctx context.Context) error {
	m.mu.Lock()
	empty := !m.seeded || len(m.bots) == 0
	m.mu.Unlock()
	if empty {
		return m.Seed(ctx)
	}
	return nil
}

func (m *Memory) ListBots(ctx context.Context, q BotQuery) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	limit, offset := ClampPage(q.Limit, q.Offset)
	m.mu.Lock()
	defer m.mu.Unlock()
	var matched []Bot
	for _, bot := range m.bots {
		if q.Role == "house" && !bot.House || q.Role == "contender" && bot.House {
			continue
		}
		if q.Q != "" && !containsFold(bot.Name+" "+bot.Tag+" "+bot.Persona+" "+bot.Quote, q.Q) {
			continue
		}
		matched = append(matched, bot)
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Rating == matched[j].Rating {
			return matched[i].Name < matched[j].Name
		}
		return matched[i].Rating > matched[j].Rating
	})
	return PageOf(sliceBots(matched, offset, limit), len(matched), limit, offset), nil
}

func (m *Memory) GetBot(ctx context.Context, id string) (Bot, error) {
	if err := ctx.Err(); err != nil {
		return Bot{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, bot := range m.bots {
		if bot.ID == id {
			return bot, nil
		}
	}
	return Bot{}, ErrNotFound
}

func (m *Memory) ListMatches(ctx context.Context, q MatchQuery) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	limit, offset := ClampPage(q.Limit, q.Offset)
	m.mu.Lock()
	defer m.mu.Unlock()
	var matched []Match
	for _, match := range m.matches {
		if !matchAllowed(match, q) {
			continue
		}
		matched = append(matched, match)
	}
	return PageOf(sliceMatches(matched, offset, limit), len(matched), limit, offset), nil
}

func (m *Memory) GetMatch(ctx context.Context, id string) (Match, error) {
	if err := ctx.Err(); err != nil {
		return Match{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, match := range m.matches {
		if match.ID == id {
			return match, nil
		}
	}
	return Match{}, ErrNotFound
}

func (m *Memory) Exhibition(ctx context.Context) (Exhibition, error) {
	if err := ctx.Err(); err != nil {
		return Exhibition{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.exhibition.Left.ID == "" {
		return Exhibition{}, ErrNotFound
	}
	return m.exhibition, nil
}

func (m *Memory) CreateBot(ctx context.Context, in CreateBot) (Bot, error) {
	if err := ctx.Err(); err != nil {
		return Bot{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	clean, verr := ValidateCreate(in)
	if verr != nil {
		return Bot{}, verr
	}
	if conflict := conflictWith(m.bots, clean); conflict != nil {
		return Bot{}, conflict
	}
	bot := Bot{
		ID: NewBotSlug(clean.Name), Name: clean.Name, Email: clean.Email, Tag: "NEW CONTENDER",
		Rating: 1500, RD: 350, Color: "mint", Face: "visor", Persona: clean.Persona,
		Tactics: []int{34, 33, 33}, Quote: QuoteFromPersona(clean.Persona), Facts: EmptyFacts(1500),
	}
	m.bots = append(m.bots, bot)
	return bot, nil
}

func matchAllowed(match Match, q MatchQuery) bool {
	if q.Stake > 0 && match.Stake != q.Stake {
		return false
	}
	if q.Bot != "" && match.A.ID != q.Bot && match.B.ID != q.Bot {
		return false
	}
	switch q.Outcome {
	case "draw":
		if match.Winner != "draw" {
			return false
		}
	case "house":
		if match.Winner == "draw" || !matchSide(match, match.Winner).House {
			return false
		}
	case "contender":
		if match.Winner == "draw" || matchSide(match, match.Winner).House {
			return false
		}
	}
	if q.Q == "" {
		return true
	}
	var b strings.Builder
	b.WriteString(match.ID)
	b.WriteByte(' ')
	b.WriteString(match.A.Name)
	b.WriteByte(' ')
	b.WriteString(match.B.Name)
	b.WriteByte(' ')
	b.WriteString(match.Headline)
	for _, highlight := range match.Highlights {
		b.WriteByte(' ')
		b.WriteString(highlight)
	}
	for _, round := range match.Rounds {
		b.WriteByte(' ')
		b.WriteString(round.A)
		b.WriteByte(' ')
		b.WriteString(round.B)
		b.WriteByte(' ')
		b.WriteString(round.QuoteA)
		b.WriteByte(' ')
		b.WriteString(round.QuoteB)
	}
	return containsFold(b.String(), q.Q)
}

func matchSide(match Match, side string) Bot {
	if side == "a" {
		return match.A
	}
	return match.B
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(strings.TrimSpace(needle)))
}

func sliceBots(items []Bot, offset, limit int) []Bot {
	if offset >= len(items) {
		return []Bot{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]Bot(nil), items[offset:end]...)
}

func sliceMatches(items []Match, offset, limit int) []Match {
	if offset >= len(items) {
		return []Match{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]Match(nil), items[offset:end]...)
}
