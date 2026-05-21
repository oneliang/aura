package handlers

import (
	"context"
	"net/http"

	commands "github.com/oneliang/aura/commands/pkg"
)

// SkillsService provides skills listing capabilities.
type SkillsService interface {
	ListSkills() (string, error)
}

// skillsServiceWrapper wraps CommandProvider to implement SkillsService.
type skillsServiceWrapper struct {
	commandProvider commands.Command
}

// NewSkillsServiceWrapper creates a new skills service wrapper.
func NewSkillsServiceWrapper(cmdProvider commands.Command) *skillsServiceWrapper {
	return &skillsServiceWrapper{
		commandProvider: cmdProvider,
	}
}

// ListSkills lists all loaded skills.
func (w *skillsServiceWrapper) ListSkills() (string, error) {
	if w.commandProvider == nil {
		return "Skills not available", nil
	}
	return w.commandProvider.Execute(context.Background(), commands.CmdNameSkillList, nil)
}

// SkillsHandler handles skills-related HTTP requests.
type SkillsHandler struct {
	service SkillsService
}

// NewSkillsHandler creates a new skills handler.
func NewSkillsHandler(service SkillsService) *SkillsHandler {
	return &SkillsHandler{service: service}
}

// HandleListSkills handles GET /api/skills.
func (h *SkillsHandler) HandleListSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := h.service.ListSkills()
	if err != nil {
		WriteError(w, "Failed to list skills: "+err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, Response{
		Status: "ok",
		Data: map[string]interface{}{
			"skills": result,
		},
	})
}
