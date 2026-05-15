// Package internal provides tests for http_helper.go.
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestBuildHTTPRequest tests BuildHTTPRequest function.
func TestBuildHTTPRequest(t *testing.T) {
	ctx := context.Background()
	body := []byte(`{"test": "data"}`)
	headers := map[string]string{
		"Authorization": "Bearer token",
		"Custom-Header": "value",
	}

	req, err := BuildHTTPRequest(ctx, "POST", "http://example.com/api", "application/json", body, headers)
	if err != nil {
		t.Fatalf("BuildHTTPRequest() error = %v", err)
	}
	if req == nil {
		t.Fatal("BuildHTTPRequest() returned nil")
	}
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.URL.String() != "http://example.com/api" {
		t.Errorf("URL = %q, want http://example.com/api", req.URL.String())
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("Authorization") != "Bearer token" {
		t.Errorf("Authorization = %q, want Bearer token", req.Header.Get("Authorization"))
	}
	if req.Header.Get("Custom-Header") != "value" {
		t.Errorf("Custom-Header = %q, want value", req.Header.Get("Custom-Header"))
	}
}

// TestBuildHTTPRequest_EmptyBody tests BuildHTTPRequest with empty body.
func TestBuildHTTPRequest_EmptyBody(t *testing.T) {
	ctx := context.Background()

	req, err := BuildHTTPRequest(ctx, "GET", "http://example.com/api", "", []byte{}, nil)
	if err != nil {
		t.Fatalf("BuildHTTPRequest() error = %v", err)
	}
	if req == nil {
		t.Fatal("BuildHTTPRequest() returned nil")
	}
	if req.Body != nil {
		t.Error("Request body should be nil for empty body")
	}
}

// TestBuildHTTPRequest_InvalidURL tests BuildHTTPRequest with invalid URL.
func TestBuildHTTPRequest_InvalidURL(t *testing.T) {
	ctx := context.Background()

	_, err := BuildHTTPRequest(ctx, "GET", "://invalid-url", "", []byte{}, nil)
	if err == nil {
		t.Error("BuildHTTPRequest() with invalid URL should return error")
	}
}

// MockHTTPClient is a mock HTTP client for testing.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// TestSendRequest tests SendRequest function.
func TestSendRequest(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("OK")),
			}, nil
		},
	}

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)

	resp, err := SendRequest(mockClient, req)
	if err != nil {
		t.Fatalf("SendRequest() error = %v", err)
	}
	if resp == nil {
		t.Fatal("SendRequest() returned nil")
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

// TestSendRequest_Error tests SendRequest with error.
func TestSendRequest_Error(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection failed")
		},
	}

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)

	_, err := SendRequest(mockClient, req)
	if err == nil {
		t.Error("SendRequest() should return error")
	}
}

// TestReadResponseBody tests ReadResponseBody function.
func TestReadResponseBody(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("test response body")),
	}

	body, err := ReadResponseBody(resp)
	if err != nil {
		t.Fatalf("ReadResponseBody() error = %v", err)
	}
	if string(body) != "test response body" {
		t.Errorf("Body = %q, want 'test response body'", string(body))
	}
}

// TestReadResponseBody_Error tests ReadResponseBody with error.
func TestReadResponseBody_Error(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(&errorReader{}),
	}

	_, err := ReadResponseBody(resp)
	if err == nil {
		t.Error("ReadResponseBody() should return error")
	}
}

// TestCheckStatusCode tests CheckStatusCode function.
func TestCheckStatusCode(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("OK")),
	}

	err := CheckStatusCode(resp, 200)
	if err != nil {
		t.Fatalf("CheckStatusCode() error = %v", err)
	}
}

// TestCheckStatusCode_WrongStatus tests CheckStatusCode with wrong status.
func TestCheckStatusCode_WrongStatus(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
	}

	err := CheckStatusCode(resp, 200)
	if err == nil {
		t.Error("CheckStatusCode() should return error for non-matching status")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("Error = %q, want to contain 'unexpected status 500'", err.Error())
	}
}

// TestCheckStatusWithAPIError tests CheckStatusWithAPIError function.
func TestCheckStatusWithAPIError(t *testing.T) {
	// Test with parseable error
	errorResp := map[string]interface{}{"error": map[string]interface{}{"message": "API error message"}}
	body, _ := json.Marshal(errorResp)

	resp := &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	parseError := func(b []byte) (string, error) {
		var resp map[string]interface{}
		if err := json.Unmarshal(b, &resp); err != nil {
			return "", err
		}
		if err, ok := resp["error"].(map[string]interface{}); ok {
			if msg, ok := err["message"].(string); ok {
				return msg, nil
			}
		}
		return "", errors.New("could not parse error")
	}

	err := CheckStatusWithAPIError(resp, "http://example.com/api", parseError)
	if err == nil {
		t.Error("CheckStatusWithAPIError() should return error")
	}
	if !strings.Contains(err.Error(), "API error message") {
		t.Errorf("Error = %q, want to contain 'API error message'", err.Error())
	}
}

// TestCheckStatusWithAPIError_Unparseable tests CheckStatusWithAPIError with unparseable error.
func TestCheckStatusWithAPIError_Unparseable(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader("Plain text error")),
	}

	parseError := func(b []byte) (string, error) {
		return "", errors.New("could not parse")
	}

	err := CheckStatusWithAPIError(resp, "http://example.com/api", parseError)
	if err == nil {
		t.Error("CheckStatusWithAPIError() should return error")
	}
	if !strings.Contains(err.Error(), "Plain text error") {
		t.Errorf("Error = %q, want to contain 'Plain text error'", err.Error())
	}
}

// TestStreamSSE tests StreamSSE function.
func TestStreamSSE(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("data: {\"chunk\": 1}\n\ndata: {\"chunk\": 2}\n\ndata: [DONE]\n")),
	}

	var chunks [][]byte
	handler := func(data []byte) error {
		chunks = append(chunks, data)
		return nil
	}

	err := StreamSSE(resp, handler)
	if err != nil {
		t.Fatalf("StreamSSE() error = %v", err)
	}
	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}
	if !bytes.Contains(chunks[0], []byte("chunk")) {
		t.Errorf("First chunk = %q, want to contain 'chunk'", string(chunks[0]))
	}
}

// TestStreamSSE_HandlerError tests StreamSSE with handler error.
func TestStreamSSE_HandlerError(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("data: {\"chunk\": 1}\n")),
	}

	handler := func(data []byte) error {
		return errors.New("handler error")
	}

	err := StreamSSE(resp, handler)
	if err == nil {
		t.Error("StreamSSE() should return handler error")
	}
}

// TestStreamSSE_EmptyLines tests StreamSSE skips empty lines.
func TestStreamSSE_EmptyLines(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("\n\n\ndata: valid\n\n")),
	}

	var chunks [][]byte
	handler := func(data []byte) error {
		chunks = append(chunks, data)
		return nil
	}

	err := StreamSSE(resp, handler)
	if err != nil {
		t.Fatalf("StreamSSE() error = %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
}

// TestStreamSSE_NonDataLines tests StreamSSE skips non-data lines.
func TestStreamSSE_NonDataLines(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("event: message\ndata: valid\ncomment: test\n")),
	}

	var chunks [][]byte
	handler := func(data []byte) error {
		chunks = append(chunks, data)
		return nil
	}

	err := StreamSSE(resp, handler)
	if err != nil {
		t.Fatalf("StreamSSE() error = %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
}

// TestMarshalJSON tests MarshalJSON function.
func TestMarshalJSON(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	result, err := MarshalJSON(data)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if !bytes.Contains(result, []byte("key")) {
		t.Errorf("Result = %q, want to contain 'key'", string(result))
	}
}

// TestMarshalJSON_Error tests MarshalJSON with unmarshalable type.
func TestMarshalJSON_Error(t *testing.T) {
	_, err := MarshalJSON(make(chan int))
	if err == nil {
		t.Error("MarshalJSON() should return error for unmarshalable type")
	}
}

// TestUnmarshalJSON tests UnmarshalJSON function.
func TestUnmarshalJSON(t *testing.T) {
	data := []byte(`{"key": "value"}`)
	var result map[string]interface{}
	err := UnmarshalJSON(data, &result)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("Result = %v, want value", result["key"])
	}
}

// TestUnmarshalJSON_Error tests UnmarshalJSON with invalid JSON.
func TestUnmarshalJSON_Error(t *testing.T) {
	data := []byte(`invalid json`)
	var result map[string]interface{}
	err := UnmarshalJSON(data, &result)
	if err == nil {
		t.Error("UnmarshalJSON() should return error for invalid JSON")
	}
}

// TestDecodeJSON tests DecodeJSON function.
func TestDecodeJSON(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{"key": "value"}`)),
	}

	var result map[string]interface{}
	err := DecodeJSON(resp, &result)
	if err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("Result = %v, want value", result["key"])
	}
}

// TestDecodeJSON_Error tests DecodeJSON with invalid JSON.
func TestDecodeJSON_Error(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`invalid json`)),
	}

	var result map[string]interface{}
	err := DecodeJSON(resp, &result)
	if err == nil {
		t.Error("DecodeJSON() should return error for invalid JSON")
	}
}

// errorReader is a reader that always returns an error.
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}
