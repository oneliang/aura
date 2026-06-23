package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	corememory "github.com/oneliang/aura/core/pkg/memory"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/memory"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	tools "github.com/oneliang/aura/tools/pkg"
)

// toolResult holds the outcome of a single tool execution.
type toolResult struct {
	Result *tools.ToolResult
	Err    error
}

// accumulateToolCallDelta merges a streaming tool call delta into existing ToolCalls.
// Deltas with the same ID are merged (later values override earlier).
// Deltas with no ID and no name are ignored (partial fragments).
func accumulateToolCallDelta(existing []llm.ToolCall, delta llm.ToolCall) []llm.ToolCall {
	if delta.ID != "" {
		for i, tc := range existing {
			if tc.ID == delta.ID {
				if delta.Name != "" {
					existing[i].Name = delta.Name
				}
				if delta.Parameters != nil {
					if existing[i].Parameters == nil {
						existing[i].Parameters = make(map[string]any)
					}
					for k, v := range delta.Parameters {
						existing[i].Parameters[k] = v
					}
				}
				return existing
			}
		}
		return append(existing, delta)
	}
	// No ID — append if it has a name (non-streaming provider)
	if delta.Name != "" {
		return append(existing, delta)
	}
	return existing
}

// executeToolsParallel executes multiple actions concurrently with a semaphore.
// Returns a slice of results (one per action, in order).
func (e *Engine) executeToolsParallel(ctx context.Context, actions []*ToolAction, eventsCh chan<- events.Event, requestID string) []toolResult {
	maxParallel := e.config.MaxParallelTools
	if maxParallel <= 0 {
		maxParallel = defaultMaxParallelTools
	}

	cwd, _ := os.Getwd()
	results := make([]toolResult, len(actions))
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup

	for i, action := range actions {
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore slot
		go func(idx int, a *ToolAction) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			result, err := e.executeToolWithEvents(ctx, a, eventsCh, requestID)
			if err != nil {
				e.hookEngine.FireWithToolName(ctx, hooks.EventPostToolUseFail, a.Tool, map[string]any{
					"tool_name":  a.Tool,
					"tool_input": a.Parameters,
					"error":      err.Error(),
					"directory":  cwd,
				})
				results[idx] = toolResult{Err: err}
				return
			}
			e.hookEngine.FireWithToolName(ctx, hooks.EventPostToolUse, a.Tool, map[string]any{
				"tool_name":   a.Tool,
				"tool_input":  a.Parameters,
				"tool_result": result,
				"directory":   cwd,
			})
			results[idx] = toolResult{Result: result}
		}(i, action)
	}

	wg.Wait()
	return results
}

// executeToolsSerial executes actions sequentially for backward compatibility.
// Returns a slice of results (one per action, in order).
func (e *Engine) executeToolsSerial(ctx context.Context, actions []*ToolAction, eventsCh chan<- events.Event, requestID string) []toolResult {
	cwd, _ := os.Getwd()
	results := make([]toolResult, len(actions))
	for i, action := range actions {
		result, err := e.executeToolWithEvents(ctx, action, eventsCh, requestID)
		if err != nil {
			e.hookEngine.FireWithToolName(ctx, hooks.EventPostToolUseFail, action.Tool, map[string]any{
				"tool_name":  action.Tool,
				"tool_input": action.Parameters,
				"error":      err.Error(),
				"directory":  cwd,
			})
			results[i] = toolResult{Err: err}
			continue
		}
		e.hookEngine.FireWithToolName(ctx, hooks.EventPostToolUse, action.Tool, map[string]any{
			"tool_name":   action.Tool,
			"tool_input":  action.Parameters,
			"tool_result": result,
			"directory":   cwd,
		})
		results[i] = toolResult{Result: result}
	}
	return results
}

// streamAndBufferResponse streams LLM response while buffering for action parsing.
// Each chunk is sent as EventTypeResponseChunk for real-time display.
// toolChoice controls whether the LLM should auto-select or be forced to use tools
// ("auto"|"required"|"none"). Set to "required" when previous step had no actions
// to force tool usage. For providers that don't support tool_choice, it is ignored.
// Returns the complete response, any tool calls from the LLM, accumulated thinking content, and error.
func (e *Engine) streamAndBufferResponse(ctx context.Context, eventsCh chan<- events.Event, messages []llm.Message, requestID string, toolChoice string) (string, []llm.ToolCall, string, error) {
	req := e.buildRequest(messages)
	req.ToolChoice = toolChoice // Override tool choice for this request
	ch, err := e.client.Stream(ctx, req)
	if err != nil {
		return "", nil, "", err
	}

	var fullResponse strings.Builder
	var thinkingContent strings.Builder // NEW: accumulate thinking content
	var skipFinalAnswerPrefix bool = true // Track if we should skip "Final Answer:" prefix
	var chunkCount int
	var toolCalls []llm.ToolCall

	// Track active states for independent event streams
	var thinkingActive bool // reasoning_content or thinking tags stream active
	var responseActive bool // content stream active

	// Filter to strip LLM thinking tags from stream chunks
	tf := &thinkingFilter{}

	for chunk := range ch {
		chunkCount++
		// Log chunk content for debugging
		e.logger.Debug("streamAndBufferResponse: received chunk", "chunk", chunkCount, "content", chunk.Content, "reasoning", chunk.ReasoningContent)
		// Check for cancellation during streaming
		select {
		case <-ctx.Done():
			// Close any active streams before returning
			if thinkingActive {
				eventsCh <- events.NewEvent(events.EventTypeThinkingEnd, "", requestID)
			}
			if responseActive {
				eventsCh <- events.NewEvent(events.EventTypeResponseEnd, "", requestID)
			}
			eventsCh <- events.NewEvent(
				events.EventTypeResponse,
				constants.MessageInterrupted,
				requestID,
			)
			return fullResponse.String(), toolCalls, thinkingContent.String(), ctx.Err()
		default:
		}

		// Handle reasoning_content (OpenAI native reasoning) - independent stream
		if chunk.ReasoningContent != "" {
			// NEW: Accumulate thinking content for storage
			thinkingContent.WriteString(chunk.ReasoningContent)

			// Type transition: response -> thinking
			if responseActive {
				eventsCh <- events.NewEvent(events.EventTypeResponseEnd, "", requestID)
				responseActive = false
			}
			// Start thinking stream if not active
			if !thinkingActive {
				eventsCh <- events.NewEvent(events.EventTypeThinkingStart, "", requestID)
				thinkingActive = true
			}
			eventsCh <- events.NewEvent(events.EventTypeThinkingChunk, chunk.ReasoningContent, requestID)
		}

		// Handle tool_calls - ends both active streams, then accumulates
		if chunk.ToolCallDelta != nil {
			if thinkingActive {
				eventsCh <- events.NewEvent(events.EventTypeThinkingEnd, "", requestID)
				thinkingActive = false
			}
			if responseActive {
				eventsCh <- events.NewEvent(events.EventTypeResponseEnd, "", requestID)
				responseActive = false
			}
			toolCalls = accumulateToolCallDelta(toolCalls, *chunk.ToolCallDelta)
		}

		// Handle content (may contain inline thinking tags) - independent stream
		if chunk.Content != "" {
			content := chunk.Content

			// Strip LLM thinking content from inline tags
			cleaned, thinking := tf.stripThinking(content)

			// Emit captured thinking content from inline tags
			if thinking != "" {
				// NEW: Accumulate thinking content for storage
				thinkingContent.WriteString(thinking)

				// Type transition: response -> thinking
				if responseActive {
					eventsCh <- events.NewEvent(events.EventTypeResponseEnd, "", requestID)
					responseActive = false
				}
				// Start thinking stream if not active
				if !thinkingActive {
					eventsCh <- events.NewEvent(events.EventTypeThinkingStart, "", requestID)
					thinkingActive = true
				}
				eventsCh <- events.NewEvent(events.EventTypeThinkingChunk, thinking, requestID)
				// If thinking tag ended, close thinking stream immediately
				if !tf.IsInThinking() {
					eventsCh <- events.NewEvent(events.EventTypeThinkingEnd, "", requestID)
					thinkingActive = false
				}
			}

			// Handle cleaned content (response stream)
			if cleaned != "" {
				// Filter out "Final Answer:" prefix if at the beginning of response
				if skipFinalAnswerPrefix {
					cleaned = finalAnswerPattern.ReplaceAllString(cleaned, "")
					skipFinalAnswerPrefix = false // Only check first chunk
				}

				// Strip code fence tags (e.g., ```tool_call, ```)
				cleaned = codeFencePattern.ReplaceAllString(cleaned, "")

				// Buffer for action parsing
				fullResponse.WriteString(cleaned)

				// Type transition: thinking -> response
				if thinkingActive {
					eventsCh <- events.NewEvent(events.EventTypeThinkingEnd, "", requestID)
					thinkingActive = false
				}
				// Start response stream if not active
				if !responseActive {
					eventsCh <- events.NewEvent(events.EventTypeResponseStart, "", requestID)
					responseActive = true
				}
				eventsCh <- events.NewEvent(events.EventTypeResponseChunk, cleaned, requestID)
			}
		}

		if chunk.Done {
			break
		}
	}

	// Emit any remaining pending thinking content (edge case: closing tag never arrived)
	if pending := tf.extractThinkingContent(); pending != "" {
		if responseActive {
			eventsCh <- events.NewEvent(events.EventTypeResponseEnd, "", requestID)
			responseActive = false
		}
		if !thinkingActive {
			eventsCh <- events.NewEvent(events.EventTypeThinkingStart, "", requestID)
			thinkingActive = true
		}
		eventsCh <- events.NewEvent(events.EventTypeThinkingChunk, pending, requestID)
	}

	// Stream ended - close any active streams
	if thinkingActive {
		eventsCh <- events.NewEvent(events.EventTypeThinkingEnd, "", requestID)
	}
	if responseActive {
		eventsCh <- events.NewEvent(events.EventTypeResponseEnd, "", requestID)
	}

	return fullResponse.String(), toolCalls, thinkingContent.String(), nil
}

// runReActLoop runs the ReAct (Reasoning + Acting) loop.
func (e *Engine) runReActLoop(ctx context.Context, eventsCh chan<- events.Event, requestID string) {
	defer e.handleReActLoopPanic(eventsCh, requestID)

	step := 0
	e.logger.Debug("runReActLoop: starting", "module", "engine", "requestID", requestID)

	for {
		// Check for cancellation
		if e.checkReActCancellation(ctx, eventsCh, requestID) {
			return
		}

		// Emit step event and increment
		step++
		e.emitReActStepEvent(eventsCh, step, requestID)

		// Max steps guard
		if e.checkReActMaxSteps(ctx, eventsCh, step, requestID) {
			return
		}

		// Build messages and get LLM response
		response, toolCalls, thinkingContent, ok := e.getReActLLMResponse(ctx, eventsCh, requestID)
		if !ok {
			return
		}

		// Parse actions
		actions := e.extractActions(response, toolCalls)
		e.logReActActions(actions, toolCalls, requestID)

		// Handle actions or final answer
		if len(actions) > 0 {
			if !e.handleReActActions(ctx, eventsCh, actions, toolCalls, response, thinkingContent, requestID) {
				return
			}
			// Trigger compression if memory supports it
			if sessionMem, ok := e.memory.(*corememory.SessionMemory); ok {
				meta, err := sessionMem.MaybeCompact(ctx)
				if err == nil && meta != nil {
					eventsCh <- events.NewEventWithExtra(
						events.EventTypeMemoryCompacted,
						"",
						map[string]any{
							"pre_tokens":  meta.PreTokens,
							"post_tokens": meta.PostTokens,
						},
						requestID,
					)
				}
			}
			continue
		}

		// Final answer - check if hook provides reflection feedback first
		finalResponse := response

		// Fire PreResponse hook (blocking) for external validation
		// If hook provides ReflectionFeedback, skip internal reflection to avoid duplication
		hookProvidedReflection := false
		if e.hookEngine != nil {
			hookResult, _ := e.hookEngine.FireBlocking(ctx, hooks.EventPreResponse, map[string]any{
				"request_id": requestID,
				"response":   response,
			})
			// Check if hookResult is valid before using it
			if hookResult != nil && e.hookEngine.ShouldBlock(hookResult) {
				// Hook blocked the response
				e.logger.Info("PreResponse hook blocked response", "requestID", requestID)
				if hookResult.Parsed != nil && hookResult.Parsed.RetryReason != "" {
					eventsCh <- events.NewEvent(events.EventTypeResponse, fmt.Sprintf("Response blocked: %s", hookResult.Parsed.RetryReason), requestID)
				} else {
					eventsCh <- events.NewEvent(events.EventTypeResponse, "Response blocked by hook", requestID)
				}
				return
			}
			// Check if hook provides reflection feedback
			if hookResult != nil && hookResult.Parsed != nil && hookResult.Parsed.ReflectionFeedback != "" {
				finalResponse = fmt.Sprintf("%s\n\n[Reflection feedback: %s]", response, hookResult.Parsed.ReflectionFeedback)
				hookProvidedReflection = true
				e.logger.Debug("Using hook-provided reflection feedback, skipping internal reflection", "requestID", requestID)
			}
			// Check if hook requests retry (logged for awareness, retry requires loop restart)
			if hookResult != nil && hookResult.Parsed != nil && hookResult.Parsed.ShouldRetry {
				e.logger.Info("PreResponse hook signaled retry request", "requestID", requestID, "reason", hookResult.Parsed.RetryReason)
				// Note: Full retry would require restarting ReAct loop - currently logged but proceeds
			}
		}

		// Only run internal reflection if hook didn't provide feedback
		if e.config.EnableReflection && !hookProvidedReflection {
			finalResponse = e.reflectOnAnswer(ctx, finalResponse, requestID)
		}

		e.handleReActFinalAnswer(eventsCh, finalResponse, thinkingContent, requestID)
		return
	}
}

// handleReActLoopPanic handles panic recovery for ReAct loop.
func (e *Engine) handleReActLoopPanic(eventsCh chan<- events.Event, requestID string) {
	if r := recover(); r != nil {
		err := fmt.Errorf("ReAct loop panicked: %v", r)
		e.logger.Error("ReAct loop panic recovered", "error", err.Error(), "requestID", requestID)
		e.hookEngine.Fire(context.Background(), hooks.EventStopFailure, map[string]any{
			"request_id":  requestID,
			"panic_value": r,
		})

		// Send error event to user (避免静默失败)
		eventsCh <- events.NewEvent(events.EventTypeError, fmt.Sprintf("Internal error occurred: %v. Please retry.", r), requestID)
		// Send Done event to unlock UI (必须发送，否则UI会卡住)
		eventsCh <- events.NewEvent(events.EventTypeDone, "", requestID)
	}
}

// checkReActCancellation checks if context is cancelled and emits interrupt event.
func (e *Engine) checkReActCancellation(ctx context.Context, eventsCh chan<- events.Event, requestID string) bool {
	select {
	case <-ctx.Done():
		e.logger.Debug("runReActLoop: context cancelled", "module", "engine", "requestID", requestID)
		eventsCh <- events.NewEvent(events.EventTypeResponse, constants.MessageInterrupted, requestID)
		return true
	default:
		return false
	}
}

// emitReActStepEvent emits the step progress event.
func (e *Engine) emitReActStepEvent(eventsCh chan<- events.Event, step int, requestID string) {
	e.logger.Debug("runReActLoop: step", "module", "engine", "requestID", requestID, "step", step)
	eventsCh <- events.NewEventWithExtra(events.EventTypeStep, "", map[string]any{"step": step}, requestID)
}

// checkReActMaxSteps checks max steps limit and emits best-effort answer if exceeded.
func (e *Engine) checkReActMaxSteps(ctx context.Context, eventsCh chan<- events.Event, step int, requestID string) bool {
	if !checkMaxSteps(step, e.config.MaxSteps) {
		return false
	}
	emitMaxStepsExceededEvent(eventsCh, e.config.MaxSteps, requestID)
	response := e.generateBestEffortAnswer(ctx, requestID)
	eventsCh <- events.NewEvent(events.EventTypeResponse, response, requestID)
	return true
}

// getReActLLMResponse builds messages and gets streaming LLM response.
// Returns response text, tool calls, thinking content, and success status.
func (e *Engine) getReActLLMResponse(ctx context.Context, eventsCh chan<- events.Event, requestID string) (string, []llm.ToolCall, string, bool) {
	buildStart := time.Now()
	messages := e.buildReActMessages(ctx)
	e.logger.Info("[DIAG] buildReActMessages completed", "phase", "build_messages", "duration", time.Since(buildStart), "message_count", len(messages))

	// Note: ThinkingStart event is now managed inside streamAndBufferResponse
	// to avoid duplicate events and ensure proper lifecycle management

	llmStart := time.Now()
	response, toolCalls, thinkingContent, err := e.streamAndBufferResponse(ctx, eventsCh, messages, requestID, "")
	e.logger.Info("[DIAG] LLM stream completed", "phase", "llm_stream", "duration", time.Since(llmStart), "response_length", len(response))
	if err != nil {
		e.logger.Debug("runReActLoop: LLM error", "module", "engine", "requestID", requestID, "error", err.Error())
		eventsCh <- events.NewEvent(events.EventTypeError, err.Error(), requestID)
		return "", nil, "", false
	}

	// Detect empty response (no response text and no tool calls)
	if response == "" && len(toolCalls) == 0 {
		if thinkingContent != "" {
			// Has thinking but no response - model may have stopped after thinking
			e.logger.Warn("LLM returned thinking but no response",
				"module", "engine",
				"requestID", requestID,
				"thinking_length", len(thinkingContent))

			errorMsg := "LLM completed thinking but did not generate a response. " +
				"This may be due to:\n" +
				"1. Model decided to stop after thinking\n" +
				"2. API configuration issue (thinking mode parameters)\n" +
				"3. Token budget exhausted\n\n" +
				"Please try rephrasing your request or check LLM configuration."

			eventsCh <- events.NewEvent(events.EventTypeError, errorMsg, requestID)
		} else {
			// Completely empty response (no thinking and no response)
			e.logger.Error("LLM returned empty response",
				"module", "engine",
				"requestID", requestID)

			errorMsg := "LLM returned an empty response. This is unexpected. " +
				"Please try again or check your LLM provider status."

			eventsCh <- events.NewEvent(events.EventTypeError, errorMsg, requestID)
		}
		return "", nil, "", false
	}

	e.logger.Debug("runReActLoop: LLM response received", "module", "engine", "requestID", requestID, "response_length", len(response))
	return response, toolCalls, thinkingContent, true
}

// logReActActions logs parsed actions for debugging.
func (e *Engine) logReActActions(actions []*ToolAction, toolCalls []llm.ToolCall, requestID string) {
	if len(toolCalls) > 0 {
		e.logger.Debug("runReActLoop: extracted tool calls", "module", "engine", "requestID", requestID, "tool_call_count", len(toolCalls))
	}
	if len(actions) > 0 {
		e.logger.Debug("runReActLoop: parsed actions", "module", "engine", "requestID", requestID, "action_count", len(actions))
	} else {
		e.logger.Debug("runReActLoop: no actions found, final response", "module", "engine", "requestID", requestID)
	}
}

// handleReActActions executes actions and records observations.
// Returns false if cancelled during execution.
// thinkingContent is the accumulated LLM thinking/reasoning content to store.
// toolCalls contains the structured tool calls from LLM (with IDs for ToolUseBlock).
func (e *Engine) handleReActActions(ctx context.Context, eventsCh chan<- events.Event, actions []*ToolAction, toolCalls []llm.ToolCall, response string, thinkingContent string, requestID string) bool {
	// Clean response text first
	response = codeFencePattern.ReplaceAllString(response, "")

	// Persist thinking/text FIRST (logical order: thinking → text → tool_use)
	if thinkingContent != "" {
		blocks := []sharedmemory.ContentBlock{
			sharedmemory.ThinkingBlock{
				Type:     sharedmemory.BlockTypeThinking,
				Thinking: thinkingContent,
			},
			sharedmemory.TextBlock{
				Type: sharedmemory.BlockTypeText,
				Text: response,
			},
		}
		e.memory.AddWithBlocks(sharedmemory.RoleAssistant, blocks, memory.MessageTypeAction)
	} else if response != "" {
		e.memory.AddWithType(sharedmemory.RoleAssistant, response, memory.MessageTypeAction)
	}

	// Check if any action requires confirmation - if so, use serial execution
	// to avoid multiple simultaneous confirmation requests that TUI cannot handle
	needsSerial := false
	for _, action := range actions {
		tool, exists := e.regTools[action.Tool]
		if exists && tool != nil {
			// Check if tool requires confirmation via SensitiveTool interface
			if sensitive, ok := tool.(SensitiveTool); ok && sensitive.RequiresConfirmation() {
				needsSerial = true
				break
			}
			// Check via PermissionTool interface
			if permTool, ok := tool.(PermissionTool); ok {
				permLevel := permTool.PermissionLevel()
				if parsePermissionLevel(permLevel).RequiresConfirmation() {
					needsSerial = true
					break
				}
			}
		}
	}

	// Execute actions (parallel or serial)
	var results []toolResult
	if len(actions) == 1 || needsSerial {
		results = e.executeToolsSerial(ctx, actions, eventsCh, requestID)
	} else {
		results = e.executeToolsParallel(ctx, actions, eventsCh, requestID)
	}

	// Check if any tool was denied by user - stop execution
	for _, tr := range results {
		if tr.Result != nil && tr.Result.Error == constants.MsgDeniedByUser {
			e.logger.Info("handleReActActions: tool denied by user, stopping", "module", "engine", "requestID", requestID)
			eventsCh <- events.NewEvent(events.EventTypeResponse, "Task cancelled: tool execution denied by user.", requestID)
			return false
		}
	}

	// Check cancellation
	if e.checkReActCancellation(ctx, eventsCh, requestID) {
		return false
	}

	// Persist tool_use and tool_result in interleaved order (tool_use_1 → tool_result_1 → tool_use_2 → tool_result_2)
	// This ensures IN/OUT pairs are together when loading history
	for i, action := range actions {
		// Determine tool_use ID
		var toolUseID string
		if i < len(toolCalls) {
			toolUseID = toolCalls[i].ID
		} else {
			toolUseID = fmt.Sprintf("action_%d_%s", i, action.Tool)
		}

		// 1. Persist tool_use
		inputJSON, _ := json.Marshal(action.Parameters)
		toolUseBlock := sharedmemory.ToolUseBlock{
			Type:  sharedmemory.BlockTypeToolUse,
			ID:    toolUseID,
			Name:  action.Tool,
			Input: inputJSON,
		}
		e.memory.AddWithBlocks(sharedmemory.RoleAssistant, []sharedmemory.ContentBlock{toolUseBlock}, memory.MessageTypeAction)
		e.logger.Debug("handleReActActions: persisted tool_use record", "module", "engine", "requestID", requestID, "tool", action.Tool, "tool_id", toolUseID)

		// 2. Persist corresponding tool_result
		tr := results[i]
		e.recordToolResultWithBlock(tr, action.Tool, toolUseID)
	}

	e.logger.Debug("runReActLoop: added observations to memory, continuing loop", "module", "engine", "requestID", requestID)
	return true
}

// recordToolResultWithBlock records a single tool result as ToolResultBlock.
func (e *Engine) recordToolResultWithBlock(tr toolResult, toolName string, toolUseID string) {
	content := ""
	isError := tr.Err != nil
	if tr.Err != nil {
		content = fmt.Sprintf("Error: %v", tr.Err)
	} else if tr.Result != nil {
		content = tr.Result.Content
		if len(tr.Result.Data) > 0 {
			dataJSON, _ := json.Marshal(tr.Result.Data)
			content = fmt.Sprintf("%s\nData: %s", content, string(dataJSON))
		}
	}

	textBlock := sharedmemory.TextBlock{
		Type: sharedmemory.BlockTypeText,
		Text: content,
	}
	toolResultBlock := sharedmemory.ToolResultBlock{
		Type:      sharedmemory.BlockTypeToolResult,
		ToolUseID: toolUseID,
		Content:   []sharedmemory.ContentBlock{textBlock},
		IsError:   isError,
	}
	e.memory.AddWithBlocks(sharedmemory.RoleUser, []sharedmemory.ContentBlock{toolResultBlock}, memory.MessageTypeObservation)
	e.logger.Debug("recordToolResultWithBlock: persisted tool_result", "module", "engine", "tool", toolName, "tool_use_id", toolUseID)
}

// handleReActFinalAnswer handles final answer emission and memory recording.
// thinkingContent is the accumulated LLM thinking/reasoning content to store.
func (e *Engine) handleReActFinalAnswer(eventsCh chan<- events.Event, response string, thinkingContent string, requestID string) {
	response = codeFencePattern.ReplaceAllString(response, "")
	eventsCh <- events.NewEvent(events.EventTypeResponse, response, requestID)

	// Store with ContentBlocks if thinking present
	if thinkingContent != "" {
		blocks := []sharedmemory.ContentBlock{
			sharedmemory.ThinkingBlock{
				Type:     sharedmemory.BlockTypeThinking,
				Thinking: thinkingContent,
			},
			sharedmemory.TextBlock{
				Type: sharedmemory.BlockTypeText,
				Text: response,
			},
		}
		e.memory.AddWithBlocks(sharedmemory.RoleAssistant, blocks, memory.MessageTypeAssistant)
	} else {
		e.memory.AddWithType(sharedmemory.RoleAssistant, response, memory.MessageTypeAssistant)
	}
	e.logger.Debug("runReActLoop: final response saved to memory", "module", "engine", "requestID", requestID, "response_length", len(response))
}

// reflectOnAnswer performs a reflection step on the answer before final emission.
// This allows the LLM to self-verify the quality and completeness of its response.
func (e *Engine) reflectOnAnswer(ctx context.Context, response string, requestID string) string {
	// Build reflection prompt
	reflectionPrompt := e.buildReflectionPrompt(response)

	// Get messages with reflection prompt appended
	messages := e.buildReActMessages(ctx)
	messages = append(messages, llm.Message{
		Role:          sharedmemory.RoleAssistant,
		ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: response}},
	})
	messages = append(messages, llm.Message{
		Role:          sharedmemory.RoleUser,
		ContentBlocks: []sharedmemory.ContentBlock{sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: reflectionPrompt}},
	})

	// Call LLM for reflection
	req := e.buildRequest(messages)
	resp, err := e.client.Complete(ctx, req)
	if err != nil {
		e.logger.Debug("reflectOnAnswer: LLM error, returning original response", "module", "engine", "requestID", requestID, "error", err.Error())
		return response
	}

	// Parse reflection response - look for VERDICT: PASS/FAIL/PARTIAL pattern
	var reflectionContent string
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			reflectionContent = tb.Text
			break
		}
	}
	if strings.Contains(reflectionContent, "VERDICT: PASS") {
		e.logger.Debug("reflectOnAnswer: reflection passed, returning original response", "module", "engine", "requestID", requestID)
		return response
	}

	// If reflection suggests improvements, append feedback
	if strings.Contains(reflectionContent, "VERDICT: PARTIAL") || strings.Contains(reflectionContent, "VERDICT: FAIL") {
		e.logger.Info("reflectOnAnswer: reflection identified issues, appending feedback", "module", "engine", "requestID", requestID)
		// Extract improvement suggestions
		improvements := e.extractReflectionImprovements(reflectionContent)
		if improvements != "" {
			return fmt.Sprintf("%s\n\n[Self-review feedback: %s]", response, improvements)
		}
	}

	return response
}

// buildReflectionPrompt builds the prompt for reflection step.
// Supports template placeholders: {{response}}, {{context}}, {{task}}
// Falls back to single %s for backward compatibility.
func (e *Engine) buildReflectionPrompt(response string) string {
	template := e.config.ReflectionConfig.PromptTemplate
	if template != "" {
		// Check if template uses {{...}} placeholders (new format)
		if strings.Contains(template, "{{") {
			// Replace placeholders
			result := strings.ReplaceAll(template, "{{response}}", response)
			// Get summary from memory if it supports SummarizingMemory interface
			if sumMem, ok := e.memory.(sharedmemory.SummarizingMemory); ok {
				msgs := sumMem.GetMessagesWithSummary()
				// Extract summary from first system message if present
				for _, msg := range msgs {
					if msg.Role == sharedmemory.RoleSystem {
						// Extract text from ContentBlocks
						for _, block := range msg.GetContentBlocks() {
							if tb, ok := block.(sharedmemory.TextBlock); ok {
								if strings.Contains(tb.Text, "Previous conversation summary:") {
									result = strings.ReplaceAll(result, "{{context}}", tb.Text)
									break
								}
							}
						}
					}
				}
			}
			result = strings.ReplaceAll(result, "{{context}}", "")
			// Get task from first user message
			result = strings.ReplaceAll(result, "{{task}}", e.getCurrentTask())
			return result
		}
		// Backward compatibility: single %s placeholder
		return fmt.Sprintf(template, response)
	}

	// Default reflection prompt
	return fmt.Sprintf(`Review your previous response for quality and completeness:

%s

Please provide:
1. VERDICT: PASS, PARTIAL, or FAIL (based on completeness and accuracy)
2. If PARTIAL or FAIL, list specific improvements needed

Keep your review concise (max 100 words).`, response)
}

// getCurrentTask returns the current task description from memory.
func (e *Engine) getCurrentTask() string {
	// Get the first user message as the task
	msgs := e.memory.Get()
	for _, msg := range msgs {
		if msg.Role == sharedmemory.RoleUser {
			// Extract text from ContentBlocks
			for _, block := range msg.GetContentBlocks() {
				if tb, ok := block.(sharedmemory.TextBlock); ok {
					if len(tb.Text) > 100 {
						return tb.Text[:100] + "..."
					}
					return tb.Text
				}
			}
		}
	}
	return ""
}

// extractReflectionImprovements extracts improvement suggestions from reflection response.
// Supports multiple output formats for robustness.
func (e *Engine) extractReflectionImprovements(reflectionContent string) string {
	// Define patterns to match (order matters - more specific first)
	patterns := []string{
		"improvements needed:",
		"Improvements needed:",
		"Improvements:",
		"Suggestions:",
		"改进建议：",
		"需要改进：",
		"改进：",
		"建议：",
		"2.", // Match numbered list after VERDICT
	}

	for _, pattern := range patterns {
		idx := strings.Index(reflectionContent, pattern)
		if idx != -1 {
			// Extract from pattern to end
			improvements := reflectionContent[idx:]
			// Clean up and limit length
			improvements = strings.TrimSpace(improvements)
			if len(improvements) > 200 {
				improvements = improvements[:200] + "..."
			}
			return improvements
		}
	}

	// Fallback: try to extract content after VERDICT line
	verdictIdx := strings.Index(reflectionContent, "VERDICT:")
	if verdictIdx != -1 {
		// Find the next line after VERDICT
		afterVerdict := reflectionContent[verdictIdx:]
		lines := strings.Split(afterVerdict, "\n")
		if len(lines) > 1 {
			// Collect non-empty lines after VERDICT (skip the VERDICT line itself)
			var improvementLines []string
			for i, line := range lines {
				if i == 0 {
					continue // Skip VERDICT line
				}
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "VERDICT:") {
					improvementLines = append(improvementLines, line)
				}
				if len(improvementLines) >= 3 {
					break // Limit to 3 lines
				}
			}
			if len(improvementLines) > 0 {
				return strings.Join(improvementLines, "; ")
			}
		}
	}

	return ""
}

// generateBestEffortAnswer returns the best available answer when max steps exceeded.
func (e *Engine) generateBestEffortAnswer(ctx context.Context, requestID string) string {
	messages := e.buildReActMessages(ctx)
	messages = append(messages, llm.Message{
		Role:          sharedmemory.RoleUser,
		ContentBlocks: []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: "You have reached the maximum number of steps. Please provide the best answer you can with what you have gathered so far."},
		},
	})
	req := e.buildRequest(messages)
	resp, err := e.client.Complete(ctx, req)
	if err != nil {
		e.logger.Debug("generateBestEffortAnswer: LLM error", "module", "engine", "requestID", requestID, "error", err.Error())
		return "I've reached my thinking limit and cannot provide a more detailed response. Here's what I know so far based on our conversation."
	}
	// Extract text from ContentBlocks
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(sharedmemory.TextBlock); ok {
			return tb.Text
		}
	}
	return "I've reached my thinking limit and cannot provide a more detailed response. Here's what I know so far based on our conversation."
}
