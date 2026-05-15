package runtime

import (
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
)

func TestDefaultRuntimeConfig(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	if cfg == nil {
		t.Fatal("DefaultRuntimeConfig() returned nil")
	}

	if cfg.Config == nil {
		t.Fatal("DefaultRuntimeConfig() Config is nil")
	}

	if cfg.Memory.MaxContext != 50 {
		t.Errorf("DefaultRuntimeConfig() Memory.MaxContext = %d, want 50", cfg.Memory.MaxContext)
	}
}

func TestFromConfig(t *testing.T) {
	srcCfg := &config.Config{
		LLM: config.LLMConfig{
			Provider:       "ollama",
			BaseURL:        "http://localhost:11434",
			Model:          "qwen3:8b",
			EmbeddingModel: "nomic-embed-text",
		},
		Agent: config.AgentConfig{
			PlanningMode: "explicit",
		},
		Tools: config.ToolsConfig{
			Enabled: []string{"file_read", "file_write"},
		},
		Permissions: config.PermissionsConfig{
			DefaultLevel: "ask",
		},
		Skills: config.SkillsConfig{
			Enabled:     true,
			Directories: []string{"/tmp/skills"},
		},
		SSH: config.SSHConfig{
			Servers: []config.SSHServerConfig{
				{
					Name: "test-server",
					Host: "test.example.com",
					Port: 22,
					User: "test",
				},
			},
		},
		Memory: config.MemoryConfig{
			Type:       "sqlite",
			MaxContext: 100,
		},
	}

	runtimeCfg := FromConfig(srcCfg)

	if runtimeCfg == nil {
		t.Fatal("FromConfig() returned nil")
	}

	if runtimeCfg.Config == nil {
		t.Fatal("FromConfig() Config is nil")
	}

	// Verify values are copied correctly
	if runtimeCfg.LLM.Provider != "ollama" {
		t.Errorf("FromConfig() LLM.Provider = %q, want %q", runtimeCfg.LLM.Provider, "ollama")
	}

	if runtimeCfg.Memory.MaxContext != 100 {
		t.Errorf("FromConfig() Memory.MaxContext = %d, want 100", runtimeCfg.Memory.MaxContext)
	}
}

func TestFromConfig_EmptyConfig(t *testing.T) {
	srcCfg := &config.Config{}

	runtimeCfg := FromConfig(srcCfg)

	if runtimeCfg == nil {
		t.Fatal("FromConfig() with empty config returned nil")
	}

	// Config should reference the same object, fields accessible
	if runtimeCfg.Config == nil {
		t.Error("FromConfig() Config is nil")
	}
}
