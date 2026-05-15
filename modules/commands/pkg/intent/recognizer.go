// Package intent provides natural language intent recognition for command execution.
package intent

import (
	"context"
	"strings"

	"github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/commands/pkg/alias"
)

// Recognizer recognizes intent from natural language input.
type Recognizer struct {
	aliasManager        *alias.Manager
	commands            []commands.CommandInfo
	keywordLoader       *KeywordLoader
	confidenceThreshold float64
}

// Result contains the recognition result.
type Result struct {
	Matched    bool           `json:"matched"`
	Command    string         `json:"command"`
	Params     map[string]any `json:"params"`
	Confidence float64        `json:"confidence"`
	Source     string         `json:"source"`
}

// NewRecognizer creates a new intent recognizer.
// If threshold is <= 0, defaults to 0.7.
func NewRecognizer(aliasMgr *alias.Manager, cmds []commands.CommandInfo, threshold float64) *Recognizer {
	if threshold <= 0 {
		threshold = ConfidenceLow // Default to 0.7
	}
	return &Recognizer{
		aliasManager:        aliasMgr,
		commands:            cmds,
		keywordLoader:       GetKeywordLoader(),
		confidenceThreshold: threshold,
	}
}

// Recognize attempts to recognize a command from natural language input.
// It first tries alias matching, then falls back to command name matching.
// Returns unmatched result if confidence is below threshold.
func (r *Recognizer) Recognize(ctx context.Context, input string) (*Result, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	// 1. Try alias resolution
	if r.aliasManager != nil {
		if cmd, ok := r.aliasManager.ResolveWithPrefix(input); ok {
			result := &Result{
				Matched:    true,
				Command:    cmd,
				Params:     r.extractParams(input, cmd),
				Confidence: ConfidenceHigh,
				Source:     SourceAlias,
			}
			return r.applyThreshold(result), nil
		}
	}

	// 2. Try direct command name matching
	for _, cmd := range r.commands {
		if strings.Contains(input, cmd.Name) || strings.Contains(input, cmd.DisplayName) {
			result := &Result{
				Matched:    true,
				Command:    cmd.Name,
				Params:     r.extractParams(input, cmd.Name),
				Confidence: ConfidenceMedium,
				Source:     SourceCommand,
			}
			return r.applyThreshold(result), nil
		}
	}

	// 3. Try keyword-based matching
	if result := r.matchByKeywords(input); result != nil {
		return r.applyThreshold(result), nil
	}

	return &Result{
		Matched:    false,
		Command:    "",
		Params:     nil,
		Confidence: 0,
		Source:     "",
	}, nil
}

// applyThreshold filters results below the confidence threshold.
func (r *Recognizer) applyThreshold(result *Result) *Result {
	if result.Confidence < r.confidenceThreshold {
		return &Result{
			Matched:    false,
			Command:    "",
			Params:     nil,
			Confidence: result.Confidence,
			Source:     result.Source,
		}
	}
	return result
}

// matchByKeywords attempts to match input using keyword patterns.
func (r *Recognizer) matchByKeywords(input string) *Result {
	kw := r.keywordLoader

	// Exit keywords
	if kw.ContainsKeyword(input, KeywordExit) {
		return &Result{Matched: true, Command: commands.CmdNameExit, Confidence: ConfidenceLow, Source: SourceKeyword}
	}

	// Help keywords
	if kw.ContainsKeyword(input, KeywordHelp) {
		return &Result{Matched: true, Command: commands.CmdNameHelp, Confidence: ConfidenceLow, Source: SourceKeyword}
	}

	// Session-related keywords
	if kw.ContainsKeyword(input, KeywordSession) {
		if kw.ContainsKeyword(input, KeywordCreate) {
			return &Result{Matched: true, Command: commands.CmdNameSessionCreate, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
		if kw.ContainsKeyword(input, KeywordList) {
			return &Result{Matched: true, Command: commands.CmdNameSessions, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
		if kw.ContainsKeyword(input, KeywordDelete) {
			return &Result{Matched: true, Command: commands.CmdNameSessionDelete, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
		// Clear session maps to CmdNameClear (clears current session memory)
		if kw.ContainsKeyword(input, KeywordClear) {
			return &Result{Matched: true, Command: commands.CmdNameClear, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
	}

	// Skill-related keywords
	if kw.ContainsKeyword(input, KeywordSkill) {
		if kw.ContainsKeyword(input, KeywordCreate) {
			return &Result{Matched: true, Command: commands.CmdNameSkillCreate, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
		if kw.ContainsKeyword(input, KeywordList) {
			return &Result{Matched: true, Command: commands.CmdNameSkillList, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
		if kw.ContainsKeyword(input, KeywordDelete) {
			return &Result{Matched: true, Command: commands.CmdNameSkillDelete, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
	}

	// Agent-related keywords
	if kw.ContainsKeyword(input, KeywordAgent) {
		if kw.ContainsKeyword(input, KeywordCreate) {
			return &Result{Matched: true, Command: commands.CmdNameAgentCreate, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
		if kw.ContainsKeyword(input, KeywordList) {
			return &Result{Matched: true, Command: commands.CmdNameAgentList, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
		if kw.ContainsKeyword(input, KeywordDelete) {
			return &Result{Matched: true, Command: commands.CmdNameAgentDelete, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
	}

	// Config-related keywords
	if kw.ContainsKeyword(input, KeywordConfig) {
		if kw.ContainsKeyword(input, KeywordShow) {
			return &Result{Matched: true, Command: commands.CmdNameConfigShow, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
	}

	// Memory-related keywords
	if kw.ContainsKeyword(input, KeywordMemory) {
		if kw.ContainsKeyword(input, KeywordClear) {
			return &Result{Matched: true, Command: commands.CmdNameClear, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
		if kw.ContainsKeyword(input, KeywordStats) {
			return &Result{Matched: true, Command: commands.CmdNameMemory, Confidence: ConfidenceLow, Source: SourceKeyword}
		}
	}

	return nil
}

// extractParams attempts to extract parameters from the input.
func (r *Recognizer) extractParams(input, cmd string) map[string]any {
	params := make(map[string]any)

	switch cmd {
	case commands.CmdNameSessionCreate, commands.CmdNameSkillCreate, commands.CmdNameAgentCreate:
		name := r.keywordLoader.ExtractName(input)
		if name != "" {
			params["name"] = name
		}
	case commands.CmdNameConfigGet:
		key := r.keywordLoader.ExtractConfigKey(input)
		if key != "" {
			params["key"] = key
		}
	}

	return params
}
