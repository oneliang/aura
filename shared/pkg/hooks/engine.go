package hooks

import (
	"context"
	"regexp"
	"sync"

	"github.com/oneliang/aura/shared/pkg/logger"
)

// Engine is the hook engine that dispatches events to configured hooks.
// It is stateless and safe for concurrent use across multiple sessions.
type Engine struct {
	config   *HooksConfig
	logger   *logger.Logger
	matchers map[HookEventType][]compiledMatcher
}

// compiledMatcher holds a precompiled regex matcher and its hooks.
type compiledMatcher struct {
	regex *regexp.Regexp // nil means match all
	hooks []HookConfig
}

// NewEngine creates a new hook engine from the given configuration.
// If cfg is nil or cfg.Enabled is false, returns nil (zero overhead).
func NewEngine(cfg *HooksConfig, log *logger.Logger) *Engine {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	e := &Engine{
		config:   cfg,
		logger:   log,
		matchers: make(map[HookEventType][]compiledMatcher),
	}
	e.compileMatchers()
	return e
}

// HasEvents returns true if there are hooks configured for the given event type.
// Use this for fast-path checks before building event input data.
func (e *Engine) HasEvents(eventType HookEventType) bool {
	if e == nil {
		return false
	}
	matchers, ok := e.matchers[eventType]
	if !ok || len(matchers) == 0 {
		return false
	}
	// Check if any matcher has hooks
	for _, m := range matchers {
		if len(m.hooks) > 0 {
			return true
		}
	}
	return false
}

// Fire triggers all hooks for the given event type asynchronously.
// Hooks run concurrently; this method returns immediately.
// Returns nil immediately (fire-and-forget semantics).
func (e *Engine) Fire(ctx context.Context, eventType HookEventType, input any) error {
	if e == nil || !e.HasEvents(eventType) {
		return nil
	}

	matchers := e.matchers[eventType]
	if len(matchers) == 0 {
		return nil
	}

	go func() {
		for _, cm := range matchers {
			for _, hc := range cm.hooks {
				if hc.Type != "command" {
					continue
				}
				go func(h HookConfig) {
					result, err := executeCommand(ctx, h.Command, input, h.Timeout)
					if err != nil {
						e.logger.Error("hook execution error", "event", string(eventType), "command", h.Command, "error", err.Error())
					} else {
						e.logResult(eventType, h.Command, result)
					}
				}(hc)
			}
		}
	}()

	return nil
}

// FireBlocking triggers all hooks for the given event type and waits for
// the first hook to return a blocking result (Continue=false or exit code 2).
// If no hook blocks, returns a result with Continue=true.
func (e *Engine) FireBlocking(ctx context.Context, eventType HookEventType, input any) (*HookResult, error) {
	if e == nil || !e.HasEvents(eventType) {
		return nil, nil
	}

	matchers := e.matchers[eventType]
	if len(matchers) == 0 {
		return nil, nil
	}

	for _, cm := range matchers {
		for _, hc := range cm.hooks {
			if hc.Type != "command" {
				continue
			}
			result, err := executeCommand(ctx, hc.Command, input, hc.Timeout)
			if err != nil {
				e.logger.Error("blocking hook execution error", "event", string(eventType), "command", hc.Command, "error", err.Error())
				// Error in blocking hook: don't block the main flow
				continue
			}
			e.logResult(eventType, hc.Command, result)

			if e.ShouldBlock(result) {
				return result, nil
			}
		}
	}

	return nil, nil
}

// Shutdown shuts down the hook engine (closes file watchers if any).
func (e *Engine) Shutdown() {
	if e == nil {
		return
	}
	e.logger.Info("hooks engine shut down")
}

// compileMatchers precompiles regex matchers for all event types.
func (e *Engine) compileMatchers() {
	eventConfigs := map[HookEventType][]HookEvent{
		EventSessionStart:     e.config.SessionStart,
		EventUserPromptSubmit: e.config.UserPromptSubmit,
		EventPreToolUse:       e.config.PreToolUse,
		EventPostToolUse:      e.config.PostToolUse,
		EventPostToolUseFail:  e.config.PostToolUseFail,
		EventStop:             e.config.Stop,
		EventStopFailure:      e.config.StopFailure,
		EventSubagentStop:     e.config.SubagentStop,
		EventPreCompact:       e.config.PreCompact,
		EventPostCompact:      e.config.PostCompact,
		EventTaskCreated:      e.config.TaskCreated,
		EventTaskCompleted:    e.config.TaskCompleted,
		EventFileChanged:      e.config.FileChanged,
		EventSessionEnd:       e.config.SessionEnd,
	}

	for evt, events := range eventConfigs {
		var cms []compiledMatcher
		for _, ev := range events {
			cm := compiledMatcher{hooks: ev.Hooks}
			if ev.Matcher != "" {
				if re, err := regexp.Compile(ev.Matcher); err != nil {
					e.logger.Warn("invalid hook matcher regex", "matcher", ev.Matcher, "error", err.Error())
				} else {
					cm.regex = re
				}
			}
			cms = append(cms, cm)
		}
		if len(cms) > 0 {
			e.matchers[evt] = cms
		}
	}
}

// ShouldBlock determines if a hook result should block the main flow.
func (e *Engine) ShouldBlock(result *HookResult) bool {
	if result == nil {
		return false
	}
	// Explicit exit code 2 = blocking
	if result.ExitCode == HookExitCodeBlocking {
		return true
	}
	// Explicit Continue=false in parsed output
	if result.Parsed != nil && result.Parsed.Continue != nil && !*result.Parsed.Continue {
		return true
	}
	return false
}

// logResult logs the hook execution result at debug level.
func (e *Engine) logResult(eventType HookEventType, command string, result *HookResult) {
	if result == nil {
		return
	}
	// Build key-value args
	args := []any{"event", string(eventType), "command", command, "exit_code", result.ExitCode}
	if result.Stderr != "" {
		args = append(args, "stderr", truncateString(result.Stderr, 200))
	}
	if result.Parsed != nil {
		if result.Parsed.StopReason != "" {
			args = append(args, "stop_reason", result.Parsed.StopReason)
		}
		if result.Parsed.Continue != nil {
			args = append(args, "continue", *result.Parsed.Continue)
		}
		if result.Parsed.SystemMessage != "" {
			args = append(args, "system_message_len", len(result.Parsed.SystemMessage))
			args = append(args, "system_message_preview", truncateString(result.Parsed.SystemMessage, 100))
		}
	}
	e.logger.Debug("hook executed", args...)
}

// matchToolName checks if a tool name matches the given compiled matchers.
// Returns true if any matcher matches (or if there are no regex matchers,
// meaning match-all).
func matchToolName(matchers []compiledMatcher, toolName string) ([]HookConfig, bool) {
	var matched []HookConfig
	hasMatch := false

	for _, cm := range matchers {
		if cm.regex != nil {
			if cm.regex.MatchString(toolName) {
				matched = append(matched, cm.hooks...)
				hasMatch = true
			}
		} else {
			// No regex = match all (non-tool events)
			matched = append(matched, cm.hooks...)
			hasMatch = true
		}
	}

	return matched, hasMatch
}

// FireWithToolName fires hooks that are filtered by tool name.
// Used for tool-related events (PreToolUse, PostToolUse, PostToolUseFailure).
func (e *Engine) FireWithToolName(ctx context.Context, eventType HookEventType, toolName string, input any) error {
	if e == nil || !e.HasEvents(eventType) {
		return nil
	}

	matchers, ok := e.matchers[eventType]
	if !ok || len(matchers) == 0 {
		return nil
	}

	hooks, matched := matchToolName(matchers, toolName)
	if !matched || len(hooks) == 0 {
		return nil
	}

	go func() {
		var wg sync.WaitGroup
		for _, hc := range hooks {
			if hc.Type != "command" {
				continue
			}
			wg.Add(1)
			go func(h HookConfig) {
				defer wg.Done()
				result, err := executeCommand(ctx, h.Command, input, h.Timeout)
				if err != nil {
					e.logger.Error("hook execution error", "event", string(eventType), "command", h.Command, "error", err.Error())
				} else {
					e.logResult(eventType, h.Command, result)
				}
			}(hc)
		}
		wg.Wait()
	}()

	return nil
}

// FireBlockingWithToolName fires hooks filtered by tool name and waits
// for a blocking result.
func (e *Engine) FireBlockingWithToolName(ctx context.Context, eventType HookEventType, toolName string, input any) (*HookResult, error) {
	if e == nil || !e.HasEvents(eventType) {
		return nil, nil
	}

	matchers, ok := e.matchers[eventType]
	if !ok || len(matchers) == 0 {
		return nil, nil
	}

	hooks, matched := matchToolName(matchers, toolName)
	if !matched || len(hooks) == 0 {
		return nil, nil
	}

	for _, hc := range hooks {
		if hc.Type != "command" {
			continue
		}
		result, err := executeCommand(ctx, hc.Command, input, hc.Timeout)
		if err != nil {
			e.logger.Error("blocking hook execution error", "event", string(eventType), "command", hc.Command, "error", err.Error())
			continue
		}
		e.logResult(eventType, hc.Command, result)

		if e.ShouldBlock(result) {
			return result, nil
		}
	}

	return nil, nil
}
