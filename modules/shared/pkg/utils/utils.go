// Package utils provides common utility functions.
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// GenerateID generates a unique ID based on timestamp and content.
func GenerateID(content string) string {
	hash := sha256.Sum256([]byte(content + time.Now().String()))
	return hex.EncodeToString(hash[:8])
}

// Truncate truncates a string to a maximum length.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Contains checks if a string slice contains a string.
func Contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveEmpty removes empty strings from a slice.
func RemoveEmpty(slice []string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}
	return result
}

// Coalesce returns the first non-empty string.
func Coalesce(strings ...string) string {
	for _, s := range strings {
		if s != "" {
			return s
		}
	}
	return ""
}

// Now returns current time in UTC.
func Now() time.Time {
	return time.Now().UTC()
}

// FormatTime formats time in ISO 8601 format.
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseTime parses ISO 8601 time format.
func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// FormatTimestamp formats a timestamp for UI display (HH:MM:SS).
// This is a common utility function used by TUI and other UI components.
func FormatTimestamp(t time.Time) string {
	return t.Format("15:04:05")
}

// Min returns the minimum of two integers.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
