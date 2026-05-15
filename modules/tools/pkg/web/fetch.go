// Package web provides web-related tools.
package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/httpclient"
	tools "github.com/oneliang/aura/tools/pkg"
)

// FetchTool fetches content from a URL.
type FetchTool struct {
	client  *http.Client
	timeout time.Duration
}

// FetchOption is a configuration option for FetchTool.
type FetchOption func(*FetchTool)

// WithFetchTimeout sets the timeout for HTTP requests.
func WithFetchTimeout(d time.Duration) FetchOption {
	return func(t *FetchTool) {
		t.timeout = d
		t.client.Timeout = d
	}
}

// WithHTTPClient injects an external HTTP client.
func WithHTTPClient(client *http.Client) FetchOption {
	return func(t *FetchTool) {
		t.client = client
	}
}

// NewFetchTool creates a new fetch tool.
func NewFetchTool(opts ...FetchOption) *FetchTool {
	t := &FetchTool{
		timeout: constants.DefaultWebTimeout,
		client:  httpclient.DefaultWebClient(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *FetchTool) Name() string {
	return constants.ToolWebFetch
}

// Description returns the tool description.
func (t *FetchTool) Description() string {
	return "Fetch content from a URL. Parameters: url (string, required), max_length (number, optional, default 10000)"
}

// Execute fetches content from a URL.
func (t *FetchTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url parameter is required")
	}

	maxLength := 10000
	if ml, ok := params["max_length"].(float64); ok {
		maxLength = int(ml)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to create request: %v", err),
			Data: map[string]any{
				"url":   url,
				"error": err.Error(),
			},
		}, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AuraAgent/1.0)")

	resp, err := t.client.Do(req)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to fetch URL: %v", err),
			Data: map[string]any{
				"url":   url,
				"error": err.Error(),
			},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("HTTP error: %d %s", resp.StatusCode, resp.Status),
			Data: map[string]any{
				"url":        url,
				"status":     resp.StatusCode,
				"statusText": resp.Status,
			},
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to read response: %v", err),
			Data: map[string]any{
				"url":   url,
				"error": err.Error(),
			},
		}, nil
	}

	content := string(body)

	// Truncate if too long
	if len(content) > maxLength {
		content = content[:maxLength] + "\n... (truncated)"
	}

	// Extract text content (basic HTML stripping)
	content = stripHTML(content)

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: content,
	}, nil
}

// SearchTool performs a web search (using DuckDuckGo HTML version).
type SearchTool struct {
	client *http.Client
}

// SearchOption is a configuration option for SearchTool.
type SearchOption func(*SearchTool)

// WithSearchHTTPClient injects an external HTTP client for SearchTool.
func WithSearchHTTPClient(client *http.Client) SearchOption {
	return func(t *SearchTool) {
		t.client = client
	}
}

// NewSearchTool creates a new search tool.
func NewSearchTool(opts ...SearchOption) *SearchTool {
	t := &SearchTool{
		client: httpclient.DefaultWebClient(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *SearchTool) Name() string {
	return constants.ToolWebSearch
}

// Description returns the tool description.
func (t *SearchTool) Description() string {
	return "Search the web for information. Parameters: query (string, required)"
}

// Execute performs a web search.
func (t *SearchTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	// Use DuckDuckGo HTML version for simple search
	url := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to create request: %v", err),
			Data: map[string]any{
				"query": query,
				"error": err.Error(),
			},
		}, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AuraAgent/1.0)")

	resp, err := t.client.Do(req)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to search: %v", err),
			Data: map[string]any{
				"query": query,
				"error": err.Error(),
			},
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.ToolResult{
			Status: tools.ToolStatusError,
			Error:  fmt.Sprintf("failed to read response: %v", err),
			Data: map[string]any{
				"query": query,
				"error": err.Error(),
			},
		}, nil
	}

	// Extract search results (basic parsing)
	results := parseSearchResults(string(body))

	return &tools.ToolResult{
		Status:  tools.ToolStatusSuccess,
		Content: results,
	}, nil
}

func stripHTML(s string) string {
	// Remove script and style blocks first (including their content)
	s = removeBlock(s, "script")
	s = removeBlock(s, "style")

	// Basic HTML tag stripping
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	// Clean up HTML entities
	content := result.String()
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&#39;", "'")
	content = strings.ReplaceAll(content, "&apos;", "'")

	// Remove multiple spaces and newlines
	content = strings.Join(strings.Fields(content), " ")

	return strings.TrimSpace(content)
}

// removeBlock strips all content between the specified HTML tags (e.g., <script>...</script>).
func removeBlock(s, tag string) string {
	opening := "<" + tag
	closing := "</" + tag + ">"
	for {
		start := strings.Index(strings.ToLower(s), opening)
		if start == -1 {
			break
		}
		end := strings.Index(strings.ToLower(s[start:]), closing)
		if end == -1 {
			s = s[:start]
			break
		}
		s = s[:start] + s[start+end+len(closing):]
	}
	return s
}

func parseSearchResults(html string) string {
	var results strings.Builder
	results.WriteString("Search results:\n\n")

	// Simple extraction of links and snippets
	lines := strings.Split(html, "\n")
	count := 0

	for _, line := range lines {
		if strings.Contains(line, "class=\"result__a\"") {
			// Extract title
			start := strings.Index(line, ">")
			end := strings.Index(line, "</a>")
			if start != -1 && end != -1 && start < end {
				title := stripHTML(line[start+1 : end])
				if title != "" {
					count++
					results.WriteString(fmt.Sprintf("%d. %s\n", count, title))
				}
			}
		}

		if count >= 5 {
			break
		}
	}

	if count == 0 {
		return "No results found."
	}

	return results.String()
}
