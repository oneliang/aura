// Package intent provides intent recognition service for the Core layer.
package intent

import (
	"context"
	"testing"

	commands "github.com/oneliang/aura/commands/pkg"
)

// mockCommandProvider is a mock implementation of commands.Command for testing.
type mockCommandProvider struct {
	commands []commands.CommandInfo
}

func (m *mockCommandProvider) GetCommands() []commands.CommandInfo {
	return m.commands
}

func (m *mockCommandProvider) Execute(ctx context.Context, cmd string, params map[string]any) (string, error) {
	return "executed: " + cmd, nil
}

func TestNewService(t *testing.T) {
	t.Run("with command provider", func(t *testing.T) {
		mock := &mockCommandProvider{
			commands: []commands.CommandInfo{
				{Name: commands.CmdNameExit, DisplayName: "Exit", Description: "Exit the application"},
			},
		}
		svc := NewService(mock, 0) // Use default threshold
		if svc == nil {
			t.Fatal("expected non-nil service")
		}
		if !svc.IsEnabled() {
			t.Error("expected service to be enabled by default")
		}
	})

	t.Run("with nil command provider", func(t *testing.T) {
		svc := NewService(nil, 0)
		if svc == nil {
			t.Fatal("expected non-nil service")
		}
	})
}

func TestService_Recognize(t *testing.T) {
	t.Run("recognizes exit command via alias", func(t *testing.T) {
		mock := &mockCommandProvider{
			commands: []commands.CommandInfo{
				{Name: commands.CmdNameExit, DisplayName: "Exit", Description: "Exit the application"},
			},
		}
		svc := NewService(mock, 0)

		result, err := svc.Recognize(context.Background(), "exit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result for 'exit'")
		}
		if !result.Matched {
			t.Error("expected 'exit' to match")
		}
		if result.Command != commands.CmdNameExit {
			t.Errorf("expected command %s, got %s", commands.CmdNameExit, result.Command)
		}
	})

	t.Run("recognizes quit command via alias", func(t *testing.T) {
		mock := &mockCommandProvider{
			commands: []commands.CommandInfo{
				{Name: commands.CmdNameQuit, DisplayName: "Quit", Description: "Quit the application"},
			},
		}
		svc := NewService(mock, 0)

		result, err := svc.Recognize(context.Background(), "quit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result for 'quit'")
		}
		if !result.Matched {
			t.Error("expected 'quit' to match")
		}
	})

	t.Run("returns nil for non-matching input", func(t *testing.T) {
		mock := &mockCommandProvider{
			commands: []commands.CommandInfo{
				{Name: commands.CmdNameExit, DisplayName: "Exit", Description: "Exit the application"},
			},
		}
		svc := NewService(mock, 0)

		result, err := svc.Recognize(context.Background(), "hello world")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result for non-matching input, got %+v", result)
		}
	})

	t.Run("returns nil when disabled", func(t *testing.T) {
		mock := &mockCommandProvider{
			commands: []commands.CommandInfo{
				{Name: commands.CmdNameExit, DisplayName: "Exit", Description: "Exit the application"},
			},
		}
		svc := NewService(mock, 0)
		svc.SetEnabled(false)

		result, err := svc.Recognize(context.Background(), "exit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result when disabled, got %+v", result)
		}
	})
}

func TestService_ExecuteCommand(t *testing.T) {
	t.Run("executes matched command", func(t *testing.T) {
		mock := &mockCommandProvider{
			commands: []commands.CommandInfo{
				{Name: commands.CmdNameExit, DisplayName: "Exit", Description: "Exit the application"},
			},
		}
		svc := NewService(mock, 0)

		result, _ := svc.Recognize(context.Background(), "exit")
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		output, err := svc.ExecuteCommand(context.Background(), result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != "executed: "+commands.CmdNameExit {
			t.Errorf("unexpected output: %s", output)
		}
	})

	t.Run("returns error for nil result", func(t *testing.T) {
		mock := &mockCommandProvider{
			commands: []commands.CommandInfo{},
		}
		svc := NewService(mock, 0)

		_, err := svc.ExecuteCommand(context.Background(), nil)
		if err == nil {
			t.Error("expected error for nil result")
		}
	})
}

func TestService_SetEnabled(t *testing.T) {
	mock := &mockCommandProvider{
		commands: []commands.CommandInfo{
			{Name: commands.CmdNameExit, DisplayName: "Exit", Description: "Exit"},
		},
	}
	svc := NewService(mock, 0)

	if !svc.IsEnabled() {
		t.Error("expected enabled by default")
	}

	svc.SetEnabled(false)
	if svc.IsEnabled() {
		t.Error("expected disabled after SetEnabled(false)")
	}

	svc.SetEnabled(true)
	if !svc.IsEnabled() {
		t.Error("expected enabled after SetEnabled(true)")
	}
}
