package adapters

import (
	"context"
	"fmt"
	"sync"
)

// AdapterFactory is a function that creates an adapter from configuration.
type AdapterFactory func(config map[string]any) (Adapter, error)

// globalFactories stores adapter factories for auto-registration.
var globalFactories = struct {
	mu       sync.RWMutex
	factories map[string]AdapterFactory
}{
	factories: make(map[string]AdapterFactory),
}

// RegisterFactory registers an adapter factory globally.
// This enables config-driven adapter creation without direct imports.
func RegisterFactory(name string, factory AdapterFactory) {
	globalFactories.mu.Lock()
	defer globalFactories.mu.Unlock()
	globalFactories.factories[name] = factory
}

// GetFactory returns a registered factory by name.
func GetFactory(name string) (AdapterFactory, error) {
	globalFactories.mu.RLock()
	defer globalFactories.mu.RUnlock()
	factory, exists := globalFactories.factories[name]
	if !exists {
		return nil, fmt.Errorf("adapter factory %s not found", name)
	}
	return factory, nil
}

// CreateAdapter creates an adapter using a registered factory.
func CreateAdapter(name string, config map[string]any) (Adapter, error) {
	factory, err := GetFactory(name)
	if err != nil {
		return nil, err
	}
	return factory(config)
}

// ListFactories returns names of all registered factories.
func ListFactories() []string {
	globalFactories.mu.RLock()
	defer globalFactories.mu.RUnlock()
	names := make([]string, 0, len(globalFactories.factories))
	for name := range globalFactories.factories {
		names = append(names, name)
	}
	return names
}

// Registry maintains a collection of adapters and manages their lifecycle.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry creates a new adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register registers an adapter with the registry.
// Returns an error if an adapter with the same name already exists.
func (r *Registry) Register(adapter Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := adapter.Name()
	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("adapter %s already registered", name)
	}

	r.adapters[name] = adapter
	return nil
}

// Unregister removes an adapter from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.adapters, name)
}

// Get returns an adapter by name.
func (r *Registry) Get(name string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", name)
	}
	return adapter, nil
}

// List returns the names of all registered adapters.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered adapters.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.adapters)
}

// InitializeAll initializes all registered adapters.
// If any adapter fails to initialize, an error is returned.
func (r *Registry) InitializeAll(ctx context.Context, mgr ResourceManager) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, adapter := range r.adapters {
		if err := adapter.Initialize(ctx, mgr); err != nil {
			return fmt.Errorf("failed to initialize adapter %s: %w", adapter.Name(), err)
		}
	}
	return nil
}

// ShutdownAll gracefully shuts down all registered adapters.
// Returns the last error encountered, if any.
func (r *Registry) ShutdownAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lastErr error
	for _, adapter := range r.adapters {
		if err := adapter.Shutdown(ctx); err != nil {
			lastErr = fmt.Errorf("failed to shutdown adapter %s: %w", adapter.Name(), err)
		}
	}
	return lastErr
}

// StatusAll returns the status of all registered adapters.
func (r *Registry) StatusAll() map[string]AdapterStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make(map[string]AdapterStatus, len(r.adapters))
	for name, adapter := range r.adapters {
		statuses[name] = adapter.Status()
	}
	return statuses
}

// GetStatus returns the status of a specific adapter.
func (r *Registry) GetStatus(name string) (AdapterStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[name]
	if !exists {
		return AdapterStatus{}, fmt.Errorf("adapter %s not found", name)
	}
	return adapter.Status(), nil
}
