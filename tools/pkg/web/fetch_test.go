// Package web provides web-related tools.
package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tools "github.com/oneliang/aura/tools/pkg"
)

// Test FetchTool
func TestFetchToolName(t *testing.T) {
	tool := NewFetchTool()
	name := tool.Name()
	if name != "web_fetch" {
		t.Errorf("Name() = %v, want 'web_fetch'", name)
	}
}

func TestFetchToolDescription(t *testing.T) {
	tool := NewFetchTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !contains(desc, "url") {
		t.Error("Description() should mention 'url' parameter")
	}
	if !contains(desc, "max_length") {
		t.Error("Description() should mention 'max_length' parameter")
	}
}

func TestFetchToolExecute(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx := context.Background()

	params := map[string]any{"url": server.URL}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result.Content != "Hello, World!" {
		t.Errorf("Execute() result = %v, want 'Hello, World!'", result.Content)
	}
}

func TestFetchToolExecuteWithMaxLength(t *testing.T) {
	longContent := "This is a long content. "
	for i := 0; i < 1000; i++ {
		longContent += "More content. "
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(longContent))
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx := context.Background()

	params := map[string]any{
		"url":        server.URL,
		"max_length": 50,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !contains(result.Content, "(truncated)") {
		t.Errorf("Result should be truncated, got: %s", result.Content)
	}
}

func TestFetchToolExecuteMissingURL(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Execute() should error when url is missing")
	}
}

func TestFetchToolExecuteInvalidURLType(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.Background()

	params := map[string]any{"url": 123}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when url is not a string")
	}
}

func TestFetchToolExecuteInvalidMaxLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx := context.Background()

	// max_length as string should be ignored (only float64 is valid)
	params := map[string]any{
		"url":        server.URL,
		"max_length": "invalid",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	// Should use default max_length
	if contains(result.Content, "(truncated)") {
		t.Error("Result should not be truncated with invalid max_length")
	}
}

func TestFetchToolExecuteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx := context.Background()

	params := map[string]any{"url": server.URL}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Errorf("Execute() status = %v, want error status", result.Status)
	}
	if !contains(result.Error, "HTTP error") {
		t.Errorf("Error should mention HTTP error, got: %v", result.Error)
	}
}

func TestFetchToolExecuteWithCustomTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Delayed response"))
	}))
	defer server.Close()

	tool := NewFetchTool(WithFetchTimeout(500 * time.Millisecond))
	ctx := context.Background()

	params := map[string]any{"url": server.URL}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result.Content != "Delayed response" {
		t.Errorf("Execute() result = %v, want 'Delayed response'", result.Content)
	}
}

func TestFetchToolExecuteCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tool := NewFetchTool()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	params := map[string]any{"url": server.URL}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Errorf("Execute() should error with cancelled context, status = %v", result.Status)
	}
}

func TestFetchToolExecuteConnectionError(t *testing.T) {
	tool := NewFetchTool()
	ctx := context.Background()

	// Use an invalid URL to trigger connection error
	params := map[string]any{"url": "http://invalid-host-that-does-not-exist-12345.com"}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Errorf("Execute() should error on connection failure, status = %v", result.Status)
	}
}

// Test with custom timeout
func TestFetchToolWithTimeoutOption(t *testing.T) {
	tool := NewFetchTool(WithFetchTimeout(5 * time.Second))
	if tool.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", tool.timeout)
	}
	if tool.client.Timeout != 5*time.Second {
		t.Errorf("client.Timeout = %v, want 5s", tool.client.Timeout)
	}
}

// Test SearchTool
func TestSearchToolName(t *testing.T) {
	tool := NewSearchTool()
	name := tool.Name()
	if name != "web_search" {
		t.Errorf("Name() = %v, want 'web_search'", name)
	}
}

func TestSearchToolDescription(t *testing.T) {
	tool := NewSearchTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !contains(desc, "query") {
		t.Error("Description() should mention 'query' parameter")
	}
}

func TestSearchToolExecuteMissingQuery(t *testing.T) {
	tool := NewSearchTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Execute() should error when query is missing")
	}
}

func TestSearchToolExecuteInvalidQueryType(t *testing.T) {
	tool := NewSearchTool()
	ctx := context.Background()

	params := map[string]any{"query": 123}
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("Execute() should error when query is not a string")
	}
}

func TestSearchToolExecuteCancelledContext(t *testing.T) {
	tool := NewSearchTool()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	params := map[string]any{"query": "test"}
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}
	if result.Status != tools.ToolStatusError {
		t.Errorf("Execute() should error with cancelled context, status = %v", result.Status)
	}
}

// Test stripHTML
func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple tag", "<p>Hello</p>", "Hello"},
		{"multiple tags", "<div><p>Hello</p> <b>World</b></div>", "Hello World"},
		{"with entities", "<p>Hello&nbsp;World</p>", "Hello World"},
		{"with ampersand", "<p>A&amp;B</p>", "A&B"},
		{"no tags", "Plain text", "Plain text"},
		{"empty", "", ""},
		{"only tags", "<div></div>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTML(tt.input)
			if result != tt.expected {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStripHTMLMultipleSpaces(t *testing.T) {
	input := "Hello    World"
	result := stripHTML(input)
	// Should collapse multiple spaces to single space
	if !contains(result, "Hello World") || contains(result, "  ") {
		t.Errorf("stripHTML should collapse multiple spaces, got: %q", result)
	}
}

// Test parseSearchResults
func TestParseSearchResultsEmpty(t *testing.T) {
	html := "<html><body>No results</body></html>"
	result := parseSearchResults(html)

	if result == "No results found." {
		// Expected behavior for empty results
	} else if !contains(result, "Search results") {
		t.Errorf("parseSearchResults should return search results header, got: %s", result)
	}
}

func TestParseSearchResults(t *testing.T) {
	// Note: parseSearchResults looks for <a class="result__a"> tags
	// and extracts content between > and </a>
	html := `<a class="result__a">Title 1</a>
<a class="result__a">Title 2</a>
<a class="result__a">Title 3</a>`

	result := parseSearchResults(html)

	if !contains(result, "Search results") {
		t.Error("parseSearchResults should contain header")
	}
	if !contains(result, "1. Title 1") {
		t.Error("parseSearchResults should contain '1. Title 1'")
	}
	if !contains(result, "2. Title 2") {
		t.Error("parseSearchResults should contain '2. Title 2'")
	}
	if !contains(result, "3. Title 3") {
		t.Error("parseSearchResults should contain '3. Title 3'")
	}
}

func TestParseSearchResultsLimitsTo5(t *testing.T) {
	// Note: parseSearchResults looks for <a class="result__a"> tags
	html := `<a class="result__a">Title 1</a>
<a class="result__a">Title 2</a>
<a class="result__a">Title 3</a>
<a class="result__a">Title 4</a>
<a class="result__a">Title 5</a>
<a class="result__a">Title 6</a>
<a class="result__a">Title 7</a>`

	result := parseSearchResults(html)

	// Count number of results
	count := 0
	for i := 1; i <= 10; i++ {
		if contains(result, string(rune('0'+i))+".") {
			count++
		}
	}

	// Should have at most 5 results
	if count > 5 {
		t.Errorf("parseSearchResults should limit to 5 results, got %d", count)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
