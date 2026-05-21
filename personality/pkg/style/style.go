// Package style provides communication style management.
package style

import (
	"fmt"
	"strings"
)

// Tone constants.
const (
	ToneFormal    = "formal"
	ToneCasual    = "casual"
	ToneTechnical = "technical"
)

// Verbosity constants.
const (
	VerbosityConcise  = "concise"
	VerbosityDetailed = "detailed"
)

// Style holds communication style parameters.
type Style struct {
	Tone       string  `yaml:"tone"`
	Vocabulary string  `yaml:"vocabulary"`
	Humor      float64 `yaml:"humor"` // 0.0-1.0
	Verbosity  string  `yaml:"verbosity"`
}

// Default returns a default casual style.
func Default() Style {
	return Style{
		Tone:       ToneCasual,
		Vocabulary: "simple",
		Humor:      0.3,
		Verbosity:  VerbosityConcise,
	}
}

// ToPromptFragment renders the style as a system-prompt fragment.
func (s Style) ToPromptFragment() string {
	var sb strings.Builder
	sb.WriteString("Communication style:\n")
	sb.WriteString(fmt.Sprintf("- Tone: %s\n", s.Tone))
	sb.WriteString(fmt.Sprintf("- Vocabulary: %s\n", s.Vocabulary))
	sb.WriteString(fmt.Sprintf("- Verbosity: %s\n", s.Verbosity))

	if s.Humor >= 0.7 {
		sb.WriteString("- Feel free to use humor and light-hearted remarks.\n")
	} else if s.Humor <= 0.1 {
		sb.WriteString("- Maintain a serious and professional tone.\n")
	}

	return sb.String()
}

// Adjust tweaks the style based on feedback signal (-1 to +1).
// positive signal = be more like this, negative = less like this.
type FeedbackSignal struct {
	Humor     float64 // +: more humor, -: less humor
	Verbosity float64 // +: more detailed, -: more concise
}

// Apply applies a feedback signal to the style (clamped 0-1).
func (s *Style) Apply(sig FeedbackSignal) {
	s.Humor = clamp(s.Humor+sig.Humor*0.1, 0, 1)
	// verbosity uses a continuous scale mapped to string
	vf := verbosityToFloat(s.Verbosity) + sig.Verbosity*0.1
	s.Verbosity = floatToVerbosity(clamp(vf, 0, 1))
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func verbosityToFloat(v string) float64 {
	switch v {
	case VerbosityConcise:
		return 0.25
	case VerbosityDetailed:
		return 0.75
	default:
		return 0.5
	}
}

func floatToVerbosity(f float64) string {
	if f < 0.5 {
		return VerbosityConcise
	}
	return VerbosityDetailed
}
