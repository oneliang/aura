// Package runtime provides agent delegation functionality.
package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	agentpkg "github.com/oneliang/aura/agent/pkg/agent"
	"github.com/oneliang/aura/core/pkg/engine"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/logger"
)

// findAgent finds an agent by name from the loaded agents.
func (r *AgentRuntime) findAgent(agentName string) (*agentpkg.Agent, error) {
	if r.agentLoader == nil {
		return nil, fmt.Errorf("agent loader not initialized")
	}

	agents := r.agentLoader.GetAgents()
	for _, a := range agents {
		if a.Name == agentName {
			return &a, nil
		}
	}

	return nil, fmt.Errorf("agent '%s' not found", agentName)
}

// buildSubAgentConfig builds the runtime config for a sub-agent.
func buildSubAgentConfig(parent *AgentRuntime, foundAgent *agentpkg.Agent, task string) *RuntimeConfig {
	systemPrompt := fmt.Sprintf("%s\n\n## Task\n\nYou have been delegated the following task. Complete it and provide a clear, comprehensive result.\n\n%s", foundAgent.Body, task)

	subAgentCfg := &RuntimeConfig{
		Config:          parent.config.Config,
		SessionID:       "",
		Role:            fmt.Sprintf("sub-agent-%s", foundAgent.Name),
		DisableTools:    false,
		SystemPrompt:    systemPrompt,
		PermissionMode:  foundAgent.Meta.PermissionMode,
		PermissionLevel: foundAgent.Meta.PermissionLevel,
	}

	// Note: We do NOT apply the sub-agent's LLM model override here.
	// The agent's llm_model field is intended for standalone execution,
	// not for sub-agent delegation. Sub-agents inherit the parent's LLM
	// configuration to ensure compatibility (e.g., an Ollama model like
	// "qwen3:8b" cannot be used with a DashScope/OpenAI endpoint).
	// If a different model is needed, the parent config should be changed.

	// Phase-based permission auto-downgrade:
	// If parent is in exploration/planning phase, auto-downgrade to readonly
	// unless agent explicitly requests independent permissions.
	var permissionMode string
	if parent.agent != nil {
		parentPhase := parent.agent.GetPhase()

		if parentPhase == engine.PhaseExploration || parentPhase == engine.PhasePlanning {
			// Parent in restricted phase
			if foundAgent.Meta.PermissionMode == string(permissions.PermissionIndependent) {
				// Explicit independent: respect it
				permissionMode = string(permissions.PermissionIndependent)
			} else {
				// Auto-downgrade to readonly
				permissionMode = string(permissions.PermissionReadonly)
			}
		} else {
			// Normal phase: use agent's configured mode or default
			permissionMode = foundAgent.Meta.PermissionMode
			if permissionMode == "" {
				permissionMode = string(permissions.PermissionInherit)
			}
		}
	} else {
		// No parent agent: use agent's configured mode
		permissionMode = foundAgent.Meta.PermissionMode
		if permissionMode == "" {
			permissionMode = string(permissions.PermissionInherit)
		}
	}

	subAgentCfg.PermissionMode = permissionMode

	// Apply AgentConfig inheritance
	applyAgentConfigInheritance(subAgentCfg, &foundAgent.Meta)

	return subAgentCfg
}

// applyAgentConfigInheritance applies agent config inheritance rules.
func applyAgentConfigInheritance(cfg *RuntimeConfig, meta *agentpkg.AgentMeta) {
	// PlanningMode: use SubAgent's if specified, otherwise inherit
	if meta.PlanningMode != "" {
		cfg.Agent.PlanningMode = meta.PlanningMode
	}

	// Temperature: use SubAgent's if > 0, otherwise inherit
	if meta.Temperature > 0 {
		cfg.Agent.Temperature = meta.Temperature
	}

	// SummaryTemp: use SubAgent's if > 0, otherwise inherit
	if meta.SummaryTemp > 0 {
		cfg.Agent.SummaryTemp = meta.SummaryTemp
	}
}

// executeSubAgent runs the sub-agent and collects the result.
func executeSubAgent(ctx context.Context, subAgentRuntime *AgentRuntime, task string, subAgentID string, hookEngine *hooks.Engine) (string, error) {
	var (
		mu         sync.Mutex
		result     string
		lastError  error
		eventCount int
	)

	subAgentLog := subAgentRuntime.logger.WithField("subAgentID", subAgentID)
	subAgentLog.Debug("executeSubAgent: starting sub-agent execution", "task_preview", task[:min(len(task), 100)])

	// Start sub-agent runtime event stream
	if err := subAgentRuntime.Start(ctx); err != nil {
		return "", fmt.Errorf("failed to start sub-agent: %w", err)
	}

	// Send task as user input event
	requestID := fmt.Sprintf("subagent_%d", time.Now().UnixNano())
	taskEvent := events.NewEvent(events.EventTypeUserInput, task, requestID)
	if err := subAgentRuntime.SendEvent(ctx, taskEvent); err != nil {
		subAgentRuntime.Stop(ctx)
		return "", fmt.Errorf("failed to send task to sub-agent: %w", err)
	}

	// Process events from stream
	for ev := range subAgentRuntime.Events() {
		eventCount++
		mu.Lock()
		switch ev.Type() {
		case EventTypeResponse:
			result = ev.Content()
		case EventTypeError:
			lastError = fmt.Errorf("%s", ev.Content())
		case EventTypeDone:
			// Done event signals completion
			break
		}
		mu.Unlock()
	}
	subAgentRuntime.Stop(ctx)

	mu.Lock()
	defer mu.Unlock()

	// Fire SubagentStop hook (non-blocking)
	if hookEngine != nil {
		hookEngine.Fire(ctx, hooks.EventSubagentStop, map[string]any{
			"agent_name": subAgentID,
			"task":       task,
			"result":     result,
			"duration":   0, // could track separately if needed
		})
	}

	if lastError != nil {
		return "", fmt.Errorf("sub-agent execution failed: %w", lastError)
	}

	subAgentLog.Debug("executeSubAgent: sub-agent completed", "events", eventCount, "result_len", len(result))
	return result, nil
}

// createAgentDelegateFn creates the agent delegation function.
func (r *AgentRuntime) createAgentDelegateFn(ctx context.Context) func(ctx context.Context, agentName string, task string) (string, error) {
	return func(ctx context.Context, agentName string, task string) (string, error) {
		subAgentID := fmt.Sprintf("sa-%s", uuid.New().String()[:8])
		r.logger.Info("createAgentDelegateFn: delegation started", "agent", agentName, "task_len", len(task), "subAgentID", subAgentID)
		dl := logger.GetDelegationAuditLogger()
		reqID := uuid.New().String()
		startTime := time.Now()

		// Find the agent
		foundAgent, err := r.findAgent(agentName)
		if err != nil {
			r.logger.Warn("createAgentDelegateFn: agent not found", "agent", agentName, "subAgentID", subAgentID, "error", err.Error())
			dl.Error(reqID, agentName, "find_agent", time.Since(startTime).Milliseconds(), err)
			return "", err
		}
		r.logger.Info("createAgentDelegateFn: agent found, starting delegation", "agent", agentName, "subAgentID", subAgentID)
		dl.Start(reqID, r.sessionID, agentName, task)

		// Create per-delegation independent log file
		fileLogger, err := logger.NewDelegationFileLogger(reqID, agentName)
		if err != nil {
			dl.Error(reqID, agentName, "file_logger", time.Since(startTime).Milliseconds(), err)
			// Continue without independent log
		} else {
			defer fileLogger.Close()
		}
		dl.Step(reqID, "find_agent", time.Since(startTime).Milliseconds())

		// Build sub-agent config (inherits parent LLM, applies config inheritance)
		subAgentCfg := buildSubAgentConfig(r, foundAgent, task)
		dl.Step(reqID, "build_config", time.Since(startTime).Milliseconds())

		// Create lightweight sub-agent runtime (shares parent's expensive resources)
		subAgentRuntime, err := NewSubAgentRuntime(r, subAgentCfg, foundAgent.Meta.DisableTools, fileLogger)
		if err != nil {
			if fileLogger != nil {
				fileLogger.Close()
			}
			dl.Error(reqID, agentName, "create_runtime", time.Since(startTime).Milliseconds(), err)
			return "", fmt.Errorf("failed to create sub-agent runtime: %w", err)
		}
		// Tag logger with sub-agent ID for tracking (use fileLogger if available)
		if fileLogger != nil {
			subAgentLogger := fileLogger.Logger().WithModule("sub-agent:" + agentName)
			subAgentRuntime.logger = subAgentLogger
		}
		dl.Step(reqID, "create_runtime", time.Since(startTime).Milliseconds())

		// Lightweight initialization (shares LLM client, skips MCP/tools/disk reads)
		subCtx, subCancel := context.WithTimeout(ctx, 10*time.Minute)
		defer subCancel()

		r.logger.Info("createAgentDelegateFn: initializing sub-agent runtime", "subAgentID", subAgentID, "agent", agentName)
		if err := subAgentRuntime.Initialize(subCtx); err != nil {
			if fileLogger != nil {
				fileLogger.Close()
			}
			dl.Error(reqID, agentName, "initialize", time.Since(startTime).Milliseconds(), err)
			return "", fmt.Errorf("failed to initialize sub-agent: %w", err)
		}
		defer subAgentRuntime.Shutdown()
		dl.Step(reqID, "initialize", time.Since(startTime).Milliseconds())

		// Execute and get result
		result, err := executeSubAgent(subCtx, subAgentRuntime, task, subAgentID, r.hookEngine)
		if err != nil {
			if fileLogger != nil {
				fileLogger.Close()
			}
			dl.Error(reqID, agentName, "execute", time.Since(startTime).Milliseconds(), err)
			return "", err
		}
		dl.Step(reqID, "execute", time.Since(startTime).Milliseconds())

		// Optional reviewer validation (if UseReviewer is enabled)
		if foundAgent.Meta.UseReviewer && r.agentDelegateFn != nil {
			// Use configured reviewer agent name (must be explicitly set)
			reviewerName := foundAgent.Meta.ReviewerAgent
			if reviewerName == "" {
				r.logger.Warn("createAgentDelegateFn: UseReviewer enabled but ReviewerAgent not configured, skipping review")
				// Skip review if reviewer agent name not configured
			} else {
				// Validate reviewer agent exists before delegating
				_, reviewErr := r.findAgent(reviewerName)
				if reviewErr != nil {
					r.logger.Warn("createAgentDelegateFn: reviewer agent not found, skipping review", "reviewer", reviewerName, "error", reviewErr.Error())
					// Skip review if reviewer agent doesn't exist (log warning, don't fail)
				} else {
					r.logger.Debug("createAgentDelegateFn: reviewer agent found, starting review", "reviewer", reviewerName)
					reviewResult, reviewErr := r.agentDelegateFn(ctx, reviewerName, fmt.Sprintf("Review the following sub-agent result for quality and correctness:\n\nAgent: %s\nTask: %s\nResult: %s", agentName, task, result))
					if reviewErr == nil && strings.Contains(reviewResult, "VERDICT: FAIL") {
						if fileLogger != nil {
							fileLogger.Close()
						}
						dl.Error(reqID, agentName, "review", time.Since(startTime).Milliseconds(), fmt.Errorf("reviewer rejected result"))
						return "", fmt.Errorf("sub-agent result rejected by reviewer '%s': %s", reviewerName, extractReviewReason(reviewResult))
					}
					dl.Step(reqID, "review", time.Since(startTime).Milliseconds())
					r.logger.Debug("createAgentDelegateFn: review completed", "reviewer", reviewerName, "passed", reviewErr != nil || !strings.Contains(reviewResult, "VERDICT: FAIL"))
				}
			}
		}

		totalDuration := time.Since(startTime).Milliseconds()
		formattedResult := fmt.Sprintf("【SubAgent %s completed the task】\n\n%s", agentName, result)
		r.logger.Info("createAgentDelegateFn: delegation completed", "agent", agentName, "subAgentID", subAgentID, "duration_ms", totalDuration, "result_len", len(formattedResult))
		dl.Complete(reqID, agentName, totalDuration, "fast", formattedResult)

		return formattedResult, nil
	}
}

// extractReviewReason extracts the reason from a review result.
// Returns default message if reviewResult is empty or no reason found.
func extractReviewReason(reviewResult string) string {
	if reviewResult == "" {
		return "no specific reason provided"
	}
	// Look for patterns like "VERDICT: FAIL - reason" or "Reason: ..."
	lines := strings.Split(reviewResult, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VERDICT: FAIL") {
			// Extract reason after "VERDICT: FAIL"
			reason := strings.TrimPrefix(line, "VERDICT: FAIL")
			reason = strings.TrimSpace(reason)
			if strings.HasPrefix(reason, "-") {
				reason = strings.TrimPrefix(reason, "-")
				reason = strings.TrimSpace(reason)
			}
			if reason != "" {
				return reason
			}
		}
		if strings.HasPrefix(line, "Reason:") || strings.HasPrefix(line, "原因:") {
			reason := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "Reason:"), "原因:"))
			if reason != "" {
				return reason
			}
		}
	}
	return "no specific reason provided"
}