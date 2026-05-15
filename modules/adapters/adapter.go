// Package adapters provides a unified interface for external IM platform adapters.
// Adapters enable the agent to receive messages from and send messages to external platforms
// such as Feishu, WeChat, DingTalk, Telegram, etc.
//
// Unlike plugins which are passively invoked tools, adapters are long-running services
// that actively receive events from external platforms and push them to the agent for processing.
package adapters

import (
	"context"

	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
)

// Adapter is the unified interface for all external platform adapters.
// Adapters bridge external IM platforms with the agent's core capabilities.
type Adapter interface {
	// Name returns the adapter name (e.g., "feishu", "wechat", "telegram")
	Name() string

	// Description returns a human-readable description of the adapter
	Description() string

	// Initialize starts the adapter (establishes connections, starts HTTP servers, etc.)
	// The ResourceManager is used to interact with Aura's session and runtime system
	Initialize(ctx context.Context, mgr ResourceManager) error

	// Shutdown gracefully stops the adapter
	Shutdown(ctx context.Context) error

	// Status returns the current running status of the adapter
	Status() AdapterStatus
}

// AdapterStatus represents the runtime status of an adapter
type AdapterStatus struct {
	// Running indicates whether the adapter is running
	Running bool

	// Health indicates the health status: "healthy", "degraded", "error"
	Health string

	// Message provides additional status details (e.g., error messages)
	Message string
}

// SessionAccessor provides access to session management operations.
// Use this interface when you only need session-related operations.
type SessionAccessor interface {
	// GetOrCreateSession gets an existing session or creates a new one.
	// source: the event source (e.g., "feishu", "wechat")
	// identifier: unique identifier for the user/chat from the external platform
	GetOrCreateSession(ctx context.Context, source string, identifier string) (string, error)

	// CreateSession creates a new session with the given configuration
	CreateSession(ctx context.Context, name string, subscriptions []model.Subscription, role string) (*model.Session, error)

	// SessionStore returns the session store for persisting messages
	SessionStore() *storage.JSONLStore
}

// RuntimeAccessor provides access to runtime operations.
// Use this interface when you only need runtime access.
type RuntimeAccessor interface {
	// GetRuntime returns the AgentRuntime for a given session
	GetRuntime(ctx context.Context, sessionID string) (*sdk.Runtime, error)
}

// MessageProcessor provides message processing capabilities.
// Use this interface when you only need to process messages.
type MessageProcessor interface {
	// ProcessMessage processes a message through the session's runtime
	// and returns the event stream
	ProcessMessage(ctx context.Context, sessionID, content string) (<-chan sdk.Event, error)
}

// ResourceManager provides the interface for adapters to interact with Aura's core.
// It combines session management, runtime access, and message processing capabilities.
// This is a composite interface that includes all three smaller interfaces.
type ResourceManager interface {
	SessionAccessor
	RuntimeAccessor
	MessageProcessor
}

// Subscription defines how events are routed to sessions
// Reuses the model from session package
type Subscription = model.Subscription

// Session represents a conversation session
// Reuses the model from session package
type Session = model.Session
