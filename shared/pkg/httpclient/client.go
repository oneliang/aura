// Package httpclient provides HTTP client utilities with connection pooling.
package httpclient

import (
	"net/http"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
)

// NewClient creates a new HTTP client with connection pooling.
// The timeout parameter controls ResponseHeaderTimeout (TTFB): the max time to wait
// for response headers after sending the request. http.Client.Timeout is intentionally
// NOT set — it covers the entire request lifecycle including reading the response body,
// which kills active streaming connections. Use idle timeout in the stream reader instead.
// 调用方负责管理单例，这里只提供创建能力。
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: timeout,
		},
	}
}

// DefaultLLMClient creates a default HTTP client for LLM operations.
func DefaultLLMClient() *http.Client {
	return NewClient(constants.DefaultLLMTimeout)
}

// DefaultWebClient creates a default HTTP client for web tools.
func DefaultWebClient() *http.Client {
	return NewClient(constants.DefaultWebTimeout)
}
