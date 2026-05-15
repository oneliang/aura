// Package commands provides tests for ConfigExecutor.
package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewConfigExecutor tests the NewConfigExecutor function.
func TestNewConfigExecutor(t *testing.T) {
	configPath := "/test/config/path.yaml"

	executor := NewConfigExecutor(configPath)

	if executor == nil {
		t.Fatal("NewConfigExecutor() returned nil")
	}
	if executor.configPath != configPath {
		t.Errorf("configPath = %q, want %q", executor.configPath, configPath)
	}
}

// TestConfigExecutor_ExecuteCommand tests the ExecuteCommand method.
func TestConfigExecutor_ExecuteCommand(t *testing.T) {
	// Create a temporary config file for testing
	tempDir, err := os.MkdirTemp("", "config-executor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.yaml")
	testConfig := `llm:
  provider: ollama
  model: qwen3:8b
`
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	executor := NewConfigExecutor(configPath)
	ctx := context.Background()

	tests := []struct {
		name    string
		cmd     string
		params  map[string]any
		wantErr bool
	}{
		{name: "show config", cmd: "show", params: nil, wantErr: false},
		{name: "show config path", cmd: "path", params: nil, wantErr: false},
		{name: "unknown command", cmd: "unknown", params: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteCommand(ctx, tt.cmd, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == "" {
				t.Error("ExecuteCommand() returned empty result")
			}
		})
	}
}

// TestConfigExecutor_showConfig tests the showConfig method.
func TestConfigExecutor_showConfig(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "config-executor-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "config.yaml")
		testConfig := "key: value\n"
		if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		executor := NewConfigExecutor(configPath)
		result, err := executor.showConfig()
		if err != nil {
			t.Fatalf("showConfig() error = %v", err)
		}
		if result == "" {
			t.Fatal("showConfig() returned empty result")
		}
		if result != testConfig {
			t.Errorf("showConfig() = %q, want %q", result, testConfig)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		executor := NewConfigExecutor("/non/existent/path.yaml")
		_, err := executor.showConfig()
		if err == nil {
			t.Error("showConfig() should return error for non-existent file")
		}
	})
}

// TestConfigExecutor_showConfigPath tests the showConfigPath method.
func TestConfigExecutor_showConfigPath(t *testing.T) {
	t.Run("explicit path", func(t *testing.T) {
		executor := NewConfigExecutor("/explicit/path.yaml")
		result := executor.showConfigPath()
		if result == "" {
			t.Fatal("showConfigPath() returned empty result")
		}
		if !strings.Contains(result, "Config file:") {
			t.Error("Result should contain 'Config file:'")
		}
	})

	t.Run("empty path", func(t *testing.T) {
		executor := NewConfigExecutor("")
		result := executor.showConfigPath()
		if result == "" {
			t.Fatal("showConfigPath() returned empty result")
		}
		if !strings.Contains(result, "Config file:") {
			t.Error("Result should contain 'Config file:'")
		}
	})
}

// TestLoadConfig tests the LoadConfig function.
func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "config-load-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "config.yaml")
		testConfig := `llm:
  provider: ollama
  model: qwen3:8b
memory:
  max_tokens: 8000
`
		if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if config == nil {
			t.Fatal("LoadConfig() returned nil config")
		}

		// Verify parsed content
		llm, ok := config["llm"].(map[string]any)
		if !ok {
			t.Fatal("llm section not found or wrong type")
		}
		if llm["provider"] != "ollama" {
			t.Errorf("llm.provider = %v, want 'ollama'", llm["provider"])
		}
		if llm["model"] != "qwen3:8b" {
			t.Errorf("llm.model = %v, want 'qwen3:8b'", llm["model"])
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := LoadConfig("/non/existent/config.yaml")
		if err == nil {
			t.Error("LoadConfig() should return error for non-existent file")
		}
	})
}

// TestSaveConfig tests the SaveConfig function.
func TestSaveConfig(t *testing.T) {
	t.Run("save new config", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "config-save-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "config.yaml")
		testConfig := map[string]any{
			"llm": map[string]any{
				"provider": "ollama",
				"model":    "qwen3:8b",
			},
		}

		err = SaveConfig(configPath, testConfig)
		if err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("Config file was not created")
		}

		// Read back and verify content
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read saved config: %v", err)
		}
		if len(data) == 0 {
			t.Error("Saved config is empty")
		}
	})

	t.Run("save creates directory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "config-save-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "subdir", "config.yaml")
		testConfig := map[string]any{"key": "value"}

		err = SaveConfig(configPath, testConfig)
		if err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		// Verify file was created in subdirectory
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("Config file was not created in subdirectory")
		}
	})
}

// TestLoadConfig_And_SaveConfig_RoundTrip tests round-trip config loading and saving.
func TestLoadConfig_And_SaveConfig_RoundTrip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-roundtrip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.yaml")

	// Original config
	original := map[string]any{
		"llm": map[string]any{
			"provider": "ollama",
			"model":    "test-model",
		},
		"memory": map[string]any{
			"max_tokens": 8000,
		},
	}

	// Save
	err = SaveConfig(configPath, original)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Load
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Compare high-level structure
	if loaded == nil {
		t.Fatal("Loaded config is nil")
	}

	llm, ok := loaded["llm"].(map[string]any)
	if !ok {
		t.Fatal("llm section not found")
	}
	if llm["provider"] != "ollama" {
		t.Errorf("llm.provider = %v, want 'ollama'", llm["provider"])
	}
	if llm["model"] != "test-model" {
		t.Errorf("llm.model = %v, want 'test-model'", llm["model"])
	}
}
