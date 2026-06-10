// Package server provides SSE handling functionality.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// sseRequest represents the parsed SSE request body.
type sseRequest struct {
	Content string `json:"content"`
	Source  string `json:"source,omitempty"`
}

// setupSSEHeaders sets the required SSE headers on the response writer.
func setupSSEHeaders(w http.ResponseWriter) http.Flusher {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	return flusher
}

// parseSSERequest parses the SSE request body from the HTTP request.
func (s *Server) parseSSERequest(r *http.Request) (*sseRequest, error) {
	var req sseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("invalid request body")
	}

	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	return &req, nil
}
