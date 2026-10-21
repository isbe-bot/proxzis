package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		bot, err := a.server.authenticateBot(r) // authenticated before SDK dispatch
		if err != nil {
			return nil
		}
		return a.buildMCPServerForRequest(bot, r.Context())
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		MaxRequestBodyBytes:          maxRequestBodyBytes,
		PropagateRequestCancellation: true,
	})
	authenticated := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.server.requireBot(w, r); !ok {
			return
		}
		handler.ServeHTTP(w, r)
	})
	return boundaryHeaders(http.NewCrossOriginProtection().Handler(authenticated))
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
	return a.buildMCPServerForRequest(bot, context.Background())
}

func (a *MCPAdapter) buildMCPServerForRequest(bot matchmaker.BotInfo, requestCtx context.Context) *mcp.Server {
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
		ctx, cancel := originatingContext(ctx, requestCtx)
		defer cancel()
		session, err := a.server.findMatch(ctx, bot, FindMatchRequest{PreferredStake: args.PreferredStake, MaxStake: args.MaxStake})
		if err != nil {
			return errorToolResult(err.Error()), nil, nil
		}

		resp := a.server.buildEnvelope(session, bot.ID)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return textToolResult(string(data)), nil, nil
	})

	// Tool: play_turn
	mcp.AddTool(s, &mcp.Tool{
		Name:        "play_turn",
		Description: "Commit your stance (rock, paper, or scissors), side bet, and optional banter for the current Parley round.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args PlayTurnArgs) (*mcp.CallToolResult, any, error) {
		ctx, cancel := originatingContext(ctx, requestCtx)
		defer cancel()
		resp, err := a.server.playTurn(ctx, bot, TurnRequest{Stance: args.Stance, SideBet: args.SideBet, Say: args.Say})
		if err != nil {
			return errorToolResult(err.Error()), nil, nil
		}

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

		resp := a.server.buildEnvelope(session, bot.ID)

		data, _ := json.MarshalIndent(resp, "", "  ")
		return textToolResult(string(data)), nil, nil
	})

	// Tool: forfeit_match
	mcp.AddTool(s, &mcp.Tool{
		Name:        "forfeit_match",
		Description: "Concede and forfeit the active Parley match.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ForfeitMatchArgs) (*mcp.CallToolResult, any, error) {
		if !args.Confirm {
			return errorToolResult("confirmation required"), nil, nil
		}
		ctx, cancel := originatingContext(ctx, requestCtx)
		defer cancel()
		if err := a.server.forfeitMatch(ctx, bot); err != nil {
			return errorToolResult(err.Error()), nil, nil
		}

		return textToolResult(`{"status": "forfeited", "message": "Match conceded successfully"}`), nil, nil
	})

	// Tool: bot_status
	mcp.AddTool(s, &mcp.Tool{
		Name:        "bot_status",
		Description: "Retrieve demonstration contestant telemetry; purse, honor, and rating are not authoritative persisted account state.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args EmptyArgs) (*mcp.CallToolResult, any, error) {
		status := botStatus(bot)
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

// Older MCP protocols detach tool contexts from POST cancellation. Keep game
// long-polls bound to the originating request without losing SDK context values.
func originatingContext(ctx, requestCtx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(requestCtx, cancel)
	if requestCtx.Err() != nil {
		cancel()
	}
	return ctx, func() { stop(); cancel() }
}
