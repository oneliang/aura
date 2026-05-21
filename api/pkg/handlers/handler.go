package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/oneliang/aura/shared/pkg/logger"
)

// Response represents a standard API response.
type Response struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// WriteJSON writes a JSON response with the given status code and data.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Default().Error().Err(err).Msg("WriteJSON: failed to encode response")
	}
}

// WriteOK writes a success response with optional data and message.
func WriteOK(w http.ResponseWriter, data interface{}, message string) {
	resp := Response{Status: "ok"}
	if data != nil {
		resp.Data = data
	}
	if message != "" {
		resp.Message = message
	}
	WriteJSON(w, http.StatusOK, resp)
}

// WriteError writes an error response in JSON format.
func WriteError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Response{
		Status: "error",
		Error:  message,
	}); err != nil {
		logger.Default().Error().Err(err).Msg("WriteError: failed to encode response")
	}
}
