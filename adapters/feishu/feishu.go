package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/oneliang/aura/adapters"
	sdk "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/logger"
)

// init registers the Feishu adapter factory globally.
func init() {
	adapters.RegisterFactory("feishu", func(config map[string]any) (adapters.Adapter, error) {
		// Extract config values
		appID, _ := config["app_id"].(string)
		appSecret, _ := config["app_secret"].(string)
		encryptKey, _ := config["encrypt_key"].(string)
		verificationToken, _ := config["verification_token"].(string)
		dataDir, _ := config["data_dir"].(string)

		// Extract bool config values
		asyncProcessing, _ := config["async_processing"].(bool)
		autoReply, _ := config["auto_reply"].(bool)
		showProcessingIndicator, _ := config["show_processing_indicator"].(bool)

		cfg := &Config{
			Enabled:                 true, // Factory called means adapter is enabled
			AppID:                   appID,
			AppSecret:               appSecret,
			EncryptKey:              encryptKey,
			VerificationToken:       verificationToken,
			DataDir:                 dataDir,
			AsyncProcessing:         asyncProcessing,
			AutoReply:               autoReply,
			ShowProcessingIndicator: showProcessingIndicator,
		}

		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid feishu config: %w", err)
		}

		return NewAdapter(cfg), nil
	})
}

// Adapter is the Feishu IM platform adapter using WebSocket long connection.
type Adapter struct {
	config    *Config
	client    *Client
	wsClient  *ws.Client
	mgr       adapters.ResourceManager
	logger    *logger.Logger
	userStore *UserStore

	mu     sync.RWMutex
	status adapters.AdapterStatus
	wg     sync.WaitGroup
	done   chan struct{}
}

// NewAdapter creates a new Feishu adapter.
func NewAdapter(config *Config) *Adapter {
	return &Adapter{
		config: config,
		status: adapters.AdapterStatus{
			Running: false,
			Health:  "initializing",
		},
		logger: logger.NewNamed(logger.Config{
			Level:  "debug", // Use debug level to capture all details
			Format: "text",
			Output: "stdout",
			Module: "feishu",
		}),
	}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "feishu"
}

// Description returns a description of the Feishu adapter.
func (a *Adapter) Description() string {
	return "飞书 IM 平台适配器（长连接模式），支持消息接收、自动回复和主动推送"
}

// Initialize initializes the Feishu adapter with WebSocket long connection.
func (a *Adapter) Initialize(ctx context.Context, mgr adapters.ResourceManager) error {
	// Phase 1: Validate and setup config
	if err := a.setupConfig(mgr); err != nil {
		return err
	}

	// Phase 2: Create clients (API client, user store, WebSocket)
	if err := a.setupClients(ctx); err != nil {
		return err
	}

	// Phase 3: Start WebSocket connection in background
	a.startWebSocketLoop()

	// Phase 4: Mark as healthy
	a.markHealthy()

	return nil
}

// setupConfig validates configuration and initializes basic fields.
func (a *Adapter) setupConfig(mgr adapters.ResourceManager) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.config.Validate(); err != nil {
		a.status = adapters.AdapterStatus{
			Running: false,
			Health:  "error",
			Message: fmt.Sprintf("Invalid config: %v", err),
		}
		return err
	}

	a.mgr = mgr
	a.done = make(chan struct{})
	return nil
}

// setupClients creates API client, user store, and WebSocket client.
func (a *Adapter) setupClients(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create Feishu API client for sending messages
	a.client = NewClient(a.config.AppID, a.config.AppSecret, a.logger)

	// Create user store for identity mapping
	userStoreFile := filepath.Join(a.config.DataDir, "feishu_users.json")
	userStore, err := NewUserStore(userStoreFile)
	if err != nil {
		a.logger.Warn().Err(err).Str("module", "feishu").Msg("Failed to create user store, using in-memory only")
		userStore, _ = NewUserStore("")
	}
	a.userStore = userStore

	// Create event dispatcher
	eventDispatcher := dispatcher.NewEventDispatcher("", "")
	eventDispatcher.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		msgID := ""
		if event.Event != nil && event.Event.Message != nil && event.Event.Message.MessageId != nil {
			msgID = *event.Event.Message.MessageId
		}
		a.logger.Info().Str("module", "feishu").Str("msg_id", msgID).Msg("Received message")
		return a.handleMessageEvent(ctx, event)
	})
	eventDispatcher.OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
		a.logger.Debug().Str("module", "feishu").Msg("Ignored message_read event")
		return nil
	})

	// Create WebSocket client
	a.wsClient = ws.NewClient(a.config.AppID, a.config.AppSecret, ws.WithEventHandler(eventDispatcher))

	return nil
}

// startWebSocketLoop starts the WebSocket connection goroutine.
func (a *Adapter) startWebSocketLoop() {
	go a.wsConnectionLoop()
}

// wsConnectionLoop manages WebSocket connection with reconnection.
func (a *Adapter) wsConnectionLoop() {
	backoff := 5 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-a.done:
			return
		default:
		}

		a.updateStatus(adapters.AdapterStatus{
			Running: true,
			Health:  "connecting",
			Message: "Establishing WebSocket connection...",
		})

		// Create a context tied to the adapter's lifecycle
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-a.done
			cancel()
		}()

		// Start the WebSocket client (blocking call)
		err := a.wsClient.Start(ctx)
		cancel()

		if err == nil {
			// Clean exit (shutdown requested)
			return
		}

		// Connection dropped — attempt reconnect with exponential backoff
		select {
		case <-a.done:
			return
		case <-time.After(backoff):
		}

		a.updateStatus(adapters.AdapterStatus{
			Running: true,
			Health:  "reconnecting",
			Message: fmt.Sprintf("WebSocket error: %v, reconnecting in %v", err, backoff),
		})

		// Exponential backoff with cap
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// updateStatus safely updates the adapter status.
func (a *Adapter) updateStatus(status adapters.AdapterStatus) {
	a.mu.Lock()
	a.status = status
	a.mu.Unlock()
}

// markHealthy marks the adapter as healthy after initialization.
func (a *Adapter) markHealthy() {
	a.updateStatus(adapters.AdapterStatus{
		Running: true,
		Health:  "healthy",
		Message: "WebSocket connection established",
	})
}

// Shutdown gracefully shuts down the Feishu adapter.
func (a *Adapter) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.status.Running = false
	a.status.Health = "shutting_down"

	// Signal shutdown
	close(a.done)

	// The WebSocket connection is cancelled via context when a.done is closed
	// (see Initialize). No explicit Close method is exposed by ws.Client.

	// Wait for pending message processing
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.status = adapters.AdapterStatus{
			Running: false,
			Health:  "stopped",
			Message: "Shutdown complete",
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Status returns the current adapter status.
func (a *Adapter) Status() adapters.AdapterStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// handleMessageEvent handles a direct message event from Feishu.
func (a *Adapter) handleMessageEvent(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	// Get sender identifier
	senderID := a.getSenderID(event)
	if senderID == "" {
		a.logger.Error().Str("module", "feishu").Msg("Cannot get sender ID")
		return fmt.Errorf("cannot get sender id")
	}

	a.logger.Info().Str("module", "feishu").Str("sender", senderID).Msg("Processing message from sender")

	// Get or create session
	sessionID, err := a.mgr.GetOrCreateSession(ctx, "feishu", senderID)
	if err != nil {
		a.logger.Error().Str("module", "feishu").Err(err).Msg("Get or create session failed")
		return fmt.Errorf("get or create session: %w", err)
	}

	a.logger.Debug().Str("module", "feishu").Str("session_id", sessionID).Msg("Session created/retrieved")

	// Auto-save user identity mapping
	a.saveUserInfo(ctx, sessionID, senderID, event)

	// Parse message content
	content := a.getMessageText(event)
	if strings.TrimSpace(content) == "" {
		a.logger.Debug().Str("module", "feishu").Msg("Skipping empty message")
		return nil // Skip empty messages
	}

	a.logger.Info().Str("module", "feishu").Str("content", content).Msg("Message content")

	// Process message asynchronously if configured
	if a.config.AsyncProcessing {
		a.mu.RLock()
		shuttingDown := !a.status.Running
		a.mu.RUnlock()
		if shuttingDown {
			a.logger.Debug().Str("module", "feishu").Msg("Skipping async processing, adapter shutting down")
			return nil
		}
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.processMessage(ctx, sessionID, content, &MessageEventInfo{
				ChatID:    a.getChatID(event),
				MessageID: a.getMessageID(event),
				SenderID:  senderID,
				IsGroup:   false,
			})
		}()
	} else {
		a.logger.Debug().Str("module", "feishu").Msg("Processing message synchronously")
		a.processMessage(ctx, sessionID, content, &MessageEventInfo{
			ChatID:    a.getChatID(event),
			MessageID: a.getMessageID(event),
			SenderID:  senderID,
			IsGroup:   false,
		})
	}

	return nil
}

// MessageEventInfo contains message event metadata.
type MessageEventInfo struct {
	ChatID      string
	MessageID   string
	SenderID    string
	IsGroup     bool
	ReceiveType string // "open_id" or "chat_id" for sending messages back
}

// processMessage processes a message through Aura and sends reply if configured.
func (a *Adapter) processMessage(ctx context.Context, sessionID, content string, msgInfo *MessageEventInfo) {
	a.logger.Debug().Str("module", "feishu").Str("session", sessionID).Str("content", content).Msg("Processing message")

	// Get runtime for session
	rt, err := a.mgr.GetRuntime(ctx, sessionID)
	if err != nil {
		a.logger.Error().Str("module", "feishu").Err(err).Msg("Get runtime failed")
		return
	}

	// Collect response for reply
	var responseBuilder strings.Builder
	var hasResponse bool

	// Create event handler to capture responses
	eventHandler := func(ev sdk.Event) {
		// Only process response events for reply
		if ev.Type() != sdk.EventTypeResponse {
			return
		}
		responseBuilder.WriteString(ev.Content())
		hasResponse = true
	}

	// Create confirmation handler (auto-approve for Feishu)
	confirmHandler := func(req sdk.ConfirmationRequest) {
		req.ResponseCh <- true
	}

	// Set handlers on runtime
	rt.SetEventHandler(eventHandler)
	rt.SetConfirmationHandler(confirmHandler)

	// Determine target for sending messages
	var targetID, targetType string
	if msgInfo.IsGroup {
		targetID = msgInfo.ChatID
		targetType = larkim.ReceiveIdTypeChatId
	} else {
		targetID = msgInfo.SenderID
		targetType = larkim.ReceiveIdTypeOpenId
	}

	// Add THINKING emoji reaction as "processing" indicator if enabled
	var hasProcessingReaction bool
	if a.config.ShowProcessingIndicator && msgInfo.MessageID != "" {
		if err := a.addProcessingReaction(ctx, msgInfo.MessageID); err != nil {
			a.logger.Debug().Str("module", "feishu").Err(err).Msg("Failed to add processing reaction")
		} else {
			hasProcessingReaction = true
		}
	}

	// Process message (agent.Run will automatically add user message to memory)
	events, err := rt.Process(ctx, content)
	if err != nil {
		a.logger.Error().Str("module", "feishu").Err(err).Msg("Process message failed")
		// Remove processing reaction on error
		if hasProcessingReaction && msgInfo.MessageID != "" {
			a.removeProcessingReaction(ctx, msgInfo.MessageID)
		}
		return
	}

	// Consume event stream (handler already set via SetEventHandler, called by runtime's event pump)
	for ev := range events {
		// Just drain the channel, handler is called internally by processing.go
		// Log for debugging
		a.logger.Debug().Str("module", "feishu").Str("event_type", string(ev.Type())).Msg("Event received")
	}

	a.logger.Debug().Str("module", "feishu").Bool("has_response", hasResponse).Msg("Processed message")

	// Remove processing reaction before sending reply
	if hasProcessingReaction && msgInfo.MessageID != "" {
		a.removeProcessingReaction(ctx, msgInfo.MessageID)
	}

	// Send reply if auto-reply is enabled and we have a response
	if a.config.AutoReply && hasResponse {
		responseText := responseBuilder.String()
		a.logger.Info().Str("module", "feishu").Str("response", responseText).Msg("Sending reply")
		if err := a.client.SendTextMessage(ctx, targetID, targetType, responseText); err != nil {
			a.logger.Error().Str("module", "feishu").Err(err).Msg("Send reply message failed")
		} else {
			a.logger.Info().Str("module", "feishu").Str("to", targetID).Msg("Sent reply message")
		}
	}
}

// addProcessingReaction adds a THINKING emoji reaction to the user's message.
func (a *Adapter) addProcessingReaction(ctx context.Context, messageID string) error {
	a.logger.Debug().Str("module", "feishu").Str("message_id", messageID).Msg("Adding THINKING reaction")
	return a.client.AddMessageReaction(ctx, messageID, "THINKING")
}

// removeProcessingReaction removes the THINKING emoji reaction from the user's message.
func (a *Adapter) removeProcessingReaction(ctx context.Context, messageID string) {
	a.logger.Debug().Str("module", "feishu").Str("message_id", messageID).Msg("Removing THINKING reaction")
	if err := a.client.RemoveMessageReactionByType(ctx, messageID, "THINKING"); err != nil {
		a.logger.Debug().Str("module", "feishu").Err(err).Msg("Failed to remove processing reaction")
	}
}

// getSenderID extracts the sender ID from a message event.
func (a *Adapter) getSenderID(event *larkim.P2MessageReceiveV1) string {
	if event.Event != nil && event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		if event.Event.Sender.SenderId.OpenId != nil {
			return *event.Event.Sender.SenderId.OpenId
		}
	}
	return ""
}

// getChatID extracts the chat ID from a message event.
func (a *Adapter) getChatID(event *larkim.P2MessageReceiveV1) string {
	if event.Event != nil && event.Event.Message != nil && event.Event.Message.ChatId != nil {
		return *event.Event.Message.ChatId
	}
	return ""
}

// getMessageID extracts the message ID from a message event.
func (a *Adapter) getMessageID(event *larkim.P2MessageReceiveV1) string {
	if event.Event != nil && event.Event.Message != nil && event.Event.Message.MessageId != nil {
		return *event.Event.Message.MessageId
	}
	return ""
}

// getMessageText extracts the message text from a message event.
func (a *Adapter) getMessageText(event *larkim.P2MessageReceiveV1) string {
	if event.Event == nil || event.Event.Message == nil || event.Event.Message.Content == nil {
		return ""
	}

	// Content is a JSON string, parse it to get the text
	contentStr := *event.Event.Message.Content

	// For text messages, content is like: {"text":"hello"}
	if strings.Contains(contentStr, "\"text\"") {
		var content map[string]string
		if err := json.Unmarshal([]byte(contentStr), &content); err != nil {
			return contentStr
		}
		if text, ok := content["text"]; ok {
			return text
		}
	}

	return contentStr
}

// saveUserInfo saves user identity information to the user store.
func (a *Adapter) saveUserInfo(ctx context.Context, sessionID, openID string, event *larkim.P2MessageReceiveV1) {
	userInfo := &UserInfo{
		SessionID: sessionID,
		OpenID:    openID,
		IsGroup:   false,
		ChatID:    a.getChatID(event),
	}

	// Extract additional user info if available
	if event.Event != nil && event.Event.Sender != nil {
		sender := event.Event.Sender
		if sender.SenderId != nil {
			if sender.SenderId.UserId != nil {
				userInfo.UserID = *sender.SenderId.UserId
			}
			if sender.SenderId.UnionId != nil {
				userInfo.UnionID = *sender.SenderId.UnionId
			}
		}
	}

	if err := a.userStore.Set(userInfo); err != nil {
		a.logger.Warn().Err(err).Str("module", "feishu").
			Str("session_id", sessionID).
			Str("open_id", openID).
			Msg("Failed to save user info")
	} else {
		a.logger.Debug().Str("module", "feishu").
			Str("session_id", sessionID).
			Str("open_id", openID).
			Msg("Saved user info")
	}
}

// PushMessage sends a proactive message to a target (implements adapters.MessagePusher).
func (a *Adapter) PushMessage(ctx context.Context, targetType, targetID, msgType string, content map[string]interface{}) error {
	switch msgType {
	case adapters.MsgTypeText:
		text, ok := content["text"].(string)
		if !ok {
			return fmt.Errorf("text content required for text message")
		}
		return a.client.SendTextMessage(ctx, targetID, targetType, text)
	case adapters.MsgTypePost:
		return a.client.SendPostMessage(ctx, targetID, targetType, content)
	case adapters.MsgTypeCard:
		return a.client.SendCardMessage(ctx, targetID, targetType, content)
	default:
		return fmt.Errorf("unsupported message type: %s", msgType)
	}
}

// BroadcastMessage sends the same message to multiple targets (implements adapters.MessagePusher).
func (a *Adapter) BroadcastMessage(ctx context.Context, targets []adapters.MessageTarget, msgType string, content map[string]interface{}) map[string]error {
	results := make(map[string]error)
	for _, target := range targets {
		if err := a.PushMessage(ctx, target.TargetType, target.TargetID, msgType, content); err != nil {
			results[target.TargetID] = err
			a.logger.Error().Str("module", "feishu").
				Str("target", target.TargetID).
				Err(err).
				Msg("Broadcast message failed")
		} else {
			results[target.TargetID] = nil
			a.logger.Debug().Str("module", "feishu").
				Str("target", target.TargetID).
				Msg("Broadcast message sent")
		}
	}
	return results
}

// GetUserStore returns the user store for identity mapping.
func (a *Adapter) GetUserStore() *UserStore {
	return a.userStore
}

// PushToSession sends a message to a session by session ID.
func (a *Adapter) PushToSession(ctx context.Context, sessionID, msgType string, content map[string]interface{}) error {
	// Look up user info by session ID
	userInfo, ok := a.userStore.GetBySessionID(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Determine target type and ID
	var targetType, targetID string
	if userInfo.IsGroup {
		targetType = larkim.ReceiveIdTypeChatId
		targetID = userInfo.ChatID
	} else {
		targetType = larkim.ReceiveIdTypeOpenId
		targetID = userInfo.OpenID
	}

	return a.PushMessage(ctx, targetType, targetID, msgType, content)
}
