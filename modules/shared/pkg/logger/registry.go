// Package logger provides structured logging for aura.
package logger

import (
	"fmt"
	"sync"
)

// Registry manages logger instances and their lifecycle.
type Registry struct {
	mu      sync.RWMutex
	loggers map[string]*Logger
}

var (
	globalRegistry = NewRegistry()
)

// NewRegistry creates a new logger registry.
func NewRegistry() *Registry {
	return &Registry{
		loggers: make(map[string]*Logger),
	}
}

// Register creates or returns an existing named logger.
// If a logger with the same name already exists, it is returned unchanged.
// The "default" name is reserved for Default().
func (r *Registry) Register(name string, cfg Config) *Logger {
	r.mu.Lock()
	defer r.mu.Unlock()

	if l, ok := r.loggers[name]; ok {
		return l
	}

	l := New(cfg)
	r.loggers[name] = l
	return l
}

// Get returns a registered logger by name.
func (r *Registry) Get(name string) (*Logger, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	l, ok := r.loggers[name]
	return l, ok
}

// MustGet returns a registered logger, falling back to Default() if not found.
func (r *Registry) MustGet(name string) *Logger {
	if l, ok := r.Get(name); ok {
		return l
	}
	return Default()
}

// Default returns the default logger, registering it if needed.
func (r *Registry) Default() *Logger {
	return r.Register("default", Config{
		Level:  "info",
		Format: "text",
		Output: "file",
	})
}

// CloseAll closes all registered loggers.
func (r *Registry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	for name, l := range r.loggers {
		if err := l.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("logger %s: %w", name, err)
		}
	}
	r.loggers = make(map[string]*Logger)
	return firstErr
}

// Register creates or returns an existing named logger on the global registry.
func Register(name string, cfg Config) *Logger {
	return globalRegistry.Register(name, cfg)
}

// Get returns a registered logger from the global registry.
func Get(name string) (*Logger, bool) {
	return globalRegistry.Get(name)
}

// MustGet returns a registered logger from the global registry, falling back to Default().
func MustGet(name string) *Logger {
	return globalRegistry.MustGet(name)
}

// Default returns the default logger from the global registry.
// Replaces the original Default() function — delegates to registry.
func RegistryDefault() *Logger {
	return globalRegistry.Default()
}

// CloseAll closes all loggers on the global registry.
func CloseAll() error {
	return globalRegistry.CloseAll()
}
