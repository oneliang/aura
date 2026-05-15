package feishu

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
)

func TestLoadConfig_NilGlobalConfig(t *testing.T) {
	_, err := LoadConfig(nil)
	if err == nil {
		t.Error("Expected error for nil global config, got nil")
	}
}

func TestLoadConfig_WithFeishuConfig(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "feishu-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .aura directory
	configDir := filepath.Join(tmpDir, ".aura")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create .aura dir: %v", err)
	}

	// Create config file with feishu settings
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: qwen3:8b

adapters:
  enabled: true
  enabled_adapters:
    - feishu
  data_dir: /tmp/feishu-test-data
  feishu:
    enabled: true
    app_id: "test_app_id_from_config"
    app_secret: "test_secret_from_config"
    webhook_path: "/webhook/feishu"
    port: "8080"
    async_processing: false
    auto_reply: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	// Load global config
	globalCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load global config: %v", err)
	}

	// Load Feishu config
	cfg, err := LoadConfig(globalCfg)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.AppID != "test_app_id_from_config" {
		t.Errorf("Expected AppID 'test_app_id_from_config', got '%s'", cfg.AppID)
	}
	if cfg.AppSecret != "test_secret_from_config" {
		t.Errorf("Expected AppSecret 'test_secret_from_config', got '%s'", cfg.AppSecret)
	}
	if cfg.AsyncProcessing {
		t.Error("Expected AsyncProcessing to be false")
	}
	if cfg.AutoReply {
		t.Error("Expected AutoReply to be false")
	}
	if cfg.DataDir != "/tmp/feishu-test-data" {
		t.Errorf("Expected DataDir '/tmp/feishu-test-data', got '%s'", cfg.DataDir)
	}
}

func TestLoadConfig_WithEmptyFeishuSection(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "feishu-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .aura directory
	configDir := filepath.Join(tmpDir, ".aura")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create .aura dir: %v", err)
	}

	// Create config file with empty feishu section
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: qwen3:8b

adapters:
  enabled: false
  enabled_adapters: []
  data_dir: /tmp/feishu-test-data-empty
  feishu:
    enabled: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load global config
	globalCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load global config: %v", err)
	}

	// Load Feishu config
	cfg, err := LoadConfig(globalCfg)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should return config with defaults
	if cfg.Enabled {
		t.Error("Expected Enabled to be false")
	}
}

func TestLoadConfig_MissingCredentials(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "feishu-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .aura directory
	configDir := filepath.Join(tmpDir, ".aura")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create .aura dir: %v", err)
	}

	// Create config file with enabled but missing credentials
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: qwen3:8b

adapters:
  enabled: true
  enabled_adapters:
    - feishu
  data_dir: /tmp/feishu-test-data-empty
  feishu:
    enabled: true
    app_id: ""
    app_secret: ""
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load global config
	globalCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load global config: %v", err)
	}

	// Load Feishu config - should fail validation
	_, err = LoadConfig(globalCfg)
	if err == nil {
		t.Error("Expected error for missing credentials, got nil")
	}
}

func TestValidate_Disabled(t *testing.T) {
	cfg := &Config{
		Enabled: false,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should not error for disabled config, got: %v", err)
	}
}

func TestValidate_MissingAppID(t *testing.T) {
	cfg := &Config{
		Enabled:   true,
		AppSecret: "secret",
	}

	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for missing AppID, got nil")
	}
}

func TestValidate_MissingAppSecret(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		AppID:   "app123",
	}

	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for missing AppSecret, got nil")
	}
}

func TestValidate_Valid(t *testing.T) {
	cfg := &Config{
		Enabled:   true,
		AppID:     "app123",
		AppSecret: "secret456",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should not error for valid config, got: %v", err)
	}
}
