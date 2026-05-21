package cli

import (
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
)

func TestInitSessionManager(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Model:    "qwen3:8b",
		},
	}

	sessionMgr, sessionID, err := initSessionManager(tmpDir, cfg, "test-user")
	if err != nil {
		t.Fatalf("initSessionManager() error = %v", err)
	}

	if sessionMgr == nil {
		t.Error("initSessionManager() returned nil session manager")
	}

	if sessionID == "" {
		t.Error("initSessionManager() returned empty session ID")
	}
}

func TestInitSessionManagerForTUI(t *testing.T) {
	// This test checks if the function runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("initSessionManagerForTUI() panicked: %v", r)
		}
	}()

	_, _ = initSessionManagerForTUI()
}
