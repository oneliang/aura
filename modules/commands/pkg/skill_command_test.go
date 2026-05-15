// Package commands provides tests for SkillCommand.
package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/oneliang/aura/skill/pkg/loader"
)

// TestNewSkillCommand tests the NewSkillCommand function.
func TestNewSkillCommand(t *testing.T) {
	var ld *loader.Loader

	cmd := NewSkillCommand(ld)

	if cmd == nil {
		t.Fatal("NewSkillCommand() returned nil")
	}
	if cmd.loader != ld {
		t.Error("loader not set correctly")
	}
}

// TestSkillCommand_GetCommands_NilLoader tests GetCommands with nil loader.
func TestSkillCommand_GetCommands_NilLoader(t *testing.T) {
	cmd := NewSkillCommand(nil)
	commands := cmd.GetCommands()

	if commands != nil {
		t.Errorf("GetCommands() with nil loader should return nil, got %v", commands)
	}
}

// TestSkillCommand_Execute_NilLoader tests Execute with nil loader.
func TestSkillCommand_Execute_NilLoader(t *testing.T) {
	cmd := NewSkillCommand(nil)
	ctx := context.Background()

	_, err := cmd.Execute(ctx, "any-skill", nil)
	if err == nil {
		t.Error("Execute() with nil loader should return error")
	}
}

// TestSkillCommand_Execute_SkillNotFound tests skill not found error.
func TestSkillCommand_Execute_SkillNotFound(t *testing.T) {
	tempDir := t.TempDir()
	ld := loader.NewLoader([]string{tempDir})
	cmd := NewSkillCommand(ld)
	ctx := context.Background()

	_, err := cmd.Execute(ctx, "non-existent-skill", nil)
	if err == nil {
		t.Error("Execute() should return error for non-existent skill")
	}
	if !strings.Contains(err.Error(), "skill not found") {
		t.Errorf("Error should contain 'skill not found', got: %v", err)
	}
}

// TestSkillCommand_buildCommandName tests the buildCommandName method.
func TestSkillCommand_buildCommandName(t *testing.T) {
	cmd := &SkillCommand{}

	tests := []struct {
		skillName string
		want      string
	}{
		{"my-skill", "skill_my-skill"},
		{"test", "skill_test"},
		{"complex-skill-name", "skill_complex-skill-name"},
	}

	for _, tt := range tests {
		t.Run(tt.skillName, func(t *testing.T) {
			got := cmd.buildCommandName(tt.skillName)
			if got != tt.want {
				t.Errorf("buildCommandName(%q) = %q, want %q", tt.skillName, got, tt.want)
			}
		})
	}
}

// TestSkillCommand_extractSkillName tests the extractSkillName method.
func TestSkillCommand_extractSkillName(t *testing.T) {
	cmd := &SkillCommand{}

	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{name: "with prefix", cmd: "skill_my-skill", want: "my-skill"},
		{name: "without prefix", cmd: "my-skill", want: "my-skill"},
		{name: "short prefix", cmd: "skill_x", want: "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cmd.extractSkillName(tt.cmd)
			if got != tt.want {
				t.Errorf("extractSkillName(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

// TestSkillCommand_buildSkillContext tests the buildSkillContext method.
func TestSkillCommand_buildSkillContext(t *testing.T) {
	cmd := &SkillCommand{}

	t.Run("nil params", func(t *testing.T) {
		result := cmd.buildSkillContext(nil)
		if result != "" {
			t.Errorf("buildSkillContext(nil) = %q, want ''", result)
		}
	})

	t.Run("empty params", func(t *testing.T) {
		result := cmd.buildSkillContext(map[string]any{})
		if result != "" {
			t.Errorf("buildSkillContext(empty) = %q, want ''", result)
		}
	})

	t.Run("with params", func(t *testing.T) {
		params := map[string]any{
			"key1": "value1",
			"key2": 123,
		}
		result := cmd.buildSkillContext(params)
		if result == "" {
			t.Fatal("buildSkillContext() returned empty result")
		}
		if !strings.Contains(result, "key1: value1") {
			t.Error("Result should contain key1")
		}
		if !strings.Contains(result, "key2: 123") {
			t.Error("Result should contain key2")
		}
	})
}

// TestSkillCommand_GetSkills_NilLoader tests GetSkills with nil loader.
func TestSkillCommand_GetSkills_NilLoader(t *testing.T) {
	cmd := NewSkillCommand(nil)
	skills := cmd.GetSkills()
	if skills != nil {
		t.Errorf("GetSkills() with nil loader should return nil, got %v", skills)
	}
}
