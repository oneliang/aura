// Package learner provides a dialogue-based style learner.
package learner

import (
	"strings"

	"github.com/oneliang/aura/personality/pkg/style"
)

// Learner observes dialogue patterns and adjusts style parameters.
type Learner struct {
	style *style.Style
}

// New creates a new Learner backed by the given style pointer.
func New(s *style.Style) *Learner {
	return &Learner{style: s}
}

// Observe updates the style based on a user feedback message.
// Simple heuristic: look for sentiment keywords.
func (l *Learner) Observe(userMsg string) {
	lower := strings.ToLower(userMsg)

	var sig style.FeedbackSignal

	// Humor feedback
	if contains(lower, "funny", "haha", "lol", "humor", "joke") {
		sig.Humor += 1
	}
	if contains(lower, "serious", "formal", "professional") {
		sig.Humor -= 1
	}

	// Verbosity feedback
	if contains(lower, "more detail", "elaborate", "explain more", "tell me more") {
		sig.Verbosity += 1
	}
	if contains(lower, "brief", "short", "concise", "too long", "tldr", "tl;dr") {
		sig.Verbosity -= 1
	}

	if sig.Humor != 0 || sig.Verbosity != 0 {
		l.style.Apply(sig)
	}
}

func contains(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
