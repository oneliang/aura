package skilltool

import (
	"os"
	"path/filepath"
)

// createTestSkill creates a test skill directory with SKILL.md
func createTestSkill(dir, name, description string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	content := `---
name: "` + name + `"
description: "` + description + `"
---

# ` + name + ` Skill

This is a test skill for ` + name + `.

## Usage

Run the ` + name + ` command.
`

	return os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644)
}
