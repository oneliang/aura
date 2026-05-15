// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oneliang/aura/skill/pkg/manager"
	"github.com/oneliang/aura/shared/pkg/i18n"
)

// SkillHandler handles skill management commands.
type SkillHandler struct {
	manager *manager.SkillManager
}

// NewSkillHandler creates a new skill handler.
func NewSkillHandler(mgr *manager.SkillManager) *SkillHandler {
	return &SkillHandler{
		manager: mgr,
	}
}

// ExecuteCommand executes a skill management command.
// Commands: create, update, delete, reload, list, get
func (h *SkillHandler) ExecuteCommand(ctx context.Context, cmd string, params map[string]any) (string, error) {
	if h.manager == nil {
		return "", errors.New(i18n.T("error.skill.manager_not_configured"))
	}

	switch cmd {
	case "create":
		return h.createSkill(ctx, params)
	case "update":
		return h.updateSkill(ctx, params)
	case "delete":
		return h.deleteSkill(ctx, params)
	case "reload":
		return h.reloadSkills(ctx)
	case "list":
		return h.listSkills()
	case "get":
		return h.getSkill(params)
	default:
		return "", fmt.Errorf("%s: %s", i18n.T("error.skill.unknown"), cmd)
	}
}

// createSkill creates a new skill.
func (h *SkillHandler) createSkill(ctx context.Context, params map[string]any) (string, error) {
	name, err := requireStringParam(params, "name")
	if err != nil {
		return "", err
	}

	description, err := requireStringParam(params, "description")
	if err != nil {
		return "", err
	}

	body, err := requireStringParam(params, "body")
	if err != nil {
		return "", err
	}

	req := &manager.CreateSkillRequest{
		Name:        name,
		Description: description,
		Body:        body,
	}

	created, err := h.manager.Create(ctx, req)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(i18n.T("command.skill_created"), created.Name, created.FilePath), nil
}

// updateSkill updates an existing skill.
func (h *SkillHandler) updateSkill(ctx context.Context, params map[string]any) (string, error) {
	name, err := requireStringParam(params, "name")
	if err != nil {
		return "", err
	}

	req := &manager.UpdateSkillRequest{}

	if desc, ok := params["description"].(string); ok {
		req.Description = &desc
	}
	if body, ok := params["body"].(string); ok {
		req.Body = &body
	}

	updated, err := h.manager.Update(ctx, name, req)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(i18n.T("command.skill_updated"), updated.Name), nil
}

// deleteSkill deletes a skill.
func (h *SkillHandler) deleteSkill(ctx context.Context, params map[string]any) (string, error) {
	name, err := requireStringParam(params, "name")
	if err != nil {
		return "", err
	}

	if err := h.manager.Delete(ctx, name); err != nil {
		return "", err
	}

	return fmt.Sprintf(i18n.T("command.skill_deleted"), name), nil
}

// reloadSkills reloads all skills from disk.
func (h *SkillHandler) reloadSkills(ctx context.Context) (string, error) {
	if err := h.manager.Reload(ctx); err != nil {
		return "", err
	}

	skills := h.manager.List()
	return fmt.Sprintf(i18n.T("command.skills_reloaded"), len(skills)), nil
}

// listSkills lists all skills.
func (h *SkillHandler) listSkills() (string, error) {
	skills := h.manager.List()
	if len(skills) == 0 {
		return i18n.T("command.no_skills_available"), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(i18n.T("command.available_skills"), len(skills)))
	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("  - %s: %s\n", s.Name, s.Description))
	}
	return sb.String(), nil
}

// getSkill gets a specific skill.
func (h *SkillHandler) getSkill(params map[string]any) (string, error) {
	name, err := requireStringParam(params, "name")
	if err != nil {
		return "", err
	}

	s := h.manager.Get(name)
	if s == nil {
		return "", fmt.Errorf(i18n.T("error.skill.not_found"), name)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(i18n.T("command.skill_info"), s.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", s.Description))
	sb.WriteString(fmt.Sprintf(i18n.T("command.skill_file"), s.FilePath))
	sb.WriteString(i18n.T("command.skill_body_separator"))
	sb.WriteString(s.Body)
	return sb.String(), nil
}

// GetManager returns the underlying manager for integration.
func (h *SkillHandler) GetManager() *manager.SkillManager {
	return h.manager
}

// Helper function to require a string parameter.
func requireStringParam(params map[string]any, key string) (string, error) {
	val, ok := params[key]
	if !ok {
		return "", fmt.Errorf(i18n.T("error.param.required"), key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf(i18n.T("error.param.type_string"), key)
	}
	if str == "" {
		return "", fmt.Errorf(i18n.T("error.param.empty"), key)
	}
	return str, nil
}
