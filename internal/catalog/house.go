package catalog

import "github.com/isbe-bot/bottagon/internal/house"

func houseBots() []Bot {
	agents := house.Agents()
	bots := make([]Bot, 0, len(agents))
	for _, agent := range agents {
		bots = append(bots, Bot{
			ID: agent.Slug, Name: agent.Name, Tag: agent.Tag, House: true,
			Rating: agent.Rating, RD: agent.RD, Color: agent.Color, Face: agent.Face,
			Persona: agent.Persona, Tactics: []int{agent.Rock, agent.Paper, agent.Scissors},
			Quote: agent.Quote, Facts: EmptyFacts(agent.Rating),
		})
	}
	return bots
}

// mergeHouse adds the resident roster and refreshes their public profile.
// Match facts already stored on an existing row are kept.
func mergeHouse(bots []Bot) []Bot {
	index := make(map[string]int, len(bots))
	for i, bot := range bots {
		index[bot.ID] = i
	}
	for _, resident := range houseBots() {
		if i, ok := index[resident.ID]; ok {
			facts := bots[i].Facts
			email := bots[i].Email
			bots[i] = resident
			bots[i].Facts = facts
			bots[i].Email = email
			continue
		}
		bots = append(bots, resident)
	}
	return bots
}
