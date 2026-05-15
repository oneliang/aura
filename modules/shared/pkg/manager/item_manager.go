// Package manager provides shared item management with strategy/callback pattern.
package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Item is the interface all manageable items must implement.
type Item interface {
	GetName() string
	GetDescription() string
	GetFilePath() string
	GetContent() string
	GetBody() string
}

// ValidateFunc validates a create request map.
type ValidateFunc func(req map[string]any) error

// BuildContentFunc builds file content from a request map.
type BuildContentFunc func(req map[string]any) string

// ConstructItemFunc creates a typed item from fields and file path.
type ConstructItemFunc func(fields map[string]any, filePath string) Item

// MergeUpdateFunc merges update request into an existing item, returning merged fields.
type MergeUpdateFunc func(existing Item, req map[string]any) map[string]any

// ReloadFunc reloads the loader and returns error if any.
type ReloadFunc func() error

// ManagerConfig holds type-specific callbacks for the shared Manager.
type ManagerConfig struct {
	ItemName string // "skill" / "agent" (for error messages)
	FileName string // "SKILL.md" / "AGENT.md"

	Validate      ValidateFunc
	BuildContent  BuildContentFunc
	ConstructItem ConstructItemFunc
	MergeUpdate   MergeUpdateFunc
	Reload        ReloadFunc
	FindByName    func(name string) Item
}

// ItemLister is the interface for listing loaded items.
type ItemLister interface {
	GetItems() []Item
}

// Manager provides shared CRUD operations for file-based items.
type Manager struct {
	lister   ItemLister
	baseDirs []string
	config   ManagerConfig
	mutex    sync.RWMutex
}

// NewManager creates a new shared item manager.
func NewManager(lister ItemLister, baseDirs []string, cfg ManagerConfig) *Manager {
	return &Manager{
		lister:   lister,
		baseDirs: baseDirs,
		config:   cfg,
	}
}

// Create creates a new item (directory + file).
func (m *Manager) Create(ctx context.Context, req map[string]any) (Item, error) {
	if err := m.config.Validate(req); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := req["name"].(string)
	if m.config.FindByName(name) != nil {
		return nil, fmt.Errorf("%s '%s' already exists", m.config.ItemName, name)
	}

	if len(m.baseDirs) == 0 {
		return nil, fmt.Errorf("no %s directories configured", m.config.ItemName)
	}
	baseDir := m.baseDirs[0]

	dirPath := filepath.Join(baseDir, name)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create %s directory: %w", m.config.ItemName, err)
	}

	filePath := filepath.Join(dirPath, m.config.FileName)
	content := m.config.BuildContent(req)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write %s file: %w", m.config.ItemName, err)
	}

	item := m.config.ConstructItem(req, filePath)

	if m.lister != nil {
		if err := m.config.Reload(); err != nil {
			fmt.Printf("Warning: failed to reload %ss: %v\n", m.config.ItemName, err)
		}
	}

	return item, nil
}

// Update updates an existing item.
func (m *Manager) Update(ctx context.Context, name string, req map[string]any) (Item, error) {
	if name == "" {
		return nil, fmt.Errorf("%s name is required", m.config.ItemName)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing := m.config.FindByName(name)
	if existing == nil {
		return nil, fmt.Errorf("%s '%s' not found", m.config.ItemName, name)
	}

	merged := m.config.MergeUpdate(existing, req)
	merged["name"] = name

	content := m.config.BuildContent(merged)

	if err := os.WriteFile(existing.GetFilePath(), []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to update %s file: %w", m.config.ItemName, err)
	}

	updated := m.config.ConstructItem(merged, existing.GetFilePath())

	if m.lister != nil {
		if err := m.config.Reload(); err != nil {
			fmt.Printf("Warning: failed to reload %ss: %v\n", m.config.ItemName, err)
		}
	}

	return updated, nil
}

// Delete removes an item by deleting its directory.
func (m *Manager) Delete(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("%s name is required", m.config.ItemName)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing := m.config.FindByName(name)
	if existing == nil {
		return fmt.Errorf("%s '%s' not found", m.config.ItemName, name)
	}

	dir := filepath.Dir(existing.GetFilePath())
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to delete %s directory: %w", m.config.ItemName, err)
	}

	if m.lister != nil {
		if err := m.config.Reload(); err != nil {
			fmt.Printf("Warning: failed to reload %ss: %v\n", m.config.ItemName, err)
		}
	}

	return nil
}

// Get retrieves an item by name.
func (m *Manager) Get(name string) Item {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.FindByName(name)
}

// List returns all items.
func (m *Manager) List() []Item {
	if m.lister == nil {
		return nil
	}
	return m.lister.GetItems()
}

// Reload reloads all items from disk.
func (m *Manager) Reload(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.config.Reload()
}
