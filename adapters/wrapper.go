package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	cmds "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/llm"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/shared/pkg/utils"
	skillloader "github.com/oneliang/aura/skill/pkg/loader"
	skillmanager "github.com/oneliang/aura/skill/pkg/manager"
)

const maxCachedRuntimes = 100

// mapEventTypeToMessageType maps SDK event types to memory message types.
// This ensures consistent message handling across the system.
// Only conversation messages (assistant) are returned - internal events return empty string.
func mapEventTypeToMessageType(eventType sdk.EventType) sharedmemory.MessageType {
	switch eventType {
	case sdk.EventTypeResponse:
		return sharedmemory.MessageTypeAssistant
	// Internal events - not persisted to session storage by memory layer
	case sdk.EventTypeToolStart, sdk.EventTypeToolEnd:
		return sharedmemory.MessageTypeToolStart
	case sdk.EventTypeError:
		return sharedmemory.MessageTypeError
	default:
		// For other event types, don't persist to session memory
		return ""
	}
}

// sessionRuntime represents a running session runtime.
type sessionRuntime struct {
	runtime   *sdk.Runtime
	createdAt time.Time
}

// AdapterResourceManager wraps SessionManager to implement the ResourceManager interface.
// This allows adapters to interact with the agent's session and runtime system.
type AdapterResourceManager struct {
	sessionMgr *manager.SessionManager
	llmClient  llm.Client
	config     *config.Config
	store      *storage.JSONLStore
	userID     string
	runtimes   map[string]*sessionRuntime // In-memory runtime instances
	mu         sync.RWMutex               // Protects runtimes map
}

// NewAdapterResourceManager creates a new AdapterResourceManager.
func NewAdapterResourceManager(
	sessionMgr *manager.SessionManager,
	llmClient llm.Client,
	cfg *config.Config,
	store *storage.JSONLStore,
	userID string,
) *AdapterResourceManager {
	return &AdapterResourceManager{
		sessionMgr: sessionMgr,
		llmClient:  llmClient,
		config:     cfg,
		store:      store,
		userID:     userID,
		runtimes:   make(map[string]*sessionRuntime),
	}
}

// GetOrCreateSession gets an existing session or creates a new one.
// For adapters, sessions are created based on external platform user identifiers.
func (w *AdapterResourceManager) GetOrCreateSession(ctx context.Context, source string, identifier string) (string, error) {
	userID := w.userID

	// Try to find existing session with matching subscription
	sessions, err := w.sessionMgr.ListSessions(userID)
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}

	// Look for session with matching source subscription
	sourceIdentifier := fmt.Sprintf("%s:%s", source, identifier)
	for _, session := range sessions {
		for _, sub := range session.Subscriptions {
			if sub.Active && sub.Source == sourceIdentifier {
				return session.ID, nil
			}
		}
	}

	// No matching session, create new one
	sessionName := fmt.Sprintf("%s-%s", source, identifier[:utils.Min(8, len(identifier))])
	subscriptions := []model.Subscription{
		{
			ID:      fmt.Sprintf("sub_%s_%s", source, identifier),
			Trigger: "", // Match all messages from this source
			Source:  sourceIdentifier,
			Active:  true,
		},
	}

	session, err := w.sessionMgr.CreateSession(sessionName, subscriptions, "", userID)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return session.ID, nil
}

// GetRuntime gets or creates an AgentRuntime for a session.
// Note: For adapter mode, we don't persist tool events to session storage
// to avoid inconsistency. The Runtime's SessionMemory handles all persistence.
func (w *AdapterResourceManager) GetRuntime(ctx context.Context, sessionID string) (*sdk.Runtime, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if runtime already exists
	if sr, ok := w.runtimes[sessionID]; ok {
		return sr.runtime, nil
	}

	// Use config values for runtime settings
	var runtimeCfg *sdk.RuntimeConfig
	if w.config != nil {
		runtimeCfg = sdk.FromConfig(w.config)
	} else {
		runtimeCfg = sdk.DefaultRuntimeConfig()
	}
	runtimeCfg.SessionID = sessionID

	// Load session to get system prompt
	userID := w.userID
	session, err := w.sessionMgr.GetSession(sessionID, userID)
	if err == nil {
		runtimeCfg.SystemPrompt = session.SystemPrompt
	}

	// Create CommandProvider for internal commands
	var skillLoader *skillloader.Loader
	var skillMgr *skillmanager.SkillManager
	if w.config.Skills.Enabled && len(w.config.Skills.Directories) > 0 {
		skillLoader = skillloader.NewLoader(w.config.Skills.Directories)
		skillMgr = skillmanager.NewSkillManager(skillLoader, w.config.Skills.Directories)
		if _, err := skillLoader.Load(); err != nil {
			skillLoader = nil
			skillMgr = nil
		}
	}
	cmdProvider := cmds.NewCommandProvider(cmds.CommandProviderDeps{
		Config:       w.config,
		UserID:       w.userID,
		SkillLoader:  skillLoader,
		SkillManager: skillMgr,
	})

	// Create runtime (with auto-approve for adapter mode)
	rt, err := sdk.NewRuntime(
		runtimeCfg,
		sdk.WithAutoApprove(),
		sdk.WithSessionStore(w.store.MessageStore()),
		sdk.WithSessionID(sessionID),
		sdk.WithCommands(cmdProvider),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime: %w", err)
	}

	// Register event bus handlers for command execution (after runtime exists for taskProvider)
	bus := cmdProvider.GetEventBus()
	cmds.RegisterDefaultCommandHandlers(bus, nil, rt, w.config)

	// Initialize runtime
	if err := rt.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize runtime: %w", err)
	}

	// Cache runtime with eviction protection
	if len(w.runtimes) >= maxCachedRuntimes {
		// Evict oldest runtime
		oldestID := ""
		oldest := time.Now()
		for id, sr := range w.runtimes {
			if sr.createdAt.Before(oldest) {
				oldest = sr.createdAt
				oldestID = id
			}
		}
		if oldestID != "" {
			w.runtimes[oldestID].runtime.Shutdown()
			delete(w.runtimes, oldestID)
		}
	}

	sr := &sessionRuntime{
		runtime:   rt,
		createdAt: time.Now(),
	}
	w.runtimes[sessionID] = sr

	return rt, nil
}

// ProcessMessage processes a message through the session's runtime.
func (w *AdapterResourceManager) ProcessMessage(ctx context.Context, sessionID, content string) (<-chan sdk.Event, error) {
	rt, err := w.GetRuntime(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return rt.Process(ctx, content)
}

// SessionStore returns the session store.
func (w *AdapterResourceManager) SessionStore() *storage.JSONLStore {
	return w.store
}

// CreateSession creates a new session with the given configuration.
func (w *AdapterResourceManager) CreateSession(ctx context.Context, name string, subscriptions []model.Subscription, role string) (*model.Session, error) {
	userID := w.userID
	return w.sessionMgr.CreateSession(name, subscriptions, role, userID)
}

// GetSession retrieves a session by ID.
func (w *AdapterResourceManager) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	userID := w.userID
	return w.sessionMgr.GetSession(sessionID, userID)
}

// Close closes all cached runtimes.
func (w *AdapterResourceManager) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for id, sr := range w.runtimes {
		sr.runtime.Shutdown()
		delete(w.runtimes, id)
	}
}
