// Package intent provides natural language intent recognition for command execution.
package intent

import (
	"context"
	"testing"

	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/commands/pkg/alias"
)

func TestNewRecognizer(t *testing.T) {
	t.Run("with default threshold", func(t *testing.T) {
		aliasMgr := alias.NewManager()
		cmds := []commands.CommandInfo{
			{Name: commands.CmdNameExit, DisplayName: "Exit"},
		}
		r := NewRecognizer(aliasMgr, cmds, 0)
		if r == nil {
			t.Fatal("expected non-nil recognizer")
		}
		if r.confidenceThreshold != ConfidenceLow {
			t.Errorf("expected threshold %f, got %f", ConfidenceLow, r.confidenceThreshold)
		}
	})

	t.Run("with custom threshold", func(t *testing.T) {
		aliasMgr := alias.NewManager()
		cmds := []commands.CommandInfo{}
		r := NewRecognizer(aliasMgr, cmds, 0.8)
		if r.confidenceThreshold != 0.8 {
			t.Errorf("expected threshold 0.8, got %f", r.confidenceThreshold)
		}
	})
}

func TestRecognizer_Recognize(t *testing.T) {
	t.Run("alias match returns high confidence", func(t *testing.T) {
		aliasMgr := alias.NewManager()
		cmds := []commands.CommandInfo{
			{Name: commands.CmdNameExit, DisplayName: "Exit"},
		}
		r := NewRecognizer(aliasMgr, cmds, 0)

		result, err := r.Recognize(context.Background(), "exit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Matched {
			t.Error("expected match for 'exit'")
		}
		if result.Confidence != ConfidenceHigh {
			t.Errorf("expected high confidence %f, got %f", ConfidenceHigh, result.Confidence)
		}
		if result.Source != SourceAlias {
			t.Errorf("expected source %s, got %s", SourceAlias, result.Source)
		}
	})

	t.Run("command name match returns medium confidence", func(t *testing.T) {
		aliasMgr := alias.NewManager()
		cmds := []commands.CommandInfo{
			{Name: "test_command", DisplayName: "Test Command"},
		}
		r := NewRecognizer(aliasMgr, cmds, 0)

		result, err := r.Recognize(context.Background(), "test_command")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Matched {
			t.Error("expected match for 'test_command'")
		}
		if result.Confidence != ConfidenceMedium {
			t.Errorf("expected medium confidence %f, got %f", ConfidenceMedium, result.Confidence)
		}
		if result.Source != SourceCommand {
			t.Errorf("expected source %s, got %s", SourceCommand, result.Source)
		}
	})

	t.Run("keyword match returns low confidence if i18n loaded", func(t *testing.T) {
		aliasMgr := alias.NewManager()
		cmds := []commands.CommandInfo{}
		r := NewRecognizer(aliasMgr, cmds, 0)

		// Check if keywords are loaded from i18n
		keywords := r.keywordLoader.GetKeywords(KeywordExit)
		if len(keywords) == 0 {
			t.Skip("skip: exit keywords not loaded (i18n not initialized)")
		}

		result, err := r.Recognize(context.Background(), "再见") // Chinese "bye"
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Matched {
			t.Error("expected match for '再见' (exit keyword)")
		}
		if result.Confidence != ConfidenceLow {
			t.Errorf("expected low confidence %f, got %f", ConfidenceLow, result.Confidence)
		}
		if result.Source != SourceKeyword {
			t.Errorf("expected source %s, got %s", SourceKeyword, result.Source)
		}
	})

	t.Run("no match returns unmatched result", func(t *testing.T) {
		aliasMgr := alias.NewManager()
		cmds := []commands.CommandInfo{}
		r := NewRecognizer(aliasMgr, cmds, 0)

		result, err := r.Recognize(context.Background(), "hello world")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Matched {
			t.Error("expected no match for 'hello world'")
		}
	})
}

func TestRecognizer_Recognize_Threshold(t *testing.T) {
	t.Run("filters low confidence when threshold is high", func(t *testing.T) {
		aliasMgr := alias.NewManager()
		cmds := []commands.CommandInfo{}
		r := NewRecognizer(aliasMgr, cmds, 0.9) // High threshold

		result, err := r.Recognize(context.Background(), "再见")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Low confidence (0.7) should be filtered by threshold (0.9)
		if result.Matched {
			t.Error("expected no match due to threshold filtering")
		}
	})

	t.Run("keeps high confidence when threshold is high", func(t *testing.T) {
		aliasMgr := alias.NewManager()
		cmds := []commands.CommandInfo{
			{Name: commands.CmdNameExit, DisplayName: "Exit"},
		}
		r := NewRecognizer(aliasMgr, cmds, 0.9)

		result, err := r.Recognize(context.Background(), "exit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// High confidence (1.0) should pass threshold (0.9)
		if !result.Matched {
			t.Error("expected match for 'exit' (alias has high confidence)")
		}
	})
}

func TestRecognizer_matchByKeywords(t *testing.T) {
	aliasMgr := alias.NewManager()
	cmds := []commands.CommandInfo{}
	r := NewRecognizer(aliasMgr, cmds, 0)

	// Check if keywords are loaded from i18n
	keywords := r.keywordLoader.GetKeywords(KeywordExit)
	if len(keywords) == 0 {
		t.Skip("skip: keywords not loaded (i18n not initialized)")
	}

	t.Run("exit keywords", func(t *testing.T) {
		result := r.matchByKeywords("exit")
		if result == nil || !result.Matched {
			t.Error("expected match for exit keyword")
		}
		if result != nil && result.Command != commands.CmdNameExit {
			t.Errorf("expected command %s, got %s", commands.CmdNameExit, result.Command)
		}
	})

	t.Run("help keywords", func(t *testing.T) {
		result := r.matchByKeywords("help")
		if result == nil || !result.Matched {
			t.Error("expected match for help keyword")
		}
		if result != nil && result.Command != commands.CmdNameHelp {
			t.Errorf("expected command %s, got %s", commands.CmdNameHelp, result.Command)
		}
	})

	t.Run("session create keywords", func(t *testing.T) {
		result := r.matchByKeywords("create session")
		if result == nil || !result.Matched {
			t.Error("expected match for 'create session'")
		}
		if result != nil && result.Command != commands.CmdNameSessionCreate {
			t.Errorf("expected command %s, got %s", commands.CmdNameSessionCreate, result.Command)
		}
	})

	t.Run("session list keywords", func(t *testing.T) {
		result := r.matchByKeywords("list sessions")
		if result == nil || !result.Matched {
			t.Error("expected match for 'list sessions'")
		}
		if result != nil && result.Command != commands.CmdNameSessions {
			t.Errorf("expected command %s, got %s", commands.CmdNameSessions, result.Command)
		}
	})
}

func TestRecognizer_extractParams(t *testing.T) {
	aliasMgr := alias.NewManager()
	cmds := []commands.CommandInfo{}
	r := NewRecognizer(aliasMgr, cmds, 0)

	// Check if patterns are loaded from i18n
	patterns := r.keywordLoader.GetPatterns(PatternName)
	if len(patterns) == 0 {
		t.Skip("skip: patterns not loaded (i18n not initialized)")
	}

	t.Run("extracts name for session create", func(t *testing.T) {
		params := r.extractParams("create session called test", commands.CmdNameSessionCreate)
		if params == nil {
			t.Fatal("expected non-nil params")
		}
		if name, ok := params["name"].(string); !ok || name == "" {
			t.Error("expected name param to be extracted")
		}
	})
}
