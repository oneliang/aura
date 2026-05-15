package planner

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/memory"
)

// Helper function to create a message with text content
func newTextMessage(role, text string) llm.Message {
	msg := llm.Message{Role: role}
	msg.SetContentBlocks([]memory.ContentBlock{
		memory.TextBlock{Type: memory.BlockTypeText, Text: text},
	})
	return msg
}

// Helper function to extract text content from message
func getTextContent(msg llm.Message) string {
	blocks := msg.GetContentBlocks()
	for _, block := range blocks {
		if tb, ok := block.(memory.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// MockLLMClient is a mock implementation of the llm.Client interface.
type MockLLMClient struct {
	CompleteFunc func(ctx context.Context, req *llm.Request) (*llm.Response, error)
	StreamFunc   func(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error)
	EmbedFunc    func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *MockLLMClient) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	if m.CompleteFunc != nil {
		return m.CompleteFunc(ctx, req)
	}
	return &llm.Response{
		Message: newTextMessage("assistant", "1. Step one\n2. Step two\n3. Step three"),
	}, nil
}

func (m *MockLLMClient) Stream(ctx context.Context, req *llm.Request) (<-chan llm.Chunk, error) {
	if m.StreamFunc != nil {
		return m.StreamFunc(ctx, req)
	}
	ch := make(chan llm.Chunk)
	close(ch)
	return ch, nil
}

func (m *MockLLMClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.EmbedFunc != nil {
		return m.EmbedFunc(ctx, texts)
	}
	return [][]float32{}, nil
}

func TestNew(t *testing.T) {
	client := &MockLLMClient{}
	planner := New(client)

	if planner == nil {
		t.Fatal("Expected planner to be created")
	}
	if planner.client != client {
		t.Error("Expected client to be set")
	}
}

func TestPlanner_CreatePlan(t *testing.T) {
	tests := []struct {
		name           string
		goal           string
		explorationCtx string
		wantErr        bool
	}{
		{"simple goal", "Write a hello world program", "", false},
		{"complex goal", "Build a web application with user authentication", "", false},
		{"empty goal", "", "", false},
		{"with exploration context", "Add logging", "Found existing log package in utils/logger.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockLLMClient{}
			planner := New(client)

			ctx := context.Background()
			plan, err := planner.CreatePlan(ctx, tt.goal, tt.explorationCtx)

			if (err != nil) != tt.wantErr {
				t.Fatalf("CreatePlan() error = %v, wantErr %v", err, tt.wantErr)
			}

			if plan == nil {
				t.Fatal("Expected plan to be created")
			}

			if plan.Goal != tt.goal {
				t.Errorf("Expected goal %q, got %q", tt.goal, plan.Goal)
			}
		})
	}
}

func TestPlanner_CreatePlan_Error(t *testing.T) {
	client := &MockLLMClient{
		CompleteFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
			return nil, context.Canceled
		},
	}
	planner := New(client)

	ctx := context.Background()
	plan, err := planner.CreatePlan(ctx, "test goal", "")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if plan != nil {
		t.Error("Expected nil plan on error")
	}
}

func TestParseSteps(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLen  int
		wantDesc string
	}{
		{
			name:     "simple content",
			content:  "Just do it",
			wantLen:  1,
			wantDesc: "Just do it",
		},
		{
			name:    "numbered steps",
			content: "1. First step\n2. Second step\n3. Third step",
			wantLen: 3,
		},
		{
			name:     "empty content",
			content:  "",
			wantLen:  1,
			wantDesc: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := parseSteps(tt.content)

			if len(steps) != tt.wantLen {
				t.Errorf("Expected %d steps, got %d", tt.wantLen, len(steps))
			}

			if tt.wantDesc != "" && steps[0].Description != tt.wantDesc {
				t.Errorf("Expected first step description %q, got %q", tt.wantDesc, steps[0].Description)
			}
		})
	}
}

func TestPlan_GetTotalSteps(t *testing.T) {
	tests := []struct {
		name      string
		plan      *Plan
		wantTotal int
	}{
		{
			name:      "has steps",
			plan:      &Plan{Steps: []Step{{ID: "1", Description: "Step 1"}, {ID: "2", Description: "Step 2"}}},
			wantTotal: 2,
		},
		{
			name:      "no steps",
			plan:      &Plan{Steps: []Step{}},
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := tt.plan.GetTotalSteps()
			if total != tt.wantTotal {
				t.Errorf("Expected %d total steps, got %d", tt.wantTotal, total)
			}
		})
	}
}

func TestPlan_String(t *testing.T) {
	plan := &Plan{
		Goal: "Test goal",
		Steps: []Step{
			{ID: "1", Description: "Step 1"},
			{ID: "2", Description: "Step 2", RiskLevel: "high"},
		},
	}

	s := plan.String()

	if !strings.Contains(s, "Test goal") {
		t.Error("Expected string to contain goal")
	}
	if !strings.Contains(s, "Step 1") {
		t.Error("Expected string to contain step 1")
	}
	if !strings.Contains(s, "high") {
		t.Error("Expected string to contain risk level")
	}
	if !strings.Contains(s, "- [ ]") {
		t.Error("Expected checkbox format")
	}
}

func TestPlan_MarkStepCompleted(t *testing.T) {
	plan := &Plan{
		Goal: "Test goal",
		Steps: []Step{
			{ID: "1", Description: "Step 1"},
			{ID: "2", Description: "Step 2"},
			{ID: "3", Description: "Step 3"},
		},
	}

	if plan.IsStepCompleted(0) {
		t.Error("Step 0 should not be completed initially")
	}
	if plan.IsStepCompleted(99) {
		t.Error("Out-of-bounds step should not be completed")
	}

	plan.MarkStepCompleted(0)
	if !plan.IsStepCompleted(0) {
		t.Error("Step 0 should be completed")
	}
	if plan.IsStepCompleted(1) {
		t.Error("Step 1 should still be pending")
	}

	// Out-of-bounds should not panic
	plan.MarkStepCompleted(-1)
	plan.MarkStepCompleted(99)
}

func TestPlan_String_WithCompletion(t *testing.T) {
	plan := &Plan{
		Goal: "Test goal",
		Steps: []Step{
			{ID: "1", Description: "Done step"},
			{ID: "2", Description: "Pending step"},
		},
	}
	plan.MarkStepCompleted(0)

	s := plan.String()

	if !strings.Contains(s, "- [x] 1. Done step") {
		t.Errorf("Expected completed step with [x], got: %s", s)
	}
	if !strings.Contains(s, "- [ ] 2. Pending step") {
		t.Errorf("Expected pending step with [ ], got: %s", s)
	}
}

func TestPlan_WriteToFile(t *testing.T) {
	dir := t.TempDir()
	plan := &Plan{
		Goal: "Build auth system",
		Steps: []Step{
			{ID: "1", Description: "Read existing code", RiskLevel: "low", Dependencies: []string{"file_read"}, FilesToCheck: []string{"auth.go"}},
			{ID: "2", Description: "Implement JWT", RiskLevel: "high"},
			{ID: "3", Description: "Add tests"},
		},
	}

	path := dir + "/plan.md"
	if err := plan.WriteToFile(path); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read plan file: %v", err)
	}
	content := string(data)

	// Verify checkbox format
	if !strings.Contains(content, "# Plan: Build auth system") {
		t.Error("Expected plan title")
	}
	if !strings.Contains(content, "## Steps") {
		t.Error("Expected Steps section header")
	}
	if !strings.Contains(content, "- [ ] 1. Read existing code") {
		t.Error("Expected checkbox for step 1")
	}
	if !strings.Contains(content, "- [ ] 2. Implement JWT") {
		t.Error("Expected checkbox for step 2")
	}
	if !strings.Contains(content, "- [ ] 3. Add tests") {
		t.Error("Expected checkbox for step 3")
	}
	// Verify metadata
	if !strings.Contains(content, "Risk: high") {
		t.Error("Expected risk metadata")
	}
	if !strings.Contains(content, "Dependencies: file_read") {
		t.Error("Expected dependency metadata")
	}
	if !strings.Contains(content, "Files: auth.go") {
		t.Error("Expected file metadata")
	}
}

func TestPlan_WriteToFile_WithCompletion(t *testing.T) {
	dir := t.TempDir()
	plan := &Plan{
		Goal: "Build feature",
		Steps: []Step{
			{ID: "1", Description: "Read code"},
			{ID: "2", Description: "Write code"},
		},
	}
	plan.MarkStepCompleted(0)

	path := dir + "/plan.md"
	if err := plan.WriteToFile(path); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read plan file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "- [x] 1. Read code") {
		t.Errorf("Expected completed step with [x], got: %s", content)
	}
	if !strings.Contains(content, "- [ ] 2. Write code") {
		t.Errorf("Expected pending step with [ ], got: %s", content)
	}
}

func TestPlanner_RevisePlan(t *testing.T) {
	client := &MockLLMClient{
		CompleteFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
			return &llm.Response{
				Message: newTextMessage("assistant", `{"goal": "Test goal", "steps": [{"description": "Replacement step 1"}, {"description": "Replacement step 2"}]}`),
			}, nil
		},
	}
	planner := New(client)

	plan := &Plan{
		Goal: "Test goal",
		Steps: []Step{
			{ID: "1", Description: "Step 1"},
			{ID: "2", Description: "Step 2"},
			{ID: "3", Description: "Step 3"},
		},
	}
	failedStep := &plan.Steps[1]

	ctx := context.Background()
	revised, err := planner.RevisePlan(ctx, plan, 1, failedStep.Description, "timeout")

	if err != nil {
		t.Fatalf("RevisePlan() error = %v", err)
	}
	if revised == nil {
		t.Fatal("Expected revised plan")
	}
	if revised.Goal != plan.Goal {
		t.Errorf("Expected goal %q, got %q", plan.Goal, revised.Goal)
	}
	// 1 preserved step + 2 new replacement steps
	if len(revised.Steps) != 3 {
		t.Errorf("Expected 3 steps (1 preserved + 2 new), got %d", len(revised.Steps))
	}
	if revised.Steps[0].Description != "Step 1" {
		t.Errorf("Expected preserved first step 'Step 1', got %q", revised.Steps[0].Description)
	}
	if revised.Steps[1].Description != "Replacement step 1" {
		t.Errorf("Expected first new step 'Replacement step 1', got %q", revised.Steps[1].Description)
	}
}

func TestPlanner_RevisePlan_Error(t *testing.T) {
	client := &MockLLMClient{
		CompleteFunc: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
			return nil, context.Canceled
		},
	}
	planner := New(client)

	plan := &Plan{
		Goal:  "Test",
		Steps: []Step{{ID: "1", Description: "Step 1"}},
	}

	ctx := context.Background()
	revised, err := planner.RevisePlan(ctx, plan, 0, plan.Steps[0].Description, "error")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if revised != nil {
		t.Error("Expected nil plan on error")
	}
}

func TestParseNumberedLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantNum  int
		wantDesc string
	}{
		{"dot format", "1. First step", 1, "First step"},
		{"paren format", "2) Second step", 2, "Second step"},
		{"extra spaces", "  3.   Third step", 3, "Third step"},
		{"not numbered", "Just text", 0, ""},
		{"empty", "", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, desc := parseNumberedLine(tt.line)
			if num != tt.wantNum {
				t.Errorf("Expected num %d, got %d", tt.wantNum, num)
			}
			if desc != tt.wantDesc {
				t.Errorf("Expected desc %q, got %q", tt.wantDesc, desc)
			}
		})
	}
}

func TestPlan_ValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid short plan",
			plan: &Plan{
				Goal: "Short goal",
				Steps: []Step{
					{ID: "1", Description: "Step 1"},
					{ID: "2", Description: "Step 2"},
				},
			},
			wantErr: false,
		},
		{
			name: "plan exceeds max lines",
			plan: &Plan{
				Goal: "Very long goal",
				Steps: make([]Step, 50), // 50 steps will exceed 40 lines
			},
			wantErr: true,
			errMsg:  "exceeds",
		},
		{
			name: "plan with Context section",
			plan: &Plan{
				Goal: "Goal\n\n## Context\nSome context here\n\n## Steps\n\n- [ ] 1. Step",
				Steps: []Step{{ID: "1", Description: "Step"}},
			},
			wantErr: true,
			errMsg:  "prohibited prose section",
		},
		{
			name: "plan with Background section",
			plan: &Plan{
				Goal: "Goal\n\n## Background\nSome background\n\n## Steps\n\n- [ ] 1. Step",
				Steps: []Step{{ID: "1", Description: "Step"}},
			},
			wantErr: true,
			errMsg:  "prohibited prose section",
		},
		{
			name: "plan with Overview section",
			plan: &Plan{
				Goal: "Goal\n\n## Overview\nOverview text\n\n## Steps\n\n- [ ] 1. Step",
				Steps: []Step{{ID: "1", Description: "Step"}},
			},
			wantErr: true,
			errMsg:  "prohibited prose section",
		},
		{
			name: "empty plan",
			plan: &Plan{
				Goal:  "",
				Steps: []Step{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Populate step descriptions for the long plan test
			if tt.name == "plan exceeds max lines" {
				for i := range tt.plan.Steps {
					tt.plan.Steps[i] = Step{ID: string(rune(i + 1)), Description: "Step description"}
				}
			}

			err := tt.plan.ValidateFormat()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}