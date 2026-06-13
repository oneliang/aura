// Package profile provides personal profile management.
package profile

import (
	"os"
	"path/filepath"
	"strings"

	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// Profile represents a personal profile stored as markdown.
type Profile struct {
	Content string // Raw markdown content
}

// defaultTemplate is the default profile markdown template.
const defaultTemplate = `# 关于我

- 名字：
- 职业：
- 位置：

# 背景

简要描述你的背景和经验。

# 技能

- 技能1
- 技能2

# 偏好

- 回复语言：中文
- 代码风格：简洁
`

// DefaultProfile returns a default profile with a markdown template.
func DefaultProfile() *Profile {
	return &Profile{Content: defaultTemplate}
}

// Load loads a profile from a markdown file.
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &Profile{Content: string(data)}, nil
}

// Save saves the profile to a markdown file.
func (p *Profile) Save(path string) error {
	dir := filepath.Dir(path)
	if err := ffp.EnsureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(p.Content), 0644)
}

// ToSystemPrompt converts the profile to a system prompt fragment.
func (p *Profile) ToSystemPrompt() string {
	return "User Profile:\n\n" + p.Content
}

// DisplayName extracts a display name from the profile markdown.
// It looks for the first non-empty value after "名字" or "Name" in the content.
// Falls back to "Aura" if no name is found.
func (p *Profile) DisplayName() string {
	for _, line := range strings.Split(p.Content, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.Contains(lower, "名字") || strings.Contains(lower, "name") {
			// Try to extract value after colon (supports both ： and :)
			for _, sep := range []string{"：", ":"} {
				if idx := strings.Index(line, sep); idx >= 0 {
					val := strings.TrimSpace(line[idx+len(sep):])
					if val != "" {
						return val
					}
				}
			}
		}
	}
	return "Aura"
}
