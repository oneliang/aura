package style

import (
	"fmt"
	"strings"
	"testing"
)

// TestDefaultStyle tests default style creation.
func TestDefaultStyle(t *testing.T) {
	s := Default()

	if s.Tone != ToneCasual {
		t.Errorf("Tone = %q, want %q", s.Tone, ToneCasual)
	}
	if s.Vocabulary != "simple" {
		t.Errorf("Vocabulary = %q, want %q", s.Vocabulary, "simple")
	}
	if s.Humor != 0.3 {
		t.Errorf("Humor = %v, want %v", s.Humor, 0.3)
	}
	if s.Verbosity != VerbosityConcise {
		t.Errorf("Verbosity = %q, want %q", s.Verbosity, VerbosityConcise)
	}
}

// TestStyleToPromptFragment tests ToPromptFragment method.
func TestStyleToPromptFragment(t *testing.T) {
	tests := []struct {
		name        string
		style       Style
		contains    []string
		notContains []string
	}{
		{
			name:  "default style",
			style: Default(),
			contains: []string{
				"Communication style:",
				"Tone:",
				"Vocabulary:",
				"Verbosity:",
			},
		},
		{
			name: "high humor style",
			style: Style{
				Tone:       ToneCasual,
				Vocabulary: "simple",
				Humor:      0.8,
				Verbosity:  VerbosityConcise,
			},
			contains: []string{
				"humor",
				"light-hearted",
			},
		},
		{
			name: "low humor style",
			style: Style{
				Tone:       ToneFormal,
				Vocabulary: "technical",
				Humor:      0.05,
				Verbosity:  VerbosityDetailed,
			},
			contains: []string{
				"serious",
				"professional",
			},
		},
		{
			name: "medium humor style",
			style: Style{
				Tone:       ToneTechnical,
				Vocabulary: "technical",
				Humor:      0.5,
				Verbosity:  VerbosityConcise,
			},
			notContains: []string{
				"humor",
				"serious",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.style.ToPromptFragment()

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("ToPromptFragment() missing %q", want)
				}
			}

			for _, notWant := range tt.notContains {
				if strings.Contains(result, notWant) {
					t.Errorf("ToPromptFragment() should not contain %q", notWant)
				}
			}
		})
	}
}

// TestStyleApply tests Apply method with feedback signals.
func TestStyleApply(t *testing.T) {
	tests := []struct {
		name        string
		initial     Style
		signal      FeedbackSignal
		expectHumor float64
		expectVerb  string
	}{
		{
			name: "positive humor feedback",
			initial: Style{
				Tone:       ToneCasual,
				Vocabulary: "simple",
				Humor:      0.3,
				Verbosity:  VerbosityConcise,
			},
			signal: FeedbackSignal{
				Humor:     1,
				Verbosity: 0,
			},
			expectHumor: 0.4, // 0.3 + 1*0.1
			expectVerb:  VerbosityConcise,
		},
		{
			name: "negative humor feedback",
			initial: Style{
				Tone:       ToneCasual,
				Vocabulary: "simple",
				Humor:      0.5,
				Verbosity:  VerbosityConcise,
			},
			signal: FeedbackSignal{
				Humor:     -1,
				Verbosity: 0,
			},
			expectHumor: 0.4, // 0.5 - 1*0.1
			expectVerb:  VerbosityConcise,
		},
		{
			name: "positive verbosity feedback",
			initial: Style{
				Tone:       ToneCasual,
				Vocabulary: "simple",
				Humor:      0.3,
				Verbosity:  VerbosityConcise,
			},
			signal: FeedbackSignal{
				Humor:     0,
				Verbosity: 1,
			},
			expectHumor: 0.3,
			expectVerb:  VerbosityConcise, // 0.25 + 0.1 = 0.35 -> still concise (< 0.5)
		},
		{
			name: "clamp humor max",
			initial: Style{
				Tone:       ToneCasual,
				Vocabulary: "simple",
				Humor:      0.95,
				Verbosity:  VerbosityConcise,
			},
			signal: FeedbackSignal{
				Humor:     1,
				Verbosity: 0,
			},
			expectHumor: 1.0, // Should clamp to 1.0
			expectVerb:  VerbosityConcise,
		},
		{
			name: "clamp humor min",
			initial: Style{
				Tone:       ToneCasual,
				Vocabulary: "simple",
				Humor:      0.05,
				Verbosity:  VerbosityConcise,
			},
			signal: FeedbackSignal{
				Humor:     -1,
				Verbosity: 0,
			},
			expectHumor: 0.0, // Should clamp to 0.0
			expectVerb:  VerbosityConcise,
		},
		{
			name: "verbosity change to detailed",
			initial: Style{
				Tone:       ToneCasual,
				Vocabulary: "simple",
				Humor:      0.3,
				Verbosity:  VerbosityConcise,
			},
			signal: FeedbackSignal{
				Humor:     0,
				Verbosity: 3, // Large positive signal
			},
			expectHumor: 0.3,
			expectVerb:  VerbosityDetailed, // Should cross threshold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.Apply(tt.signal)

			if s.Humor != tt.expectHumor {
				t.Errorf("Humor = %v, want %v", s.Humor, tt.expectHumor)
			}
			if s.Verbosity != tt.expectVerb {
				t.Errorf("Verbosity = %q, want %q", s.Verbosity, tt.expectVerb)
			}
		})
	}
}

// TestClamp tests the clamp helper function.
func TestClamp(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		min    float64
		max    float64
		expect float64
	}{
		{"in range", 0.5, 0, 1, 0.5},
		{"below min", -0.5, 0, 1, 0},
		{"above max", 1.5, 0, 1, 1},
		{"at min", 0, 0, 1, 0},
		{"at max", 1, 0, 1, 1},
		{"negative range", -5, -10, 0, -5},
		{"negative below min", -15, -10, 0, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clamp(tt.value, tt.min, tt.max)
			if result != tt.expect {
				t.Errorf("clamp(%v, %v, %v) = %v, want %v", tt.value, tt.min, tt.max, result, tt.expect)
			}
		})
	}
}

// TestVerbosityToFloat tests verbosityToFloat function.
func TestVerbosityToFloat(t *testing.T) {
	tests := []struct {
		verbosity string
		expect    float64
	}{
		{VerbosityConcise, 0.25},
		{VerbosityDetailed, 0.75},
		{"unknown", 0.5},
		{"", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.verbosity, func(t *testing.T) {
			result := verbosityToFloat(tt.verbosity)
			if result != tt.expect {
				t.Errorf("verbosityToFloat(%q) = %v, want %v", tt.verbosity, result, tt.expect)
			}
		})
	}
}

// TestFloatToVerbosity tests floatToVerbosity function.
func TestFloatToVerbosity(t *testing.T) {
	tests := []struct {
		value  float64
		expect string
	}{
		{0.0, VerbosityConcise},
		{0.25, VerbosityConcise},
		{0.49, VerbosityConcise},
		{0.5, VerbosityDetailed},
		{0.75, VerbosityDetailed},
		{1.0, VerbosityDetailed},
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(fmt.Sprintf("%.2f", tt.value), ".", "_"), func(t *testing.T) {
			result := floatToVerbosity(tt.value)
			if result != tt.expect {
				t.Errorf("floatToVerbosity(%v) = %q, want %q", tt.value, result, tt.expect)
			}
		})
	}
}

// TestStyleToneConstants tests tone constants.
func TestStyleToneConstants(t *testing.T) {
	if ToneFormal != "formal" {
		t.Errorf("ToneFormal = %q, want %q", ToneFormal, "formal")
	}
	if ToneCasual != "casual" {
		t.Errorf("ToneCasual = %q, want %q", ToneCasual, "casual")
	}
	if ToneTechnical != "technical" {
		t.Errorf("ToneTechnical = %q, want %q", ToneTechnical, "technical")
	}
}

// TestStyleVerbosityConstants tests verbosity constants.
func TestStyleVerbosityConstants(t *testing.T) {
	if VerbosityConcise != "concise" {
		t.Errorf("VerbosityConcise = %q, want %q", VerbosityConcise, "concise")
	}
	if VerbosityDetailed != "detailed" {
		t.Errorf("VerbosityDetailed = %q, want %q", VerbosityDetailed, "detailed")
	}
}

// TestStyleApplyMultipleSignals tests applying multiple signals.
func TestStyleApplyMultipleSignals(t *testing.T) {
	s := Default()

	// Apply multiple positive humor signals
	for i := 0; i < 5; i++ {
		s.Apply(FeedbackSignal{Humor: 1, Verbosity: 0})
	}

	// 0.3 + 5*0.1 = 0.8 (with floating point tolerance)
	if s.Humor < 0.79 || s.Humor > 0.81 {
		t.Errorf("After 5 positive humor signals, Humor = %v, want ~0.8", s.Humor)
	}

	// Apply multiple negative humor signals
	for i := 0; i < 10; i++ {
		s.Apply(FeedbackSignal{Humor: -1, Verbosity: 0})
	}

	if s.Humor != 0.0 { // Should be clamped to 0
		t.Errorf("After 10 negative humor signals, Humor = %v, want %v", s.Humor, 0.0)
	}
}

// TestStylePromptFragmentFormat tests the format of ToPromptFragment output.
func TestStylePromptFragmentFormat(t *testing.T) {
	s := Style{
		Tone:       ToneTechnical,
		Vocabulary: "technical",
		Humor:      0.5,
		Verbosity:  VerbosityDetailed,
	}

	result := s.ToPromptFragment()

	// Check format
	lines := strings.Split(result, "\n")
	if len(lines) < 4 {
		t.Errorf("ToPromptFragment() should have at least 4 lines, got %d", len(lines))
	}

	// First line should be the header
	if !strings.Contains(lines[0], "Communication style") {
		t.Errorf("First line should contain 'Communication style', got %q", lines[0])
	}
}
