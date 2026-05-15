package alias

import (
	"testing"

	"github.com/oneliang/aura/commands/pkg"
)

// TestNewManager tests NewManager function.
func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.aliases == nil {
		t.Error("aliases map should be initialized")
	}
}

// TestManager_Register tests Register method.
func TestManager_Register(t *testing.T) {
	m := NewManager()

	err := m.Register("test alias", "test_command")
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	cmd, ok := m.Resolve("test alias")
	if !ok {
		t.Error("Expected alias to be registered")
	}
	if cmd != "test_command" {
		t.Errorf("Expected 'test_command', got '%s'", cmd)
	}
}

// TestManager_Unregister tests Unregister method.
func TestManager_Unregister(t *testing.T) {
	m := NewManager()

	m.Register("test alias", "test_command")
	err := m.Unregister("test alias")
	if err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	_, ok := m.Resolve("test alias")
	if ok {
		t.Error("Expected alias to be unregistered")
	}
}

// TestManager_Resolve tests Resolve method.
func TestManager_Resolve(t *testing.T) {
	m := NewManager()

	// Test built-in alias
	cmd, ok := m.Resolve("create session")
	if !ok {
		t.Error("Expected to find built-in alias 'create session'")
	}
	if cmd != commands.CmdNameSessionCreate {
		t.Errorf("Expected '%s', got '%s'", commands.CmdNameSessionCreate, cmd)
	}

	// Test non-existent alias
	_, ok = m.Resolve("nonexistent alias")
	if ok {
		t.Error("Expected not to find non-existent alias")
	}
}

// TestManager_List tests List method.
func TestManager_List(t *testing.T) {
	m := NewManager()

	list := m.List()
	if list == nil {
		t.Fatal("List() returned nil")
	}

	// Check that built-in aliases are present
	if _, ok := list["create session"]; !ok {
		t.Error("Expected 'create session' alias to be present")
	}

	// Verify it's a copy by modifying the returned map
	list["new test alias"] = "test_command"
	_, ok := m.Resolve("new test alias")
	if ok {
		t.Error("Modifying returned map should not affect internal state")
	}
}

// TestManager_ResolveWithPrefix tests ResolveWithPrefix method.
func TestManager_ResolveWithPrefix(t *testing.T) {
	m := NewManager()

	tests := []struct {
		input     string
		wantCmd   string
		wantFound bool
	}{
		{
			input:     "create session",
			wantCmd:   commands.CmdNameSessionCreate,
			wantFound: true,
		},
		{
			input:     "create session called mysession",
			wantCmd:   commands.CmdNameSessionCreate,
			wantFound: true,
		},
		{
			input:     "clear memory",
			wantCmd:   commands.CmdNameClear,
			wantFound: true,
		},
		{
			input:     "nonexistent command",
			wantCmd:   "",
			wantFound: false,
		},
		{
			input:     "xyz",
			wantCmd:   "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd, found := m.ResolveWithPrefix(tt.input)
			if found != tt.wantFound {
				t.Errorf("ResolveWithPrefix(%q) found = %v, want %v", tt.input, found, tt.wantFound)
			}
			if cmd != tt.wantCmd {
				t.Errorf("ResolveWithPrefix(%q) cmd = %q, want %q", tt.input, cmd, tt.wantCmd)
			}
		})
	}
}

// TestManager_BuiltInAliases tests that all built-in aliases are registered.
func TestManager_BuiltInAliases(t *testing.T) {
	m := NewManager()

	builtInAliases := []struct {
		alias   string
		command string
	}{
		{"create session", commands.CmdNameSessionCreate},
		{"new session", commands.CmdNameSessionCreate},
		{"list sessions", commands.CmdNameSessions},
		{"delete session", commands.CmdNameSessionDelete},
		{"show profile", commands.CmdNameProfileShow},
		{"show config", commands.CmdNameConfigShow},
		{"clear memory", commands.CmdNameClear},
		{"memory stats", commands.CmdNameMemory},
		{"search knowledge", commands.CmdNameKnowledgeSearch},
		{"list skills", commands.CmdNameSkillList},
		{"list agents", commands.CmdNameAgentList},
		{"help", commands.CmdNameHelp},
		{"list tools", commands.CmdNameTools},
		{"status", commands.CmdNameStatus},
		{"quit", commands.CmdNameQuit},
		{"exit", commands.CmdNameExit},
	}

	for _, tt := range builtInAliases {
		t.Run(tt.alias, func(t *testing.T) {
			cmd, ok := m.Resolve(tt.alias)
			if !ok {
				t.Errorf("Built-in alias '%s' not found", tt.alias)
			}
			if cmd != tt.command {
				t.Errorf("Alias '%s' = '%s', want '%s'", tt.alias, cmd, tt.command)
			}
		})
	}
}

// TestManager_ConcurrentAccess tests concurrent access to the manager.
func TestManager_ConcurrentAccess(t *testing.T) {
	m := NewManager()

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			m.Register("concurrent alias", "test_command")
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			m.Resolve("concurrent alias")
		}
		done <- true
	}()

	// List goroutine
	go func() {
		for i := 0; i < 100; i++ {
			m.List()
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done
}

// TestManager_Overwrite tests that Register overwrites existing aliases.
func TestManager_Overwrite(t *testing.T) {
	m := NewManager()

	m.Register("test alias", "command1")
	m.Register("test alias", "command2")

	cmd, ok := m.Resolve("test alias")
	if !ok {
		t.Error("Expected alias to exist")
	}
	if cmd != "command2" {
		t.Errorf("Expected 'command2', got '%s'", cmd)
	}
}
