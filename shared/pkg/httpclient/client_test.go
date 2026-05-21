package httpclient

import (
	"net/http"
	"testing"
	"time"
)

// TestNewClient tests NewClient function.
func TestNewClient(t *testing.T) {
	timeout := 30 * time.Second
	client := NewClient(timeout)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.Timeout)
	}
	if client.Transport == nil {
		t.Error("Transport should not be nil")
	}
}

// TestNewClient_TransportConfig tests that transport is properly configured.
func TestNewClient_TransportConfig(t *testing.T) {
	client := NewClient(10 * time.Second)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport should be *http.Transport")
	}

	if transport.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("Expected MaxIdleConnsPerHost 10, got %d", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout 90s, got %v", transport.IdleConnTimeout)
	}
}

// TestDefaultLLMClient tests DefaultLLMClient function.
func TestDefaultLLMClient(t *testing.T) {
	client := DefaultLLMClient()

	if client == nil {
		t.Fatal("DefaultLLMClient() returned nil")
	}
	if client.Timeout <= 0 {
		t.Errorf("Expected positive timeout, got %v", client.Timeout)
	}
}

// TestDefaultWebClient tests DefaultWebClient function.
func TestDefaultWebClient(t *testing.T) {
	client := DefaultWebClient()

	if client == nil {
		t.Fatal("DefaultWebClient() returned nil")
	}
	if client.Timeout <= 0 {
		t.Errorf("Expected positive timeout, got %v", client.Timeout)
	}
}

// TestNewClient_ZeroTimeout tests NewClient with zero timeout.
func TestNewClient_ZeroTimeout(t *testing.T) {
	client := NewClient(0)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	// Zero timeout means no timeout
	if client.Timeout != 0 {
		t.Errorf("Expected timeout 0, got %v", client.Timeout)
	}
}

// TestNewClient_DifferentTimeouts tests creating clients with different timeouts.
func TestNewClient_DifferentTimeouts(t *testing.T) {
	timeouts := []time.Duration{
		1 * time.Second,
		10 * time.Second,
		30 * time.Second,
		1 * time.Minute,
		5 * time.Minute,
	}

	for _, timeout := range timeouts {
		client := NewClient(timeout)
		if client.Timeout != timeout {
			t.Errorf("Expected timeout %v, got %v", timeout, client.Timeout)
		}
	}
}
