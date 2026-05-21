// Package planner provides task planning capabilities.
package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/oneliang/aura/core/pkg/llm"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
)

// Plan format constraints (Claude Code alignment).
const planMaxLines = 40

// Plan represents a task plan (static, no execution state).
type Plan struct {
	Goal           string           `json:"goal"`
	Steps          []Step           `json:"steps"`
	completedSteps map[int]struct{} // completed step indexes (0-based)
}

// Step represents a plan step (static description only).
type Step struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	FilesToCheck []string `json:"files_to_check,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	RiskLevel    string   `json:"risk_level,omitempty"`
}

// Planner creates and manages plans.
type Planner struct {
	client llm.Client
}

// New creates a new planner.
func New(client llm.Client) *Planner {
	return &Planner{client: client}
}

// CreatePlan creates a plan for the given goal, optionally enriched with
// exploration context discovered during a read-only codebase exploration phase.
func (p *Planner) CreatePlan(ctx context.Context, goal string, explorationContext string) (*Plan, error) {
	var explorationBlock string
	if explorationContext != "" {
		explorationBlock = fmt.Sprintf(`
Exploration findings (from codebase read-only exploration):
%s

Use the exploration findings above to avoid duplicating existing implementations.`, explorationContext)
	}

	prompt := fmt.Sprintf(`You are a task planner. Break down the following goal into concrete, actionable steps.%s

Goal: %s

RULES FOR EACH STEP:
- Each step must be a single, concrete action (not a phase or category).
- Specify exactly WHICH files to create or modify.
- Describe WHAT change to make in each file (not just "update X" or "add Y").
- Reference existing functions/types to reuse when applicable.
- Include how to verify the step is correct (e.g., a test, a build command, a visual check).
- Assign risk_level: "low" (pure read/refactor), "medium" (additive change), "high" (delete/rewrite/external API).

CRITICAL: Do NOT write a context/background section. Do NOT restate the goal in the steps. Each step should be specific enough that a developer can execute it without asking follow-up questions.

Respond with a JSON object in this exact format:
{
  "goal": "Restate the goal",
  "steps": [
    {
      "description": "Create pkg/auth/handler.go with login and register handlers",
      "files_to_check": ["pkg/auth/handler.go"],
      "dependencies": ["token_name"],
      "risk_level": "medium"
    }
  ]
}

If you cannot produce valid JSON, provide a numbered list with the same level of specificity:
1. Create pkg/auth/handler.go with login and register handlers (risk: medium, files: pkg/auth/handler.go)`, explorationBlock, goal)

	resp, err := p.client.Complete(ctx, &llm.Request{
		Messages: []llm.Message{
			{
				Role:          "system",
				ContentBlocks: []sharedmemory.ContentBlock{
					sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "You are a helpful task planner."},
				},
			},
			{
				Role:          "user",
				ContentBlocks: []sharedmemory.ContentBlock{
					sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: prompt},
				},
			},
		},
		Thinking: &llm.ThinkingConfig{Enabled: false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Extract text from response ContentBlocks
	var respContent string
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			respContent = tb.Text
			break
		}
	}
	steps := parsePlanJSON(respContent)
	if len(steps) == 0 {
		steps = parseSteps(respContent)
	}
	return &Plan{
		Goal:  goal,
		Steps: steps,
	}, nil
}

type planJSON struct {
	Goal  string         `json:"goal"`
	Steps []planStepJSON `json:"steps"`
}

type planStepJSON struct {
	Description  string   `json:"description"`
	FilesToCheck []string `json:"files_to_check,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	RiskLevel    string   `json:"risk_level,omitempty"`
}

// parsePlanJSON tries to parse the response as a structured JSON plan.
// Returns nil slice if parsing fails, caller should fall back to parseSteps.
func parsePlanJSON(content string) []Step {
	content = strings.TrimSpace(content)
	// Strip markdown code fences if present
	if idx := strings.Index(content, "```"); idx >= 0 {
		content = content[idx+3:]
		if nl := strings.Index(content, "\n"); nl >= 0 {
			content = content[nl+1:]
		}
		if end := strings.LastIndex(content, "```"); end >= 0 {
			content = content[:end]
		}
	}
	var plan planJSON
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil
	}
	steps := make([]Step, 0, len(plan.Steps))
	for i, s := range plan.Steps {
		if s.Description != "" {
			risk := s.RiskLevel
			if risk == "" {
				risk = "medium"
			}
			steps = append(steps, Step{
				ID:           strconv.Itoa(i + 1),
				Description:  s.Description,
				FilesToCheck: s.FilesToCheck,
				Dependencies: s.Dependencies,
				RiskLevel:    risk,
			})
		}
	}
	return steps
}

func parseSteps(content string) []Step {
	var steps []Step

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		stepNum, description := parseNumberedLine(line)
		if description != "" {
			steps = append(steps, Step{
				ID:          strconv.Itoa(stepNum),
				Description: description,
			})
		}
	}

	if len(steps) == 0 {
		steps = append(steps, Step{
			ID:          "1",
			Description: strings.TrimSpace(content),
		})
	}

	return steps
}

// parseNumberedLine tries to parse a line like "1. Step" or "1) Step"
func parseNumberedLine(line string) (int, string) {
	line = strings.TrimSpace(line)

	for _, pattern := range []string{".", ")"} {
		if idx := strings.Index(line, pattern); idx > 0 {
			numStr := line[:idx]
			if num, err := strconv.Atoi(strings.TrimSpace(numStr)); err == nil {
				description := strings.TrimSpace(line[idx+1:])
				if description != "" {
					return num, description
				}
			}
		}
	}

	return 0, ""
}

// GetTotalSteps returns the total number of steps.
func (p *Plan) GetTotalSteps() int {
	return len(p.Steps)
}

// MarkStepCompleted marks a step as completed by 0-based index.
func (p *Plan) MarkStepCompleted(stepIndex int) {
	if p.completedSteps == nil {
		p.completedSteps = make(map[int]struct{})
	}
	if stepIndex >= 0 && stepIndex < len(p.Steps) {
		p.completedSteps[stepIndex] = struct{}{}
	}
}

// IsStepCompleted returns whether a step is completed.
func (p *Plan) IsStepCompleted(stepIndex int) bool {
	if p.completedSteps == nil {
		return false
	}
	_, ok := p.completedSteps[stepIndex]
	return ok
}

// String returns a human-readable representation of the plan.
func (p *Plan) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Plan: %s\n\n", p.Goal))
	for i, step := range p.Steps {
		checkbox := "- [ ]"
		if p.IsStepCompleted(i) {
			checkbox = "- [x]"
		}
		sb.WriteString(fmt.Sprintf("%s %d. %s", checkbox, i+1, step.Description))
		if step.RiskLevel != "" {
			sb.WriteString(fmt.Sprintf("  *(%s)*", step.RiskLevel))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// RevisePlan revises the plan after a failed step at the given index.
// It replaces the failed step and all subsequent steps.
func (p *Planner) RevisePlan(ctx context.Context, plan *Plan, failedStepIndex int, failedStepDesc string, failureReason string) (*Plan, error) {
	prompt := fmt.Sprintf(`A plan step has failed. Revise the plan by replacing the failed step and all subsequent steps.

Original plan:
%s

Failed step (index %d): %s
Failure reason: %s

RULES:
- Keep all steps before index %d exactly as they are.
- Replace step %d and all subsequent steps with new, concrete steps.
- Each new step must be specific: which files to modify, what changes to make, how to verify.
- The new steps should account for the failure reason — do NOT suggest the same approach that already failed.

Respond with JSON:
{"goal": "Same goal", "steps": [{"description": "replacement step with file-level detail", "risk_level": "low|medium|high"}]}
`, plan.String(), failedStepIndex, failedStepDesc, failureReason, failedStepIndex, failedStepIndex)

	resp, err := p.client.Complete(ctx, &llm.Request{
		Messages: []llm.Message{
			{
				Role:          "system",
				ContentBlocks: []sharedmemory.ContentBlock{
					sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "You are a task planner that revises plans after failures."},
				},
			},
			{
				Role:          "user",
				ContentBlocks: []sharedmemory.ContentBlock{
					sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: prompt},
				},
			},
		},
		Thinking: &llm.ThinkingConfig{Enabled: false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to revise plan: %w", err)
	}

	// Extract text from response ContentBlocks
	var respContent string
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			respContent = tb.Text
			break
		}
	}
	steps := parsePlanJSON(respContent)
	if len(steps) == 0 {
		steps = parseSteps(respContent)
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("revision produced no steps")
	}

	// Preserve steps before the failed index
	if failedStepIndex < 0 || failedStepIndex > len(plan.Steps) {
		failedStepIndex = len(plan.Steps)
	}
	preserved := plan.Steps[:failedStepIndex]
	allSteps := append(preserved, steps...)

	return &Plan{
		Goal:  plan.Goal,
		Steps: allSteps,
	}, nil
}

// WriteToFile writes the plan to a Markdown file with checkbox format.
func (p *Plan) WriteToFile(path string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Plan: %s\n\n", p.Goal))
	sb.WriteString("## Steps\n\n")
	for i, step := range p.Steps {
		checkbox := "- [ ]"
		if p.IsStepCompleted(i) {
			checkbox = "- [x]"
		}
		sb.WriteString(fmt.Sprintf("%s %d. %s\n", checkbox, i+1, step.Description))
		if step.RiskLevel != "" {
			sb.WriteString(fmt.Sprintf("   - Risk: %s\n", step.RiskLevel))
		}
		if len(step.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("   - Dependencies: %s\n", strings.Join(step.Dependencies, ", ")))
		}
		if len(step.FilesToCheck) > 0 {
			sb.WriteString(fmt.Sprintf("   - Files: %s\n", strings.Join(step.FilesToCheck, ", ")))
		}
		sb.WriteString("\n")
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// ValidateFormat checks if the plan meets Claude Code format constraints.
// Returns error if plan exceeds max lines or contains prohibited sections.
func (p *Plan) ValidateFormat() error {
	content := p.String()
	lines := strings.Split(content, "\n")

	// Check line count
	if len(lines) > planMaxLines {
		return fmt.Errorf("plan exceeds %d lines (has %d)", planMaxLines, len(lines))
	}

	// Check for prohibited prose sections
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## Context") ||
			strings.HasPrefix(line, "## Background") ||
			strings.HasPrefix(line, "## Overview") {
			return fmt.Errorf("plan contains prohibited prose section: %s", line)
		}
	}

	return nil
}
