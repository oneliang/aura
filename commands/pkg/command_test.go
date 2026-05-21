// Package commands provides tests for the commands package.
package commands

import (
	"testing"
)

// TestCommandInfo_GetName tests the GetName method.
func TestCommandInfo_GetName(t *testing.T) {
	tests := []struct {
		name string
		info CommandInfo
		want string
	}{
		{
			name: "basic command",
			info: CommandInfo{Name: "command_test"},
			want: "command_test",
		},
		{
			name: "empty name",
			info: CommandInfo{Name: ""},
			want: "",
		},
		{
			name: "command with prefix",
			info: CommandInfo{Name: CmdNameSessionCreate},
			want: CmdNameSessionCreate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.GetName()
			if got != tt.want {
				t.Errorf("GetName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCommandInfo_GetDescription tests the GetDescription method.
func TestCommandInfo_GetDescription(t *testing.T) {
	tests := []struct {
		name string
		info CommandInfo
		want string
	}{
		{
			name: "basic description",
			info: CommandInfo{Description: "Test description"},
			want: "Test description",
		},
		{
			name: "empty description",
			info: CommandInfo{Description: ""},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.GetDescription()
			if got != tt.want {
				t.Errorf("GetDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGetInternalCommands tests the GetInternalCommands function.
func TestGetInternalCommands(t *testing.T) {
	cmds := GetInternalCommands()

	if len(cmds) == 0 {
		t.Fatal("GetInternalCommands() returned empty slice")
	}

	// Verify command structure
	for i, cmd := range cmds {
		if cmd.Name == "" {
			t.Errorf("Command %d has empty Name", i)
		}
		if cmd.DisplayName == "" {
			t.Errorf("Command %d has empty DisplayName", i)
		}
		if cmd.Description == "" {
			t.Errorf("Command %d has empty Description", i)
		}
	}

	// Verify expected commands exist
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

	cmdMap := make(map[string]bool)
	for _, cmd := range cmds {
		cmdMap[cmd.Name] = true
	}

	for _, expected := range expectedCommands {
		if !cmdMap[expected] {
			t.Errorf("Expected command %q not found", expected)
		}
	}
}

// TestGetInternalCommands_ParamDefinitions tests parameter definitions for commands.
func TestGetInternalCommands_ParamDefinitions(t *testing.T) {
	cmds := GetInternalCommands()

	tests := []struct {
		commandName    string
		wantParamCount int
	}{
		{CmdNameExit, 0},
		{CmdNameClear, 0},
		{CmdNameSessionCreate, 2},   // name, role
		{CmdNameSessionShow, 1},     // id
		{CmdNameSessionDelete, 1},   // id
		{CmdNameKnowledgeSearch, 1}, // query
		{CmdNameKnowledgeImport, 1}, // path
		{CmdNameSubscriptionAdd, 3}, // session_id, trigger, source
	}

	cmdMap := make(map[string]CommandInfo)
	for _, cmd := range cmds {
		cmdMap[cmd.Name] = cmd
	}

	for _, tt := range tests {
		t.Run(tt.commandName, func(t *testing.T) {
			cmd, ok := cmdMap[tt.commandName]
			if !ok {
				t.Fatalf("Command %q not found", tt.commandName)
			}

			if len(cmd.Params) != tt.wantParamCount {
				t.Errorf("Command %q has %d params, want %d", tt.commandName, len(cmd.Params), tt.wantParamCount)
			}
		})
	}
}

// TestParamInfo tests ParamInfo structure.
func TestParamInfo(t *testing.T) {
	param := ParamInfo{
		Name:     "test_param",
		Type:     "string",
		Required: true,
		Desc:     "Test parameter description",
	}

	if param.Name != "test_param" {
		t.Errorf("Name = %q, want %q", param.Name, "test_param")
	}
	if param.Type != "string" {
		t.Errorf("Type = %q, want %q", param.Type, "string")
	}
	if !param.Required {
		t.Error("Required should be true")
	}
	if param.Desc != "Test parameter description" {
		t.Errorf("Desc = %q, want %q", param.Desc, "Test parameter description")
	}
}
