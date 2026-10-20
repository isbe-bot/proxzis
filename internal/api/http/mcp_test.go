package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

func TestEndpoints_RulesAndLLMsTxt(t *testing.T) {
	q := matchmaker.NewQueue()
	server := NewServer(q)

	// Test /v1/rules
	req := httptest.NewRequest(http.MethodGet, "/rules", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /v1/rules, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "BOTTAGON: Parley Game Rules") {
		t.Fatalf("expected rules markdown, got: %s", rr.Body.String())
	}
}

func TestMCPAdapter_ToolsDirectCall(t *testing.T) {
	q := matchmaker.NewQueue()
	server := NewServer(q)
	adapter := NewMCPAdapter(server)

	bot := matchmaker.BotInfo{
		ID:      uuid.New(),
		Name:    "MCPTestBot",
		Rating:  1500.0,
		Honor:   100,
		IsHouse: false,
	}

	mcpServer := adapter.buildMCPServer(bot)
	if mcpServer == nil {
		t.Fatal("expected non-nil MCP server")
	}

	// Test handler response via HTTP POST to /mcp
	mcpHandler := adapter.Handler()
	initReqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0"}}}`
	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(initReqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+bot.ID.String())
	w := httptest.NewRecorder()

	mcpHandler.ServeHTTP(w, httpReq)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from MCP initialize handler, got %d: %s", w.Code, w.Body.String())
	}

	// Extract JSON payload from SSE event body: "event: message\ndata: {json}\n\n"
	bodyStr := w.Body.String()
	var jsonLine string
	for _, line := range strings.Split(bodyStr, "\n") {
		if strings.HasPrefix(line, "data: ") {
			jsonLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	if jsonLine == "" {
		t.Fatalf("could not find data line in SSE response: %s", bodyStr)
	}

	var initResp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(jsonLine), &initResp); err != nil {
		t.Fatalf("failed to decode MCP init response: %v", err)
	}
	if initResp.Result.ServerInfo.Name != "bottagon-mcp" {
		t.Fatalf("unexpected server name: %s", initResp.Result.ServerInfo.Name)
	}
}

func TestMCPAdapter_FullSessionToolLifecycle(t *testing.T) {
	q := matchmaker.NewQueue()
	server := NewServer(q)
	adapter := NewMCPAdapter(server)

	bot := matchmaker.BotInfo{
		ID:      uuid.New(),
		Name:    "MCPCombatant",
		Rating:  1500.0,
		Honor:   100,
		IsHouse: false,
	}

	mcpServer := adapter.buildMCPServer(bot)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverSession, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server session: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client session: %v", err)
	}
	defer clientSession.Close()

	// 1. List tools
	toolsRes, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	toolNames := make(map[string]bool)
	for _, tool := range toolsRes.Tools {
		toolNames[tool.Name] = true
	}
	expectedTools := []string{"find_match", "play_turn", "current_match", "forfeit_match", "bot_status", "get_rules"}
	for _, exp := range expectedTools {
		if !toolNames[exp] {
			t.Errorf("missing expected MCP tool: %s", exp)
		}
	}

	// 2. Call bot_status
	statusRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "bot_status",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool bot_status failed: %v", err)
	}
	if len(statusRes.Content) == 0 {
		t.Fatal("empty content in bot_status")
	}

	// 3. Call get_rules
	rulesRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_rules",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool get_rules failed: %v", err)
	}
	textRule, ok := rulesRes.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(textRule.Text, "BOTTAGON: Parley") {
		t.Fatalf("unexpected rules content: %v", rulesRes.Content)
	}
}
