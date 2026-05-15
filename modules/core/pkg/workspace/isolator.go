// Package workspace provides workspace isolation utilities for multi-agent scenarios.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Workspace represents an isolated workspace for a single agent.
type Workspace struct {
	ID      string // Agent ID
	Dir     string // workspace/{agentID}/ - agent-specific files
	DocsDir string // workspace/docs/ - shared collaboration documents
}

// Isolator manages workspace directories for multiple agents.
type Isolator struct {
	baseDir    string
	docsDir    string
	mu         sync.RWMutex
	workspaces map[string]*Workspace
}

// NewIsolator creates a new workspace isolator.
// baseDir is the root directory for all workspaces (e.g., ~/.aura/workspace).
func NewIsolator(baseDir string) (*Isolator, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("baseDir is required")
	}

	docsDir := filepath.Join(baseDir, "docs")

	// Create base directory
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create docs directory
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create docs directory: %w", err)
	}

	return &Isolator{
		baseDir:    baseDir,
		docsDir:    docsDir,
		workspaces: make(map[string]*Workspace),
	}, nil
}

// Create creates a new workspace for the given agent ID.
// Returns the workspace if it already exists.
func (i *Isolator) Create(agentID string) (*Workspace, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agentID is required")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// Check if workspace already exists
	if ws, exists := i.workspaces[agentID]; exists {
		return ws, nil
	}

	agentDir := filepath.Join(i.baseDir, agentID)

	// Create agent workspace directory
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	ws := &Workspace{
		ID:      agentID,
		Dir:     agentDir,
		DocsDir: i.docsDir,
	}

	i.workspaces[agentID] = ws
	return ws, nil
}

// Get returns the workspace for the given agent ID, or nil if not found.
func (i *Isolator) Get(agentID string) *Workspace {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.workspaces[agentID]
}

// Cleanup removes the workspace directory and unregisters the workspace.
// This is called when an agent is terminated or finishes its task.
func (i *Isolator) Cleanup(agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agentID is required")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	ws, exists := i.workspaces[agentID]
	if !exists {
		return nil // Already cleaned up
	}

	// Remove the agent directory
	if err := os.RemoveAll(ws.Dir); err != nil {
		return fmt.Errorf("failed to remove workspace directory: %w", err)
	}

	delete(i.workspaces, agentID)
	return nil
}

// CleanupAll removes all workspace directories and clears the registry.
func (i *Isolator) CleanupAll() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	var lastErr error
	for agentID, ws := range i.workspaces {
		if err := os.RemoveAll(ws.Dir); err != nil {
			lastErr = err
		}
		delete(i.workspaces, agentID)
	}

	// Also clean up docs directory
	if err := os.RemoveAll(i.docsDir); err != nil {
		lastErr = err
	}

	return lastErr
}

// BaseDir returns the base directory for all workspaces.
func (i *Isolator) BaseDir() string {
	return i.baseDir
}

// SharedDocsDir returns the shared documents directory.
func (i *Isolator) SharedDocsDir() string {
	return i.docsDir
}

// ListWorkspaces returns a list of all active workspace IDs.
func (i *Isolator) ListWorkspaces() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	ids := make([]string, 0, len(i.workspaces))
	for id := range i.workspaces {
		ids = append(ids, id)
	}
	return ids
}
