// Package loader provides agent loading functionality from directories.
package loader

import (
	"github.com/oneliang/aura/agent/pkg/agent"
	sharedloader "github.com/oneliang/aura/shared/pkg/loader"
	sharedmanager "github.com/oneliang/aura/shared/pkg/manager"
)

// Loader loads agents from configured directories.
type Loader struct {
	baseDirs []string
	agents   []agent.Agent
}

// NewLoader creates a new agent loader.
func NewLoader(baseDirs []string) *Loader {
	return &Loader{
		baseDirs: baseDirs,
	}
}

// Load loads all agents from configured directories.
// Each directory can contain multiple agent subdirectories, each with an AGENT.md file.
func (l *Loader) Load() ([]agent.Agent, error) {
	results, err := sharedloader.LoadFromDirectories(
		l.baseDirs,
		sharedloader.FileSpec{FileName: "AGENT.md"},
		parseAgent,
	)
	if err != nil {
		return nil, err
	}

	l.agents = make([]agent.Agent, len(results))
	for i, r := range results {
		l.agents[i] = r.Item
	}
	return l.agents, nil
}

// parseAgent implements sharedloader.ParseFunc[agent.Agent].
func parseAgent(content, filePath string) (agent.Agent, string, error) {
	var meta agent.AgentMeta

	body, err := sharedloader.UnmarshalYAMLFrontmatter(content, &meta)
	if err != nil {
		return agent.Agent{}, "", err
	}

	if err := meta.Validate(); err != nil {
		return agent.Agent{}, "", err
	}

	return agent.Agent{
		Name:        meta.Name,
		Description: meta.Description,
		FilePath:    filePath,
		Content:     content,
		Body:        body,
		Meta:        meta,
	}, body, nil
}

// GetAgents returns the loaded agents.
func (l *Loader) GetAgents() []agent.Agent {
	return l.agents
}

// GetItems returns agents as shared manager Item slice.
func (l *Loader) GetItems() []sharedmanager.Item {
	if l == nil {
		return nil
	}
	agents := l.GetAgents()
	if agents == nil {
		return nil
	}
	items := make([]sharedmanager.Item, len(agents))
	for i := range agents {
		items[i] = &agents[i]
	}
	return items
}
