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
		r.logger.Debug().Str("module", "runtime").Str("tool", toolName).Bool("allowed", allowed).Bool("requiresConfirm", requiresConfirm).Msg("ConfirmationHandler: permission check result")
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
	r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Msg("checkSkillPermission: called")

	// Check permission manager first (respects config default_level and per-tool settings)
	if r.permMgr != nil {
		toolName := fmt.Sprintf("skill:%s", sk.Name)
		allowed, requiresConfirm, reason := r.permMgr.CheckPermission(ctx, toolName, map[string]any{
			"skill":            sk.Name,
			"description":      sk.Description,
			"permission_level": sk.PermissionLevel,
		})
		if !allowed {
			r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Str("reason", reason).Msg("checkSkillPermission: denied by PermissionMgr")
			return false
		}
		if !requiresConfirm {
			r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Msg("checkSkillPermission: allowed by PermissionMgr, no confirmation needed")
			return true
		}
		// PermissionMgr says ask, fall through to user confirmation below
	}

	// Use event stream for confirmation
	// If outer context is already cancelled, deny immediately
	if ctx.Err() != nil {
		r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Msg("checkSkillPermission: outer context cancelled, denying")
		return false
	}

	req := events.InteractionRequest{
		Type:       events.InteractionTypeToolConfirmation,
		ToolName:   fmt.Sprintf("skill:%s", sk.Name),
		ToolParams: map[string]any{"skill": sk.Name, "description": sk.Description, "permission_level": sk.PermissionLevel},
		Timeout:    60 * time.Second,
	}

	r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Msg("checkSkillPermission: sending confirmation request via event stream")
	resp := r.requestInteraction(ctx, req)

	r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Bool("approved", resp.Approved).Msg("checkSkillPermission: received response")
	return resp.Approved
}
