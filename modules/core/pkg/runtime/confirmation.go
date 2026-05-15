package runtime

import (
	"context"
	"fmt"
	"time"

	enginepkg "github.com/oneliang/aura/core/pkg/engine"
	"github.com/oneliang/aura/skill/pkg/skill"
)

// createConfirmationHandler creates a confirmation handler.
// Relies on the injected onConfirm callback for all modes.
// Returns an error if no handler is configured when confirmation is required.
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

		// Requires confirmation - must have handler configured
		if r.onConfirm == nil {
			return false, fmt.Errorf("confirmation required for tool %q but no confirmation handler is configured", toolName)
		}

		respCh := make(chan bool, 1)
		r.handlerMu.RLock()
		handler := r.onConfirm
		r.handlerMu.RUnlock()
		handler(ConfirmationRequest{
			ToolName:   toolName,
			Params:     params,
			ResponseCh: respCh,
		})
		// Use independent timeout for tool confirmation too
		// Increased to 60s to accommodate multiple sequential confirmations
		confirmCtx, confirmCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer confirmCancel()
		select {
		case confirmed := <-respCh:
			return confirmed, nil
		case <-confirmCtx.Done():
			return false, confirmCtx.Err()
		}
	}
}

// checkSkillPermission checks if a skill with required confirmation should be injected.
// Returns true if confirmed by user, false otherwise.
func (r *AgentRuntime) checkSkillPermission(ctx context.Context, sk *skill.Skill) bool {
	r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Int("mode", int(r.mode)).Bool("onConfirm_nil", r.onConfirm == nil).Msg("checkSkillPermission: called")

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

	// Use confirmation handler if configured
	if r.onConfirm != nil {
		// If outer context is already cancelled, deny immediately
		if ctx.Err() != nil {
			r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Msg("checkSkillPermission: outer context cancelled, denying")
			return false
		}

		respCh := make(chan bool, 1)
		r.handlerMu.RLock()
		handler := r.onConfirm
		r.handlerMu.RUnlock()
		r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Msg("checkSkillPermission: sending confirmation request via onConfirm")
		handler(ConfirmationRequest{
			ToolName:   fmt.Sprintf("skill:%s", sk.Name),
			Params:     map[string]any{"skill": sk.Name, "description": sk.Description, "permission_level": sk.PermissionLevel},
			ResponseCh: respCh,
		})

		// Use independent timeout so one skill's cancellation doesn't cascade to others
		// Increased to 60s to accommodate multiple sequential confirmations
		confirmCtx, confirmCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer confirmCancel()

		r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Msg("checkSkillPermission: waiting for response")
		select {
		case result := <-respCh:
			r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Bool("result", result).Msg("checkSkillPermission: received response")
			return result
		case <-confirmCtx.Done():
			r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Msg("checkSkillPermission: confirmation timeout or context cancelled")
			return false
		}
	}

	// No handler configured - log and deny
	r.logger.Debug().Str("module", "runtime").Str("skill", sk.Name).Int("mode", int(r.mode)).Msg("checkSkillPermission: no confirmation handler configured, denying")
	return false
}
