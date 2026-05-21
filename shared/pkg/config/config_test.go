// Package config provides tests for configuration management.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfig tests default configuration creation.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.LLM.Provider != "ollama" {
		t.Errorf("LLM.Provider = %q, want %q", cfg.LLM.Provider, "ollama")
	}
	if cfg.LLM.BaseURL != "http://localhost:11434" {
		t.Errorf("LLM.BaseURL = %q, want %q", cfg.LLM.BaseURL, "http://localhost:11434")
	}
	if cfg.LLM.Model != "qwen3:8b" {
		t.Errorf("LLM.Model = %q, want %q", cfg.LLM.Model, "qwen3:8b")
	}
	if cfg.Memory.MaxContext != 50 {
		t.Errorf("Memory.MaxContext = %d, want %d", cfg.Memory.MaxContext, 50)
	}
	if cfg.Permissions.DefaultLevel != "ask" {
		t.Errorf("Permissions.DefaultLevel = %q, want %q", cfg.Permissions.DefaultLevel, "ask")
	}
}

// TestDefaultConfigMemoryType tests default memory type.
func TestDefaultConfigMemoryType(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Memory.Type != "sqlite" {
		t.Errorf("Memory.Type = %q, want %q", cfg.Memory.Type, "sqlite")
	}
}

// TestDefaultConfigSkillsEnabled tests default skills setting.
func TestDefaultConfigSkillsEnabled(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Skills.Enabled {
		t.Error("Skills.Enabled should be true")
	}
}

// TestDefaultConfigLogConfig tests default log configuration.
func TestDefaultConfigLogConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want %q", cfg.Log.Format, "text")
	}
}

// TestDefaultConfigPermissions tests default permissions.
func TestDefaultConfigPermissions(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Permissions.Tools["file_read"] != "allow" {
		t.Errorf("Tools[file_read] = %q, want %q", cfg.Permissions.Tools["file_read"], "allow")
	}
	if cfg.Permissions.Tools["bash"] != "ask" {
		t.Errorf("Tools[bash] = %q, want %q", cfg.Permissions.Tools["bash"], "ask")
	}
}

// TestDefaultConfigShellRestrictions tests shell command restrictions.
func TestDefaultConfigShellRestrictions(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.Permissions.ShellRestrictions.DeniedCommands) == 0 {
		t.Error("ShellRestrictions.DeniedCommands should not be empty")
	}

	// Check for some expected denied commands
	denied := cfg.Permissions.ShellRestrictions.DeniedCommands
	foundRmRf := false
	for _, cmd := range denied {
		if cmd == "rm -rf /" {
			foundRmRf = true
			break
		}
	}
	if !foundRmRf {
		t.Error("Should deny 'rm -rf /' command")
	}
}

// TestApplyEnvOverrides tests environment variable overrides.
func TestApplyEnvOverrides(t *testing.T) {
	// Save original values
	origProvider := os.Getenv(EnvLLMProvider)
	origBaseURL := os.Getenv(EnvLLMBaseURL)
	origModel := os.Getenv(EnvLLMModel)
	origAPIKey := os.Getenv(EnvLLMAPIKey)
	defer func() {
		os.Setenv(EnvLLMProvider, origProvider)
		os.Setenv(EnvLLMBaseURL, origBaseURL)
		os.Setenv(EnvLLMModel, origModel)
		os.Setenv(EnvLLMAPIKey, origAPIKey)
	}()

	// Set test values
	os.Setenv(EnvLLMProvider, "openai")
	os.Setenv(EnvLLMBaseURL, "https://test.openai.com")
	os.Setenv(EnvLLMModel, "gpt-4-test")
	os.Setenv(EnvLLMAPIKey, "test-key")

	cfg := DefaultConfig()
	applyEnvOverrides(cfg)

	if cfg.LLM.Provider != "openai" {
		t.Errorf("LLM.Provider = %q, want %q", cfg.LLM.Provider, "openai")
	}
	if cfg.LLM.BaseURL != "https://test.openai.com" {
		t.Errorf("LLM.BaseURL = %q, want %q", cfg.LLM.BaseURL, "https://test.openai.com")
	}
	if cfg.LLM.Model != "gpt-4-test" {
		t.Errorf("LLM.Model = %q, want %q", cfg.LLM.Model, "gpt-4-test")
	}
	if cfg.LLM.APIKey != "test-key" {
		t.Errorf("LLM.APIKey = %q, want %q", cfg.LLM.APIKey, "test-key")
	}
}

// TestApplyEnvOverridesPartial tests partial environment overrides.
func TestApplyEnvOverridesPartial(t *testing.T) {
	// Save original values
	origProvider := os.Getenv(EnvLLMProvider)
	origBaseURL := os.Getenv(EnvLLMBaseURL)
	origModel := os.Getenv(EnvLLMModel)
	origAPIKey := os.Getenv(EnvLLMAPIKey)
	defer func() {
		os.Setenv(EnvLLMProvider, origProvider)
		os.Setenv(EnvLLMBaseURL, origBaseURL)
		os.Setenv(EnvLLMModel, origModel)
		os.Setenv(EnvLLMAPIKey, origAPIKey)
	}()

	// Clear all first
	os.Unsetenv(EnvLLMProvider)
	os.Unsetenv(EnvLLMBaseURL)
	os.Unsetenv(EnvLLMModel)
	os.Unsetenv(EnvLLMAPIKey)

	// Only set provider
	os.Setenv(EnvLLMProvider, "anthropic")

	cfg := DefaultConfig()
	applyEnvOverrides(cfg)

	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("LLM.Provider = %q, want %q", cfg.LLM.Provider, "anthropic")
	}
	// Other values should remain default
	if cfg.LLM.BaseURL != "http://localhost:11434" {
		t.Errorf("LLM.BaseURL should remain default, got %q", cfg.LLM.BaseURL)
	}
}

// TestLoad_NonExistentFile tests loading non-existent config file.
func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")

	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}
	// Should return default config
	if cfg.LLM.Provider != "ollama" {
		t.Errorf("Should return default config, Provider = %q", cfg.LLM.Provider)
	}
}

// TestLoad_EmptyPath tests loading with empty path.
func TestLoad_EmptyPath(t *testing.T) {
	cfg, err := Load("")

	if err != nil {
		t.Fatalf("Load(\"\") error = %v, want nil", err)
	}
	if cfg == nil {
		t.Fatal("Load(\"\") returned nil")
	}
}

// TestSave tests saving configuration.
func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.LLM.Model = "test-model"

	err := cfg.Save(tmpFile)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Config file should be created")
	}

	// Load it back and verify
	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.LLM.Model != "test-model" {
		t.Errorf("Loaded Model = %q, want %q", loaded.LLM.Model, "test-model")
	}
}

// TestSave_EmptyPath tests saving with empty path.
func TestSave_EmptyPath(t *testing.T) {
	// This test would create file in home directory, so we just verify it doesn't panic
	cfg := DefaultConfig()

	// Create a temp dir to use as fake home
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	err := cfg.Save("")
	// May fail due to directory creation, but should not panic
	if err != nil {
		t.Logf("Save(\"\") error (expected in test env): %v", err)
	}
}

// TestValidate tests configuration validation.
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "ollama",
					BaseURL:  "http://localhost:11434",
					Model:    "test",
				},
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			cfg: &Config{
				LLM: LLMConfig{
					BaseURL: "http://localhost:11434",
					Model:   "test",
				},
			},
			wantErr: true,
		},
		{
			name: "missing base URL",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "ollama",
					Model:    "test",
				},
			},
			wantErr: true,
		},
		{
			name: "missing model",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "ollama",
					BaseURL:  "http://localhost:11434",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "invalid",
					BaseURL:  "http://localhost:11434",
					Model:    "test",
				},
			},
			wantErr: true,
		},
		{
			name: "valid openai provider",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "openai",
					BaseURL:  "https://api.openai.com",
					Model:    "gpt-4",
				},
			},
			wantErr: false,
		},
		{
			name: "valid anthropic provider",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "anthropic",
					BaseURL:  "https://api.anthropic.com",
					Model:    "claude-3",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "ollama",
					BaseURL:  "http://localhost:11434",
					Model:    "test",
				},
				Log: LogConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "valid empty log level",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "ollama",
					BaseURL:  "http://localhost:11434",
					Model:    "test",
				},
				Log: LogConfig{
					Level: "",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfigStructTags tests that config struct has proper tags.
func TestConfigStructTags(t *testing.T) {
	cfg := &Config{}

	// Just verify struct fields are accessible
	_ = cfg.LLM
	_ = cfg.Memory
	_ = cfg.Tools
	_ = cfg.Log
	_ = cfg.SSH
	_ = cfg.Permissions
	_ = cfg.Skills
}

// TestSSHServerConfig tests SSH server config struct.
func TestSSHServerConfig(t *testing.T) {
	s := SSHServerConfig{
		Name:     "test",
		Host:     "example.com",
		Port:     22,
		User:     "testuser",
		KeyPath:  "/path/to/key",
		Password: "secret",
	}

	if s.Name != "test" {
		t.Errorf("Name = %q, want %q", s.Name, "test")
	}
	if s.Host != "example.com" {
		t.Errorf("Host = %q, want %q", s.Host, "example.com")
	}
	if s.Port != 22 {
		t.Errorf("Port = %d, want %d", s.Port, 22)
	}
	if s.User != "testuser" {
		t.Errorf("User = %q, want %q", s.User, "testuser")
	}
}

// TestCommandRestrictions tests command restrictions struct.
func TestCommandRestrictions(t *testing.T) {
	cr := CommandRestrictions{
		AllowedCommands: []string{"ls", "cat"},
		DeniedCommands:  []string{"rm"},
	}

	if len(cr.AllowedCommands) != 2 {
		t.Errorf("AllowedCommands len = %d, want 2", len(cr.AllowedCommands))
	}
	if len(cr.DeniedCommands) != 1 {
		t.Errorf("DeniedCommands len = %d, want 1", len(cr.DeniedCommands))
	}
}

// TestSSHRestrictions tests SSH restrictions struct.
func TestSSHRestrictions(t *testing.T) {
	sr := SSHRestrictions{
		AllowedHosts:    []string{"trusted.com"},
		DeniedHosts:     []string{"untrusted.com"},
		AllowedCommands: []string{"uptime"},
		DeniedCommands:  []string{"rm"},
	}

	if len(sr.AllowedHosts) != 1 {
		t.Errorf("AllowedHosts len = %d, want 1", len(sr.AllowedHosts))
	}
	if len(sr.DeniedHosts) != 1 {
		t.Errorf("DeniedHosts len = %d, want 1", len(sr.DeniedHosts))
	}
}
