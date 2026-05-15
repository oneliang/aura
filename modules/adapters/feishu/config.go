package feishu

import (
	"fmt"

	"github.com/oneliang/aura/shared/pkg/config"
)

// Config holds the Feishu adapter configuration (WebSocket long connection mode).
type Config struct {
	Enabled                 bool   `yaml:"enabled"`
	AppID                   string `yaml:"app_id"`
	AppSecret               string `yaml:"app_secret"`
	EncryptKey              string `yaml:"encrypt_key"`
	VerificationToken       string `yaml:"verification_token"`
	WebhookPath             string `yaml:"webhook_path"`
	Port                    string `yaml:"port"`
	AsyncProcessing         bool   `yaml:"async_processing"`
	AutoReply               bool   `yaml:"auto_reply"`
	ShowProcessingIndicator bool   `yaml:"show_processing_indicator"`
	DataDir                 string `yaml:"data_dir"`
}

// DefaultConfig returns the default Feishu adapter configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:                 false,
		WebhookPath:             "/webhook/feishu",
		Port:                    "8080",
		AsyncProcessing:         true,
		AutoReply:               true,
		ShowProcessingIndicator: true,
		DataDir:                 "", // Will be set from global config
	}
}

// LoadConfig loads the Feishu adapter configuration from global config.
func LoadConfig(globalCfg *config.Config) (*Config, error) {
	if globalCfg == nil {
		return nil, fmt.Errorf("global config is nil")
	}

	// Use centralized configuration from shared/config
	feishuCfg := globalCfg.Adapters.Feishu

	cfg := &Config{
		Enabled:                 feishuCfg.Enabled,
		AppID:                   feishuCfg.AppID,
		AppSecret:               feishuCfg.AppSecret,
		EncryptKey:              feishuCfg.EncryptKey,
		VerificationToken:       feishuCfg.VerificationToken,
		WebhookPath:             feishuCfg.WebhookPath,
		Port:                    feishuCfg.Port,
		AsyncProcessing:         feishuCfg.AsyncProcessing,
		AutoReply:               feishuCfg.AutoReply,
		ShowProcessingIndicator: feishuCfg.ShowProcessingIndicator,
		DataDir:                 globalCfg.Adapters.DataDir,
	}

	// Validate required fields
	if cfg.Enabled {
		if cfg.AppID == "" || cfg.AppSecret == "" {
			return nil, fmt.Errorf("feishu adapter enabled but app_id or app_secret is missing")
		}
	}

	return cfg, nil
}

// Validate validates the Feishu adapter configuration.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.AppID == "" {
		return fmt.Errorf("app_id is required")
	}

	if c.AppSecret == "" {
		return fmt.Errorf("app_secret is required")
	}

	return nil
}
