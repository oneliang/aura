// Package manager provides agent lifecycle management (CRUD operations).
package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/oneliang/aura/agent/pkg/agent"
	"github.com/oneliang/aura/agent/pkg/loader"
	sharedmanager "github.com/oneliang/aura/shared/pkg/manager"
)

// CreateAgentRequest represents a request to create a new agent.
type CreateAgentRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Body         string   `json:"body"`
	LLMModel     string   `json:"llm_model,omitempty"`
	Temperature  float64  `json:"temperature,omitempty"`
	DisableTools []string `json:"disable_tools,omitempty"`
}

// UpdateAgentRequest represents a request to update an existing agent.
type UpdateAgentRequest struct {
	Description  *string  `json:"description,omitempty"`
	Body         *string  `json:"body,omitempty"`
	LLMModel     *string  `json:"llm_model,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	DisableTools []string `json:"disable_tools,omitempty"`
}

// agentLoaderAdapter adapts loader.Loader to TypedLoader[*agent.Agent].
type agentLoaderAdapter struct {
	ldr *loader.Loader
}

func (a *agentLoaderAdapter) Load() ([]*agent.Agent, error) {
	if a == nil || a.ldr == nil {
		return nil, nil
	}
	agents, err := a.ldr.Load()
	if err != nil {
		return nil, err
	}
	if agents == nil {
		return nil, nil
	}
	result := make([]*agent.Agent, len(agents))
	for i := range agents {
		result[i] = &agents[i]
	}
	return result, nil
}

func (a *agentLoaderAdapter) GetItems() []*agent.Agent {
	if a == nil || a.ldr == nil {
		return nil
	}
	agents := a.ldr.GetAgents()
	if agents == nil {
		return nil
	}
	result := make([]*agent.Agent, len(agents))
	for i := range agents {
		result[i] = &agents[i]
	}
	return result
}

// AgentManager manages agent lifecycle (CRUD operations).
// Wraps TypedManager[*agent.Agent] with API-compatible methods.
type AgentManager struct {
	typed    *sharedmanager.TypedManager[*agent.Agent]
	baseDirs []string
}

// NewAgentManager creates a new agent manager.
func NewAgentManager(ldr *loader.Loader, baseDirs []string) *AgentManager {
	cfg := sharedmanager.TypedConfig[*agent.Agent]{
		ItemName:       "agent",
		FileName:       "AGENT.md",
		RequiredFields: []string{"name", "description", "body"},
		BuildContent: func(req map[string]any) string {
			var sb strings.Builder
			sb.WriteString("---\n")
			sb.WriteString(fmt.Sprintf("name: %s\n", req["name"]))
			sb.WriteString(fmt.Sprintf("description: %s\n", req["description"]))
			if llmModel, ok := req["llm_model"].(string); ok && llmModel != "" {
				sb.WriteString(fmt.Sprintf("llm_model: %s\n", llmModel))
			}
			if temp, ok := req["temperature"].(float64); ok && temp > 0 {
				sb.WriteString(fmt.Sprintf("temperature: %.2f\n", temp))
			}
			if tools, ok := req["disable_tools"].([]string); ok && len(tools) > 0 {
				sb.WriteString("disable_tools:\n")
				for _, tool := range tools {
					sb.WriteString(fmt.Sprintf("  - %s\n", tool))
				}
			}
			sb.WriteString("---\n\n")
			sb.WriteString(req["body"].(string))
			return sb.String()
		},
		ConstructItem: func(fields map[string]any, filePath string) *agent.Agent {
			name, _ := fields["name"].(string)
			desc, _ := fields["description"].(string)
			body, _ := fields["body"].(string)
			llmModel, _ := fields["llm_model"].(string)
			disableTools, _ := fields["disable_tools"].([]string)
			content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", name, desc, body)
			return &agent.Agent{
				Name:        name,
				Description: desc,
				FilePath:    filePath,
				Content:     content,
				Body:        body,
				Meta: agent.AgentMeta{
					Name:         name,
					Description:  desc,
					LLMModel:     llmModel,
					DisableTools: disableTools,
				},
			}
		},
		MergeUpdate: func(existing *agent.Agent, req map[string]any) map[string]any {
			merged := map[string]any{
				"name":          existing.Name,
				"description":   existing.Description,
				"body":          existing.Body,
				"llm_model":     existing.Meta.LLMModel,
				"temperature":   existing.Meta.Temperature,
				"disable_tools": existing.Meta.DisableTools,
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
			if v, ok := req["llm_model"]; ok {
				if ps, ok := v.(*string); ok && ps != nil {
					merged["llm_model"] = *ps
				}
			}
			if v, ok := req["temperature"]; ok {
				if pf, ok := v.(*float64); ok && pf != nil {
					merged["temperature"] = *pf
				}
			}
			if v, ok := req["disable_tools"]; ok {
				if ss, ok := v.([]string); ok {
					merged["disable_tools"] = ss
				}
			}
			return merged
		},
		Loader: &agentLoaderAdapter{ldr: ldr},
	}

	typed := sharedmanager.NewTypedManager[*agent.Agent](cfg.Loader, baseDirs, cfg)
	return &AgentManager{typed: typed, baseDirs: baseDirs}
}

// Create creates a new agent.
func (m *AgentManager) Create(ctx context.Context, req *CreateAgentRequest) (*agent.Agent, error) {
	return m.typed.Create(ctx, map[string]any{
		"name":          req.Name,
		"description":   req.Description,
		"body":          req.Body,
		"llm_model":     req.LLMModel,
		"temperature":   req.Temperature,
		"disable_tools": req.DisableTools,
	})
}

// Update updates an existing agent.
func (m *AgentManager) Update(ctx context.Context, name string, req *UpdateAgentRequest) (*agent.Agent, error) {
	return m.typed.Update(ctx, name, map[string]any{
		"description":   req.Description,
		"body":          req.Body,
		"llm_model":     req.LLMModel,
		"temperature":   req.Temperature,
		"disable_tools": req.DisableTools,
	})
}

// Delete removes an agent by deleting its directory.
func (m *AgentManager) Delete(ctx context.Context, name string) error {
	return m.typed.Delete(ctx, name)
}

// Get retrieves an agent by name.
func (m *AgentManager) Get(name string) *agent.Agent {
	return m.typed.Get(name)
}

// List lists all agents (returns values, not pointers, for API compatibility).
func (m *AgentManager) List() []agent.Agent {
	items := m.typed.List()
	if items == nil {
		return nil
	}
	agents := make([]agent.Agent, len(items))
	for i, item := range items {
		agents[i] = *item
	}
	return agents
}

// Reload reloads all agents from disk.
func (m *AgentManager) Reload(ctx context.Context) error {
	return m.typed.Reload(ctx)
}