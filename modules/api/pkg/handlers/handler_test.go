package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "success response",
			status:     http.StatusOK,
			data:       map[string]string{"status": "ok"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "created response",
			status:     http.StatusCreated,
			data:       map[string]string{"id": "123"},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("WriteJSON() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("WriteJSON() Content-Type = %v, want application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		status     int
		wantStatus int
	}{
		{
			name:       "bad request",
			message:    "Invalid input",
			status:     http.StatusBadRequest,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal server error",
			message:    "Server error",
			status:     http.StatusInternalServerError,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tt.message, tt.status)

			if w.Code != tt.wantStatus {
				t.Errorf("WriteError() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if !strings.Contains(w.Body.String(), tt.message) {
				t.Errorf("WriteError() body = %v, want to contain %v", w.Body.String(), tt.message)
			}
		})
	}
}

func TestWriteOK(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "success message",
			message: "Operation successful",
		},
		{
			name:    "empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteOK(w, nil, tt.message)

			if w.Code != http.StatusOK {
				t.Errorf("WriteOK() status = %v, want %v", w.Code, http.StatusOK)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("WriteOK() failed to decode JSON: %v", err)
			}

			if resp["status"] != "ok" {
				t.Errorf("WriteOK() status = %v, want ok", resp["status"])
			}

			if resp["message"] != tt.message {
				t.Errorf("WriteOK() message = %v, want %v", resp["message"], tt.message)
			}
		})
	}
}

func TestWriteErrorJSON(t *testing.T) {
	// Note: WriteError is now the unified error handler that writes JSON
	// This test is kept for backward compatibility
	tests := []struct {
		name    string
		message string
		status  int
	}{
		{
			name:    "bad request",
			message: "Invalid input",
			status:  http.StatusBadRequest,
		},
		{
			name:    "not found",
			message: "Resource not found",
			status:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tt.message, tt.status)

			if w.Code != tt.status {
				t.Errorf("WriteError() status = %v, want %v", w.Code, tt.status)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("WriteError() Content-Type = %v, want application/json", w.Header().Get("Content-Type"))
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("WriteError() failed to decode JSON: %v", err)
			}

			if resp["error"] != tt.message {
				t.Errorf("WriteError() error = %v, want %v", resp["error"], tt.message)
			}
		})
	}
}
