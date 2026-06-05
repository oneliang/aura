// Package commands provides command orchestration logic.
// This package is UI-agnostic and can be used across TUI, CLI, and adapter scenarios.
package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	agentmanager "github.com/oneliang/aura/agent/pkg/manager"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/logger"
)

// AgentHandler handles agent delegation and management commands.
type AgentHandler struct {
	// delegateFn is a callback function that performs the actual delegation.
	// It takes the agent name and task description, and returns the result.
	// This is injected by the runtime to avoid circular dependencies.
	delegateFn func(ctx context.Context, agentName string, task string) (string, error)
	// manager is the agent manager for CRUD operations.
	manager *agentmanager.AgentManager
}

// NewAgentHandler creates a new agent handler.
func NewAgentHandler(delegateFn func(ctx context.Context, agentName string, task string) (string, error)) *AgentHandler {
	return &AgentHandler{
		delegateFn: delegateFn,
	}
}

// NewAgentHandlerWithManager creates a new agent handler with manager support.
func NewAgentHandlerWithManager(delegateFn func(ctx context.Context, agentName string, task string) (string, error), mgr *agentmanager.AgentManager) *AgentHandler {
	return &AgentHandler{
		delegateFn: delegateFn,
		manager:    mgr,
	}
}

// ExecuteCommand executes an agent-related command.
func (h *AgentHandler) ExecuteCommand(ctx context.Context, cmd string, params map[string]any) (string, error) {
	log := logger.RegistryDefault().WithModule("agent_handler")
	log.Debug("ExecuteCommand: dispatching agent command", "cmd", cmd, "params", params, "delegateFn_nil", h.delegateFn == nil)
	switch cmd {
	case "delegate_to_agent", "delegate":
		return h.delegateToAgent(ctx, params)
	case "create":
		return h.createAgent(ctx, params)
	case "update":
		return h.updateAgent(ctx, params)
	case "delete":
		return h.deleteAgent(ctx, params)
	case "reload":
		return h.reloadAgents(ctx)
	case "list":
		return h.listAgents()
	case "get":
		return h.getAgent(params)
	default:
		return "", fmt.Errorf("%s: %s", i18n.T("error.agent.unknown"), cmd)
	}
}

// delegateToAgent delegates a task to a specified agent.
// Parameters:
//   - agent: The name of the agent to delegate to
//   - task: The task description to delegate
func (h *AgentHandler) delegateToAgent(ctx context.Context, params map[string]any) (string, error) {
	if h.delegateFn == nil {
		return "", errors.New(i18n.T("error.agent.delegation_not_configured"))
	}

	agentName, ok := params["agent"].(string)
	if !ok || agentName == "" {
		return "", errors.New(i18n.T("error.agent.param_agent_required"))
	}

	task, ok := params["task"].(string)
	if !ok || task == "" {
		return "", errors.New(i18n.T("error.agent.param_task_required"))
	}

	log := logger.RegistryDefault().WithModule("agent_handler")
	log.Debug("delegateToAgent: delegating task to agent", "agent", agentName, "task_len", len(task))
	result, err := h.delegateFn(ctx, agentName, task)
	log.Debug("delegateToAgent: delegation result", "agent", agentName, "result_len", len(result), "error", err)
	return result, err
}

// SetDelegateFn sets the delegate function callback.
// This allows the runtime to inject the actual delegation logic.
func (h *AgentHandler) SetDelegateFn(fn func(ctx context.Context, agentName string, task string) (string, error)) {
	h.delegateFn = fn
}

// SetManager sets the agent manager.
func (h *AgentHandler) SetManager(mgr *agentmanager.AgentManager) {
	h.manager = mgr
}

// createAgent creates a new agent.
func (h *AgentHandler) createAgent(ctx context.Context, params map[string]any) (string, error) {
	if h.manager == nil {
		return "", errors.New(i18n.T("error.agent.manager_not_configured"))
	}

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

	req := &agentmanager.CreateAgentRequest{
		Name:        name,
		Description: description,
		Body:        body,
	}

	// Optional parameters
	if llmModel, ok := params["llm_model"].(string); ok {
		req.LLMModel = llmModel
	}
	if temperature, ok := params["temperature"].(float64); ok {
		req.Temperature = temperature
	}
	if disableTools, ok := params["disable_tools"].([]interface{}); ok {
		for _, t := range disableTools {
			if tool, ok := t.(string); ok {
				req.DisableTools = append(req.DisableTools, tool)
			}
		}
	}

	created, err := h.manager.Create(ctx, req)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(i18n.T("command.agent_created"), created.Name, created.FilePath), nil
}

// updateAgent updates an existing agent.
func (h *AgentHandler) updateAgent(ctx context.Context, params map[string]any) (string, error) {
	if h.manager == nil {
		return "", errors.New(i18n.T("error.agent.manager_not_configured"))
	}

	name, err := requireStringParam(params, "name")
	if err != nil {
		return "", err
	}

	req := &agentmanager.UpdateAgentRequest{}

	if desc, ok := params["description"].(string); ok {
		req.Description = &desc
	}
	if body, ok := params["body"].(string); ok {
		req.Body = &body
	}
	if llmModel, ok := params["llm_model"].(string); ok {
		req.LLMModel = &llmModel
	}
	if temperature, ok := params["temperature"].(float64); ok {
		req.Temperature = &temperature
	}

	updated, err := h.manager.Update(ctx, name, req)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(i18n.T("command.agent_updated"), updated.Name), nil
}

// deleteAgent deletes an agent.
func (h *AgentHandler) deleteAgent(ctx context.Context, params map[string]any) (string, error) {
	if h.manager == nil {
		return "", errors.New(i18n.T("error.agent.manager_not_configured"))
	}

	name, err := requireStringParam(params, "name")
	if err != nil {
		return "", err
	}

	if err := h.manager.Delete(ctx, name); err != nil {
		return "", err
	}

	return fmt.Sprintf(i18n.T("command.agent_deleted"), name), nil
}

// reloadAgents reloads all agents from disk.
func (h *AgentHandler) reloadAgents(ctx context.Context) (string, error) {
	if h.manager == nil {
		return "", errors.New(i18n.T("error.agent.manager_not_configured"))
	}

	if err := h.manager.Reload(ctx); err != nil {
		return "", err
	}

	agents := h.manager.List()
	return fmt.Sprintf(i18n.T("command.agents_reloaded"), len(agents)), nil
}

// listAgents lists all agents.
func (h *AgentHandler) listAgents() (string, error) {
	if h.manager == nil {
		return "", errors.New(i18n.T("error.agent.manager_not_configured"))
	}

	agents := h.manager.List()
	if len(agents) == 0 {
		return i18n.T("command.no_agents_available"), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(i18n.T("command.available_agents"), len(agents)))
	for _, a := range agents {
		sb.WriteString(fmt.Sprintf("  - %s: %s\n", a.Name, a.Description))
	}
	return sb.String(), nil
}

// getAgent gets a specific agent.
func (h *AgentHandler) getAgent(params map[string]any) (string, error) {
	if h.manager == nil {
		return "", errors.New(i18n.T("error.agent.manager_not_configured"))
	}

	name, err := requireStringParam(params, "name")
	if err != nil {
		return "", err
	}

	a := h.manager.Get(name)
	if a == nil {
		return "", fmt.Errorf(i18n.T("error.agent.not_found"), name)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(i18n.T("command.agent_info"), a.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", a.Description))
	sb.WriteString(fmt.Sprintf(i18n.T("command.skill_file"), a.FilePath))
	if a.Meta.LLMModel != "" {
		sb.WriteString(fmt.Sprintf(i18n.T("command.agent_llm_model"), a.Meta.LLMModel))
	}
	sb.WriteString(i18n.T("command.agent_body_separator"))
	sb.WriteString(a.Body)
	return sb.String(), nil
}
