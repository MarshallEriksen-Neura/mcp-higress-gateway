package mcpserver_test

import (
	"context"
	"testing"
	"time"

	"bridge/internal/logging"
	"bridge/internal/mcpbridge"
	"bridge/internal/mcpserver"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAggregatorStdioServer_ListAndCall(t *testing.T) {
	logger, _ := logging.NewLogger(logging.Options{Level: "error", Format: "text"})
	agg := mcpbridge.NewAggregator(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := agg.Start(ctx, nil); err != nil {
		t.Fatalf("aggregator start: %v", err)
	}

	server := mcpserver.NewAggregatorStdioServer(logger, agg)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()

	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "v0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	toolsRes, err := session.ListTools(ctx, &sdk.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	found := false
	for _, tool := range toolsRes.Tools {
		if tool != nil && tool.Name == "bridge__echo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tool bridge__echo in tools/list")
	}

	callRes, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "bridge__echo",
		Arguments: map[string]any{"text": "hi"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if callRes == nil {
		t.Fatalf("call tool: nil result")
	}
	if callRes.IsError {
		t.Fatalf("call tool returned isError=true")
	}
	if callRes.StructuredContent == nil {
		t.Fatalf("expected structuredContent")
	}
}
