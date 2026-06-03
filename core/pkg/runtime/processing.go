package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/hooks"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"

	"github.com/oneliang/aura/core/pkg/intent"

	habitmodel "github.com/oneliang/aura/habit/pkg/model"
)

// Process processes an input message and returns an event channel.
//
// Runtime Layer Responsibilities:
// - Entry point for user messages (L351: r.memory.Add(User, input))
// - Skill/system message injection (L339: r.memory.AddWithType(System, ...))
// - Event conversion and routing
//
// Engine Layer Responsibilities (delegated to agent.Run):
// - All LLM conversation messages (Thought/Action/Observation/Assistant)
// - Tool execution and result handling
// - Message persistence coordination via SessionMemory
func (r *AgentRuntime) Process(ctx context.Context, input string) (<-chan Event, error) {
	if !r.initialized {
		return nil, fmt.Errorf("runtime not initialized")
	}

	// === DIAGNOSTIC: Measure each phase timing ===
	startTime := time.Now()
	r.logger.Info().Str("phase", "start").Msg("[DIAG] Process: starting")

	// Try intent recognition for non-slash input
	var intentResult *intent.IntentResult
	intentSvc := r.intentService
	if r.skills != nil && r.skills.intentSvc != nil {
		intentSvc = r.skills.intentSvc
	}
	if intentSvc != nil && !strings.HasPrefix(input, "/") {
		intentStart := time.Now()
		var err error
		intentResult, err = intentSvc.Recognize(ctx, input)
		r.logger.Info().Str("phase", "intent").Dur("duration", time.Since(intentStart)).Msg("[DIAG] Intent recognition completed")
		if err != nil {
			r.logger.Warn().Err(err).Str("input", input).Msg("Intent recognition error, continuing with normal processing")
		}
		// Skip skill commands for natural language - skill activation is handled via skill_activate tool
		// Natural language: LLM decides when to call skill_activate based on system prompt skill descriptions
		if intentResult != nil && intentResult.Matched && !strings.HasPrefix(intentResult.Command, "skill_") {
			return r.processIntentCommand(ctx, intentResult)
		}
	}

	// Fire UserPromptSubmit hook via component (blocking) — before skill matching and processing
	userID := r.userID
	sessionID := r.sessionID
	if r.session != nil {
		userID = r.session.GetUserID()
		sessionID = r.session.GetSessionID()
	}
	hookResult, err := r.hooks.FireBlocking(ctx, hooks.EventUserPromptSubmit, map[string]any{
		"input":      input,
		"session_id": sessionID,
		"user_id":    userID,
	})
	r.logger.Info().Str("phase", "hook").Msg("[DIAG] UserPromptSubmit hook completed")
	if err != nil {
		r.logger.Warn().Err(err).Msg("UserPromptSubmit hook error")
	} else if hookResult != nil && r.hooks.GetEngine() != nil && r.hooks.GetEngine().ShouldBlock(hookResult) {
		r.logger.Debug().Str("module", "runtime").Msg("UserPromptSubmit hook blocked processing")
		out := make(chan Event, constants.EventChannelBufferSize)
		go func() {
			defer close(out)
			if hookResult.Parsed != nil && hookResult.Parsed.SystemMessage != "" {
				out <- NewEvent(EventTypeResponse, hookResult.Parsed.SystemMessage)
			} else {
				out <- NewEvent(EventTypeResponse, "Request blocked by hook")
			}
			out <- NewEvent(EventTypeDone, "")
		}()
		return out, nil
	}

	// Skill activation is handled via skill_activate tool
	// LLM calls the tool when it determines a skill is needed
	// No automatic matching/injection here

	r.logger.Info().Str("phase", "pre_llm").Dur("total_duration", time.Since(startTime)).Msg("[DIAG] Pre-LLM phase completed")

	// Create output channel
	out := make(chan Event, constants.EventChannelBufferSize)

	// Add user message to memory
	r.memory.AddWithType(sharedmemory.RoleUser, input, sharedmemory.MessageTypeUser)

	// Run agent - returns per-request event channel
	agentEvents, err := r.agent.Run(ctx, input)
	if err != nil {
		return nil, err
	}

	// Start event conversion goroutine
	// Note: We need this goroutine because agentEvents is <-chan events.Event
	// and we need to convert to Event type
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				err := fmt.Errorf("event pump panicked: %v", rec)
				r.logger.Error().Err(err).Msg("Event pump panic recovered")
			}
			close(out)
		}()

		var toolsUsed []string

		for ev := range agentEvents {
			r.logger.Debug().Str("type", string(ev.Type())).Msg("Runtime: converting event")

			// Track tool usage for habit learning
			if ev.Type() == events.EventTypeToolStart {
				if toolName, ok := ev.Extra()["tool"].(string); ok {
					toolsUsed = append(toolsUsed, toolName)
				}
			}

			convertedEvent := r.convertEvent(ev)
			if convertedEvent == nil {
				continue
			}
			out <- convertedEvent
		}

		// Record action for habit tracking (fire-and-forget)
		if r.habitManager != nil && len(toolsUsed) > 0 {
			go func() {
				action := &habitmodel.Action{
					UserID:    r.userID,
					SessionID: r.sessionID,
					Input:     input,
					ToolsUsed: toolsUsed,
				}
				if err := r.habitManager.RecordAction(context.Background(), r.userID, action); err != nil {
					r.logger.Warn().Err(err).Str("module", "runtime").Msg("Failed to record habit action")
				}
			}()
		}

		r.logger.Debug().Msg("Runtime: agentEvents closed")

		// Explicitly send Done event before closing
		doneEv := NewEvent(EventTypeDone, "")
		out <- doneEv
	}()

	return out, nil
}

// processIntentCommand processes a recognized intent command.
func (r *AgentRuntime) processIntentCommand(ctx context.Context, result *intent.IntentResult) (<-chan Event, error) {
	out := make(chan Event, constants.EventChannelBufferSize)

	go func() {
		defer close(out)

		// Send command matched event
		ev := NewEventWithExtra(EventTypeCommandMatched, result.Command, map[string]any{
			"params":     result.Params,
			"confidence": result.Confidence,
			"source":     result.Source,
		})
		out <- ev

		// Execute command
		cmdResult, err := r.intentService.ExecuteCommand(ctx, result)
		if err != nil {
			errEv := NewEvent(EventTypeError, err.Error())
			out <- errEv
			return
		}

		// Send command result event
		resultEv := NewEvent(EventTypeCommandResult, cmdResult)
		out <- resultEv

		// Send done event to signal completion (consistent with Engine)
		doneEv := NewEvent(EventTypeDone, "")
		out <- doneEv
	}()

	return out, nil
}