// Package intent provides intent recognition service for the Core layer.
// This wraps the Layer 2 intent recognition (commands/pkg/intent) and provides
// a unified interface for the Runtime.
package intent

import (
	"context"
	"fmt"

	commands "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/commands/pkg/alias"
	intentpkg "github.com/oneliang/aura/commands/pkg/intent"
)

// IntentResult represents the result of intent recognition.
type IntentResult struct {
	Matched    bool
	Command    string
	Params     map[string]any
	Confidence float64
	Source     string
}

// Service provides intent recognition capabilities for the Core layer.
// It wraps the Layer 2 Recognizer and provides integration with CommandProvider.
type Service struct {
	recognizer      *intentpkg.Recognizer
	commandProvider commands.Command
	enabled         bool
}

// NewService creates a new intent service with the given command provider.
// If threshold is <= 0, defaults to 0.7.
func NewService(cmdProvider commands.Command, threshold float64) *Service {
	s := &Service{
		commandProvider: cmdProvider,
		enabled:         true,
	}
	if cmdProvider != nil {
		aliasMgr := alias.NewManager()
		cmdInfos := cmdProvider.GetCommands()
		s.recognizer = intentpkg.NewRecognizer(aliasMgr, cmdInfos, threshold)
	}
	return s
}

// Recognize attempts to recognize a command from natural language input.
// Returns nil if no match is found or if the service is disabled.
func (s *Service) Recognize(ctx context.Context, input string) (*IntentResult, error) {
	if !s.enabled || s.recognizer == nil {
		return nil, nil
	}
	result, err := s.recognizer.Recognize(ctx, input)
	if err != nil {
		return nil, err
	}
	if !result.Matched {
		return nil, nil
	}
	return &IntentResult{
		Matched:    result.Matched,
		Command:    result.Command,
		Params:     result.Params,
		Confidence: result.Confidence,
		Source:     result.Source,
	}, nil
}

// ExecuteCommand executes a recognized command.
func (s *Service) ExecuteCommand(ctx context.Context, result *IntentResult) (string, error) {
	if s.commandProvider == nil {
		return "", fmt.Errorf("command provider not available")
	}
	if result == nil {
		return "", fmt.Errorf("intent result is nil")
	}
	return s.commandProvider.Execute(ctx, result.Command, result.Params)
}

// SetEnabled enables or disables the intent service.
func (s *Service) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// IsEnabled returns whether the service is enabled.
func (s *Service) IsEnabled() bool {
	return s.enabled
}
