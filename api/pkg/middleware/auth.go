// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/oneliang/aura/shared/pkg/user"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware extracts userID from token and adds it to request context.
func AuthMiddleware(userManager *user.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from:
			// 1. Authorization header: "Bearer <token>"
			// 2. Query param: "?token=<token>" (for SSE)
			token := extractToken(r)
			if token == "" {
				writeUnauthorized(w, "Unauthorized: missing token")
				return
			}

			// Validate token and get user
			userCfg := userManager.GetUserByToken(token)
			if userCfg == nil {
				writeUnauthorized(w, "Invalid token")
				return
			}

			// Add userID to context
			ctx := context.WithValue(r.Context(), userIDKey, userCfg.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken extracts auth token from request.
func extractToken(r *http.Request) string {
	// From Authorization header
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// From query param (for SSE - EventSource can't set headers)
	return r.URL.Query().Get("token")
}

// writeUnauthorized writes unauthorized response.
func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// GetUserID extracts userID from request context.
// Returns empty string if not found.
func GetUserID(r *http.Request) string {
	userID, _ := r.Context().Value(userIDKey).(string)
	return userID
}