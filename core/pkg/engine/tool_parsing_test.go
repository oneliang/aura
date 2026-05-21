package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/events"
	tools "github.com/oneliang/aura/tools/pkg"
)

func mockTool(name, desc string) *MockTool {
	return &MockTool{
		NameFunc:        func() string { return name },
		DescriptionFunc: func() string { return desc },
	}
}

func TestToolCallsToActions_EmptySlice(t *testing.T) {
	e := &Engine{regTools: make(map[string]tools.Tool)}
	actions := e.toolCallsToActions(nil)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(actions))
	}
}

func TestToolCallsToActions_SingleValidCall(t *testing.T) {
	e := &Engine{regTools: map[string]tools.Tool{
		"calculator": mockTool("calculator", "Do math"),
	}}
	calls := []llm.ToolCall{
		{ID: "call_1", Name: "calculator", Parameters: map[string]any{"expr": "1+1"}},
	}
	actions := e.toolCallsToActions(calls)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Tool != "calculator" {
		t.Errorf("expected tool=calculator, got %s", actions[0].Tool)
	}
	if actions[0].Parameters["expr"] != "1+1" {
		t.Errorf("expected expr=1+1, got %v", actions[0].Parameters["expr"])
	}
}

func TestToolCallsToActions_NonExistentToolSkipped(t *testing.T) {
	e := &Engine{regTools: map[string]tools.Tool{
		"calculator": mockTool("calculator", "Do math"),
	}}
	calls := []llm.ToolCall{
		{ID: "call_1", Name: "nonexistent", Parameters: map[string]any{}},
		{ID: "call_2", Name: "calculator", Parameters: map[string]any{"expr": "2+2"}},
	}
	actions := e.toolCallsToActions(calls)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action (nonexistent skipped), got %d", len(actions))
	}
	if actions[0].Tool != "calculator" {
		t.Errorf("expected tool=calculator, got %s", actions[0].Tool)
	}
}

func TestToolCallsToActions_CommandRoutedThroughInternalCommand(t *testing.T) {
	e := &Engine{regTools: map[string]tools.Tool{
		"internal_command": mockTool("internal_command", "Run commands"),
	}}
	calls := []llm.ToolCall{
		{ID: "call_1", Name: "command_session_create", Parameters: map[string]any{"name": "test"}},
	}
	actions := e.toolCallsToActions(calls)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Tool != "internal_command" {
		t.Errorf("expected tool=internal_command, got %s", actions[0].Tool)
	}
	if actions[0].Parameters["command"] != "command_session_create" {
		t.Errorf("expected command=command_session_create, got %v", actions[0].Parameters["command"])
	}
}

func TestToolCallsToActions_EmptyNameSkipped(t *testing.T) {
	e := &Engine{regTools: map[string]tools.Tool{
		"calculator": mockTool("calculator", "Do math"),
	}}
	calls := []llm.ToolCall{
		{ID: "call_1", Name: "", Parameters: map[string]any{}},
		{ID: "call_2", Name: "calculator", Parameters: map[string]any{"expr": "3+3"}},
	}
	actions := e.toolCallsToActions(calls)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action (empty name skipped), got %d", len(actions))
	}
}

func TestToolCallsToActions_MultipleCalls(t *testing.T) {
	e := &Engine{regTools: map[string]tools.Tool{
		"calculator": mockTool("calculator", "Do math"),
		"datetime":   mockTool("datetime", "Get time"),
	}}
	calls := []llm.ToolCall{
		{ID: "call_1", Name: "calculator", Parameters: map[string]any{"expr": "1+1"}},
		{ID: "call_2", Name: "datetime", Parameters: map[string]any{"format": "RFC3339"}},
	}
	actions := e.toolCallsToActions(calls)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Tool != "calculator" {
		t.Errorf("expected action[0].Tool=calculator, got %s", actions[0].Tool)
	}
	if actions[1].Tool != "datetime" {
		t.Errorf("expected action[1].Tool=datetime, got %s", actions[1].Tool)
	}
}

func TestAccumulateToolCallDelta_SingleDeltaNoPrior(t *testing.T) {
	var existing []llm.ToolCall
	delta := llm.ToolCall{ID: "call_1", Name: "calculator", Parameters: map[string]any{"expr": "1+1"}}
	result := accumulateToolCallDelta(existing, delta)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result))
	}
	if result[0].Name != "calculator" {
		t.Errorf("expected name=calculator, got %s", result[0].Name)
	}
}

func TestAccumulateToolCallDelta_SameIDMerged(t *testing.T) {
	existing := []llm.ToolCall{
		{ID: "call_1", Name: "calculator", Parameters: map[string]any{}},
	}
	delta := llm.ToolCall{ID: "call_1", Name: "", Parameters: map[string]any{"expr": "1+1"}}
	result := accumulateToolCallDelta(existing, delta)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result))
	}
	if result[0].Name != "calculator" {
		t.Errorf("expected name=calculator, got %s", result[0].Name)
	}
	if result[0].Parameters["expr"] != "1+1" {
		t.Errorf("expected expr=1+1, got %v", result[0].Parameters["expr"])
	}
}

func TestAccumulateToolCallDelta_DifferentIDsAppended(t *testing.T) {
	existing := []llm.ToolCall{
		{ID: "call_1", Name: "calculator", Parameters: map[string]any{}},
	}
	delta := llm.ToolCall{ID: "call_2", Name: "datetime", Parameters: map[string]any{}}
	result := accumulateToolCallDelta(existing, delta)
	if len(result) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(result))
	}
	if result[0].Name != "calculator" || result[1].Name != "datetime" {
		t.Errorf("expected calculator+datetime, got %s+%s", result[0].Name, result[1].Name)
	}
}

func TestAccumulateToolCallDelta_EmptyIDAndNameIgnored(t *testing.T) {
	existing := []llm.ToolCall{
		{ID: "call_1", Name: "calculator", Parameters: map[string]any{}},
	}
	delta := llm.ToolCall{ID: "", Name: "", Parameters: map[string]any{}}
	result := accumulateToolCallDelta(existing, delta)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool call (empty delta ignored), got %d", len(result))
	}
}

func TestAccumulateToolCallDelta_LateParamsOverride(t *testing.T) {
	existing := []llm.ToolCall{
		{ID: "call_1", Name: "tool", Parameters: map[string]any{"key": "old"}},
	}
	delta := llm.ToolCall{ID: "call_1", Name: "tool", Parameters: map[string]any{"key": "new", "extra": "val"}}
	result := accumulateToolCallDelta(existing, delta)
	if result[0].Parameters["key"] != "new" {
		t.Errorf("expected key=new, got %v", result[0].Parameters["key"])
	}
	if result[0].Parameters["extra"] != "val" {
		t.Errorf("expected extra=val, got %v", result[0].Parameters["extra"])
	}
}

func TestBuildToolSchemas_NoTools(t *testing.T) {
	e := &Engine{regTools: make(map[string]tools.Tool)}
	schemas := e.buildToolSchemas()
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas, got %d", len(schemas))
	}
}

func TestBuildToolSchemas_TwoTools(t *testing.T) {
	e := &Engine{regTools: map[string]tools.Tool{
		"calculator": mockTool("calculator", "Do math"),
		"datetime":   mockTool("datetime", "Get time"),
	}}
	schemas := e.buildToolSchemas()
	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}

	names := make(map[string]bool)
	for _, s := range schemas {
		names[s.Name] = true
	}
	if !names["calculator"] || !names["datetime"] {
		t.Errorf("expected calculator+datetime schemas, got %v", names)
	}
}

func TestBuildToolSchemas_WithInputSchemaProvider(t *testing.T) {
	customSchema := map[string]any{"type": "object", "properties": map[string]any{"expr": map[string]any{"type": "string"}}}
	e := &Engine{regTools: map[string]tools.Tool{
		"calculator": &MockToolWithInputSchema{name: "calculator", description: "Do math", schema: customSchema},
	}}
	schemas := e.buildToolSchemas()
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
	if schemas[0].Parameters["properties"] == nil {
		t.Error("expected custom properties from InputSchema provider")
	}
}

// MockToolWithInputSchema implements tools.Tool and InputSchema provider.
type MockToolWithInputSchema struct {
	name        string
	description string
	schema      map[string]any
}

func (m *MockToolWithInputSchema) Name() string        { return m.name }
func (m *MockToolWithInputSchema) Description() string { return m.description }
func (m *MockToolWithInputSchema) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "ok"}, nil
}
func (m *MockToolWithInputSchema) InputSchema() map[string]any { return m.schema }

// TestStreamingToolCallAccumulation_Integration verifies that when a streaming
// LLM response fragments a single tool call across multiple chunks (like OpenAI
// does), the engine accumulates deltas by ID and produces exactly 1 action —
// not one per chunk.
func TestStreamingToolCallAccumulation_Integration(t *testing.T) {
	// Simulate OpenAI-style streaming: 5 chunks for a single tool call.
	// Chunks 1-3 carry name/params incrementally, chunk 4 is final text, chunk 5 is done.
	streamCh := make(chan llm.Chunk, 10)
	go func() {
		defer close(streamCh)
		streamCh <- llm.Chunk{Content: "Let me calculate that.\n"}
		// Chunk 2: tool call delta — ID + name only
		streamCh <- llm.Chunk{
			ToolCallDelta: &llm.ToolCall{ID: "call_abc123", Name: "calculator"},
		}
		// Chunk 3: tool call delta — same ID, partial params
		streamCh <- llm.Chunk{
			ToolCallDelta: &llm.ToolCall{ID: "call_abc123", Parameters: map[string]any{"expr": "2+"}},
		}
		// Chunk 4: tool call delta — same ID, complete params (overrides partial)
		streamCh <- llm.Chunk{
			ToolCallDelta: &llm.ToolCall{ID: "call_abc123", Parameters: map[string]any{"expr": "2+2"}},
		}
		// Chunk 5: done
		streamCh <- llm.Chunk{Done: true}
	}()

	client := &MockLLMClient{StreamCh: streamCh}

	toolCallCount := 0
	e, err := New(
		WithClient(client),
		WithMemory(NewMockMemory()),
	)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	e.AddTool(&MockTool{
		NameFunc:        func() string { return "calculator" },
		DescriptionFunc: func() string { return "Do math" },
		ExecuteFunc: func(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
			toolCallCount++
			expr, _ := params["expr"].(string)
			return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: fmt.Sprintf("Result: %s = 4", expr)}, nil
		},
	})

	ctx := context.Background()
	eventCh, err := e.Run(ctx, "What is 2+2?")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Drain events and count ToolStart / ToolEnd
	var toolStartCount, toolEndCount, responseCount int
	for event := range eventCh {
		switch event.Type() {
		case events.EventTypeToolStart:
			toolStartCount++
		case events.EventTypeToolEnd:
			toolEndCount++
		case events.EventTypeResponse:
			responseCount++
		}
	}

	// The key assertion: exactly 1 tool invocation, NOT 3 (one per delta chunk)
	if toolCallCount != 1 {
		t.Errorf("tool was called %d times, want exactly 1 (deltas should be accumulated)", toolCallCount)
	}
	if toolStartCount != 1 {
		t.Errorf("got %d ToolStart events, want 1", toolStartCount)
	}
	if toolEndCount != 1 {
		t.Errorf("got %d ToolEnd events, want 1", toolEndCount)
	}
	if responseCount < 1 {
		t.Error("expected at least 1 Response event")
	}
}

// TestStreamingToolCallAccumulation_MultiTool_Integration verifies that
// streaming deltas for TWO different tool calls (interleaved chunks) are
// accumulated into exactly 2 actions.
func TestStreamingToolCallAccumulation_MultiTool_Integration(t *testing.T) {
	streamCh := make(chan llm.Chunk, 20)
	go func() {
		defer close(streamCh)
		streamCh <- llm.Chunk{Content: "I'll check both.\n"}
		// Tool 1 delta: ID + name
		streamCh <- llm.Chunk{
			ToolCallDelta: &llm.ToolCall{ID: "call_1", Name: "calculator"},
		}
		// Tool 2 delta: ID + name
		streamCh <- llm.Chunk{
			ToolCallDelta: &llm.ToolCall{ID: "call_2", Name: "datetime"},
		}
		// Tool 1 delta: params
		streamCh <- llm.Chunk{
			ToolCallDelta: &llm.ToolCall{ID: "call_1", Parameters: map[string]any{"expr": "1+1"}},
		}
		// Tool 2 delta: params
		streamCh <- llm.Chunk{
			ToolCallDelta: &llm.ToolCall{ID: "call_2", Parameters: map[string]any{"format": "RFC3339"}},
		}
		streamCh <- llm.Chunk{Done: true}
	}()

	client := &MockLLMClient{StreamCh: streamCh}

	executed := make(map[string]int)
	var mu sync.Mutex
	e, err := New(
		WithClient(client),
		WithMemory(NewMockMemory()),
	)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	e.AddTool(&MockTool{
		NameFunc:        func() string { return "calculator" },
		DescriptionFunc: func() string { return "Do math" },
		ExecuteFunc: func(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
			mu.Lock()
			executed["calculator"]++
			mu.Unlock()
			return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "2"}, nil
		},
	})
	e.AddTool(&MockTool{
		NameFunc:        func() string { return "datetime" },
		DescriptionFunc: func() string { return "Get time" },
		ExecuteFunc: func(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
			mu.Lock()
			executed["datetime"]++
			mu.Unlock()
			return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "2026-01-01T00:00:00Z"}, nil
		},
	})

	ctx := context.Background()
	eventCh, err := e.Run(ctx, "Calculate 1+1 and get the time")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var toolStartCount int
	for event := range eventCh {
		if event.Type() == events.EventTypeToolStart {
			toolStartCount++
		}
	}

	mu.Lock()
	calcCount := executed["calculator"]
	dtCount := executed["datetime"]
	mu.Unlock()

	if calcCount != 1 {
		t.Errorf("calculator called %d times, want 1", calcCount)
	}
	if dtCount != 1 {
		t.Errorf("datetime called %d times, want 1", dtCount)
	}
	if toolStartCount != 2 {
		t.Errorf("got %d ToolStart events, want 2", toolStartCount)
	}
}

// MockToolWithOutputSchema implements tools.Tool and OutputSchemaProvider.
type MockToolWithOutputSchema struct {
	name         string
	description  string
	inputSchema  map[string]any
	outputSchema map[string]any
}

func (m *MockToolWithOutputSchema) Name() string                 { return m.name }
func (m *MockToolWithOutputSchema) Description() string          { return m.description }
func (m *MockToolWithOutputSchema) InputSchema() map[string]any  { return m.inputSchema }
func (m *MockToolWithOutputSchema) OutputSchema() map[string]any { return m.outputSchema }
func (m *MockToolWithOutputSchema) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	return &tools.ToolResult{Status: tools.ToolStatusSuccess, Content: "ok"}, nil
}

func TestValidateOutputSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  map[string]any
		data    map[string]any
		wantErr string
	}{
		{
			name: "valid output matches schema",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"create": map[string]any{
						"type":     "object",
						"required": []string{"task_id", "content", "status"},
						"properties": map[string]any{
							"task_id": map[string]any{"type": "number"},
							"content": map[string]any{"type": "string"},
							"status":  map[string]any{"type": "string"},
						},
					},
				},
			},
			data: map[string]any{
				"create": map[string]any{
					"task_id": float64(1),
					"content": "test task",
					"status":  "completed",
				},
			},
			wantErr: "",
		},
		{
			name: "missing required field",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"create": map[string]any{
						"type":     "object",
						"required": []string{"task_id", "content", "status"},
						"properties": map[string]any{
							"task_id": map[string]any{"type": "number"},
							"content": map[string]any{"type": "string"},
							"status":  map[string]any{"type": "string"},
						},
					},
				},
			},
			data: map[string]any{
				"create": map[string]any{
					"task_id": float64(1),
					"content": "test task",
				},
			},
			wantErr: `missing required field "status"`,
		},
		{
			name: "type mismatch",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id": map[string]any{"type": "number"},
				},
			},
			data: map[string]any{
				"task_id": "not a number",
			},
			wantErr: `type mismatch: expected number, got string`,
		},
		{
			name:    "no data field — skipped",
			schema:  map[string]any{"type": "object", "properties": map[string]any{}},
			data:    nil,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &tools.ToolResult{
				Status:  tools.ToolStatusSuccess,
				Content: "ok",
				Data:    tt.data,
			}
			e := &Engine{
				regTools: map[string]tools.Tool{
					"task": &MockToolWithOutputSchema{
						name:         "task",
						description:  "task tool",
						outputSchema: tt.schema,
					},
				},
			}

			errMsg := e.validateOutputSchema("task", result)
			if tt.wantErr == "" && errMsg != "" {
				t.Errorf("unexpected error: %s", errMsg)
			}
			if tt.wantErr != "" && !strings.Contains(errMsg, tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", errMsg, tt.wantErr)
			}
		})
	}
}
