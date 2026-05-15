// Package importer extracts personality data from documents.
package importer

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/oneliang/aura/personality/pkg/profile"
)

// Importer extracts profile hints from text documents.
type Importer struct{}

// New creates a new personality importer.
func New() *Importer { return &Importer{} }

// ImportFile reads a text/markdown file and returns profile fragments.
// It performs simple keyword extraction — for richer extraction, pipe
// results through an LLM.
func (imp *Importer) ImportFile(path string) (*profile.Profile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return imp.extractProfile(lines), nil
}

// ImportText extracts profile data from raw text.
func (imp *Importer) ImportText(text string) *profile.Profile {
	lines := strings.Split(text, "\n")
	return imp.extractProfile(lines)
}

// extractProfile does heuristic extraction from lines.
func (imp *Importer) extractProfile(lines []string) *profile.Profile {
	p := profile.DefaultProfile()

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Look for name hints (only "Name:" or "name:", not markdown headers)
		if strings.HasPrefix(lower, "name:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" && p.BasicInfo.Name == "User" {
					p.BasicInfo.Name = val
				}
			}
		}

		// Look for skill hints - both "Skill:" and bullet points "- ..."
		if strings.HasPrefix(lower, "skill:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				skillName := strings.TrimSpace(parts[1])
				p.Skills = append(p.Skills, profile.Skill{
					Name:  skillName,
					Level: "intermediate",
				})
			}
		} else if strings.HasPrefix(lower, "- ") && !strings.Contains(lower, ":") {
			// Bullet point without colon - likely a skill under ## Skills section
			skillName := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if skillName != "" && len(skillName) < 100 {
				p.Skills = append(p.Skills, profile.Skill{
					Name:  skillName,
					Level: "intermediate",
				})
			}
		}

		// Look for occupation hints
		if strings.HasPrefix(lower, "occupation:") || strings.HasPrefix(lower, "role:") || strings.HasPrefix(lower, "job:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				p.BasicInfo.Occupation = strings.TrimSpace(parts[1])
			}
		}
	}

	return p
}
