package location

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	tools "github.com/oneliang/aura/tools/pkg"
)

func TestLocationTool_Name(t *testing.T) {
	t.Parallel()
	tool := NewLocationTool()
	if tool.Name() != "location" {
		t.Errorf("expected name 'location', got '%s'", tool.Name())
	}
}

func TestLocationTool_Description(t *testing.T) {
	t.Parallel()
	tool := NewLocationTool()
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestLocationTool_Execute_ConfigOverride(t *testing.T) {
	t.Parallel()
	tool := NewLocationTool(
		WithConfig(LocationConfig{
			FixedCity:    "Beijing",
			FixedCountry: "China",
		}),
	)

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "City: Beijing") {
		t.Errorf("expected 'City: Beijing' in result, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Country: China") {
		t.Errorf("expected 'Country: China' in result, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Source: config") {
		t.Errorf("expected 'Source: config' in result, got: %s", result.Content)
	}
}

func TestLocationTool_Execute_AutoDetectDisabled(t *testing.T) {
	t.Parallel()
	tool := NewLocationTool(
		WithConfig(LocationConfig{
			AutoDetect: false,
		}),
	)

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Fatal("expected error status when auto-detect is disabled")
	}
	if !strings.Contains(result.Error, "auto-detection is disabled") {
		t.Errorf("expected 'auto-detection is disabled' error, got: %s", result.Error)
	}
}

func TestLocationTool_Execute_ContextCancel(t *testing.T) {
	t.Parallel()
	tool := NewLocationTool(
		WithHTTPClient(&http.Client{
			Timeout: time.Hour, // Long timeout so context cancels first
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := tool.Execute(ctx, nil)
	if err != nil {
		t.Fatal("unexpected Go error:", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Fatal("expected error status on cancelled context")
	}
}

func TestLocationTool_CacheConfigOverride(t *testing.T) {
	t.Parallel()
	tool := NewLocationTool(
		WithConfig(LocationConfig{
			FixedCity:    "Shanghai",
			FixedCountry: "China",
		}),
	)

	// Call twice, both should return config result without HTTP request
	for i := 0; i < 2; i++ {
		result, err := tool.Execute(context.Background(), nil)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i+1, err)
		}
		if !strings.Contains(result.Content, "City: Shanghai") {
			t.Errorf("call %d: expected 'City: Shanghai', got: %s", i+1, result.Content)
		}
	}
}

func TestLocationTool_Options(t *testing.T) {
	t.Parallel()
	customClient := &http.Client{Timeout: 5 * time.Second}
	tool := NewLocationTool(
		WithHTTPClient(customClient),
		WithConfig(LocationConfig{AutoDetect: true}),
	)

	if tool.client != customClient {
		t.Error("HTTP client should be set to custom client")
	}
}

func TestLocationTool_Cache_Expiry(t *testing.T) {
	t.Parallel()
	tool := NewLocationTool()

	// Manually set an expired cache entry
	tool.mu.Lock()
	tool.cache = &LocationData{
		City:   "StaleCity",
		Source: "ipinfo",
	}
	tool.expiry = time.Now().Add(-1 * time.Hour) // Expired
	tool.mu.Unlock()

	// With auto-detect disabled, it should NOT use the expired cache
	// but should return error instead
	tool2 := NewLocationTool(
		WithConfig(LocationConfig{AutoDetect: false}),
	)
	// Copy the expired cache to tool2
	tool2.mu.Lock()
	tool2.cache = tool.cache
	tool2.expiry = tool.expiry
	tool2.mu.Unlock()

	result, err := tool2.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal("unexpected Go error:", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Fatal("expected error status when cache expired and auto-detect disabled")
	}
}

// Test with mock HTTP server for IP API fallback
func TestLocationTool_IPAPIFallback(t *testing.T) {
	t.Parallel()

	// Create a mock server that always fails for primary API
	failClient := &http.Client{
		Transport: &failingTransport{failPrimary: true},
	}

	tool := NewLocationTool(
		WithHTTPClient(failClient),
	)

	// This will fail both APIs, so we expect an error
	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal("unexpected Go error:", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Fatal("expected error status when both APIs fail")
	}
}

// failingTransport is a mock RoundTripper for testing.
type failingTransport struct {
	failPrimary bool
}

func (t *failingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("network unreachable")
}
