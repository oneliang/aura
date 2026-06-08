package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/planner"
	"github.com/oneliang/aura/core/pkg/rollback"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/memory"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/shared/pkg/tasks"
)

// Phase represents the current execution phase of the engine.
type Phase int

const (
	// PhaseNormal - Normal execution, all tools available.
	PhaseNormal Phase = iota
	// PhaseExploration - Exploration phase (plan mode), only read-only tools.
	PhaseExploration
	// PhasePlanning - Planning phase, designing implementation plan.
	PhasePlanning
)

// stepResult holds the result of executing a plan step.
type stepResult struct {
	output string
	reason string
	failed bool
}

// verifyResult holds the result of the verify phase.
type verifyResult struct {
	passed      bool
	failures    []verifyFailure
	agentReview string
}

// verifyFailure represents a single verification failure.
type verifyFailure struct {
	Command string
	Output  string
	Error   string
}

// runExplicitPlanningLoop runs the explicit planning loop:
// Phase 1: Explore — read-only codebase exploration (forced)
// Phase 2: Design — create plan with exploration context
// Phase 3: Review — user approval, create rollback snapshot
// Phase 4: Execute — execute each plan step (Task tracks execution state)
// Phase 5: Verify — run verification commands
func (e *Engine) runExplicitPlanningLoop(ctx context.Context, eventsCh chan<- events.Event, input string, requestID string) {
	if e.planner == nil {
		e.runReActLoop(ctx, eventsCh, requestID)
		return
	}

	// Enter plan mode - downgrade permissions to read-only
	e.enterPlanMode()
	eventsCh <- events.NewEvent(events.EventTypeEnterPlanMode, "", requestID)
	if e.hookEngine != nil {
		e.hookEngine.Fire(ctx, hooks.EventEnterPlanMode, map[string]any{"request_id": requestID})
	}

	// Phase 1: Exploration — forced read-only codebase discovery
	e.transitionPlanModeState(PlanModeStateExplore)
	eventsCh <- events.NewEvent(events.EventTypeStep, i18n.T("engine.exploration_phase"), requestID)
	explorationSummary := e.runParallelExploration(ctx, eventsCh, input, requestID)
	if explorationSummary != "" {
		e.logger.Debug("runExplicitPlanningLoop: exploration complete", "module", "engine", "requestID", requestID, "summary_len", len(explorationSummary))
	}

	// Phase 2: Design — create plan with exploration context
	e.transitionPlanModeState(PlanModeStateDesign)
	plan, err := e.planner.CreatePlan(ctx, input, explorationSummary)
	if err != nil {
		eventsCh <- events.NewEvent(events.EventTypeError, fmt.Sprintf(i18n.T("engine.plan_create_failed"), err), requestID)
		e.exitPlanMode()
		e.runReActLoop(ctx, eventsCh, requestID)
		return
	}

	e.currentPlan = plan
	e.currentStepIndex = 0

	// Write plan to file with goal-based name (no timestamp)
	safeGoal := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, plan.Goal)
	safeGoal = strings.ToLower(strings.TrimSpace(safeGoal))
	if len(safeGoal) > planFilenameGoalMax {
		safeGoal = safeGoal[:planFilenameGoalMax]
	}
	planPath := fmt.Sprintf("plan-%s.md", safeGoal)
	if err := plan.WriteToFile(planPath); err != nil {
		e.logger.Warn("runExplicitPlanningLoop: failed to write plan file", "module", "engine", "error", err.Error())
	}
	e.SetPlanFilePath(planPath)

	// Build step descriptions slice for TUI widget
	stepDescriptions := make([]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		stepDescriptions = append(stepDescriptions, s.Description)
	}

	eventsCh <- events.NewEventWithExtra(
		events.EventTypePlanCreated,
		plan.String(),
		map[string]any{
			"total_steps": plan.GetTotalSteps(),
			"goal":        plan.Goal,
			"steps":       stepDescriptions,
		},
		requestID,
	)

	// Phase 3: Review — extract critical files and confirm with user
	e.transitionPlanModeState(PlanModeStateReview)
	criticalFiles := e.extractCriticalFiles(plan)
	eventsCh <- events.NewEventWithExtra(
		events.EventTypePlanReviewStart,
		i18n.T("engine.review_phase"),
		map[string]any{
			"total_steps": len(plan.Steps),
			"goal":        plan.Goal,
		},
		requestID,
	)
	if len(criticalFiles) > 0 {
		eventsCh <- events.NewEventWithExtra(
			events.EventTypePlanReviewFiles,
			"",
			map[string]any{
				"files": criticalFiles,
			},
			requestID,
		)
	}

	// Wait for user approval before executing
	if e.planReviewFn != nil {
		approved, err := e.planReviewFn(ctx, plan.Goal, stepDescriptions)
		if err != nil || !approved {
			eventsCh <- events.NewEvent(events.EventTypeResponse, i18n.T("engine.plan_rejected"), requestID)
			e.transitionPlanModeState(PlanModeStateNone)
			return
		}
	}

	// Create rollback snapshot before execution (if rollback manager available)
	var snapshot *rollback.Snapshot
	if e.rollbackMgr != nil {
		snapshotID := fmt.Sprintf("plan-%s-%d", requestID, time.Now().Unix())
		snapshot, err = e.rollbackMgr.CreateSnapshot(ctx, snapshotID, criticalFiles)
		if err != nil {
			e.logger.Warn("runExplicitPlanningLoop: failed to create rollback snapshot", "module", "engine", "error", err.Error())
		} else if snapshot != nil {
			e.rollbackSnapshotID = snapshot.ID
			eventsCh <- events.NewEventWithExtra(
				events.EventTypeSnapshotCreated,
				"",
				map[string]any{
					"snapshot_id": snapshot.ID,
					"files":       snapshot.Files,
				},
				requestID,
			)
		}
	}

	// Exit plan mode - restore permissions for execution
	e.exitPlanMode()
	eventsCh <- events.NewEventWithExtra(
		events.EventTypePlanModeExit,
		"",
		map[string]any{
			"plan_file":    planPath,
			"total_steps": len(plan.Steps),
		},
		requestID,
	)
	if e.hookEngine != nil {
		e.hookEngine.Fire(ctx, hooks.EventExitPlanMode, map[string]any{
			"request_id": requestID,
			"plan_file":  planPath,
		})
	}

	// Phase 4: Execute each step
	for e.currentStepIndex < len(plan.Steps) {
		select {
		case <-ctx.Done():
			eventsCh <- events.NewEvent(
				events.EventTypeResponse,
				constants.MessageInterrupted,
				requestID,
			)
			return
		default:
		}

		step := plan.Steps[e.currentStepIndex]
		eventsCh <- events.NewEventWithExtra(
			events.EventTypePlanStep,
			fmt.Sprintf("Step %d/%d: %s", e.currentStepIndex+1, len(plan.Steps), step.Description),
			map[string]any{
				"step_num":    e.currentStepIndex + 1,
				"total_steps": len(plan.Steps),
				"step_desc":   step.Description,
			},
			requestID,
		)

		// Create task on-demand if not already linked (task tool is suppressed in planning mode)
		task := e.taskList.FindByPlanStepID(step.ID)
		if task == nil {
			created := e.taskList.CreateFromPlanStep(step.ID, plan.Goal, step.Description)
			task = &created
			eventsCh <- events.NewEventWithExtra(
				events.EventTypeTaskCreate,
				task.Content,
				map[string]any{
					"task_id":      task.ID,
					"plan_step_id": step.ID,
				},
				requestID,
			)
		}

		// Update linked task to in_progress
		if _, err := e.taskList.Update(task.ID, tasks.TaskStatusInProgress, ""); err == nil {
			eventsCh <- events.NewEventWithExtra(
				events.EventTypeTaskUpdate,
				task.Content,
				map[string]any{
					"task_id":      task.ID,
					"status":       string(tasks.TaskStatusInProgress),
					"plan_step_id": step.ID,
				},
				requestID,
			)
		}

		result := e.executePlanStep(ctx, eventsCh, step, plan, requestID)

		if result.failed {
			// Update linked task back to pending
			if task := e.taskList.FindByPlanStepID(step.ID); task != nil {
				if _, err := e.taskList.Update(task.ID, tasks.TaskStatusPending, result.reason); err == nil {
					eventsCh <- events.NewEventWithExtra(
						events.EventTypeTaskUpdate,
						task.Content,
						map[string]any{
							"task_id":      task.ID,
							"status":       string(tasks.TaskStatusPending),
							"notes":        result.reason,
							"plan_step_id": step.ID,
						},
						requestID,
					)
				}
			}
			eventsCh <- events.NewEvent(
				events.EventTypeError,
				fmt.Sprintf(i18n.T("engine.step_failed"), result.reason),
				requestID,
			)

			// Offer rollback if snapshot exists
			if snapshot != nil && e.rollbackSnapshotID != "" && e.rollbackConfirmFn != nil {
				confirmed, err := e.rollbackConfirmFn(ctx, e.rollbackSnapshotID, snapshot.Files, result.reason)
				if err == nil && confirmed && e.rollbackMgr != nil {
					// Execute rollback
					rollbackResult, err := e.rollbackMgr.Rollback(ctx, snapshot.ID)
						if err != nil {
							e.logger.Error("runExplicitPlanningLoop: rollback failed", "module", "engine", "error", err.Error())
							eventsCh <- events.NewEventWithExtra(
								events.EventTypeRollbackComplete,
								"",
								map[string]any{
									"success": false,
									"files":   snapshot.Files,
									"error":   err.Error(),
								},
								requestID,
							)
						} else {
							eventsCh <- events.NewEventWithExtra(
								events.EventTypeRollbackComplete,
								"",
								map[string]any{
									"success": true,
									"files":   rollbackResult.Files,
									"message": rollbackResult.Message,
								},
								requestID,
							)
						}
						e.rollbackSnapshotID = ""
				} else {
					e.rollbackSnapshotID = ""
				}
				// Stop execution after failure
				return
			}

			// Attempt revision for high-risk failed steps
			if step.RiskLevel == "high" {
				e.logger.Debug("runExplicitPlanningLoop: attempting plan revision", "module", "engine", "step_id", step.ID)
				if revised, err := e.planner.RevisePlan(ctx, plan, e.currentStepIndex, step.Description, result.reason); err == nil && revised != nil {
					plan = revised
					e.currentPlan = revised

					// Persist revised plan to disk
					if err := plan.WriteToFile(planPath); err != nil {
						e.logger.Warn("runExplicitPlanningLoop: failed to write revised plan file", "module", "engine", "error", err.Error())
					}

					// Emit revised plan to TUI widget
					newStepDescs := make([]string, 0, len(revised.Steps))
					for _, s := range revised.Steps {
						newStepDescs = append(newStepDescs, s.Description)
					}
					eventsCh <- events.NewEventWithExtra(
						events.EventTypePlanCreated,
						revised.String(),
						map[string]any{
							"total_steps": revised.GetTotalSteps(),
							"goal":        revised.Goal,
							"steps":       newStepDescs,
						},
						requestID,
					)

					e.memory.AddWithType(sharedmemory.RoleUser, i18n.T("engine.plan_revised"), memory.MessageTypeSystem)

					continue
				}
			}

			e.currentStepIndex++
			continue
		}

		// Step succeeded — mark as completed in plan and update linked task
		plan.MarkStepCompleted(e.currentStepIndex)

		// Persist progress to disk
		if err := plan.WriteToFile(planPath); err != nil {
			e.logger.Warn("runExplicitPlanningLoop: failed to write plan progress", "module", "engine", "error", err.Error())
		}

		if task := e.taskList.FindByPlanStepID(step.ID); task != nil {
			if _, err := e.taskList.Update(task.ID, tasks.TaskStatusCompleted, result.output); err == nil {
				eventsCh <- events.NewEventWithExtra(
					events.EventTypeTaskUpdate,
					task.Content,
					map[string]any{
						"task_id":      task.ID,
						"status":       string(tasks.TaskStatusCompleted),
						"notes":        result.output,
						"plan_step_id": step.ID,
					},
					requestID,
				)
			}
		}

		e.currentStepIndex++
	}

	// Phase 4: Plan complete
	eventsCh <- events.NewEventWithExtra(
		events.EventTypePlanComplete,
		i18n.T("engine.plan_complete"),
		map[string]any{
			"plan": e.currentPlan.String(),
		},
		requestID,
	)

	// Phase 5: Verify — run verification commands and optionally delegate to reviewer agent
	verifyResult := e.runVerifyPhase(ctx, eventsCh, plan, requestID)
	if !verifyResult.passed {
		eventsCh <- events.NewEvent(events.EventTypeError, fmt.Sprintf("Verification failed: %d failures", len(verifyResult.failures)), requestID)
		// Offer rollback if snapshot exists and verification failed
		if snapshot != nil && e.rollbackSnapshotID != "" && e.rollbackConfirmFn != nil {
			confirmed, err := e.rollbackConfirmFn(ctx, e.rollbackSnapshotID, snapshot.Files, "verification failed")
			if err == nil && confirmed && e.rollbackMgr != nil {
				// Execute rollback
				rollbackResult, err := e.rollbackMgr.Rollback(ctx, snapshot.ID)
				if err != nil {
					e.logger.Error("runExplicitPlanningLoop: verify rollback failed", "module", "engine", "error", err.Error())
					eventsCh <- events.NewEventWithExtra(
						events.EventTypeRollbackComplete,
						"",
						map[string]any{
							"success": false,
							"files":   snapshot.Files,
							"error":   err.Error(),
						},
						requestID,
					)
				} else {
					eventsCh <- events.NewEventWithExtra(
						events.EventTypeRollbackComplete,
						"",
						map[string]any{
							"success": true,
							"files":   rollbackResult.Files,
							"message": rollbackResult.Message,
						},
						requestID,
					)
				}
				e.rollbackSnapshotID = ""
			} else {
				e.rollbackSnapshotID = ""
			}
		}
	} else {
		// Cleanup rollback snapshot on success
		if e.rollbackMgr != nil && e.rollbackSnapshotID != "" {
			e.rollbackMgr.Cleanup(ctx, e.rollbackSnapshotID)
			e.rollbackSnapshotID = ""
		}
	}

	summary := e.generatePlanSummary(ctx, input, requestID)
	eventsCh <- events.NewEvent(events.EventTypeResponse, summary, requestID)
}

// runExplorationPhase forces the LLM to explore the codebase with read-only tools
// before creating a plan. Returns an exploration summary string (may be empty).
func (e *Engine) runExplorationPhase(ctx context.Context, eventsCh chan<- events.Event, input string, requestID string) string {
	// Set read-only tool allowlist, reset when done
	e.SetToolAllowlist(explorationTools)
	defer e.SetToolAllowlist(nil)

	// Inject exploration system prompt into memory
	explorationPrompt := fmt.Sprintf(`You are in EXPLORATION MODE. You may ONLY use read-only tools (file_read, glob, grep, code_navigate) to understand the codebase.

Your goal: %s

IMPORTANT:
1. Do NOT create, modify, or delete any files. Do NOT execute shell commands.
2. Only explore paths within the current working directory or its parents. Do NOT guess absolute paths like /Users/<name>/go/src/... or any path you haven't confirmed exists.
3. Use glob and grep to discover files — do not assume directory structures.
4. Use read-only tools to explore existing implementations before proposing changes.

After exploring, summarize what you found.`, input)

	e.memory.AddWithType(sharedmemory.RoleUser, explorationPrompt, memory.MessageTypeSystem)

	// Run exploration loop (max explorationMaxSteps iterations)
	var lastResponse string
	for i := 0; i < explorationMaxSteps; i++ {
		select {
		case <-ctx.Done():
			e.memory.AddWithType(sharedmemory.RoleUser, i18n.T("engine.exploration_cancelled"), memory.MessageTypeSystem)
			return ""
		default:
		}

		messages := e.buildReActMessages(ctx)
		response, toolCalls, _, err := e.streamAndBufferResponse(ctx, eventsCh, messages, requestID, "")
		// thinking_end is now sent inside streamAndBufferResponse
		if err != nil {
			e.logger.Warn("runExplorationPhase: LLM error", "error", err.Error())
			return ""
		}

		// Check for tool actions (primary: native, fallback: text)
		actions := e.extractActions(response, toolCalls)

		if len(actions) == 0 {
			// LLM gave a direct response — use as exploration summary
			lastResponse = response
			break
		}

		// Execute tool(s) for this iteration
		e.memory.AddWithType(sharedmemory.RoleAssistant, response, memory.MessageTypeAction)

		var results []toolResult
		if len(actions) == 1 {
			result, err := e.executeToolWithEvents(ctx, actions[0], eventsCh, requestID)
			results = []toolResult{{Result: result, Err: err}}
		} else {
			// Multi-tool: execute in parallel
			results = e.executeToolsParallel(ctx, actions, eventsCh, requestID)
		}

		// Aggregate observations and record to memory
		observation, _ := aggregateToolObservations(results, actions)
		if observation == "" {
			observation = "(no output)"
		}
		e.memory.AddWithType(sharedmemory.RoleUser, observation, memory.MessageTypeObservation)
		lastResponse = observation
	}

	// Add a final instruction to summarize
	if lastResponse != "" {
		e.memory.AddWithType(sharedmemory.RoleUser, i18n.T("engine.summarize_discovery"), memory.MessageTypeSystem)
		messages := e.buildReActMessages(ctx)
		req := e.buildRequest(messages)
		resp, err := e.client.Complete(ctx, req)
		if err == nil {
			// Extract text from ContentBlocks
			for _, block := range resp.Message.GetContentBlocks() {
				if tb, ok := block.(memory.TextBlock); ok {
					return tb.Text
				}
			}
		}
	}

	return lastResponse
}

// executePlanStep executes a single plan step using ReAct-style tool execution.
func (e *Engine) executePlanStep(ctx context.Context, eventsCh chan<- events.Event, step planner.Step, plan *planner.Plan, requestID string) stepResult {
	// Add step context with current plan status to guide the LLM
	e.memory.AddWithType(sharedmemory.RoleUser, fmt.Sprintf("Current plan:\n%s\n\nExecute step %d/%d: %s", plan.String(), e.currentStepIndex+1, len(plan.Steps), step.Description), memory.MessageTypeSystem)

	// Use ReAct-style loop for this step
	messages := e.buildReActMessages(ctx)


	response, toolCalls, err := e.getCompleteResponse(ctx, messages, eventsCh, requestID)
	if err != nil {
		return stepResult{failed: true, reason: err.Error()}
	}

	// Send ThinkingEnd event after LLM response is complete

	// Extract actions: prefer structured tool calls, fall back to text parsing
	actions := e.extractActions(response, toolCalls)

	if len(actions) == 0 {
		// LLM provided direct answer for this step
		e.memory.AddWithType(sharedmemory.RoleAssistant, response, memory.MessageTypeThought)
		e.memory.AddWithType(sharedmemory.RoleUser, fmt.Sprintf(i18n.T("engine.step_completed"), response), memory.MessageTypeObservation)
		eventsCh <- events.NewEvent(events.EventTypeResponse, response, requestID)
		return stepResult{output: response}
	}

	// Execute tool(s) for this step
	for len(actions) > 0 {
		// Check cancellation
		select {
		case <-ctx.Done():
			return stepResult{failed: true, reason: "cancelled"}
		default:
		}

		// Execute actions (parallel for >1, serial for 1)
		var results []toolResult
		if len(actions) == 1 {
			results = e.executeToolsSerial(ctx, actions, eventsCh, requestID)
		} else {
			results = e.executeToolsParallel(ctx, actions, eventsCh, requestID)
		}

		// Record action and observations to memory
		e.memory.AddWithType(sharedmemory.RoleAssistant, response, memory.MessageTypeAction)

		// Aggregate observations and check for errors
		observation, firstErr := aggregateToolObservations(results, actions)
		if firstErr != nil {
			return stepResult{failed: true, reason: firstErr.Error()}
		}
		e.memory.AddWithType(sharedmemory.RoleUser, observation, memory.MessageTypeObservation)

		// Get next action
		messages = e.buildReActMessages(ctx)
		response, toolCalls, err = e.getCompleteResponse(ctx, messages, eventsCh, requestID)
		if err != nil {
			return stepResult{failed: true, reason: err.Error()}
		}

		actions = e.extractActions(response, toolCalls)
	}

	// Step completed with final response
	e.memory.AddWithType(sharedmemory.RoleAssistant, response, memory.MessageTypeAssistant)
	eventsCh <- events.NewEvent(events.EventTypeResponse, response, requestID)
	return stepResult{output: response}
}

// generatePlanSummary generates a summary of the plan execution.
func (e *Engine) generatePlanSummary(ctx context.Context, originalInput string, requestID string) string {
	if e.currentPlan == nil {
		return i18n.T("engine.task_completed")
	}

	// Build summary prompt
	prompt := fmt.Sprintf(`Based on the following original task and executed plan, provide a brief summary of what was accomplished.

Task: %s

%s

Provide a concise 2-3 sentence summary of the completed work.`, originalInput, e.currentPlan.String())

	messages := []llm.Message{
		{
			Role:          memory.RoleSystem,
			ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: i18n.T("engine.summary_assistant_prompt")}},
		},
		{
			Role:          memory.RoleUser,
			ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: prompt}},
		},
	}

	resp, err := e.client.Complete(ctx, &llm.Request{Messages: messages, Thinking: e.config.Thinking})
	if err != nil {
		return fmt.Sprintf("Plan completed.\n\n%s", e.currentPlan.String())
	}

	// Extract text from ContentBlocks
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(memory.TextBlock); ok {
			return tb.Text
		}
	}
	return fmt.Sprintf("Plan completed.\n\n%s", e.currentPlan.String())
}

// isComplexTask determines if a task requires explicit planning.
// Uses simple heuristics: task length, presence of multiple verbs, etc.
func (e *Engine) isComplexTask(input string) bool {
	// Simple heuristics for task complexity
	words := strings.Fields(input)

	// Long tasks likely need planning
	if len(words) > complexTaskWordThreshold {
		return true
	}

	// Multiple conjunctions suggest multiple subtasks
	conjunctions := []string{"and", "then", "after", "before", "also", "同时", "然后", "接着"}
	count := 0
	for _, conj := range conjunctions {
		if strings.Contains(strings.ToLower(input), conj) {
			count++
		}
	}
	if count >= 2 {
		return true
	}

	// Multiple action verbs suggest complexity
	actionPatterns := []string{"create", "build", "analyze", "compare", "write", "implement", "design", "分析", "创建", "编写", "实现"}
	actionCount := 0
	for _, pattern := range actionPatterns {
		if strings.Contains(strings.ToLower(input), pattern) {
			actionCount++
		}
	}
	if actionCount >= 2 {
		return true
	}

	return false
}

// extractCriticalFiles extracts unique file paths from plan steps.
// Used during Review phase to show which files will be modified.
func (e *Engine) extractCriticalFiles(plan *planner.Plan) []string {
	files := make(map[string]struct{})
	for _, step := range plan.Steps {
		for _, f := range step.FilesToCheck {
			files[f] = struct{}{}
		}
	}
	result := make([]string, 0, len(files))
	for f := range files {
		result = append(result, f)
	}
	return result
}

// runVerifyPhase executes verification commands after plan execution completes.
// Phase 5 of the 5-phase planning workflow.
func (e *Engine) runVerifyPhase(ctx context.Context, eventsCh chan<- events.Event, plan *planner.Plan, requestID string) verifyResult {
	eventsCh <- events.NewEvent(events.EventTypePlanVerifyStart, i18n.T("engine.verify_phase"), requestID)

	result := verifyResult{
		passed:   true,
		failures: []verifyFailure{},
	}

	// Get verify commands from config
	verifyCommands := e.config.PlanConfig.VerifyCommands
	if len(verifyCommands) == 0 {
		// No verify commands configured, skip command verification
		eventsCh <- events.NewEventWithExtra(events.EventTypePlanVerifyEnd, "", map[string]any{
			"passed":      true,
			"skip_reason": "no_commands_configured",
		}, requestID)
	} else {
		// Execute each verify command
		for _, cmdStr := range verifyCommands {
			// Parse command (split by space, simple approach)
			parts := strings.Fields(cmdStr)
			if len(parts) == 0 {
				continue
			}

			cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
			output, err := cmd.CombinedOutput()

			if err != nil {
				result.passed = false
				result.failures = append(result.failures, verifyFailure{
					Command: cmdStr,
					Output:  string(output),
					Error:   err.Error(),
				})
			}

			// Emit result for each command
			eventsCh <- events.NewEventWithExtra(events.EventTypePlanVerifyResult, "", map[string]any{
				"command": cmdStr,
				"passed":  err == nil,
				"output":  string(output),
			}, requestID)
		}
	}

	// Optional: delegate to code-reviewer agent for additional review (on failures)
	if e.config.PlanConfig.UseReviewerAgent && len(result.failures) > 0 && e.agentDelegateFn != nil {
		reviewPrompt := fmt.Sprintf("Review the following plan execution that had verification failures:\n\nPlan: %s\n\nFailures: %v", plan.String(), result.failures)
		agentReview, err := e.agentDelegateFn(ctx, "code-reviewer", reviewPrompt)
		if err == nil {
			result.agentReview = agentReview
		}
	}

	// Auto-trigger verification-specialist agent for attack-style verification
	if e.agentDelegateFn != nil {
		verifyPrompt := fmt.Sprintf("Verify the implementation of the following plan using attack-style verification (boundary values, concurrency, idempotency, edge cases):\n\nPlan Goal: %s\n\nSteps Executed: %d", plan.Goal, len(plan.Steps))
		agentVerifyResult, err := e.agentDelegateFn(ctx, "verification-specialist", verifyPrompt)
		if err == nil && agentVerifyResult != "" {
			// Append verification result to agent review
			if result.agentReview != "" {
				result.agentReview += "\n\n--- Verification Specialist Report ---\n" + agentVerifyResult
			} else {
				result.agentReview = agentVerifyResult
			}
		}
	}

	// Emit verify end event
	eventsCh <- events.NewEventWithExtra(events.EventTypePlanVerifyEnd, "", map[string]any{
		"passed":      result.passed,
		"failures":    len(result.failures),
		"agent_review": result.agentReview != "",
	}, requestID)

	return result
}

// askUserQuestion prompts the user with a question during plan review.
// Supports text input, single choice, and multi-choice questions.
// Returns the user's answer or error if cancelled/timeout.
func (e *Engine) askUserQuestion(ctx context.Context, eventsCh chan<- events.Event, question string, options []events.QuestionOption, questionType string, requestID string) (*events.QuestionResponse, error) {
	if e.askUserQuestionFn != nil {
		return e.askUserQuestionFn(ctx, question, options, questionType)
	}
	return nil, fmt.Errorf("no ask user question handler configured")
}

// generateClarifyingQuestions generates questions to clarify plan details.
// Returns a list of questions based on plan analysis.
func (e *Engine) generateClarifyingQuestions(ctx context.Context, plan *planner.Plan) []ClarifyingQuestion {
	// For now, return empty list - can be extended to use LLM for question generation
	// This will be enhanced in future iterations
	return []ClarifyingQuestion{}
}

// ClarifyingQuestion represents a question to ask during plan review.
type ClarifyingQuestion struct {
	ID               string
	Question         string
	Type             string // "text", "choice"
	Options          []events.QuestionOption
	RequiresRevision bool   // Whether answer should trigger plan revision
}

// ExplorationTask represents a single exploration task for parallel execution.
type ExplorationTask struct {
	Area   string // Area/module to explore
	Prompt string // Task description for the explorer agent
}

// ExplorationResult represents the result of an exploration task.
type ExplorationResult struct {
	Area     string // Area explored
	Findings string // Summary of findings
	Error    error  // Error if exploration failed
}

// runParallelExploration runs multiple exploration tasks in parallel using agent delegation.
// Decomposes the input into exploration areas and dispatches to explorer agents concurrently.
// Returns an aggregated summary of all findings.
func (e *Engine) runParallelExploration(ctx context.Context, eventsCh chan<- events.Event, input string, requestID string) string {
	// Check if parallel exploration is enabled and agent delegation is available
	if !e.config.PlanConfig.ParallelExplore || e.agentDelegateFn == nil {
		// Fall back to standard exploration
		return e.runExplorationPhase(ctx, eventsCh, input, requestID)
	}

	// Decompose exploration areas from the input
	areas := e.decomposeExplorationAreas(ctx, input)
	if len(areas) == 0 {
		// No areas identified, fall back to standard exploration
		return e.runExplorationPhase(ctx, eventsCh, input, requestID)
	}

	// Limit to max parallel explorers
	maxExplorers := e.config.PlanConfig.MaxParallelExplore
	if maxExplorers <= 0 {
		maxExplorers = defaultMaxParallelExplore
	}
	if len(areas) > maxExplorers {
		areas = areas[:maxExplorers]
	}

	e.logger.Debug("runParallelExploration: starting parallel exploration", "module", "engine", "areas", len(areas))

	// Create exploration tasks
	tasks := make([]ExplorationTask, len(areas))
	for i, area := range areas {
		tasks[i] = ExplorationTask{
			Area:   area,
			Prompt: fmt.Sprintf("Explore the %s area for: %s. Focus on understanding existing patterns, file structure, and relevant implementations. Report your findings concisely.", area, input),
		}
	}

	// Execute in parallel with semaphore
	results := e.executeParallelExploration(ctx, tasks, eventsCh, requestID, maxExplorers)

	// Aggregate results
	return e.aggregateExplorationResults(ctx, results)
}

// decomposeExplorationAreas identifies distinct areas to explore based on the input.
// Uses simple heuristics to split the task into parallel exploration targets.
func (e *Engine) decomposeExplorationAreas(ctx context.Context, input string) []string {
	// Use LLM to decompose exploration areas
	prompt := fmt.Sprintf(`Analyze this task and identify distinct codebase areas that should be explored in parallel:

Task: %s

List up to 5 specific areas/modules/components to explore. Each area should be:
1. A distinct part of the codebase (e.g., "authentication module", "database layer", "API handlers")
2. Likely to contain relevant code for this task
3. Independent enough to explore separately

Respond with a JSON array of area names, nothing else. Example: ["authentication", "database", "api-handlers"]`, input)

	messages := []llm.Message{
		{
			Role:          memory.RoleSystem,
			ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: "You are a codebase exploration coordinator. Respond only with valid JSON arrays."}},
		},
		{
			Role:          memory.RoleUser,
			ContentBlocks: []memory.ContentBlock{memory.TextBlock{Type: memory.BlockTypeText, Text: prompt}},
		},
	}

	resp, err := e.client.Complete(ctx, &llm.Request{Messages: messages})
	if err != nil {
		e.logger.Warn("decomposeExplorationAreas: LLM error", "error", err.Error())
		return nil
	}

	// Parse JSON array from response
	var content string
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(memory.TextBlock); ok {
			content = tb.Text
			break
		}
	}
	content = strings.TrimSpace(content)
	// Remove code fence if present
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 1 {
			// Find content between fences
			var jsonContent strings.Builder
			for _, line := range lines {
				if !strings.HasPrefix(line, "```") {
					jsonContent.WriteString(line)
				}
			}
			content = jsonContent.String()
		}
	}

	var areas []string
	if err := json.Unmarshal([]byte(content), &areas); err != nil {
		e.logger.Warn("decomposeExplorationAreas: JSON parse error", "error", err.Error(), "content", content)
		return nil
	}

	return areas
}

// executeParallelExploration runs exploration tasks concurrently with a semaphore limit.
func (e *Engine) executeParallelExploration(ctx context.Context, tasks []ExplorationTask, eventsCh chan<- events.Event, requestID string, maxConcurrent int) []ExplorationResult {
	results := make([]ExplorationResult, len(tasks))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, task := range tasks {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, t ExplorationTask) {
			defer wg.Done()
			defer func() { <-sem }()

			// Delegate to explorer agent
			findings, err := e.agentDelegateFn(ctx, "explorer", t.Prompt)

			mu.Lock()
			results[idx] = ExplorationResult{
				Area:     t.Area,
				Findings: findings,
				Error:    err,
			}
			mu.Unlock()

			// Emit progress event
			status := "completed"
			if err != nil {
				status = "failed"
			}
			eventsCh <- events.NewEventWithExtra(events.EventTypeStep, "", map[string]any{
				"exploration_area": t.Area,
				"status":           status,
			}, requestID)

			e.logger.Debug("executeParallelExploration: task completed", "module", "engine", "area", t.Area, "success", err == nil)
		}(i, task)
	}

	wg.Wait()
	return results
}

// aggregateExplorationResults combines findings from multiple exploration tasks.
func (e *Engine) aggregateExplorationResults(ctx context.Context, results []ExplorationResult) string {
	var sb strings.Builder
	sb.WriteString("## Exploration Summary\n\n")

	for _, r := range results {
		if r.Error != nil {
			sb.WriteString(fmt.Sprintf("### %s (Error)\n%s\n\n", r.Area, r.Error))
		} else if r.Findings != "" {
			sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", r.Area, r.Findings))
		}
	}

	return sb.String()
}
