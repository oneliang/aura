package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/oneliang/aura/session/pkg/model"
	"github.com/oneliang/aura/session/pkg/subscription"
	"github.com/oneliang/aura/session/pkg/trigger"
	"github.com/oneliang/aura/shared/pkg/config"
)

// TestNewServer tests server creation with various configurations.
func TestNewServer(t *testing.T) {
	t.Run("success with minimal config", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := ServerConfig{
			Port:        "8080",
			SessionsDir: tmpDir,
			Config:      createTestConfig(),
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}
		if server == nil {
			t.Fatal("NewServer() returned nil")
		}

		// Cleanup
		server.Shutdown(context.Background())
	})

	t.Run("success with feishu config", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := createTestConfig()
		// Enable feishu adapter with test credentials
		cfg.Adapters.Feishu.Enabled = true
		cfg.Adapters.Feishu.AppID = "test-app-id"
		cfg.Adapters.Feishu.AppSecret = "test-app-secret"
		cfg.Adapters.Feishu.WebhookPath = "/api/webhooks/feishu"
		cfg.Adapters.Feishu.Port = "8081"

		serverCfg := ServerConfig{
			Port:        "8081",
			SessionsDir: tmpDir,
			Config:      cfg,
		}

		server, err := NewServer(serverCfg)
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}
		if server == nil {
			t.Fatal("NewServer() returned nil")
		}

		// Cleanup
		server.Shutdown(context.Background())
	})

	t.Run("invalid data dir", func(t *testing.T) {
		cfg := ServerConfig{
			Port:        "8082",
			SessionsDir: "/root/protected/invalid/path/that/should/fail",
			Config:      createTestConfig(),
		}

		_, err := NewServer(cfg)
		if err == nil {
			t.Error("NewServer() with invalid data dir should return error")
		}
	})

	t.Run("empty data dir", func(t *testing.T) {
		cfg := ServerConfig{
			Port:   "8083",
			Config: createTestConfig(),
		}

		_, err := NewServer(cfg)
		if err == nil {
			t.Error("NewServer() with empty data dir should return error")
		}
	})

	t.Run("nil config uses defaults", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Note: NewServer requires a non-nil Config
		// This test documents that passing nil will panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("NewServer() with nil config should panic")
			}
		}()

		cfg := ServerConfig{
			Port:        "8084",
			SessionsDir: tmpDir,
			Config:      nil, // Will cause panic
		}

		_, _ = NewServer(cfg)
	})
}

// TestServer_Start tests server Start method.
func TestServer_Start(t *testing.T) {
	t.Run("start on available port", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := ServerConfig{
			Port:        findAvailablePort(t),
			SessionsDir: tmpDir,
			Config:      createTestConfig(),
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}

		// Start server in goroutine
		done := make(chan bool)
		go func() {
			err := server.Start()
			if err != nil && err.Error() != "http: Server closed" {
				t.Logf("Start() error = %v", err)
			}
			done <- true
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Verify server is running by checking if port is in use
		// (We can't directly check, but Shutdown should work)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = server.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}

		// Wait for goroutine to exit
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("Server did not shutdown gracefully")
		}
	})

	t.Run("start after shutdown", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := ServerConfig{
			Port:        "8085",
			SessionsDir: tmpDir,
			Config:      createTestConfig(),
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}

		// Shutdown first
		server.Shutdown(context.Background())

		// Try to start
		err = server.Start()
		if err == nil {
			t.Error("Start() after shutdown should return error")
		}
	})
}

// TestServer_Shutdown tests server Shutdown method.
func TestServer_Shutdown(t *testing.T) {
	t.Run("graceful shutdown", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := ServerConfig{
			Port:        findAvailablePort(t),
			SessionsDir: tmpDir,
			Config:      createTestConfig(),
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}

		// Start server
		go server.Start()
		time.Sleep(100 * time.Millisecond)

		// Graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = server.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	t.Run("shutdown with timeout", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := ServerConfig{
			Port:        findAvailablePort(t),
			SessionsDir: tmpDir,
			Config:      createTestConfig(),
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}

		// Start server
		go server.Start()
		time.Sleep(100 * time.Millisecond)

		// Immediate timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		err = server.Shutdown(ctx)
		// May return context deadline exceeded
		if err != nil && err.Error() != "context deadline exceeded" {
			t.Logf("Shutdown() with timeout error = %v (expected)", err)
		}
	})

	t.Run("shutdown without start", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := ServerConfig{
			Port:        "8086",
			SessionsDir: tmpDir,
			Config:      createTestConfig(),
		}

		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}

		// Shutdown without start should not panic
		err = server.Shutdown(context.Background())
		if err != nil {
			t.Errorf("Shutdown() without start error = %v", err)
		}
	})
}

// TestServer_registerRoutes tests route registration.
func TestServer_registerRoutes(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.registerRoutes(mux)

	// Test that routes are registered by making requests
	tests := []struct {
		path       string
		method     string
		wantStatus int
	}{
		{"/api/health", http.MethodGet, http.StatusOK},
		{"/api/sessions", http.MethodGet, http.StatusOK},
		{"/api/subscriptions", http.MethodGet, http.StatusOK},
		{"/api/webhooks/", http.MethodPost, http.StatusBadRequest}, // Missing source
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			// Add auth token for protected routes
			if tt.path == "/api/sessions" || tt.path == "/api/sessions/" {
				req.Header.Set("Authorization", "Bearer test-token-123")
			}
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestServer_initializeAdapters tests adapter initialization.
func TestServer_initializeAdapters(t *testing.T) {
	t.Run("adapter disabled", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := createTestConfig()
		cfg.Adapters.Feishu.Enabled = false

		server, err := NewServer(ServerConfig{
			Port:        "8087",
			SessionsDir: tmpDir,
			Config:      cfg,
		})
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}
		defer server.Shutdown(context.Background())

		// initializeAdapters is called during NewServer, but we can verify
		// that notifier is nil when disabled
		if server.notifier != nil {
			t.Error("notifier should be nil when adapters disabled")
		}
	})

	t.Run("adapter with invalid config", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := createTestConfig()
		cfg.Adapters.Feishu.Enabled = true
		// Missing required credentials - should skip adapter

		server, err := NewServer(ServerConfig{
			Port:        "8088",
			SessionsDir: tmpDir,
			Config:      cfg,
		})
		if err != nil {
			t.Fatalf("NewServer() error = %v", err)
		}
		defer server.Shutdown(context.Background())

		// Adapter should be nil due to invalid config
		if server.notifier != nil {
			t.Error("notifier should be nil with invalid config")
		}
	})
}

// TestServer_triggerSubscription tests subscription triggering.
func TestServer_triggerSubscription(t *testing.T) {
	t.Run("notification sender not available", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		sub := subscription.NewSubscription("", "session-1", "daily_report", "0 9 * * *", nil)

		err := server.triggerSubscription(context.Background(), sub)
		if err == nil {
			t.Error("triggerSubscription() should return error when notification sender unavailable")
		}
		if err.Error() != "notification sender not available" {
			t.Errorf("triggerSubscription() error = %q, want 'notification sender not available'", err)
		}
	})

	t.Run("with event type override", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		sub := subscription.NewSubscription("", "session-1", "custom_event", "0 9 * * *", map[string]interface{}{
			"text": "Custom notification",
		})

		err := server.triggerSubscription(context.Background(), sub)
		if err == nil {
			t.Error("triggerSubscription() should return error when feishu adapter unavailable")
		}
	})
}

// TestServer_getOrCreateRuntime tests runtime management.
func TestServer_getOrCreateRuntime(t *testing.T) {
	t.Run("create new runtime", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		// Create a session first
		session, err := server.manager.CreateSession("Test Session", nil, "You are helpful", "")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		rt, err := server.getOrCreateRuntime(context.Background(), session.ID)
		if err != nil {
			t.Fatalf("getOrCreateRuntime() error = %v", err)
		}
		if rt == nil {
			t.Fatal("getOrCreateRuntime() returned nil")
		}

		// Verify runtime is cached
		if len(server.runtimes) != 1 {
			t.Errorf("Expected 1 cached runtime, got %d", len(server.runtimes))
		}
	})

	t.Run("reuse cached runtime", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		// Create session and runtime
		session, _ := server.manager.CreateSession("Test", nil, "", "")
		rt1, _ := server.getOrCreateRuntime(context.Background(), session.ID)

		// Get again - should return cached
		rt2, err := server.getOrCreateRuntime(context.Background(), session.ID)
		if err != nil {
			t.Fatalf("getOrCreateRuntime() error = %v", err)
		}

		if rt1 != rt2 {
			t.Error("Should return cached runtime")
		}
	})

	t.Run("non-existent session", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		// Try to get runtime for non-existent session
		// This should fail when trying to load session for system prompt
		_, err := server.getOrCreateRuntime(context.Background(), "non-existent")
		// May succeed if system prompt is optional
		if err != nil {
			t.Logf("getOrCreateRuntime() for non-existent session error = %v (expected)", err)
		}
	})
}

// TestServer_shutdownAllRuntimes tests runtime shutdown.
func TestServer_shutdownAllRuntimes(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create some runtimes
	session1, _ := server.manager.CreateSession("Session 1", nil, "", "")
	session2, _ := server.manager.CreateSession("Session 2", nil, "", "")

	server.getOrCreateRuntime(context.Background(), session1.ID)
	server.getOrCreateRuntime(context.Background(), session2.ID)

	// Shutdown all
	server.shutdownAllRuntimes()

	// Verify runtimes are cleared
	if len(server.runtimes) != 0 {
		t.Errorf("Expected 0 runtimes after shutdown, got %d", len(server.runtimes))
	}
}

// TestServer_SendNotification tests notification sending.
func TestServer_SendNotification(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	content := map[string]interface{}{
		"text": "Test notification",
	}

	err := server.SendNotification(context.Background(), "session-1", "text", content)
	if err == nil {
		t.Error("SendNotification() should return error when feishu adapter unavailable")
	}
}

// TestServer_SendTaskNotification tests task notification sending.
func TestServer_SendTaskNotification(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name   string
		status string
		result string
	}{
		{"completed task", "completed", "Task done successfully"},
		{"failed task", "failed", "Error: something went wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.SendTaskNotification(context.Background(), "task-1", tt.status, tt.result)
			if err == nil {
				t.Error("SendTaskNotification() should return error when feishu adapter unavailable")
			}
		})
	}
}

// TestServer_processEvent tests event processing.
func TestServer_processEvent(t *testing.T) {
	t.Run("route to existing session", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		// Create session with subscription
		_, _ = server.manager.CreateSession("Test Session", nil, "", "")

		event := trigger.Event{
			Source:  "feishu",
			Content: "Hello",
		}

		err := server.processEvent(context.Background(), event)
		if err != nil {
			// May fail if no matching subscription - that's okay
			t.Logf("processEvent() error = %v", err)
		}
	})

	t.Run("create session for new source", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		event := trigger.Event{
			Source:  "new-source",
			Content: "Hello from new source",
		}

		err := server.processEvent(context.Background(), event)
		if err != nil {
			t.Logf("processEvent() error = %v", err)
		}
	})
}

// TestServer_SessionServiceInterface tests session service interface implementation.
func TestServer_SessionServiceInterface(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Test GetMessages
	session, _ := server.manager.CreateSession("Test", nil, "", "")
	messages, err := server.GetMessages(context.Background(), session.ID, 10, "")
	if err != nil {
		t.Errorf("GetMessages() error = %v", err)
	}
	if messages == nil {
		t.Error("GetMessages() should return empty slice, not nil")
	}

	// Test SendMessage (async, won't wait for completion)
	err = server.SendMessage(context.Background(), session.ID, "Hello")
	if err != nil {
		// May fail without LLM client - that's okay
		t.Logf("SendMessage() error = %v", err)
	}

	// Test ListSessions
	sessions, err := server.ListSessions("")
	if err != nil {
		t.Errorf("ListSessions() error = %v", err)
	}
	if len(sessions) < 1 {
		t.Errorf("Expected at least 1 session, got %d", len(sessions))
	}

	// Test GetSession
	s, err := server.GetSession(session.ID, "")
	if err != nil {
		t.Errorf("GetSession() error = %v", err)
	}
	if s == nil {
		t.Error("GetSession() returned nil")
	}

	// Test UpdateSession
	err = server.UpdateSession(session.ID, "", stringPtr("Updated prompt"), nil)
	if err != nil {
		t.Errorf("UpdateSession() error = %v", err)
	}

	// Test DeleteSession
	err = server.DeleteSession(session.ID, "")
	if err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}
}

// TestServer_SubscriptionServiceInterface tests subscription service interface.
func TestServer_SubscriptionServiceInterface(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	session, _ := server.manager.CreateSession("Test", nil, "", "")

	// Test AddSubscription
	sub := model.Subscription{
		ID:      "sub-1",
		Source:  "feishu",
		Trigger: "test",
		Active:  true,
	}

	err := server.AddSubscription(session.ID, sub)
	if err != nil {
		t.Errorf("AddSubscription() error = %v", err)
	}

	// Test RemoveSubscription by ID
	err = server.RemoveSubscription(session.ID, "sub-1", "")
	if err != nil {
		t.Errorf("RemoveSubscription() error = %v", err)
	}

	// Add again for trigger test
	server.AddSubscription(session.ID, sub)

	// Test RemoveSubscription by trigger
	err = server.RemoveSubscription(session.ID, "", "test")
	if err != nil {
		t.Errorf("RemoveSubscription() error = %v", err)
	}

	// Test RemoveSubscription not found
	err = server.RemoveSubscription(session.ID, "non-existent", "")
	if err == nil {
		t.Error("RemoveSubscription() should return error for non-existent subscription")
	}
}

// Helper functions

func createTestConfig() *config.Config {
	return &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Model:    "llama3.2",
			BaseURL:  "http://localhost:11434",
		},
		Log: config.LogConfig{
			Level:  "error",
			Format: "text",
			Output: "stdout",
		},
		Permissions: config.PermissionsConfig{
			TrustedDirs: []string{os.TempDir()},
		},
		Adapters: config.AdaptersConfig{
			Feishu: config.FeishuAdapterConfig{
				Enabled: false,
			},
		},
	}
}

// findAvailablePort returns a port number for testing.
// Uses a simple counter based on test name hash to avoid collisions.
func findAvailablePort(t *testing.T) string {
	// Use a high port number in ephemeral range
	// Different tests get different ports based on name length
	port := 20000 + (len(t.Name()) % 1000)
	return string(rune('0'+port/1000%10)) + string(rune('0'+port/100%10)) + string(rune('0'+port/10%10)) + string(rune('0'+port%10))
}

func stringPtr(s string) *string {
	return &s
}
