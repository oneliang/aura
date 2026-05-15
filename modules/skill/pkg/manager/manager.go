// Package manager provides skill lifecycle management (CRUD operations).
package manager

import (
	"context"
	"fmt"

	sharedmanager "github.com/oneliang/aura/shared/pkg/manager"
	"github.com/oneliang/aura/skill/pkg/loader"
	"github.com/oneliang/aura/skill/pkg/skill"
)

// CreateSkillRequest represents a request to create a new skill.
type CreateSkillRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

// UpdateSkillRequest represents a request to update an existing skill.
type UpdateSkillRequest struct {
	Description *string `json:"description,omitempty"`
	Body        *string `json:"body,omitempty"`
}

// skillLoaderAdapter adapts loader.Loader to TypedLoader[*skill.Skill].
type skillLoaderAdapter struct {
	ldr *loader.Loader
}

func (a *skillLoaderAdapter) Load() ([]*skill.Skill, error) {
	if a == nil || a.ldr == nil {
		return nil, nil
	}
	skills, err := a.ldr.Load()
	if err != nil {
		return nil, err
	}
	if skills == nil {
		return nil, nil
	}
	result := make([]*skill.Skill, len(skills))
	for i := range skills {
		result[i] = &skills[i]
	}
	return result, nil
}

func (a *skillLoaderAdapter) GetItems() []*skill.Skill {
	if a == nil || a.ldr == nil {
		return nil
	}
	skills := a.ldr.GetSkills()
	if skills == nil {
		return nil
	}
	result := make([]*skill.Skill, len(skills))
	for i := range skills {
		result[i] = &skills[i]
	}
	return result
}

// SkillManager manages skill lifecycle (CRUD operations).
// Wraps TypedManager[*skill.Skill] with API-compatible methods.
type SkillManager struct {
	typed   *sharedmanager.TypedManager[*skill.Skill]
	baseDirs []string
}

// NewSkillManager creates a new skill manager.
func NewSkillManager(ldr *loader.Loader, baseDirs []string) *SkillManager {
	cfg := sharedmanager.TypedConfig[*skill.Skill]{
		ItemName:       "skill",
		FileName:       "SKILL.md",
		RequiredFields: []string{"name", "description", "body"},
		BuildContent: func(req map[string]any) string {
			return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s",
				req["name"], req["description"], req["body"])
		},
		ConstructItem: func(fields map[string]any, filePath string) *skill.Skill {
			name, _ := fields["name"].(string)
			desc, _ := fields["description"].(string)
			body, _ := fields["body"].(string)
			permLevel, _ := fields["permission_level"].(skill.SkillPermissionLevel)
			content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", name, desc, body)
			return &skill.Skill{
				Name:            name,
				Description:     desc,
				FilePath:        filePath,
				Content:         content,
				Body:            body,
				PermissionLevel: permLevel,
			}
		},
		MergeUpdate: func(existing *skill.Skill, req map[string]any) map[string]any {
			merged := map[string]any{
				"name":             existing.Name,
				"description":      existing.Description,
				"body":             existing.Body,
				"permission_level": existing.PermissionLevel,
			}
			if v, ok := req["description"]; ok {
				if ps, ok := v.(*string); ok && ps != nil {
					merged["description"] = *ps
				}
			}
			if v, ok := req["body"]; ok {
				if ps, ok := v.(*string); ok && ps != nil {
					merged["body"] = *ps
				}
			}
			return merged
		},
		Loader: &skillLoaderAdapter{ldr: ldr},
	}

	typed := sharedmanager.NewTypedManager[*skill.Skill](cfg.Loader, baseDirs, cfg)
	return &SkillManager{typed: typed, baseDirs: baseDirs}
}

// Create creates a new skill.
func (m *SkillManager) Create(ctx context.Context, req *CreateSkillRequest) (*skill.Skill, error) {
	return m.typed.Create(ctx, map[string]any{
		"name":        req.Name,
		"description": req.Description,
		"body":        req.Body,
	})
}

// Update updates an existing skill.
func (m *SkillManager) Update(ctx context.Context, name string, req *UpdateSkillRequest) (*skill.Skill, error) {
	return m.typed.Update(ctx, name, map[string]any{
		"description": req.Description,
		"body":        req.Body,
	})
}

// Delete removes a skill by deleting its directory.
func (m *SkillManager) Delete(ctx context.Context, name string) error {
	return m.typed.Delete(ctx, name)
}

// Get retrieves a skill by name.
func (m *SkillManager) Get(name string) *skill.Skill {
	return m.typed.Get(name)
}

// List lists all skills (returns values, not pointers, for API compatibility).
func (m *SkillManager) List() []skill.Skill {
	items := m.typed.List()
	if items == nil {
		return nil
	}
	skills := make([]skill.Skill, len(items))
	for i, item := range items {
		skills[i] = *item
	}
	return skills
}

// Reload reloads all skills from disk.
func (m *SkillManager) Reload(ctx context.Context) error {
	return m.typed.Reload(ctx)
}