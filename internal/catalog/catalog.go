// Package catalog is the spectator archive: bots, matches, and the exhibition.
// Rows live in spectator tables, apart from the settlement ledger. The seeded
// rows are sample parlor history for the site. They are not live matchmaking state.
package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	OpeningBalance  = 1000
	SpectatorRounds = 7
	DefaultLimit    = 20
	MaxLimit        = 100
)

var ErrNotFound = errors.New("not found")

// FieldErrors is a rejected registration, keyed by form field.
type FieldErrors map[string]string

// ValidationError is a registration the server refused. Conflict is a duplicate
// name or email. Other failures are malformed input. Status is chosen from
// Conflict, not from the wording of a message.
type ValidationError struct {
	Fields   FieldErrors
	Conflict bool
}

func (e *ValidationError) Error() string {
	if e != nil && e.Conflict {
		return "registration conflict"
	}
	return "invalid registration"
}

type Facts struct {
	Wins       int    `json:"wins"`
	Losses     int    `json:"losses"`
	Draws      int    `json:"draws"`
	Played     int    `json:"played"`
	Earnings   int    `json:"earnings"`
	MarblesWon int    `json:"marblesWon"`
	LastPlayed string `json:"lastPlayed"`
	Balance    int    `json:"balance"`
	Tier       string `json:"tier"`
	Record     string `json:"record"`
}

type Bot struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"-"`
	Tag     string `json:"tag"`
	House   bool   `json:"house"`
	Rating  int    `json:"rating"`
	RD      int    `json:"rd"`
	Color   string `json:"color"`
	Face    string `json:"face"`
	Persona string `json:"persona"`
	Tactics []int  `json:"tactics"`
	Quote   string `json:"quote"`
	Facts   Facts  `json:"facts"`
}

type Round struct {
	A      string `json:"a"`
	B      string `json:"b"`
	QuoteA string `json:"quoteA"`
	QuoteB string `json:"quoteB"`
}

type Score struct {
	A int `json:"a"`
	B int `json:"b"`
}

type Match struct {
	ID         string   `json:"id"`
	A          Bot      `json:"a"`
	B          Bot      `json:"b"`
	Stake      int      `json:"stake"`
	Rounds     []Round  `json:"rounds"`
	Date       string   `json:"date"`
	Score      Score    `json:"score"`
	Winner     string   `json:"winner"`
	Headline   string   `json:"headline"`
	Highlights []string `json:"highlights"`
}

type Exhibition struct {
	Left   Bot     `json:"left"`
	Right  Bot     `json:"right"`
	Stake  int     `json:"stake"`
	Rounds []Round `json:"rounds"`
}

type Page struct {
	Items  any `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Page   int `json:"page"`
}

type BotQuery struct {
	Role   string
	Q      string
	Limit  int
	Offset int
}

type MatchQuery struct {
	Q       string
	Outcome string
	Stake   int
	Bot     string
	Limit   int
	Offset  int
}

type CreateBot struct {
	Name    string
	Email   string
	Persona string
}

// Store is the spectator read model. The daemon opens SQLite, or Postgres when
// DATABASE_URL is a postgres URL.
type Store interface {
	ListBots(ctx context.Context, q BotQuery) (Page, error)
	GetBot(ctx context.Context, id string) (Bot, error)
	ListMatches(ctx context.Context, q MatchQuery) (Page, error)
	GetMatch(ctx context.Context, id string) (Match, error)
	Exhibition(ctx context.Context) (Exhibition, error)
	CreateBot(ctx context.Context, in CreateBot) (Bot, error)
	Seed(ctx context.Context) error
	SeedIfEmpty(ctx context.Context) error
}

func ClampPage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func PageOf(items any, total, limit, offset int) Page {
	if items == nil {
		items = []any{}
	}
	page := 1
	if limit > 0 {
		page = offset/limit + 1
	}
	return Page{Items: items, Total: total, Limit: limit, Offset: offset, Page: page}
}

func BotUUID(slug string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("bottagon:spectator:"+slug))
}

func MatchUUID(publicID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("bottagon:match:"+publicID))
}

func Tier(rating int) string {
	switch {
	case rating >= 1650:
		return "Master"
	case rating >= 1600:
		return "Diamond"
	case rating >= 1550:
		return "Gold"
	default:
		return "Silver"
	}
}

var (
	reservedNames = map[string]struct{}{
		"admin": {}, "administrator": {}, "system": {}, "bottagon": {}, "refbot": {}, "house": {},
		"deacon": {}, "rusty": {}, "blinky": {}, "moth": {}, "sovereign": {}, "nyx": {}, "echo": {}, "grok": {},
	}
	namePattern  = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} _-]*$`)
	emailPattern = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
)

const (
	msgNameTaken  = "A bot with this name is already registered."
	msgEmailTaken = "This email is already associated with an active bot."
)

func ValidateCreate(in CreateBot) (CreateBot, *ValidationError) {
	name := strings.TrimSpace(in.Name)
	email := strings.ToLower(strings.TrimSpace(in.Email))
	persona := strings.TrimSpace(in.Persona)
	errs := FieldErrors{}
	switch {
	case name == "":
		errs["name"] = "Bot name is required."
	case utf8.RuneCountInString(name) < 3 || utf8.RuneCountInString(name) > 24:
		errs["name"] = "Bot name must be between 3 and 24 characters."
	case !namePattern.MatchString(name):
		errs["name"] = "Must start with a letter/number and use letters, numbers, spaces, _ or -."
	case reserved(name):
		errs["name"] = "This name is reserved by the BOTTAGON arena."
	}
	switch {
	case email == "":
		errs["email"] = "Owner email address is required."
	case len(email) > 254:
		errs["email"] = "Email address is too long."
	case !emailPattern.MatchString(email):
		errs["email"] = "Enter a valid owner email address."
	}
	switch {
	case persona == "":
		errs["persona"] = "Bot persona description is required."
	case utf8.RuneCountInString(persona) < 12 || utf8.RuneCountInString(persona) > 180:
		errs["persona"] = "Persona description must be between 12 and 180 characters."
	}
	if len(errs) > 0 {
		return CreateBot{}, &ValidationError{Fields: errs}
	}
	return CreateBot{Name: name, Email: email, Persona: persona}, nil
}

func conflictWith(existing []Bot, in CreateBot) *ValidationError {
	errs := FieldErrors{}
	for _, bot := range existing {
		if errs["name"] == "" && strings.EqualFold(strings.TrimSpace(bot.Name), in.Name) {
			errs["name"] = msgNameTaken
		}
		if errs["email"] == "" && bot.Email != "" && strings.EqualFold(bot.Email, in.Email) {
			errs["email"] = msgEmailTaken
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return &ValidationError{Fields: errs, Conflict: true}
}

func reserved(name string) bool {
	_, ok := reservedNames[strings.ToLower(name)]
	return ok
}

func NewBotSlug(name string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name))))
	return "c-" + hex.EncodeToString(sum[:6])
}

func EmptyFacts(rating int) Facts {
	return Facts{Balance: OpeningBalance, Tier: Tier(rating), Record: "0-0-0"}
}

func QuoteFromPersona(persona string) string {
	runes := []rune(persona)
	if len(runes) > 60 {
		return string(runes[:60]) + "..."
	}
	return persona + "..."
}
