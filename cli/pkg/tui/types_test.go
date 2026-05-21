package tui

import (
	"testing"
	"time"
)

func TestRegisterCommand(t *testing.T) {
	// Clear registry before test
	commandRegistry = make(map[string]*Command)

	cmd := &Command{
		Name:        "/test",
		Description: "Test command",
		Aliases:     []string{"/t", "/testcmd"},
		Handler:     nil,
	}

	RegisterCommand(cmd)

	// Check that command is registered
	if GetCommand("/test") == nil {
		t.Error("Command /test should be registered")
	}

	// Check aliases
	if GetCommand("/t") == nil {
		t.Error("Alias /t should be registered")
	}
	if GetCommand("/testcmd") == nil {
		t.Error("Alias /testcmd should be registered")
	}

	// All aliases should point to the same command
	if GetCommand("/test") != GetCommand("/t") {
		t.Error("Alias should point to the same command")
	}
}

func TestGetCommand_NotFound(t *testing.T) {
	// Clear registry before test
	commandRegistry = make(map[string]*Command)

	cmd := GetCommand("/nonexistent")
	if cmd != nil {
		t.Error("GetCommand() should return nil for unregistered command")
	}
}

func TestGetAllCommands(t *testing.T) {
	// Clear registry before test
	commandRegistry = make(map[string]*Command)

	// Register multiple commands
	RegisterCommand(&Command{Name: "/alpha", Description: "First"})
	RegisterCommand(&Command{Name: "/beta", Description: "Second"})
	RegisterCommand(&Command{Name: "/gamma", Description: "Third"})

	commands := GetAllCommands()

	if len(commands) != 3 {
		t.Errorf("GetAllCommands() returned %d commands, want 3", len(commands))
	}

	// Commands should be sorted by name
	if commands[0].Name != "/alpha" {
		t.Errorf("First command should be /alpha, got %s", commands[0].Name)
	}
	if commands[1].Name != "/beta" {
		t.Errorf("Second command should be /beta, got %s", commands[1].Name)
	}
	if commands[2].Name != "/gamma" {
		t.Errorf("Third command should be /gamma, got %s", commands[2].Name)
	}
}

func TestGetAllCommands_NoDuplicates(t *testing.T) {
	// Clear registry before test
	commandRegistry = make(map[string]*Command)

	// Register command with alias
	RegisterCommand(&Command{
		Name:    "/unique",
		Aliases: []string{"/u"},
	})

	commands := GetAllCommands()

	// Should only return one command (not including alias)
	if len(commands) != 1 {
		t.Errorf("GetAllCommands() returned %d commands, want 1 (no duplicates)", len(commands))
	}
}

func TestGetAvailableCommands(t *testing.T) {
	// Clear registry before test
	commandRegistry = make(map[string]*Command)

	RegisterCommand(&Command{Name: "/help", Description: "Show help"})
	RegisterCommand(&Command{Name: "/exit", Description: "Exit the program"})

	infos := GetAvailableCommands()

	if len(infos) != 2 {
		t.Errorf("GetAvailableCommands() returned %d commands, want 2", len(infos))
	}

	// Check that info contains correct data
	foundHelp := false
	foundExit := false
	for _, info := range infos {
		if info.Name == "/help" {
			foundHelp = true
			if info.Description != "Show help" {
				t.Errorf("Help description mismatch: %s", info.Description)
			}
		}
		if info.Name == "/exit" {
			foundExit = true
		}
	}

	if !foundHelp {
		t.Error("GetAvailableCommands() should include /help")
	}
	if !foundExit {
		t.Error("GetAvailableCommands() should include /exit")
	}
}

func TestMessageType_String(t *testing.T) {
	types := []MessageType{
		MessageTypeUser,
		MessageTypeAssistant,
		MessageTypeToolStart,
		MessageTypeToolEnd,
		MessageTypeError,
		MessageTypeSystem,
	}

	// Just verify these are valid enum values
	for _, mt := range types {
		if mt < 0 || mt > MessageTypeSystem {
			t.Errorf("Invalid MessageType: %d", mt)
		}
	}
}

func TestChatEvent(t *testing.T) {
	event := ChatEvent{
		Type:    EventTypeResponse,
		Content: "Test response",
		Extra: map[string]any{
			"key": "value",
		},
	}

	if event.Type != EventTypeResponse {
		t.Errorf("Event type = %v, want %v", event.Type, EventTypeResponse)
	}
	if event.Content != "Test response" {
		t.Errorf("Event content = %q, want %q", event.Content, "Test response")
	}
	if event.Extra["key"] != "value" {
		t.Errorf("Event extra[key] = %v, want %v", event.Extra["key"], "value")
	}
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Mode:       "chat",
		UserName:   "TestUser",
		Tools:      []string{"file_read", "bash"},
		ShowTokens: true,
		TokenMax:   8000,
		DebugMode:  true,
		SessionID:  "test-session",
	}

	if cfg.Mode != "chat" {
		t.Errorf("Config.Mode = %q, want %q", cfg.Mode, "chat")
	}
	if cfg.UserName != "TestUser" {
		t.Errorf("Config.UserName = %q, want %q", cfg.UserName, "TestUser")
	}
	if len(cfg.Tools) != 2 {
		t.Errorf("Config.Tools length = %d, want 2", len(cfg.Tools))
	}
}

func TestCommandInfo(t *testing.T) {
	info := CommandInfo{
		Name:        "/test",
		Description: "Test command description",
	}

	if info.Name != "/test" {
		t.Errorf("CommandInfo.Name = %q, want %q", info.Name, "/test")
	}
	if info.Description != "Test command description" {
		t.Errorf("CommandInfo.Description = %q, want %q", info.Description, "Test command description")
	}
}

func TestConfirmationRequest(t *testing.T) {
	req := ConfirmationRequest{
		Type:     ConfirmationSensitiveTool,
		ToolName: "bash",
		Params:   map[string]any{"command": "ls"},
		Message:  "Execute command?",
	}

	if req.Type != ConfirmationSensitiveTool {
		t.Errorf("ConfirmationRequest.Type = %v, want %v", req.Type, ConfirmationSensitiveTool)
	}
	if req.ToolName != "bash" {
		t.Errorf("ConfirmationRequest.ToolName = %q, want %q", req.ToolName, "bash")
	}
}

func TestConfirmState(t *testing.T) {
	state := ConfirmState{
		Waiting:  true,
		Selected: 0,
		Request: &ConfirmationRequest{
			Type:    ConfirmationSensitiveTool,
			Message: "Continue?",
		},
	}

	if !state.Waiting {
		t.Error("ConfirmState.Waiting should be true")
	}
	if state.Selected != 0 {
		t.Errorf("ConfirmState.Selected = %d, want 0", state.Selected)
	}
	if state.Request == nil {
		t.Error("ConfirmState.Request should not be nil")
	}
}

func TestModelProvider(t *testing.T) {
	provider := ModelProvider{}

	// Initially nil
	if provider.Get() != nil {
		t.Error("ModelProvider.Get() should return nil initially")
	}

	// Set model
	model := &Model{}
	provider.Set(model)

	if provider.Get() != model {
		t.Error("ModelProvider.Get() should return the set model")
	}
}

func TestUIStyles(t *testing.T) {
	styles := UIStyles{}

	// Just verify the struct can be created and fields exist
	// The actual style values depend on lipgloss
	_ = styles.UserMessage
	_ = styles.AuraMessage
	_ = styles.Thinking
	_ = styles.Action
	_ = styles.Result
	_ = styles.Error
	_ = styles.Help
	_ = styles.Processing
	_ = styles.Timestamp
	_ = styles.Separator
	_ = styles.Command
	_ = styles.CommandItemSelected
	_ = styles.CommandItem
	_ = styles.CommandDesc
}

func TestMessage(t *testing.T) {
	now := time.Now()
	msg := Message{
		ID:        "test-id",
		Type:      MessageTypeUser,
		Content:   "Hello",
		Rendered:  "[User]: Hello",
		Timestamp: now,
		Extra:     map[string]any{"source": "test"},
	}

	if msg.ID != "test-id" {
		t.Errorf("Message.ID = %q, want %q", msg.ID, "test-id")
	}
	if msg.Type != MessageTypeUser {
		t.Errorf("Message.Type = %v, want %v", msg.Type, MessageTypeUser)
	}
	if msg.Content != "Hello" {
		t.Errorf("Message.Content = %q, want %q", msg.Content, "Hello")
	}
	if msg.Rendered != "[User]: Hello" {
		t.Errorf("Message.Rendered = %q, want %q", msg.Rendered, "[User]: Hello")
	}
	if !msg.Timestamp.Equal(now) {
		t.Errorf("Message.Timestamp mismatch")
	}
}

func TestGetAvailableCommands_Mechanism(t *testing.T) {
	// Clear registry and test the GetAvailableCommands function
	commandRegistry = make(map[string]*Command)

	RegisterCommand(&Command{Name: "/test", Description: "Test"})

	// GetAvailableCommands should return registered commands
	commands := GetAvailableCommands()
	if len(commands) == 0 && len(commandRegistry) > 0 {
		t.Error("GetAvailableCommands() should not be empty when commands are registered")
	}
}
