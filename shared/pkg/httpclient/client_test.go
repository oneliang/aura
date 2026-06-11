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
	// http.Client.Timeout should NOT be set (streaming-safe)
	if client.Timeout != 0 {
		t.Errorf("Expected client.Timeout 0, got %v", client.Timeout)
	}
	if client.Transport == nil {
		t.Error("Transport should not be nil")
	}
	// Timeout should be on Transport.ResponseHeaderTimeout (TTFB)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport should be *http.Transport")
	}
	if transport.ResponseHeaderTimeout != timeout {
		t.Errorf("Expected ResponseHeaderTimeout %v, got %v", timeout, transport.ResponseHeaderTimeout)
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
	if transport.ResponseHeaderTimeout != 10*time.Second {
		t.Errorf("Expected ResponseHeaderTimeout 10s, got %v", transport.ResponseHeaderTimeout)
	}
}

// TestDefaultLLMClient tests DefaultLLMClient function.
func TestDefaultLLMClient(t *testing.T) {
	client := DefaultLLMClient()

	if client == nil {
		t.Fatal("DefaultLLMClient() returned nil")
	}
	// http.Client.Timeout should NOT be set (streaming-safe)
	if client.Timeout != 0 {
		t.Errorf("Expected client.Timeout 0, got %v", client.Timeout)
	}
	// TTFB timeout should be set on Transport
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport should be *http.Transport")
	}
	if transport.ResponseHeaderTimeout <= 0 {
		t.Errorf("Expected positive ResponseHeaderTimeout, got %v", transport.ResponseHeaderTimeout)
	}
}

// TestDefaultWebClient tests DefaultWebClient function.
func TestDefaultWebClient(t *testing.T) {
	client := DefaultWebClient()

	if client == nil {
		t.Fatal("DefaultWebClient() returned nil")
	}
	// http.Client.Timeout should NOT be set (streaming-safe)
	if client.Timeout != 0 {
		t.Errorf("Expected client.Timeout 0, got %v", client.Timeout)
	}
}

// TestNewClient_ZeroTimeout tests NewClient with zero timeout.
func TestNewClient_ZeroTimeout(t *testing.T) {
	client := NewClient(0)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	// Zero timeout means no TTFB timeout either
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport should be *http.Transport")
	}
	if transport.ResponseHeaderTimeout != 0 {
		t.Errorf("Expected ResponseHeaderTimeout 0, got %v", transport.ResponseHeaderTimeout)
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
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatal("Transport should be *http.Transport")
		}
		if transport.ResponseHeaderTimeout != timeout {
			t.Errorf("Expected ResponseHeaderTimeout %v, got %v", timeout, transport.ResponseHeaderTimeout)
		}
	}
}
