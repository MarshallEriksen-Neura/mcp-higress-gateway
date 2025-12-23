package mcpbridge_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"bridge/internal/config"
	"bridge/internal/mcpbridge"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

func TestAggregator_RemoteStreamable(t *testing.T) {
	ctx := context.Background()

	server := sdk.NewServer(&sdk.Implementation{Name: "server", Version: "v0"}, nil)
	handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{
		Stateless:      true,
		JSONResponse:   true,
		SessionTimeout: 0,
	})

	var sawHeader atomic.Bool
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") == "1" {
			sawHeader.Store(true)
		}
		handler.ServeHTTP(w, r)
	}))
	defer httpServer.Close()

	agg := mcpbridge.NewAggregator(nopLogger{})
	defer func() { _ = agg.Start(ctx, nil) }()

	err := agg.Start(ctx, []config.MCPServerConfig{
		{Name: "remote", Type: "streamable", URL: httpServer.URL, Headers: map[string]string{"X-Test": "1"}},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if !sawHeader.Load() {
		t.Fatalf("expected remote requests to include configured headers")
	}

	tools := agg.ListTools(ctx)
	found := false
	for _, td := range tools {
		if td.Name == "bridge__echo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected built-in tool bridge__echo to exist")
	}
}

func TestAggregator_RemoteLegacySSE(t *testing.T) {
	ctx := context.Background()

	server := sdk.NewServer(&sdk.Implementation{Name: "server", Version: "v0"}, nil)
	handler := sdk.NewSSEHandler(func(*http.Request) *sdk.Server { return server }, nil)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	agg := mcpbridge.NewAggregator(nopLogger{})
	defer func() { _ = agg.Start(ctx, nil) }()

	err := agg.Start(ctx, []config.MCPServerConfig{
		{Name: "remote", Type: "sse", URL: httpServer.URL},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
}

func TestAggregator_RemoteAutoFallbackToSSE(t *testing.T) {
	ctx := context.Background()

	server := sdk.NewServer(&sdk.Implementation{Name: "server", Version: "v0"}, nil)
	handler := sdk.NewSSEHandler(func(*http.Request) *sdk.Server { return server }, nil)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	agg := mcpbridge.NewAggregator(nopLogger{})
	defer func() { _ = agg.Start(ctx, nil) }()

	err := agg.Start(ctx, []config.MCPServerConfig{
		{Name: "remote", URL: httpServer.URL},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
}
