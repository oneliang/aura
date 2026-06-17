package server

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/oneliang/aura/adapters"
	_ "github.com/oneliang/aura/adapters/feishu" // Import for factory registration via init()
	"github.com/oneliang/aura/api/pkg/handlers"
	"github.com/oneliang/aura/api/pkg/middleware"
	cmds "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/core/pkg/intent"
	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/core/pkg/memory"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/session/pkg/manager"
	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/storage"
	"github.com/oneliang/aura/session/pkg/subscription"
	"github.com/oneliang/aura/session/pkg/trigger"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/logger"
	sharedmemory "github.com/oneliang/aura/shared/pkg/memory"
	"github.com/oneliang/aura/shared/pkg/user"
	"github.com/oneliang/aura/shared/pkg/utils"
	skillloader "github.com/oneliang/aura/skill/pkg/loader"
	skillmanager "github.com/oneliang/aura/skill/pkg/manager"
)

//go:embed static/*
var staticFS embed.FS

// webFS is the HTTP filesystem for static web files.
var webFS http.FileSystem

// init initializes the web filesystem.
func init() {
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(fmt.Sprintf("failed to create web filesystem: %v", err))
	}
	webFS = http.FS(subFS)
}

// Constants for logging middleware.
const (
	maxLogBodySize = 1024 * 1024 // 1MB
	maxLogPreview  = 500         // Max characters to log for request/response bodies
)

// sessionRuntime represents a running session runtime.
type sessionRuntime struct {
	runtime   *sdk.Runtime
	createdAt time.Time
}

const (
	maxCachedRuntimes = 100              // Maximum number of cached runtimes
	runtimeTTL        = 30 * time.Minute // TTL for cached runtimes
)

// Server represents the HTTP API server.
type Server struct {
	manager     *manager.SessionManager
	store       *storage.JSONLStore
	port        string
	config      *config.Config
	llmClient   llm.Client
	logger      *logger.Logger
	userID      string
	sessionsDir string // Session data directory for task persistence
	userManager *user.Manager // User manager for authentication

	// Adapter support
	notifier NotificationSender

	// Subscription scheduler
	subscriptionStore     *subscription.Store
	subscriptionScheduler *subscription.Scheduler

	// Runtime management
	runtimes   map[string]*sessionRuntime
	runtimeMu  sync.RWMutex
	mcpManager *sdk.MCPManager

	// Orchestrator management (bridges HTTP and Runtime)
	orchestrators   map[string]*SessionOrchestrator
	orchestratorsMu sync.RWMutex

	// HTTP handlers
	sessionHandler      *handlers.SessionHandler
	notificationHandler *handlers.NotificationHandler
	subscriptionHandler *handlers.SubscriptionHandler
	webhookHandler      *handlers.WebhookHandler
	skillsHandler       *handlers.SkillsHandler

	// SSE concurrency protection
	sseMu         sync.Mutex
	sseProcessing map[string]bool // sessionID -> true if SSE request is being processed

	httpServer *http.Server
	mu         sync.Mutex
	shutdown   bool
	wg         sync.WaitGroup // Tracks active request handlers
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Port        string
	LLMBaseURL  string
	LLMModel    string
	SessionsDir string
	Config      *config.Config // Optional config for agent settings
	UserID      string         // User ID for multi-user isolation
}

// NewServer creates a new API server with adapter and subscription support.
func NewServer(cfg ServerConfig) (*Server, error) {
	// Create storage
	store, err := storage.NewJSONLStore(cfg.SessionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Create session manager
	sessionManager, err := manager.NewSessionManager(store, cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Create LLM client for adapters
	llmFactory := sdk.NewLLMFactory(&cfg.Config.LLM)
	llmClient, err := llmFactory.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Create logger
	log := logger.NewNamed(logger.Config{
		Level:  cfg.Config.Log.Level,
		Format: cfg.Config.Log.Format,
		Output: cfg.Config.Log.Output,
		Path:   cfg.Config.Log.Path,
		Module: "api-server",
	})

	// Check trusted directories configuration (Serve mode does not auto-ask)
	if len(cfg.Config.Permissions.TrustedDirs) == 0 {
		log.Warn("No trusted directories configured. File operations may be restricted.")
		log.Warn("Add trusted directories to ~/.aura/config.yaml or use --allow-all flag")
	} else {
		log.Info("Trusted directories configured", "trusted_dirs", cfg.Config.Permissions.TrustedDirs)
	}

	// Create subscription store and scheduler
	subStore, err := subscription.NewStore(cfg.SessionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription store: %w", err)
	}

	// Create user manager for multi-user support (from users.yaml)
	usersCfg, err := user.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load users config: %w", err)
	}
	userManager, err := user.NewManager(usersCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create user manager: %w", err)
	}

	s := &Server{
		manager:       sessionManager,
		store:         store,
		port:          cfg.Port,
		config:        cfg.Config,
		llmClient:     llmClient,
		logger:        log,
		userID:        cfg.UserID,
		sessionsDir:   cfg.SessionsDir,
		userManager:   userManager,
		runtimes:      make(map[string]*sessionRuntime),
		orchestrators: make(map[string]*SessionOrchestrator),
		sseProcessing: make(map[string]bool),

		// Subscription support
		subscriptionStore:     subStore,
		subscriptionScheduler: nil, // Initialized after adapter
	}

	// Create MCP manager
	s.mcpManager = sdk.NewMCPManager()

	// Create subscription trigger function (uses feishu adapter)
	triggerFunc := func(ctx context.Context, sub *subscription.Subscription) error {
		return s.triggerSubscription(ctx, sub)
	}

	// Create scheduler
	s.subscriptionScheduler = subscription.NewScheduler(subStore, triggerFunc, log)

	// Load and initialize adapters
	if err := s.initializeAdapters(); err != nil {
		// Log warning but don't fail - adapters are optional
		s.logger.Warn("Failed to initialize some adapters", "module", "server", "error", err.Error())
	}

	// Create handlers after adapter initialization
	s.sessionHandler = handlers.NewSessionHandler(s)
	s.notificationHandler = handlers.NewNotificationHandler(s)
	s.subscriptionHandler = handlers.NewSubscriptionHandler(&subscriptionServiceWrapper{s.subscriptionScheduler, s.userID})
	s.webhookHandler = handlers.NewWebhookHandler(s)

	// Create skills handler (requires CommandProvider which is not available in server package)
	// Skills handler will be set externally via SetSkillsHandler method

	return s, nil
}

// initializeAdapters loads and initializes configured adapters.
func (s *Server) initializeAdapters() error {
	// Check master switch
	if !s.config.Adapters.Enabled {
		return nil
	}

	// Check if Feishu adapter is enabled
	feishuConfig := s.config.Adapters.Feishu
	if !feishuConfig.Enabled {
		return nil
	}

	// Validate required config
	if feishuConfig.AppID == "" || feishuConfig.AppSecret == "" {
		s.logger.Warn("Feishu adapter enabled but app_id or app_secret is missing", "module", "server")
		return nil
	}

	// Create adapter registry and wrapper
	adapterReg := adapters.NewRegistry()
	adapterWrapper := adapters.NewAdapterResourceManager(s.manager, s.llmClient, s.config, s.store, s.userID)

	// Create Feishu adapter using factory (no direct import needed)
	feishuAdapterConfig := map[string]any{
		"app_id":                   feishuConfig.AppID,
		"app_secret":               feishuConfig.AppSecret,
		"encrypt_key":              feishuConfig.EncryptKey,
		"verification_token":       feishuConfig.VerificationToken,
		"data_dir":                 s.config.Adapters.DataDir,
		"async_processing":         feishuConfig.AsyncProcessing,
		"auto_reply":               feishuConfig.AutoReply,
		"show_processing_indicator": feishuConfig.ShowProcessingIndicator,
	}

	feishuAdapter, err := adapters.CreateAdapter("feishu", feishuAdapterConfig)
	if err != nil {
		return fmt.Errorf("failed to create feishu adapter: %w", err)
	}

	if err := adapterReg.Register(feishuAdapter); err != nil {
		return fmt.Errorf("failed to register feishu adapter: %w", err)
	}

	// Initialize Feishu adapter
	ctx := context.Background()
	if err := feishuAdapter.Initialize(ctx, adapterWrapper); err != nil {
		return fmt.Errorf("failed to initialize feishu adapter: %w", err)
	}

	// Store as NotificationSender interface using adapterNotifier
	s.notifier = &adapterNotifier{adapter: feishuAdapter, registry: adapterReg}

	s.logger.Info("Feishu adapter initialized (long connection mode)", "module", "server")
	return nil
}

// triggerSubscription is called when a subscription is triggered.
func (s *Server) triggerSubscription(ctx context.Context, sub *subscription.Subscription) error {
	if s.notifier == nil {
		return fmt.Errorf("notification sender not available")
	}

	// Build notification content based on event type
	var content map[string]interface{}
	switch sub.EventType {
	case "daily_report":
		content = map[string]interface{}{
			"text": "Daily report: This is your scheduled daily report.",
		}
	default:
		content = map[string]interface{}{
			"text": fmt.Sprintf("Scheduled notification: %s", sub.EventType),
		}
	}

	// Allow config to override content
	if sub.Config != nil {
		if text, ok := sub.Config["text"].(string); ok {
			content["text"] = text
		}
	}

	return s.notifier.PushToSession(ctx, sub.SessionID, MsgTypeText, content)
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.mu.Lock()

	if s.shutdown {
		s.mu.Unlock()
		return fmt.Errorf("server has been shut down")
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Wrap with logging middleware
	loggedMux := http.HandlerFunc(s.loggingMiddleware(mux))

	s.httpServer = &http.Server{
		Addr:    ":" + s.port,
		Handler: loggedMux,
	}

	// Start subscription scheduler
	if s.subscriptionScheduler != nil {
		s.subscriptionScheduler.Start()
	}

	// Start runtime cleanup goroutine
	s.startRuntimeCleanup()

	// Release lock before calling ListenAndServe to avoid blocking Shutdown()
	s.mu.Unlock()

	return s.httpServer.ListenAndServe()
}

// loggingMiddleware logs HTTP request and response details for debugging.
func (s *Server) loggingMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Only log API requests, not static files
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Read and log request body for POST/PUT/PATCH requests
		var requestBody string
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if r.Body != nil && r.ContentLength > 0 && r.ContentLength < maxLogBodySize { // Only log bodies < 1MB
				bodyBytes, err := readRequestBody(r)
				if err == nil {
					requestBody = utils.Truncate(string(bodyBytes), maxLogPreview)
				}
			}
		}

		// Log request
		if requestBody != "" {
			s.logger.Info("--> HTTP request received", "module", "server", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr, "request_body", requestBody)
		} else {
			s.logger.Info("--> HTTP request received", "module", "server", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
		}

		// Wrap response writer to capture status code and body
		wrapped := &responseWriterWithBody{ResponseWriter: w, statusCode: http.StatusOK, body: &bytes.Buffer{}}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log response
		duration := time.Since(start)
		var responseBody string
		if wrapped.limited {
			responseBody = utils.Truncate(wrapped.body.String(), maxLogPreview) + "... [truncated]"
		} else {
			responseBody = utils.Truncate(wrapped.body.String(), maxLogPreview)
		}
		s.logger.Info("<-- HTTP response sent", "module", "server", "method", r.Method, "path", r.URL.Path, "status", wrapped.statusCode, "duration_ms", duration, "response_body", responseBody)
	}
}

// readRequestBody reads and returns the request body, restoring it for later use.
func readRequestBody(r *http.Request) ([]byte, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	// Restore body for next handler
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}

// responseWriterWithBody wraps http.ResponseWriter to capture status code and body.
// The body buffer is limited to maxLogBodySize to prevent memory exhaustion.
type responseWriterWithBody struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	limited    bool // true if body exceeded maxLogBodySize
}

func (rw *responseWriterWithBody) Write(b []byte) (int, error) {
	// Write to underlying ResponseWriter first
	n, err := rw.ResponseWriter.Write(b)
	if err != nil {
		return n, err
	}
	// Capture for logging only if under limit
	if !rw.limited && rw.body.Len()+len(b) <= maxLogBodySize {
		rw.body.Write(b)
	} else {
		rw.limited = true
	}
	return n, nil
}

func (rw *responseWriterWithBody) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher for SSE support.
func (rw *responseWriterWithBody) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// registerRoutes registers all HTTP routes.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Web UI (must be before API routes to catch /)
	mux.HandleFunc("/", s.handleIndex)
	s.logger.Info("Web UI handler registered", "module", "server")

	// Auth handlers (public routes - no auth required)
	mux.HandleFunc("/api/auth/token", s.handleGetToken)
	mux.HandleFunc("/api/auth/validate", s.handleValidateToken)
	s.logger.Info("Auth handlers registered", "module", "server")

	// Session handlers (protected routes - auth required)
	auth := middleware.AuthMiddleware(s.userManager)
	mux.Handle("/api/sessions", auth(http.HandlerFunc(s.handleSessions)))
	mux.Handle("/api/sessions/", auth(http.HandlerFunc(s.handleSessionByID)))
	s.logger.Info("Session handlers registered", "module", "server")

	// Notification handlers
	mux.HandleFunc("/api/notifications", s.notificationHandler.HandleSendNotification)
	mux.HandleFunc("/api/notifications/task", s.notificationHandler.HandleTaskNotification)
	s.logger.Info("Notification handlers registered", "module", "server")

	// Subscription handlers
	mux.HandleFunc("/api/subscriptions", s.handleSubscriptions)
	mux.HandleFunc("/api/subscriptions/trigger", s.subscriptionHandler.HandleTriggerSubscription)
	mux.HandleFunc("/api/subscriptions/delete", s.subscriptionHandler.HandleDeleteSubscription)
	s.logger.Info("Subscription handlers registered", "module", "server")

	// Webhook handlers
	mux.HandleFunc("/api/webhooks/", s.handleWebhooks)
	mux.HandleFunc("/api/cron", s.handleCronTrigger)
	s.logger.Info("Webhook handlers registered", "module", "server")

	// Health check
	mux.HandleFunc("/api/health", s.handleHealth)
	s.logger.Info("Health check handler registered", "module", "server")

	// Skills handler
	if s.skillsHandler != nil {
		mux.HandleFunc("/api/skills", s.handleSkills)
		s.logger.Info("Skills handler registered", "module", "server")
	}
}

// SetSkillsHandler sets the skills handler.
// This should be called before Start() if skills support is needed.
func (s *Server) SetSkillsHandler(h *handlers.SkillsHandler) {
	s.skillsHandler = h
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shutdown = true

	// Stop subscription scheduler first
	if s.subscriptionScheduler != nil {
		s.subscriptionScheduler.Stop()
	}

	// Shutdown adapters
	if s.notifier != nil {
		if err := s.notifier.Shutdown(ctx); err != nil {
			s.logger.Warn("Failed to shutdown adapters", "module", "server", "error", err.Error())
		}
	}

	// Shutdown HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("HTTP server shutdown error", "module", "server", "error", err.Error())
			return fmt.Errorf("http server shutdown: %w", err)
		}
	}

	// Wait for in-flight requests
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Shutdown runtimes
	s.shutdownAllRuntimes()

	// Close session manager
	s.manager.Close()

	return nil
}

// shutdownAllRuntimes shuts down all running runtimes and orchestrators.
func (s *Server) shutdownAllRuntimes() {
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()

	// Shutdown all orchestrators first
	s.orchestratorsMu.Lock()
	for id, orch := range s.orchestrators {
		orch.Stop()
		delete(s.orchestrators, id)
	}
	s.orchestratorsMu.Unlock()

	// Then shutdown all runtimes
	for id, sr := range s.runtimes {
		sr.runtime.Shutdown()
		delete(s.runtimes, id)
	}
}

// evictOldestRuntime removes the oldest cached runtime to free resources.
// Must be called with s.runtimeMu already held.
func (s *Server) evictOldestRuntime() {
	oldestID := ""
	oldest := time.Now()
	for id, sr := range s.runtimes {
		if sr.createdAt.Before(oldest) {
			oldest = sr.createdAt
			oldestID = id
		}
	}
	if oldestID != "" {
		s.runtimes[oldestID].runtime.Shutdown()
		delete(s.runtimes, oldestID)
	}
}

// evictExpiredRuntimes removes runtimes that have exceeded the TTL.
// Must be called with s.runtimeMu already held.
func (s *Server) evictExpiredRuntimes() {
	now := time.Now()
	for id, sr := range s.runtimes {
		if now.Sub(sr.createdAt) > runtimeTTL {
			// Stop orchestrator first
			s.orchestratorsMu.Lock()
			if orch, ok := s.orchestrators[id]; ok {
				orch.Stop()
				delete(s.orchestrators, id)
			}
			s.orchestratorsMu.Unlock()

			// Then shutdown runtime
			sr.runtime.Shutdown()
			delete(s.runtimes, id)
		}
	}
}

// startRuntimeCleanup launches a background goroutine that periodically
// evicts runtimes that have exceeded their TTL.
func (s *Server) startRuntimeCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			s.runtimeMu.Lock()
			s.evictExpiredRuntimes()
			s.runtimeMu.Unlock()
		}
	}()
}

// getOrCreateRuntime gets or creates an AgentRuntime for a session.
func (s *Server) getOrCreateRuntime(ctx context.Context, sessionID string) (*sdk.Runtime, error) {
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()

	// Check if runtime already exists
	if sr, ok := s.runtimes[sessionID]; ok {
		return sr.runtime, nil
	}

	// Use config values for runtime settings
	var runtimeCfg *sdk.RuntimeConfig
	if s.config != nil {
		runtimeCfg = sdk.FromConfig(s.config)
	} else {
		runtimeCfg = sdk.DefaultRuntimeConfig()
	}
	runtimeCfg.SessionID = sessionID
	runtimeCfg.MessageSource = string(memory.SourceAPI)

	// Load session to get system prompt
	userID := s.userID
	session, err := s.manager.GetSession(sessionID, userID)
	if err == nil {
		runtimeCfg.SystemPrompt = session.SystemPrompt
	}

	// Create CommandProvider for internal commands
	var skillLoader *skillloader.Loader
	var skillMgr *skillmanager.SkillManager
	if s.config.Skills.Enabled && len(s.config.Skills.Directories) > 0 {
		skillLoader = skillloader.NewLoader(s.config.Skills.Directories)
		skillMgr = skillmanager.NewSkillManager(skillLoader, s.config.Skills.Directories)
		if _, err := skillLoader.Load(); err != nil {
			logger.RegistryDefault().Warn("Failed to load skills for API server", "error", err.Error())
			skillLoader = nil
			skillMgr = nil
		}
	}
	cmdProvider := cmds.NewCommandProvider(cmds.CommandProviderDeps{
		Config:       s.config,
		UserID:       s.userID, // API server handles per-request user context
		SkillLoader:  skillLoader,
		SkillManager: skillMgr,
	})

	// Register event bus handlers for command execution (reordered after runtime creation)

	// Create IntentService for natural language command recognition
	var intentSvc *intent.Service
	if s.config.Intent.Enabled {
		intentSvc = intent.NewService(cmdProvider, s.config.Intent.ConfidenceThreshold)
	}

	// Create runtime (with auto-approve for API mode)
	rtOpts := []sdk.RuntimeOption{
		sdk.WithAutoApprove(),
		sdk.WithSessionStore(s.store.MessageStore()),
		sdk.WithSessionID(sessionID),
		sdk.WithCommands(cmdProvider),
		sdk.WithIntentService(intentSvc),
		sdk.WithDataDir(s.sessionsDir),
	}
	if s.mcpManager != nil {
		rtOpts = append(rtOpts, sdk.WithMCPManager(s.mcpManager))
		cmdProvider.SetMCPListFunc(s.mcpListServersAdapter())
	}
	rt, err := sdk.NewRuntime(
		runtimeCfg,
		rtOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime: %w", err)
	}

	// Initialize runtime first (memory created during Initialize)
	if err := rt.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize runtime: %w", err)
	}

	// Register command handlers on the bus with memory from initialized runtime
	bus := cmdProvider.GetEventBus()
	cmds.RegisterDefaultCommandHandlers(bus, rt.GetMemory(), rt, s.config)

	// Cache runtime with eviction protection
	if len(s.runtimes) >= maxCachedRuntimes {
		s.evictOldestRuntime()
	}

	sr := &sessionRuntime{
		runtime:   rt,
		createdAt: time.Now(),
	}
	s.runtimes[sessionID] = sr

	return rt, nil
}

// SendNotification sends a notification to a specific session.
// Implements handlers.NotificationService interface.
func (s *Server) SendNotification(ctx context.Context, sessionID, msgType string, content map[string]interface{}) error {
	if s.notifier == nil {
		return fmt.Errorf("notification sender not available")
	}
	return s.notifier.PushToSession(ctx, sessionID, msgType, content)
}

// SendTaskNotification sends a task completion/failure notification.
// Implements handlers.NotificationService interface.
func (s *Server) SendTaskNotification(ctx context.Context, taskID, status, result string) error {
	if s.notifier == nil {
		return fmt.Errorf("notification sender not available")
	}

	// Build notification content based on status
	var content map[string]interface{}
	if status == "completed" {
		content = map[string]interface{}{
			"text": "Task completed: " + result,
		}
	} else {
		content = map[string]interface{}{
			"text": "Task failed: " + result,
		}
	}

	// Note: task_id should map to a session_id in a real implementation
	// For now, using taskID directly as session identifier
	return s.notifier.PushToSession(ctx, taskID, MsgTypeText, content)
}

// subscriptionServiceWrapper wraps Scheduler to implement handlers.SubscriptionService.
type subscriptionServiceWrapper struct {
	scheduler *subscription.Scheduler
	userID    string
}

// ListSubscriptions returns all subscriptions.
func (w *subscriptionServiceWrapper) ListSubscriptions() []*subscription.Subscription {
	return w.scheduler.ListSubscriptions()
}

// CreateSubscription creates a new subscription.
func (w *subscriptionServiceWrapper) CreateSubscription(sessionID, eventType, cronExpr string, config map[string]interface{}) (*subscription.Subscription, error) {
	userID := w.userID
	sub := subscription.NewSubscription(userID, sessionID, eventType, cronExpr, config)
	if err := w.scheduler.AddSubscription(sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// TriggerSubscription manually triggers a subscription.
func (w *subscriptionServiceWrapper) TriggerSubscription(id string) error {
	return w.scheduler.TriggerSubscription(id)
}

// DeleteSubscription removes a subscription.
func (w *subscriptionServiceWrapper) DeleteSubscription(id string) error {
	return w.scheduler.RemoveSubscription(id)
}

// ProcessEvent implements handlers.WebhookService interface.
func (s *Server) ProcessEvent(ctx context.Context, event trigger.Event) error {
	return s.processEvent(ctx, event)
}

// processEvent routes an event to the appropriate session and processes it.
func (s *Server) processEvent(ctx context.Context, event trigger.Event) error {
	userID := s.userID

	// Find matching session
	sessionID, err := s.manager.RouteEvent(event.Source, event.Content, userID)
	if err != nil {
		return fmt.Errorf("failed to route event: %w", err)
	}

	// If no matching session, create a new one
	if sessionID == "" {
		session, err := s.manager.CreateSession("Auto-created from "+event.Source, nil, "", userID)
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		sessionID = session.ID
	}

	// Get or create agent runtime for this session
	rt, err := s.getOrCreateRuntime(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get runtime: %w", err)
	}

	// Run runtime in background (async processing)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// Start runtime event stream
		if err := rt.Start(ctx); err != nil {
			s.logger.Error("Failed to start runtime", "module", "server", "error", err.Error())
			return
		}
		// Send user input event
		requestID := fmt.Sprintf("webhook_%d", time.Now().UnixNano())
		userEvent := events.NewEvent(events.EventTypeUserInput, event.Content, requestID)
		if err := rt.SendEvent(ctx, userEvent); err != nil {
			s.logger.Error("Failed to send event", "module", "server", "error", err.Error())
			rt.Stop(ctx)
			return
		}
		// Consume events (they are already persisted by the runtime)
		for ev := range rt.Events() {
			if ev.Type() == sdk.EventTypeDone {
				break
			}
		}
		rt.Stop(ctx)
	}()

	return nil
}

// GetMessages implements handlers.SessionService interface.
func (s *Server) GetMessages(ctx context.Context, sessionID string, limit int, userID string) ([]model.Message, error) {
	return s.store.GetMessages(ctx, sessionID, limit, userID)
}

// AppendMessage implements handlers.SessionService interface.
func (s *Server) AppendMessage(ctx context.Context, sessionID, role, content, source string) error {
	msg := &model.Message{
		SessionID: sessionID,
		Role:      role,
		ContentBlocks: []sharedmemory.ContentBlock{
			sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: content},
		},
		Timestamp: time.Now().UnixMilli(),
		Source:    source,
	}
	return s.store.AppendMessage(ctx, msg)
}

// SendMessage implements handlers.SessionService interface.
func (s *Server) SendMessage(ctx context.Context, sessionID, content string) error {
	s.logger.Debug("SendMessage called", "module", "server", "session_id", sessionID, "content", utils.Truncate(content, 100))

	rt, err := s.getOrCreateRuntime(ctx, sessionID)
	if err != nil {
		s.logger.Error("Failed to get runtime", "module", "server", "session_id", sessionID, "error", err.Error())
		return fmt.Errorf("failed to get runtime: %w", err)
	}

	// Use a derived context with 5-minute timeout so the engine can detect cancellation
	// and clean up properly (close events channel, stop goroutines).
	processCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Start runtime event stream
	if err := rt.Start(processCtx); err != nil {
		s.logger.Error("Failed to start runtime", "module", "server", "session_id", sessionID, "error", err.Error())
		return fmt.Errorf("failed to start runtime: %w", err)
	}

	// Send user input event
	requestID := fmt.Sprintf("send_%d", time.Now().UnixNano())
	userEvent := events.NewEvent(events.EventTypeUserInput, content, requestID)
	if err := rt.SendEvent(processCtx, userEvent); err != nil {
		rt.Stop(processCtx)
		s.logger.Error("Failed to send event", "module", "server", "session_id", sessionID, "error", err.Error())
		return fmt.Errorf("failed to send event: %w", err)
	}

	// Consume events — the engine will close the channel when done or cancelled
	eventCount := 0
	for ev := range rt.Events() {
		eventCount++
		if ev.Type() == sdk.EventTypeDone {
			break
		}
	}
	rt.Stop(processCtx)

	s.logger.Debug("SendMessage completed", "module", "server", "session_id", sessionID, "event_count", eventCount)

	if processCtx.Err() != nil {
		return fmt.Errorf("SendMessage timed out or was cancelled: %w", processCtx.Err())
	}
	return nil
}

// handleSendMessageSSE handles message sending with SSE streaming response.
func (s *Server) handleSendMessageSSE(w http.ResponseWriter, r *http.Request, sessionID string) {
	flusher := setupSSEHeaders(w)
	if flusher == nil {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Prevent concurrent SSE requests to the same session
	s.sseMu.Lock()
	if s.sseProcessing[sessionID] {
		s.sseMu.Unlock()
		http.Error(w, "Session is already being processed", http.StatusConflict)
		return
	}
	s.sseProcessing[sessionID] = true
	s.sseMu.Unlock()
	defer func() {
		s.sseMu.Lock()
		delete(s.sseProcessing, sessionID)
		s.sseMu.Unlock()
	}()

	req, err := s.parseSSERequest(r)
	if err != nil {
		s.sendSSEError(w, flusher, err.Error())
		return
	}

	// Send user message echo
	s.sendSSEEvent(w, flusher, "user_message", map[string]interface{}{
		"role":    "user",
		"content": req.Content,
	})

	// Get or create orchestrator
	orch, err := s.getOrCreateOrchestrator(r.Context(), sessionID)
	if err != nil {
		s.sendSSEError(w, flusher, "Failed to get orchestrator")
		return
	}

	// Attach SSE writer to orchestrator
	writerID := fmt.Sprintf("sse_%d", time.Now().UnixNano())
	writer := orch.AttachSSEWriter(writerID, w, flusher)
	defer orch.DetachSSEWriter(writerID)

	// Send user input to runtime via orchestrator
	requestID := fmt.Sprintf("sse_%d", time.Now().UnixNano())
	if err := orch.SendUserInput(req.Content, requestID); err != nil {
		s.logger.Error("Failed to send user input", "module", "server", "session_id", sessionID, "error", err.Error())
		s.sendSSEError(w, flusher, "Failed to send user input")
		return
	}

	s.logger.Debug("Starting event streaming via orchestrator", "module", "server", "session_id", sessionID)

	// Wait for the writer to be closed (by done event or client disconnect)
	select {
	case <-writer.Done():
		s.logger.Debug("SSE writer closed", "module", "server", "session_id", sessionID)
	case <-r.Context().Done():
		s.logger.Debug("Client disconnected", "module", "server", "session_id", sessionID)
	}

	s.logger.Debug("Event streaming completed", "module", "server", "session_id", sessionID)
}

// sendSSEEvent sends an SSE event to the client.
// Returns false if the client has disconnected.
func (s *Server) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) bool {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return false
	}

	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(dataBytes))
	if err != nil {
		return false
	}
	flusher.Flush()

	s.logger.Debug("SSE event sent", "module", "server", "event", eventType)
	return true
}

// sendSSEError sends an error event to the client.
func (s *Server) sendSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	s.sendSSEEvent(w, flusher, string(sdk.EventTypeError), map[string]interface{}{
		"message": message,
	})
}

// getOrCreateOrchestrator gets or creates a SessionOrchestrator for the given session.
func (s *Server) getOrCreateOrchestrator(ctx context.Context, sessionID string) (*SessionOrchestrator, error) {
	// Try to get existing orchestrator
	s.orchestratorsMu.RLock()
	if orch, ok := s.orchestrators[sessionID]; ok {
		s.orchestratorsMu.RUnlock()
		return orch, nil
	}
	s.orchestratorsMu.RUnlock()

	// Create new orchestrator
	s.orchestratorsMu.Lock()
	defer s.orchestratorsMu.Unlock()

	// Double-check after acquiring write lock
	if orch, ok := s.orchestrators[sessionID]; ok {
		return orch, nil
	}

	// Get or create runtime
	rt, err := s.getOrCreateRuntime(context.Background(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime: %w", err)
	}

	// Start runtime if not already started
	// Use background context to match orchestrator's lifecycle
	if err := rt.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to start runtime: %w", err)
	}

	// Create orchestrator
	orch := NewSessionOrchestrator(rt, s.logger)
	orch.Run()
	orch.WaitReady() // Wait for goroutines to be ready

	s.orchestrators[sessionID] = orch
	s.logger.Debug("SessionOrchestrator created", "module", "server", "session_id", sessionID)

	return orch, nil
}

// AddSubscription implements handlers.SessionService interface.
func (s *Server) AddSubscription(sessionID string, sub model.Subscription) error {
	userID := s.userID
	session, err := s.store.GetSession(sessionID, userID)
	if err != nil {
		return err
	}

	session.Subscriptions = append(session.Subscriptions, sub)
	session.UpdatedAt = time.Now().UnixMilli()
	return s.store.SaveSession(session)
}

// RemoveSubscription implements handlers.SessionService interface.
func (s *Server) RemoveSubscription(sessionID, subID, triggerStr string) error {
	userID := s.userID
	session, err := s.store.GetSession(sessionID, userID)
	if err != nil {
		return err
	}

	found := false
	newSubs := make([]model.Subscription, 0, len(session.Subscriptions))
	for _, sub := range session.Subscriptions {
		if (subID != "" && sub.ID == subID) || (triggerStr != "" && sub.Trigger == triggerStr) {
			found = true
			continue
		}
		newSubs = append(newSubs, sub)
	}

	if !found {
		return fmt.Errorf("subscription not found")
	}

	session.Subscriptions = newSubs
	session.UpdatedAt = time.Now().UnixMilli()
	return s.store.SaveSession(session)
}

// ListSessions implements handlers.SessionService interface.
func (s *Server) ListSessions(userID string) ([]*model.Session, error) {
	return s.store.ListSessions(userID)
}

// CreateSession implements handlers.SessionService interface.
func (s *Server) CreateSession(userID string, name string, subscriptions []model.Subscription, systemPrompt string) (*model.Session, error) {
	return s.manager.CreateSession(name, subscriptions, systemPrompt, userID)
}

// GetSession implements handlers.SessionService interface.
func (s *Server) GetSession(id string, userID string) (*model.Session, error) {
	return s.store.GetSession(id, userID)
}

// UpdateSession implements handlers.SessionService interface.
func (s *Server) UpdateSession(id string, userID string, systemPrompt *string, role *string) error {
	session, err := s.store.GetSession(id, userID)
	if err != nil {
		return err
	}

	if systemPrompt != nil {
		session.SystemPrompt = *systemPrompt
	}
	// Note: role field is not used in this version, but kept for interface compatibility
	_ = role

	session.UpdatedAt = time.Now().UnixMilli()
	return s.store.SaveSession(session)
}

// DeleteSession implements handlers.SessionService interface.
func (s *Server) DeleteSession(id string, userID string) error {
	if err := s.store.DeleteSession(context.Background(), id, userID); err != nil {
		return err
	}
	// Clean up cached runtime to prevent resource leak
	s.runtimeMu.Lock()
	if sr, ok := s.runtimes[id]; ok {
		sr.runtime.Shutdown()
		delete(s.runtimes, id)
	}
	s.runtimeMu.Unlock()
	return nil
}

// mcpListServersAdapter adapts MCP manager ListServers to cmds.ListServersFunc.
func (s *Server) mcpListServersAdapter() func() []cmds.MCPInfo {
	return func() []cmds.MCPInfo {
		servers := s.mcpManager.ListServers()
		result := make([]cmds.MCPInfo, len(servers))
		for i, srv := range servers {
			args := ""
			if len(srv.Args) > 0 {
				args = srv.Args[0]
			}
			result[i] = cmds.MCPInfo{
				Name:      srv.Name,
				Command:   srv.Command,
				Args:      args,
				Status:    srv.Status,
				ToolCount: srv.ToolCount,
				Error:     srv.Error,
				LastSeen:  srv.LastSeen,
			}
		}
		return result
	}
}

// adapterNotifier bridges adapters to the NotificationSender interface.
// Uses the adapters.Adapter interface instead of concrete feishu type.
type adapterNotifier struct {
	adapter  adapters.Adapter
	registry *adapters.Registry
}

// PushToSession sends a message to a session via the adapter.
// Requires the adapter to implement MessagePusher interface.
func (n *adapterNotifier) PushToSession(ctx context.Context, sessionID, msgType string, content map[string]interface{}) error {
	// Cast to pusher interface - adapter must implement PushToSession method
	if pusher, ok := n.adapter.(interface {
		PushToSession(ctx context.Context, sessionID, msgType string, content map[string]interface{}) error
	}); ok {
		return pusher.PushToSession(ctx, sessionID, msgType, content)
	}
	return fmt.Errorf("adapter does not implement PushToSession")
}

func (n *adapterNotifier) Shutdown(ctx context.Context) error {
	return n.registry.ShutdownAll(ctx)
}

// handleGetInputHistory handles GET /api/sessions/{id}/input_history.
// Returns user messages (input history) for the session, newest first.
func (s *Server) handleGetInputHistory(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		handlers.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse limit parameter
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		var parsedLimit int
		if _, err := fmt.Sscanf(l, "%d", &parsedLimit); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get messages from session
	ctx := r.Context()
	msgs, err := s.store.GetMessages(ctx, sessionID, limit, s.userID)
	if err != nil {
		s.logger.Error("handleGetInputHistory: error getting messages", "error", err.Error(), "sessionID", sessionID)
		handlers.WriteError(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	// Filter user messages (role = "user") and reverse (newest first)
	var userInputs []string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			// Extract text from ContentBlocks
			for _, block := range msgs[i].ContentBlocks {
				if tb, ok := block.(sharedmemory.TextBlock); ok {
					userInputs = append(userInputs, tb.Text)
					break
				}
			}
		}
	}

	// Response
	handlers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"history": userInputs,
		"count":   len(userInputs),
	})
}

// handleGetToken handles POST /api/auth/token.
// Returns API token for given user ID (no password required).
func (s *Server) handleGetToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handlers.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.WriteError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		handlers.WriteError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get user by ID
	userCfg := s.userManager.GetUserByID(req.UserID)
	if userCfg == nil {
		handlers.WriteJSON(w, http.StatusNotFound, map[string]string{
			"message": "User not found",
		})
		return
	}

	// Return token and user info (token never expires)
	handlers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"token":    userCfg.APIToken,
		"user_id":  userCfg.ID,
		"name":     userCfg.Name,
	})
}

// handleValidateToken handles GET /api/auth/validate.
// Validates the auth token and returns user info.
func (s *Server) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from Authorization header
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		handlers.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Missing authorization header",
		})
		return
	}

	token := strings.TrimPrefix(auth, "Bearer ")

	// Validate token
	userCfg := s.userManager.GetUserByToken(token)
	if userCfg == nil {
		handlers.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Invalid token",
		})
		return
	}

	// Return user info
	handlers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"id":       userCfg.ID,
		"name":     userCfg.Name,
		"api_token": userCfg.APIToken,
	})
}