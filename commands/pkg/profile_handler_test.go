// Package commands provides tests for ProfileHandler.
package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/oneliang/aura/personality/pkg/profile"
)

// TestNewProfileHandler tests the NewProfileHandler function.
func TestNewProfileHandler(t *testing.T) {
	var prof *profile.Profile

	handler := NewProfileHandler(prof)

	if handler == nil {
		t.Fatal("NewProfileHandler() returned nil")
	}
	if handler.profile != prof {
		t.Error("profile not set correctly")
	}
}

// TestProfileHandler_ExecuteCommand tests the ExecuteCommand method.
func TestProfileHandler_ExecuteCommand(t *testing.T) {
	handler := &ProfileHandler{
		profile: nil,
	}
	ctx := context.Background()

	tests := []struct {
		name    string
		cmd     string
		params  map[string]any
		wantErr bool
	}{
		{name: "show profile", cmd: "show", params: nil, wantErr: false},
		{name: "update profile", cmd: "update", params: nil, wantErr: false},
		{name: "unknown command", cmd: "unknown", params: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.ExecuteCommand(ctx, tt.cmd, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == "" {
				t.Error("ExecuteCommand() returned empty result")
			}
		})
	}
}

// TestProfileHandler_showProfile tests the showProfile method.
func TestProfileHandler_showProfile(t *testing.T) {
	t.Run("nil profile", func(t *testing.T) {
		handler := &ProfileHandler{profile: nil}

		result, err := handler.showProfile()
		if err != nil {
			t.Fatalf("showProfile() error = %v", err)
		}
		if result == "" {
			t.Fatal("showProfile() returned empty result")
		}
		if !strings.Contains(result, "No profile loaded") {
			t.Error("Result should contain 'No profile loaded'")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		handler := &ProfileHandler{profile: &profile.Profile{Content: ""}}

		result, err := handler.showProfile()
		if err != nil {
			t.Fatalf("showProfile() error = %v", err)
		}
		if !strings.Contains(result, "No profile loaded") {
			t.Error("Result should contain 'No profile loaded' for empty content")
		}
	})

	t.Run("with profile", func(t *testing.T) {
		prof := &profile.Profile{
			Content: "# About Me\n\n- Name: Test User\n- Occupation: Developer\n",
		}
		handler := &ProfileHandler{profile: prof}

		result, err := handler.showProfile()
		if err != nil {
			t.Fatalf("showProfile() error = %v", err)
		}
		if result == "" {
			t.Fatal("showProfile() returned empty result")
		}
		if !strings.Contains(result, "Test User") {
			t.Error("Result should contain profile name")
		}
		if !strings.Contains(result, "Developer") {
			t.Error("Result should contain occupation")
		}
	})
}

// TestProfileHandler_ExecuteCommand_ShowProfile tests show command.
func TestProfileHandler_ExecuteCommand_ShowProfile(t *testing.T) {
	prof := &profile.Profile{
		Content: "# About Me\n\n- Name: Test User\n",
	}
	handler := NewProfileHandler(prof)
	ctx := context.Background()

	result, err := handler.ExecuteCommand(ctx, "show", nil)
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	if result == "" {
		t.Fatal("ExecuteCommand() returned empty result")
	}
}

// TestProfileHandler_ExecuteCommand_UpdateProfile tests update command.
func TestProfileHandler_ExecuteCommand_UpdateProfile(t *testing.T) {
	prof := &profile.Profile{
		Content: "# About Me\n\n- Name: Test User\n",
	}
	handler := NewProfileHandler(prof)
	ctx := context.Background()

	result, err := handler.ExecuteCommand(ctx, "update", nil)
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	if result == "" {
		t.Fatal("ExecuteCommand() returned empty result")
	}
}
