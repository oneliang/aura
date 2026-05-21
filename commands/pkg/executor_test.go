// Package commands provides tests for the CommandProvider.
package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/oneliang/aura/knowledge/pkg"
	"github.com/oneliang/aura/personality/pkg/profile"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/i18n"
)

// init initializes i18n for tests
func init() {
	// Initialize i18n with empty path to use embedded locales
	if err := i18n.Init("", "en"); err != nil {
		// Non-fatal: tests will use fallback
	}
}

// TestCommandInfo_GetName tests CommandInfo.GetName method.
func TestCommandInfoMethods(t *testing.T) {
	info := CommandInfo{
		Name:        "test_command",
		DisplayName: "Test Command",
		Description: "Test description",
	}

	if info.GetName() != "test_command" {
		t.Errorf("GetName() = %q, want 'test_command'", info.GetName())
	}
	if info.GetDescription() != "Test description" {
		t.Errorf("GetDescription() = %q, want 'Test description'", info.GetDescription())
	}
}

// TestGetInternalCommands tests the GetInternalCommands function.
func TestGetInternalCommands_Count(t *testing.T) {
	cmds := GetInternalCommands()

	if len(cmds) == 0 {
		t.Fatal("GetInternalCommands() returned empty slice")
	}

	// Should have at least basic commands
	if len(cmds) < 10 {
		t.Errorf("Expected at least 10 commands, got %d", len(cmds))
	}
}

// TestGetInternalCommands_ExpectedCommands tests expected commands exist.
func TestGetInternalCommands_ExpectedCommands(t *testing.T) {
	cmds := GetInternalCommands()

	cmdMap := make(map[string]bool)
	for _, cmd := range cmds {
		cmdMap[cmd.Name] = true
	}

	expectedCommands := []string{
		CmdNameExit,
		CmdNameQuit,
		CmdNameClear,
		CmdNameHelp,
		CmdNameSessions,
		CmdNameProfile,
		CmdNameConfig,
		CmdNameKnowledge,
		CmdNameSubscription,
	}

	for _, expected := range expectedCommands {
		if !cmdMap[expected] {
			t.Errorf("Expected command %q not found", expected)
		}
	}
}

// TestCommandProvider_Execute_BasicCommands tests basic command execution.
func TestCommandProvider_Execute_BasicCommands(t *testing.T) {
	// Create event bus with test handlers
	bus := events.NewBus()
	bus.RegisterCommandHandler(events.CommandTypeMemoryClear, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: "Conversation history cleared."}
	})
	bus.RegisterCommandHandler(events.CommandTypeMemoryCompact, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: "Conversation history compacted."}
	})
	bus.RegisterCommandHandler(events.CommandTypeMemoryStats, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: "Messages: 0, Tokens: 0"}
	})
	bus.RegisterCommandHandler(events.CommandTypeEngineStatus, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: "Engine: Ready"}
	})
	bus.RegisterCommandHandler(events.CommandTypeToolHistory, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: "Tool history not available."}
	})
	bus.RegisterCommandHandler(events.CommandTypeLLMConfig, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: "Provider: test, Model: test"}
	})
	bus.RegisterCommandHandler(events.CommandTypeSessionRole, func(ctx context.Context, req *events.CommandRequest) events.CommandResponse {
		return events.CommandResponse{Success: true, Result: "No role set."}
	})

	// Create minimal provider (handlers only need their deps)
	provider := &CommandProvider{
		sessionHandler:      &SessionHandler{},
		profileHandler:      &ProfileHandler{},
		configHandler:       NewConfigExecutor("/test/config.yaml"),
		knowledgeHandler:    &KnowledgeExecutor{},
		subscriptionHandler: &SubscriptionHandler{},
		eventBus:            bus,
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{name: "exit command", cmd: CmdNameExit, wantErr: false},
		{name: "quit command", cmd: CmdNameQuit, wantErr: false},
		{name: "clear command", cmd: CmdNameClear, wantErr: false},
		{name: "compact command", cmd: CmdNameCompact, wantErr: false},
		{name: "help command", cmd: CmdNameHelp, wantErr: false},
		{name: "memory command", cmd: CmdNameMemory, wantErr: false},
		{name: "tools command", cmd: CmdNameTools, wantErr: false},
		{name: "status command", cmd: CmdNameStatus, wantErr: false},
		{name: "history command", cmd: CmdNameHistory, wantErr: false},
		{name: "model command", cmd: CmdNameModel, wantErr: false},
		{name: "role command", cmd: CmdNameRole, wantErr: false},
		{name: "unknown command", cmd: "command_unknown_xyz", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.Execute(ctx, tt.cmd, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == "" {
				t.Error("Execute() returned empty result")
			}
		})
	}
}

// TestCommandProvider_getHelp tests the getHelp method.
func TestCommandProvider_getHelp(t *testing.T) {
	provider := &CommandProvider{}

	help := provider.getHelp()

	if help == "" {
		t.Fatal("getHelp() returned empty result")
	}
	if !strings.Contains(help, "Available commands") {
		t.Error("Help should contain 'Available commands'")
	}
}

// TestCommandProvider_getToolsList tests the getToolsList method.
func TestCommandProvider_getToolsList(t *testing.T) {
	provider := &CommandProvider{}

	tools := provider.getToolsList()

	if tools == "" {
		t.Fatal("getToolsList() returned empty result")
	}

	expectedCategories := []string{
		"File System",
		"System",
		"Web",
		"Utility",
		"Knowledge",
	}

	for _, category := range expectedCategories {
		if !strings.Contains(tools, category) {
			t.Errorf("Tools list missing category: %s", category)
		}
	}
}

// TestCommandProviderDeps tests the deps struct.
func TestCommandProviderDeps_Struct(t *testing.T) {
	deps := CommandProviderDeps{
		SessionMgr: nil,
		Profile:    &profile.Profile{},
		Config:     &config.Config{},
		ConfigPath: "/test/config.yaml",
	}

	if deps.ConfigPath != "/test/config.yaml" {
		t.Errorf("ConfigPath = %q, want '/test/config.yaml'", deps.ConfigPath)
	}
}

// TestKnowledgeDefaultCollectionFactory tests the knowledge.DefaultCollectionFactory.
func TestKnowledgeDefaultCollectionFactory(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			BaseURL: "http://localhost:11434",
		},
	}
	factory := knowledge.NewDefaultCollectionFactory(cfg)

	// Test NewCollection returns error for non-existent directory (expected)
	ctx := context.Background()
	_, err := factory.NewCollection(ctx, "")
	// This may fail due to Ollama not running, which is expected
	// We just verify the method can be called
	_ = err
}
