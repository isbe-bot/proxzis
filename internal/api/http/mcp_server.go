package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/isbe-bot/bottagon/internal/game/parley"
	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

// MCPAdapter exposes BOTTAGON game actions over the official Go MCP SDK.
type MCPAdapter struct {
	server *Server
}

// NewMCPAdapter creates a new MCPAdapter.
func NewMCPAdapter(s *Server) *MCPAdapter {
	return &MCPAdapter{server: s}
}

// Handler returns an http.Handler implementing Streamable HTTP MCP at /mcp.
func (a *MCPAdapter) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		bot := a.server.authenticateBot(r)
		return a.buildMCPServer(bot)
	}, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})
}

type FindMatchArgs struct {
	PreferredStake int `json:"preferred_stake,omitempty"`
	MaxStake       int `json:"max_stake,omitempty"`
}

type PlayTurnArgs struct {
	Stance  string `json:"stance"`
	SideBet int    `json:"side_bet,omitempty"`
	Say     string `json:"say,omitempty"`
}

type ForfeitMatchArgs struct {
	Confirm bool `json:"confirm"`
}

type EmptyArgs struct{}

func (a *MCPAdapter) buildMCPServer(bot matchmaker.BotInfo) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "bottagon-mcp",
		Version: "1.0.0",
		Title:   "BOTTAGON Parley Arena MCP Server",
	}, nil)

	// Tool: find_match
	mcp.AddTool(s, &mcp.Tool{
		Name:        "find_match",
		Description: "Enter the BOTTAGON matchmaking queue to find an opponent (or fall back to House Regular after 8s).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args FindMatchArgs) (*mcp.CallToolResult, any, error) {
		prefStake := args.PreferredStake
		if prefStake <= 0 {
			prefStake = 1
		}
		maxStake := args.MaxStake
		if maxStake <= 0 {
			maxStake = 5
		}

		session, err := a.server.queue.FindMatch(ctx, bot, prefStake, maxStake, matchmaker.MaxQueueWaitTime)
		if err != nil {
			return errorToolResult(fmt.Sprintf("Matchmaking error: %v", err)), nil, nil
		}

		resp := a.server.buildEnvelope(session, bot.ID)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return textToolResult(string(data)), nil, nil
	})

	// Tool: play_turn
	mcp.AddTool(s, &mcp.Tool{
		Name:        "play_turn",
		Description: "Commit your stance (strike, parry, or feint), side bet, and optional banter for the current Parley round.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args PlayTurnArgs) (*mcp.CallToolResult, any, error) {
		session, exists := a.server.queue.GetSession(bot.ID)
		if !exists {
			return errorToolResult("No active match in progress. Use find_match first."), nil, nil
		}

		stance, err := parley.ParseStance(args.Stance)
		if err != nil {
			return errorToolResult(fmt.Sprintf("Invalid stance '%s'. Valid: strike, parry, feint.", args.Stance)), nil, nil
		}

		session.Mu.Lock()
		role := parley.Player1
		if session.P2.ID == bot.ID {
			role = parley.Player2
		}

		err = session.Engine.CommitTurn(role, stance, args.Say, args.SideBet)
		if err != nil {
			session.Mu.Unlock()
			return errorToolResult(fmt.Sprintf("Commit error: %v", err)), nil, nil
		}

		if session.HouseBot != nil && session.Engine.Status != parley.StatusCompleted {
			houseRole := parley.Player2
			houseStance := session.HouseBot.SelectStance(session.Engine.Rounds, houseRole)
			houseBanter := session.HouseBot.SelectBanter(session.Engine.Rounds, houseRole)
			_ = session.Engine.CommitTurn(houseRole, houseStance, houseBanter, 0)
		}

		session.SequenceID++
		session.LastActive = time.Now().UTC()
		session.Mu.Unlock()

		resp := a.server.buildEnvelope(session, bot.ID)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return textToolResult(string(data)), nil, nil
	})

	// Tool: current_match
	mcp.AddTool(s, &mcp.Tool{
		Name:        "current_match",
		Description: "Query the status and pending turn details of your active match.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args EmptyArgs) (*mcp.CallToolResult, any, error) {
		session, exists := a.server.queue.GetSession(bot.ID)
		if !exists {
			return textToolResult(`{"status": "idle", "message": "No active match found"}`), nil, nil
		}

		session.Mu.Lock()
		resp := a.server.buildEnvelope(session, bot.ID)
		session.Mu.Unlock()

		data, _ := json.MarshalIndent(resp, "", "  ")
		return textToolResult(string(data)), nil, nil
	})

	// Tool: forfeit_match
	mcp.AddTool(s, &mcp.Tool{
		Name:        "forfeit_match",
		Description: "Concede and forfeit the active Parley match.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ForfeitMatchArgs) (*mcp.CallToolResult, any, error) {
		session, exists := a.server.queue.GetSession(bot.ID)
		if !exists {
			return errorToolResult("No active match to forfeit."), nil, nil
		}

		session.Mu.Lock()
		session.Engine.Status = parley.StatusCompleted
		if session.P1.ID == bot.ID {
			session.Engine.Winner = parley.Player2
		} else {
			session.Engine.Winner = parley.Player1
		}
		session.Mu.Unlock()

		a.server.queue.EndSession(session.MatchID)
		return textToolResult(`{"status": "forfeited", "message": "Match conceded successfully"}`), nil, nil
	})

	// Tool: bot_status
	mcp.AddTool(s, &mcp.Tool{
		Name:        "bot_status",
		Description: "Retrieve contestant purse, honor score, and current Glicko-2 rating.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args EmptyArgs) (*mcp.CallToolResult, any, error) {
		status := map[string]interface{}{
			"bot_id": bot.ID.String(),
			"name":   bot.Name,
			"rating": bot.Rating,
			"honor":  bot.Honor,
			"purse":  1000,
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		return textToolResult(string(data)), nil, nil
	})

	// Tool: get_rules
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_rules",
		Description: "Read the full Parley game rules, stance matrix, and modifier cards.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args EmptyArgs) (*mcp.CallToolResult, any, error) {
		return textToolResult(RulesMarkdown), nil, nil
	})

	return s
}

func textToolResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.TrimSpace(text)},
		},
	}
}

func errorToolResult(errStr string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Error: %s", errStr)},
		},
	}
}
