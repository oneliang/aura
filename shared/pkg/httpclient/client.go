// Package httpclient provides HTTP client utilities with connection pooling.
package httpclient

import (
	"net/http"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
)

// NewClient creates a new HTTP client with connection pooling.
// 调用方负责管理单例，这里只提供创建能力。
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
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
