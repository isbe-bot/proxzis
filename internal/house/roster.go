// Package house is the resident roster. Each agent is an ordinary bot with
// is_house set. The sheet's tactic triple is strike, feint, parry, which maps
// to rock, scissors, paper.
package house

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/isbe-bot/bottagon/internal/game/parley"
)

//go:embed bots.json
var rosterJSON []byte

const OpeningMarbles = 1000

// Agent is one house resident. ID is the stable ledger id.
type Agent struct {
	ID       uuid.UUID
	Slug     string
	Name     string
	Tag      string
	Rating   int
	RD       int
	Color    string
	Face     string
	Persona  string
	Quote    string
	Rock     int
	Paper    int
	Scissors int
}

var agents []Agent

func init() {
	var sheet []sheetBot
	if err := json.Unmarshal(rosterJSON, &sheet); err != nil {
		panic(err)
	}
	agents = make([]Agent, 0, len(sheet))
	seen := map[string]struct{}{}
	for _, row := range sheet {
		agent, err := convert(row)
		if err != nil {
			panic(err)
		}
		if _, ok := seen[agent.Slug]; ok {
			panic("duplicate house slug " + agent.Slug)
		}
		seen[agent.Slug] = struct{}{}
		agents = append(agents, agent)
	}
	if len(agents) != 10 {
		panic(fmt.Sprintf("house roster has %d agents", len(agents)))
	}
}

// Agents returns the roster in sheet order.
func Agents() []Agent {
	out := make([]Agent, len(agents))
	copy(out, agents)
	return out
}

// Pick returns the resident whose rating is closest to the contestant.
// Equal distances keep the earlier slug.
func Pick(rating float64) Agent {
	best := agents[0]
	bestDist := math.Abs(float64(best.Rating) - rating)
	for _, agent := range agents[1:] {
		dist := math.Abs(float64(agent.Rating) - rating)
		if dist < bestDist || (dist == bestDist && agent.Slug < best.Slug) {
			best = agent
			bestDist = dist
		}
	}
	return best
}

// Player is the stance policy for this resident. House agents do not banter.
func (a Agent) Player() *parley.WeightedBot {
	var salt uint64
	for _, b := range a.ID[:8] {
		salt = salt<<8 | uint64(b)
	}
	return parley.NewWeighted(a.Name, a.Rock, a.Paper, a.Scissors, salt)
}

type sheetBot struct {
	ID      string `json:"id"`
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Rating  int    `json:"rating"`
	RD      int    `json:"rd"`
	Tactics []int  `json:"tactics"`
	Color   string `json:"color"`
	Face    string `json:"face"`
	Persona string `json:"persona"`
	Quote   string `json:"quote"`
}

func convert(row sheetBot) (Agent, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return Agent{}, fmt.Errorf("house bot %s: %w", row.Slug, err)
	}
	if row.Slug == "" || row.Name == "" {
		return Agent{}, fmt.Errorf("house bot %s is missing slug or name", row.ID)
	}
	if len(row.Tactics) != 3 {
		return Agent{}, fmt.Errorf("house bot %s tactics", row.Slug)
	}
	rock, scissors, paper := row.Tactics[0], row.Tactics[1], row.Tactics[2]
	if rock < 0 || paper < 0 || scissors < 0 || rock+paper+scissors != 100 {
		return Agent{}, fmt.Errorf("house bot %s tactics must sum to 100", row.Slug)
	}
	return Agent{
		ID: id, Slug: row.Slug, Name: row.Name, Tag: row.Tag,
		Rating: row.Rating, RD: row.RD, Color: row.Color, Face: row.Face,
		Persona: row.Persona, Quote: row.Quote,
		Rock: rock, Paper: paper, Scissors: scissors,
	}, nil
}
