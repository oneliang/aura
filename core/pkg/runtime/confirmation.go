package runtime

import (
	"context"
	"fmt"
	"time"

	enginepkg "github.com/oneliang/aura/core/pkg/engine"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/skill/pkg/skill"
)

// createConfirmationHandler creates a confirmation handler.
// Uses event stream for confirmation requests.
// Returns an error if AutoApprove is disabled and no event stream is available.
func (r *AgentRuntime) createConfirmationHandler() enginepkg.ToolConfirmationHandler {
	if r.permMgr == nil {
		return nil
	}
	return func(ctx context.Context, toolName string, params map[string]any) (bool, error) {
		allowed, requiresConfirm, reason := r.permMgr.CheckPermission(ctx, toolName, params)
		r.logger.Debug("ConfirmationHandler: permission check result", "module", "runtime", "tool", toolName, "allowed", allowed, "requiresConfirm", requiresConfirm)
		if !allowed {
			return false, fmt.Errorf("permission denied: %s", reason)
		}
		if !requiresConfirm {
			return true, nil
		}

		// Requires confirmation - use event stream
		req := events.InteractionRequest{
			Type:       events.InteractionTypeToolConfirmation,
			ToolName:   toolName,
			ToolParams: params,
			Timeout:    60 * time.Second,
		}

		resp := r.requestInteraction(ctx, req)
		if resp.Error != nil {
			return false, resp.Error
		}
		return resp.Approved, nil
	}
}

// checkSkillPermission checks if a skill with required confirmation should be injected.
// Uses event stream for confirmation requests.
// Returns true if confirmed by user, false otherwise.
func (r *AgentRuntime) checkSkillPermission(ctx context.Context, sk *skill.Skill) bool {
	r.logger.Debug("checkSkillPermission: called", "module", "runtime", "skill", sk.Name)

	// Check permission manager first (respects config default_level and per-tool settings)
	if r.permMgr != nil {
		toolName := fmt.Sprintf("skill:%s", sk.Name)
		allowed, requiresConfirm, reason := r.permMgr.CheckPermission(ctx, toolName, map[string]any{
			"skill":            sk.Name,
			"description":      sk.Description,
			"permission_level": sk.PermissionLevel,
		})
		if !allowed {
			r.logger.Debug("checkSkillPermission: denied by PermissionMgr", "module", "runtime", "skill", sk.Name, "reason", reason)
			return false
		}
		if !requiresConfirm {
			r.logger.Debug("checkSkillPermission: allowed by PermissionMgr, no confirmation needed", "module", "runtime", "skill", sk.Name)
			return true
		}
		// PermissionMgr says ask, fall through to user confirmation below
	}

	// Use event stream for confirmation
	// If outer context is already cancelled, deny immediately
	if ctx.Err() != nil {
		r.logger.Debug("checkSkillPermission: outer context cancelled, denying", "module", "runtime", "skill", sk.Name)
		return false
	}

	req := events.InteractionRequest{
		Type:       events.InteractionTypeToolConfirmation,
		ToolName:   fmt.Sprintf("skill:%s", sk.Name),
		ToolParams: map[string]any{"skill": sk.Name, "description": sk.Description, "permission_level": sk.PermissionLevel},
		Timeout:    60 * time.Second,
	}

	r.logger.Debug("checkSkillPermission: sending confirmation request via event stream", "module", "runtime", "skill", sk.Name)
	resp := r.requestInteraction(ctx, req)

	r.logger.Debug("checkSkillPermission: received response", "module", "runtime", "skill", sk.Name, "approved", resp.Approved)
	return resp.Approved
}
