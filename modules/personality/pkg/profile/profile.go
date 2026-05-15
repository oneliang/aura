// Package profile provides personal profile management.
package profile

import (
	"os"
	"path/filepath"

	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"gopkg.in/yaml.v3"
)

// Profile represents a personal profile.
type Profile struct {
	BasicInfo   BasicInfo    `yaml:"basic"`
	Background  string       `yaml:"background"`
	Skills      []Skill      `yaml:"skills"`
	Experiences []Experience `yaml:"experiences"`
	Preferences []Preference `yaml:"preferences"`
	Style       Style        `yaml:"style"`
}

// BasicInfo contains basic personal information.
type BasicInfo struct {
	Name       string `yaml:"name"`
	Location   string `yaml:"location,omitempty"`
	Occupation string `yaml:"occupation,omitempty"`
}

// Skill represents a skill.
type Skill struct {
	Name     string `yaml:"name"`
	Level    string `yaml:"level"` // beginner, intermediate, expert
	Category string `yaml:"category,omitempty"`
}

// Experience represents a work or life experience.
type Experience struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	StartYear   int    `yaml:"start_year,omitempty"`
	EndYear     int    `yaml:"end_year,omitempty"` // 0 means current
}

// Preference represents a preference.
type Preference struct {
	Category string `yaml:"category"`
	Value    string `yaml:"value"`
}

// Style represents communication style.
type Style struct {
	Tone       string  `yaml:"tone"`       // formal, casual, technical
	Vocabulary string  `yaml:"vocabulary"` // simple, technical
	Humor      float64 `yaml:"humor"`      // 0-1
	Verbosity  string  `yaml:"verbosity"`  // concise, detailed
}

// DefaultProfile returns a default profile.
func DefaultProfile() *Profile {
	return &Profile{
		BasicInfo: BasicInfo{
			Name: "User",
		},
		Style: Style{
			Tone:       "casual",
			Vocabulary: "simple",
			Humor:      0.3,
			Verbosity:  "concise",
		},
	}
}

// Load loads a profile from a YAML file.
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// Save saves the profile to a YAML file.
func (p *Profile) Save(path string) error {
	dir := filepath.Dir(path)
	if err := ffp.EnsureDir(dir); err != nil {
		return err
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ToSystemPrompt converts the profile to a system prompt.
func (p *Profile) ToSystemPrompt() string {
	prompt := "You are a personal assistant with the following characteristics:\n\n"

	if p.BasicInfo.Name != "" {
		prompt += "User's name: " + p.BasicInfo.Name + "\n"
	}
	if p.BasicInfo.Occupation != "" {
		prompt += "User's occupation: " + p.BasicInfo.Occupation + "\n"
	}
	if p.Background != "" {
		prompt += "Background: " + p.Background + "\n"
	}

	if len(p.Skills) > 0 {
		prompt += "\nSkills:\n"
		for _, skill := range p.Skills {
			prompt += "- " + skill.Name + " (" + skill.Level + ")\n"
		}
	}

	prompt += "\nCommunication style:\n"
	prompt += "- Tone: " + p.Style.Tone + "\n"
	prompt += "- Vocabulary: " + p.Style.Vocabulary + "\n"
	prompt += "- Verbosity: " + p.Style.Verbosity + "\n"

	return prompt
}
