package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"bridge/internal/logging"
	"bridge/internal/mcpbridge"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type AggregatorStdioServer struct {
	logger logging.Logger
	agg    *mcpbridge.Aggregator

	server *sdk.Server
	mux    *eventMux
}

func NewAggregatorStdioServer(logger logging.Logger, agg *mcpbridge.Aggregator) *AggregatorStdioServer {
	impl := &sdk.Implementation{
		Name:    "ai-bridge-agent",
		Version: "v0",
	}

	mcpServer := sdk.NewServer(impl, &sdk.ServerOptions{
		Instructions: "Aggregated MCP server powered by AI Bridge",
		HasTools:     true,
	})

	s := &AggregatorStdioServer{
		logger: logger,
		agg:    agg,
		server: mcpServer,
		mux:    newEventMux(),
	}
	s.installTools()
	return s
}

func (s *AggregatorStdioServer) Run(ctx context.Context, transport sdk.Transport) error {
	if s == nil {
		return errors.New("server is nil")
	}
	if s.agg == nil {
		return errors.New("aggregator is nil")
	}
	go s.mux.run(ctx, s.agg)
	return s.server.Run(ctx, transport)
}

func (s *AggregatorStdioServer) installTools() {
	for _, td := range s.agg.ListTools(context.Background()) {
		tool := &sdk.Tool{
			Name:        td.Name,
			Description: td.Description,
			InputSchema: normalizeToolInputSchema(td.InputSchema),
		}

		toolName := td.Name

		s.server.AddTool(tool, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			if req == nil || req.Params == nil {
				return nil, errors.New("missing request params")
			}

			args := map[string]any{}
			if len(req.Params.Arguments) > 0 {
				if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
					return &sdk.CallToolResult{
						IsError: true,
						Content: []sdk.Content{&sdk.TextContent{Text: fmt.Sprintf("invalid arguments: %v", err)}},
					}, nil
				}
			}

			progressToken := req.Params.GetProgressToken()
			internalReqID := ""
			if progressToken != nil {
				if t, ok := progressToken.(string); ok && t != "" {
					internalReqID = t
				} else {
					internalReqID = "pt_" + uuid.NewString()
				}
			}

			if internalReqID != "" {
				s.mux.subscribe(internalReqID, req.Session, progressToken)
				defer s.mux.unsubscribe(internalReqID, req.Session)
			}

			out, err := s.agg.CallTool(ctx, internalReqID, toolName, args)
			if err != nil {
				if errors.Is(err, mcpbridge.ErrToolNotFound) {
					return nil, err
				}
				if errors.Is(err, context.Canceled) {
					return &sdk.CallToolResult{
						IsError: true,
						Content: []sdk.Content{&sdk.TextContent{Text: "canceled"}},
					}, nil
				}
				return &sdk.CallToolResult{
					IsError: true,
					Content: []sdk.Content{&sdk.TextContent{Text: err.Error()}},
				}, nil
			}

			structured := ensureJSONObject(out)
			text := mustJSON(structured)

			return &sdk.CallToolResult{
				Content:           []sdk.Content{&sdk.TextContent{Text: text}},
				StructuredContent: structured,
			}, nil
		})
	}
}

func normalizeToolInputSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return map[string]any{"type": "object"}
	}
	out := make(map[string]any, len(schema)+1)
	for k, v := range schema {
		out[k] = v
	}
	// Ensure it conforms to go-sdk's AddTool requirement: type must be object.
	out["type"] = "object"
	return out
}

func ensureJSONObject(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{"result": v}
}

func mustJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

type eventMux struct {
	mu   sync.RWMutex
	subs map[string]map[*sdk.ServerSession]any // reqID -> session -> progressToken (original type)
}

func newEventMux() *eventMux {
	return &eventMux{subs: make(map[string]map[*sdk.ServerSession]any)}
}

func (m *eventMux) subscribe(reqID string, ss *sdk.ServerSession, token any) {
	if reqID == "" || ss == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subs[reqID] == nil {
		m.subs[reqID] = make(map[*sdk.ServerSession]any)
	}
	m.subs[reqID][ss] = token
}

func (m *eventMux) unsubscribe(reqID string, ss *sdk.ServerSession) {
	if reqID == "" || ss == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sessions := m.subs[reqID]
	delete(sessions, ss)
	if len(sessions) == 0 {
		delete(m.subs, reqID)
	}
}

func (m *eventMux) run(ctx context.Context, agg *mcpbridge.Aggregator) {
	if agg == nil {
		return
	}
	progressCh := agg.ProgressEvents()
	logCh := agg.LogEvents()

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-progressCh:
			m.forwardProgress(ctx, ev)
		case ev := <-logCh:
			m.broadcastLog(ctx, ev)
		}
	}
}

func (m *eventMux) forwardProgress(ctx context.Context, ev mcpbridge.ProgressEvent) {
	if ev.ReqID == "" {
		return
	}
	m.mu.RLock()
	targets := m.subs[ev.ReqID]
	m.mu.RUnlock()
	if len(targets) == 0 {
		return
	}

	msg := ev.Message
	if ev.ServerName != "" {
		msg = fmt.Sprintf("[%s] %s", ev.ServerName, msg)
	}

	for ss, token := range targets {
		if ss == nil {
			continue
		}
		params := &sdk.ProgressNotificationParams{
			ProgressToken: token,
			Message:       msg,
			Progress:      ev.Progress,
			Total:         ev.Total,
		}
		_ = ss.NotifyProgress(ctx, params)
	}
}

func (m *eventMux) broadcastLog(ctx context.Context, ev mcpbridge.LogEvent) {
	level := sdk.LoggingLevel(ev.Level)
	if level == "" {
		level = "info"
	}
	data := ev.Data
	if data == nil && ev.Message != "" {
		data = ev.Message
	}
	msg := &sdk.LoggingMessageParams{
		Level:  level,
		Data:   data,
		Logger: ev.ServerName,
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sessions := range m.subs {
		for ss := range sessions {
			if ss == nil {
				continue
			}
			_ = ss.Log(ctx, msg)
		}
	}
}
