package config

import (
	"testing"
	"time"

	"github.com/oneliang/aura/mcp/pkg/constants"
)

func TestServerConfig_IsHTTP(t *testing.T) {
	tests := []struct {
		name string
		cfg  ServerConfig
		want bool
	}{
		{"empty", ServerConfig{}, false},
		{"type_stdio", ServerConfig{Type: "stdio"}, false},
		{"type_http", ServerConfig{Type: "http"}, true},
		{"url_set", ServerConfig{URL: "http://localhost:8080"}, true},
		{"type_stdio_with_url", ServerConfig{Type: "stdio", URL: "http://localhost:8080"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsHTTP(); got != tt.want {
				t.Errorf("IsHTTP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerConfig_GetTimeout(t *testing.T) {
	// Zero timeout should return default
	cfg := ServerConfig{}
	got := cfg.GetTimeout()
	if got != constants.DefaultTimeout {
		t.Errorf("GetTimeout() = %v, want %v", got, constants.DefaultTimeout)
	}

	// Negative timeout should return default
	cfg = ServerConfig{Timeout: -1 * time.Second}
	got = cfg.GetTimeout()
	if got != constants.DefaultTimeout {
		t.Errorf("GetTimeout() = %v, want %v", got, constants.DefaultTimeout)
	}

	// Positive timeout should return configured value
	cfg = ServerConfig{Timeout: 30 * time.Second}
	got = cfg.GetTimeout()
	if got != 30*time.Second {
		t.Errorf("GetTimeout() = %v, want %v", got, 30*time.Second)
	}
}
