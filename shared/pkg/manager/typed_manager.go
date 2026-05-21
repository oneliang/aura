// Package manager provides generic typed item management.
package manager

import (
	"context"
	"fmt"
)

// TypedLoader is a generic interface for loading items.
// T must be a pointer type that implements Item interface (e.g., *skill.Skill, *agent.Agent).
type TypedLoader[T Item] interface {
	Load() ([]T, error)
	GetItems() []T
}

// TypedConfig holds type-specific callbacks for TypedManager.
type TypedConfig[T Item] struct {
	ItemName       string                                     // "skill" / "agent" (for error messages)
	FileName       string                                     // "SKILL.md" / "AGENT.md"
	RequiredFields []string                                   // ["name", "description", "body"]
	BuildContent   func(req map[string]any) string            // Build file content from request
	ConstructItem  func(fields map[string]any, filePath string) T // Create typed item (returns pointer)
	MergeUpdate    func(existing T, req map[string]any) map[string]any // Merge update request
	Loader         TypedLoader[T]                             // Loader for reload/find
}

// TypedManager provides generic CRUD operations for typed items.
// T must be a pointer type that implements Item interface.
type TypedManager[T Item] struct {
	mgr    *Manager
	config TypedConfig[T]
}

// NewTypedManager creates a new typed manager.
// T must be a pointer type (e.g., *skill.Skill).
func NewTypedManager[T Item](loader TypedLoader[T], baseDirs []string, cfg TypedConfig[T]) *TypedManager[T] {
	// Convert TypedLoader to ItemLister interface
	var lister ItemLister
	if loader != nil {
		lister = &typedLoaderAdapter[T]{loader: loader}
	}

	// Build ManagerConfig with adapters
	mgrCfg := ManagerConfig{
		ItemName: cfg.ItemName,
		FileName: cfg.FileName,
		Validate: buildDefaultValidator(cfg.ItemName, cfg.RequiredFields),
		BuildContent: cfg.BuildContent,
		ConstructItem: func(fields map[string]any, filePath string) Item {
			return cfg.ConstructItem(fields, filePath)
		},
		MergeUpdate: func(existing Item, req map[string]any) map[string]any {
			// Type assertion to T
			typed, ok := existing.(T)
			if !ok {
				return map[string]any{}
			}
			return cfg.MergeUpdate(typed, req)
		},
		Reload: func() error {
			if cfg.Loader == nil {
				return nil
			}
			_, err := cfg.Loader.Load()
			return err
		},
		FindByName: func(name string) Item {
			if cfg.Loader == nil {
				return nil
			}
			items := cfg.Loader.GetItems()
			for i := range items {
				if items[i].GetName() == name {
					return items[i]
				}
			}
			return nil
		},
	}

	mgr := NewManager(lister, baseDirs, mgrCfg)
	return &TypedManager[T]{mgr: mgr, config: cfg}
}

// Create creates a new item.
func (m *TypedManager[T]) Create(ctx context.Context, req map[string]any) (T, error) {
	item, err := m.mgr.Create(ctx, req)
	if err != nil {
		var zero T
		return zero, err
	}
	typed, ok := item.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("failed to cast item to %s type", m.config.ItemName)
	}
	return typed, nil
}

// Update updates an existing item.
func (m *TypedManager[T]) Update(ctx context.Context, name string, req map[string]any) (T, error) {
	item, err := m.mgr.Update(ctx, name, req)
	if err != nil {
		var zero T
		return zero, err
	}
	typed, ok := item.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("failed to cast item to %s type", m.config.ItemName)
	}
	return typed, nil
}

// Delete removes an item by deleting its directory.
func (m *TypedManager[T]) Delete(ctx context.Context, name string) error {
	return m.mgr.Delete(ctx, name)
}

// Get retrieves an item by name.
func (m *TypedManager[T]) Get(name string) T {
	item := m.mgr.Get(name)
	if item == nil {
		var zero T
		return zero
	}
	typed, ok := item.(T)
	if !ok {
		var zero T
		return zero
	}
	return typed
}

// List returns all items as a slice of T (pointers).
func (m *TypedManager[T]) List() []T {
	items := m.mgr.List()
	if items == nil {
		return nil
	}
	result := make([]T, len(items))
	for i, item := range items {
		typed, ok := item.(T)
		if !ok {
			continue
		}
		result[i] = typed
	}
	return result
}

// Reload reloads all items from disk.
func (m *TypedManager[T]) Reload(ctx context.Context) error {
	return m.mgr.Reload(ctx)
}

// typedLoaderAdapter adapts TypedLoader[T] to ItemLister interface.
type typedLoaderAdapter[T Item] struct {
	loader TypedLoader[T]
}

func (a *typedLoaderAdapter[T]) GetItems() []Item {
	if a == nil || a.loader == nil {
		return nil
	}
	items := a.loader.GetItems()
	if items == nil {
		return nil
	}
	result := make([]Item, len(items))
	for i := range items {
		result[i] = items[i]
	}
	return result
}

// buildDefaultValidator creates a validator that checks required fields.
func buildDefaultValidator(itemName string, requiredFields []string) ValidateFunc {
	return func(req map[string]any) error {
		for _, field := range requiredFields {
			if v, ok := req[field]; !ok || v == "" {
				return fmt.Errorf("%s %s is required", itemName, field)
			}
		}
		return nil
	}
}