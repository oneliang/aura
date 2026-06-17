package hooks

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/oneliang/aura/shared/pkg/logger"
)

func newTestLogger() *logger.Logger {
	return logger.NewNamed(logger.Config{Level: "debug", Module: "hooks-test"})
}

// TestNewEngine_NilConfig_ReturnsNil verifies that nil config returns nil engine.
func TestNewEngine_NilConfig_ReturnsNil(t *testing.T) {
	t.Parallel()
	e := NewEngine(nil, newTestLogger())
	if e != nil {
		t.Error("expected nil engine for nil config")
	}
}

// TestNewEngine_Disabled_ReturnsNil verifies that disabled config returns nil engine.
func TestNewEngine_Disabled_ReturnsNil(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{Enabled: false}
	e := NewEngine(cfg, newTestLogger())
	if e != nil {
		t.Error("expected nil engine for disabled config")
	}
}

// TestNewEngine_Enabled_ReturnsEngine verifies that enabled config returns an engine.
func TestNewEngine_Enabled_ReturnsEngine(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{Enabled: true}
	e := NewEngine(cfg, newTestLogger())
	if e == nil {
		t.Fatal("expected non-nil engine for enabled config")
	}
	if e.config != cfg {
		t.Error("engine should reference the provided config")
	}
}

// TestHasEvents_NoHooks_ReturnsFalse verifies fast-path returns false when no hooks configured.
func TestHasEvents_NoHooks_ReturnsFalse(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{Enabled: true}
	e := NewEngine(cfg, newTestLogger())
	if e.HasEvents(EventPreToolUse) {
		t.Error("expected HasEvents to return false when no hooks configured")
	}
}

// TestHasEvents_WithHooks_ReturnsTrue verifies fast-path returns true when hooks exist.
func TestHasEvents_WithHooks_ReturnsTrue(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{
		Enabled:    true,
		PreToolUse: []HookEvent{{Hooks: []HookConfig{{Type: "command", Command: "echo hello"}}}},
	}
	e := NewEngine(cfg, newTestLogger())
	if !e.HasEvents(EventPreToolUse) {
		t.Error("expected HasEvents to return true when hooks are configured")
	}
}

// TestHasEvents_NilEngine_ReturnsFalse verifies nil safety.
func TestHasEvents_NilEngine_ReturnsFalse(t *testing.T) {
	t.Parallel()
	var e *Engine
	if e.HasEvents(EventPreToolUse) {
		t.Error("expected nil engine HasEvents to return false")
	}
}

// TestParseHookOutput_Empty returns nil for empty input.
func TestParseHookOutput_Empty(t *testing.T) {
	t.Parallel()
	if r := parseHookOutput(""); r != nil {
		t.Errorf("expected nil for empty output, got %+v", r)
	}
	if r := parseHookOutput("   \n  "); r != nil {
		t.Errorf("expected nil for whitespace-only output, got %+v", r)
	}
}

// TestParseHookOutput_InvalidJSON returns nil for non-JSON output.
func TestParseHookOutput_InvalidJSON(t *testing.T) {
	t.Parallel()
	r := parseHookOutput("this is not json")
	if r != nil {
		t.Errorf("expected nil for invalid JSON, got %+v", r)
	}
}

// TestParseHookOutput_ValidJSON parses correctly.
func TestParseHookOutput_ValidJSON(t *testing.T) {
	t.Parallel()
	input := `{"continue": false, "stopReason": "security policy violation"}`
	r := parseHookOutput(input)
	if r == nil {
		t.Fatal("expected parsed output for valid JSON")
	}
	if r.Continue == nil || *r.Continue != false {
		t.Errorf("expected continue=false, got %v", r.Continue)
	}
	if r.StopReason != "security policy violation" {
		t.Errorf("expected stop reason 'security policy violation', got %q", r.StopReason)
	}
}

// TestParseHookOutput_PartialJSON handles partial/extra output gracefully.
func TestParseHookOutput_WithSurroundingText(t *testing.T) {
	t.Parallel()
	// If the entire output is just the JSON, it parses fine.
	input := `{"systemMessage": "hello"}`
	r := parseHookOutput(input)
	if r == nil {
		t.Fatal("expected parsed output")
	}
	if r.SystemMessage != "hello" {
		t.Errorf("expected systemMessage='hello', got %q", r.SystemMessage)
	}
}

// TestShouldBlock_ExitCode2 returns true for blocking exit code.
func TestShouldBlock_ExitCode2(t *testing.T) {
	t.Parallel()
	e := NewEngine(&HooksConfig{Enabled: true}, newTestLogger())
	result := &HookResult{ExitCode: HookExitCodeBlocking}
	if !e.ShouldBlock(result) {
		t.Error("expected exit code 2 to block")
	}
}

// TestShouldBlock_ContinueFalse returns true when parsed output has continue=false.
func TestShouldBlock_ContinueFalse(t *testing.T) {
	t.Parallel()
	e := NewEngine(&HooksConfig{Enabled: true}, newTestLogger())
	f := false
	result := &HookResult{Parsed: &HookOutput{Continue: &f}}
	if !e.ShouldBlock(result) {
		t.Error("expected Continue=false to block")
	}
}

// TestShouldBlock_ExitCode0_NoParsed returns false for normal exit.
func TestShouldBlock_ExitCode0(t *testing.T) {
	t.Parallel()
	e := NewEngine(&HooksConfig{Enabled: true}, newTestLogger())
	result := &HookResult{ExitCode: HookExitCodeNormal}
	if e.ShouldBlock(result) {
		t.Error("expected exit code 0 with no parsed output to not block")
	}
}

// TestShouldBlock_NilResult returns false.
func TestShouldBlock_NilResult(t *testing.T) {
	t.Parallel()
	e := NewEngine(&HooksConfig{Enabled: true}, newTestLogger())
	if e.ShouldBlock(nil) {
		t.Error("expected nil result to not block")
	}
}

// TestExecuteCommand_Success runs a simple command that exits 0.
func TestExecuteCommand_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	result, err := executeCommand(ctx, "echo hello", nil, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Stdout != "hello\n" {
		t.Errorf("expected stdout 'hello\\n', got %q", result.Stdout)
	}
}

// TestExecuteCommand_Failure runs a command that exits non-zero.
func TestExecuteCommand_Failure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	result, err := executeCommand(ctx, "exit 1", nil, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

// TestExecuteCommand_BlockingExitCode returns exit code 2.
func TestExecuteCommand_BlockingExitCode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	result, err := executeCommand(ctx, "exit 2", nil, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != HookExitCodeBlocking {
		t.Errorf("expected exit code 2, got %d", result.ExitCode)
	}
}

// TestExecuteCommand_StdinPassesInput verifies that input is passed as JSON on stdin.
func TestExecuteCommand_StdinPassesInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	input := map[string]string{"tool_name": "Bash", "command": "ls"}
	result, err := executeCommand(ctx, "cat", input, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	// stdin should be JSON
	var got map[string]string
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("expected valid JSON on stdout, got %q: %v", result.Stdout, err)
	}
	if got["tool_name"] != "Bash" {
		t.Errorf("expected tool_name='Bash', got %q", got["tool_name"])
	}
}

// TestExecuteCommand_StdinPassesInput verifies that hook JSON output is parsed.
func TestExecuteCommand_ParsedOutput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Command that outputs valid HookOutput JSON
	result, err := executeCommand(ctx, `echo '{"continue": false, "stopReason": "blocked"}'`, nil, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Parsed == nil {
		t.Fatal("expected parsed output")
	}
	if result.Parsed.Continue == nil || *result.Parsed.Continue != false {
		t.Errorf("expected continue=false, got %v", result.Parsed.Continue)
	}
	if result.Parsed.StopReason != "blocked" {
		t.Errorf("expected stopReason='blocked', got %q", result.Parsed.StopReason)
	}
}

// TestExecuteCommand_Timeout verifies that long-running commands are killed.
func TestExecuteCommand_Timeout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// This command sleeps for 10 seconds but we set 1s timeout
	result, err := executeCommand(ctx, "sleep 10", nil, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != -1 {
		t.Errorf("expected exit code -1 for timeout, got %d", result.ExitCode)
	}
	if result.Stderr != "hook timed out" {
		t.Errorf("expected 'hook timed out' stderr, got %q", result.Stderr)
	}
}

// TestFire_NilEngine_ReturnsNil verifies nil safety.
func TestFire_NilEngine_ReturnsNil(t *testing.T) {
	t.Parallel()
	var e *Engine
	err := e.Fire(context.Background(), EventPreToolUse, nil)
	if err != nil {
		t.Errorf("expected nil error for nil engine Fire, got %v", err)
	}
}

// TestFireBlocking_NilEngine_ReturnsNil verifies nil safety.
func TestFireBlocking_NilEngine_ReturnsNil(t *testing.T) {
	t.Parallel()
	var e *Engine
	result, err := e.FireBlocking(context.Background(), EventPreToolUse, nil)
	if err != nil {
		t.Errorf("expected nil error for nil engine FireBlocking, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for nil engine FireBlocking, got %+v", result)
	}
}

// TestFire_NonExistentEvent_NoHooks does not panic for events with no hooks.
func TestFire_NonExistentEvent_NoHooks(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{Enabled: true}
	e := NewEngine(cfg, newTestLogger())
	err := e.Fire(context.Background(), EventPreToolUse, nil)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// TestMatchToolName_RegexMatch matches when regex matches.
func TestMatchToolName_RegexMatch(t *testing.T) {
	t.Parallel()
	matcher := compiledMatcher{}
	matcher.regex = regexp.MustCompile("Bash|Exec")
	matcher.hooks = []HookConfig{{Type: "command", Command: "echo matched"}}

	hooks, ok := matchToolName([]compiledMatcher{matcher}, "Bash")
	if !ok {
		t.Fatal("expected match for Bash")
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}
	if hooks[0].Command != "echo matched" {
		t.Errorf("expected 'echo matched', got %q", hooks[0].Command)
	}
}

// TestMatchToolName_RegexNoMatch does not match when regex doesn't match.
func TestMatchToolName_RegexNoMatch(t *testing.T) {
	t.Parallel()
	matcher := compiledMatcher{}
	matcher.regex = regexp.MustCompile("Bash|Exec")
	matcher.hooks = []HookConfig{{Type: "command", Command: "echo matched"}}

	_, ok := matchToolName([]compiledMatcher{matcher}, "file_read")
	if ok {
		t.Error("expected no match for file_read")
	}
}

// TestMatchToolName_MatchAll matches when no regex (match-all).
func TestMatchToolName_MatchAll(t *testing.T) {
	t.Parallel()
	matcher := compiledMatcher{
		hooks: []HookConfig{{Type: "command", Command: "echo all"}},
	}

	hooks, ok := matchToolName([]compiledMatcher{matcher}, "anything")
	if !ok {
		t.Fatal("expected match-all to match anything")
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}
}

// TestFireBlocking_BlocksOnExitCode2 verifies blocking behavior.
func TestFireBlocking_BlocksOnExitCode2(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{
		Enabled: true,
		PreToolUse: []HookEvent{
			{Hooks: []HookConfig{{Type: "command", Command: "exit 2"}}},
		},
	}
	e := NewEngine(cfg, newTestLogger())
	result, err := e.FireBlocking(context.Background(), EventPreToolUse, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected blocking result")
	}
	if result.ExitCode != HookExitCodeBlocking {
		t.Errorf("expected exit code 2, got %d", result.ExitCode)
	}
}

// TestFireBlocking_DoesNotBlockOnExitCode0 does not block on normal exit,
// but still returns the last result for callers to inspect.
func TestFireBlocking_DoesNotBlockOnExitCode0(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{
		Enabled: true,
		PreToolUse: []HookEvent{
			{Hooks: []HookConfig{{Type: "command", Command: "echo ok"}}},
		},
	}
	e := NewEngine(cfg, newTestLogger())
	result, err := e.FireBlocking(context.Background(), EventPreToolUse, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for non-blocking hook")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestFireBlocking_ReturnsSystemMessage verifies that FireBlocking returns the
// hook result (including SystemMessage) even when the hook doesn't block.
func TestFireBlocking_ReturnsSystemMessage(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{
		Enabled: true,
		SessionStart: []HookEvent{
			{Hooks: []HookConfig{{Type: "command", Command: `echo '{"systemMessage":"hello from hook"}'`}}},
		},
	}
	e := NewEngine(cfg, newTestLogger())
	result, err := e.FireBlocking(context.Background(), EventSessionStart, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Parsed == nil {
		t.Fatal("expected parsed output")
	}
	if result.Parsed.SystemMessage != "hello from hook" {
		t.Errorf("expected systemMessage='hello from hook', got '%s'", result.Parsed.SystemMessage)
	}
}

// TestEngine_Methods_Disabled_ReturnImmediately verifies that disabled engine returns fast.
func TestEngine_Methods_Disabled_ReturnImmediately(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{Enabled: false}
	e := NewEngine(cfg, newTestLogger())

	// Should not panic or hang
	_ = e.Fire(context.Background(), EventPreToolUse, nil)
	r, err := e.FireBlocking(context.Background(), EventPreToolUse, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if r != nil {
		t.Errorf("expected nil result, got %+v", r)
	}
	e.Shutdown()
}

// TestHookEventTypes_Constants verifies that all event type constants are defined.
func TestHookEventTypes_Constants(t *testing.T) {
	t.Parallel()
	events := []HookEventType{
		EventSessionStart,
		EventUserPromptSubmit,
		EventPreToolUse,
		EventPostToolUse,
		EventPostToolUseFail,
		EventStop,
		EventStopFailure,
		EventSubagentStop,
		EventPreCompact,
		EventPostCompact,
		EventTaskCreated,
		EventTaskCompleted,
		EventFileChanged,
		EventSessionEnd,
	}
	for _, evt := range events {
		if evt == "" {
			t.Errorf("event type constant should not be empty")
		}
	}
}

// TestHookExitCode_Constants verifies exit code constants.
func TestHookExitCode_Constants(t *testing.T) {
	t.Parallel()
	if HookExitCodeNormal != 0 {
		t.Errorf("HookExitCodeNormal should be 0, got %d", HookExitCodeNormal)
	}
	if HookExitCodeBlocking != 2 {
		t.Errorf("HookExitCodeBlocking should be 2, got %d", HookExitCodeBlocking)
	}
}

// TestFireBlocking_WithToolName verifies tool name filtering in blocking mode.
func TestFireBlocking_WithToolName(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{
		Enabled: true,
		PreToolUse: []HookEvent{
			{Matcher: "Bash", Hooks: []HookConfig{{Type: "command", Command: "exit 2"}}},
		},
	}
	e := NewEngine(cfg, newTestLogger())

	// Should block for Bash
	result, err := e.FireBlockingWithToolName(context.Background(), EventPreToolUse, "Bash", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected blocking result for Bash")
	}

	// Should not block for file_read (no match)
	result, err = e.FireBlockingWithToolName(context.Background(), EventPreToolUse, "file_read", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for non-matching tool file_read, got %+v", result)
	}
}

// TestFire_WithToolName verifies tool name filtering in async mode.
func TestFire_WithToolName(t *testing.T) {
	t.Parallel()
	cfg := &HooksConfig{
		Enabled: true,
		PostToolUse: []HookEvent{
			{Matcher: "Bash", Hooks: []HookConfig{{Type: "command", Command: "echo triggered"}}},
		},
	}
	e := NewEngine(cfg, newTestLogger())

	// Should fire for Bash (no error, async)
	err := e.FireWithToolName(context.Background(), EventPostToolUse, "Bash", nil)
	if err != nil {
		t.Errorf("unexpected error for matching tool: %v", err)
	}

	// Should not fire for file_read (no error, no match)
	err = e.FireWithToolName(context.Background(), EventPostToolUse, "file_read", nil)
	if err != nil {
		t.Errorf("unexpected error for non-matching tool: %v", err)
	}

	// Small delay for async goroutine to complete (best-effort check)
	time.Sleep(50 * time.Millisecond)
}
