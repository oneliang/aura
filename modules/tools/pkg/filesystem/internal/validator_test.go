// Package internal provides tests for the internal package.
package internal

import (
	"strings"
	"testing"

	"github.com/oneliang/aura/tools/pkg/trustedpath"
)

// TestValidatePath tests ValidatePath function.
func TestValidatePath(t *testing.T) {
	checker := trustedpath.NopChecker()

	tests := []struct {
		name    string
		params  map[string]any
		want    string
		wantErr bool
	}{
		{
			name:    "missing path",
			params:  map[string]any{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid path type",
			params:  map[string]any{"path": 123},
			want:    "",
			wantErr: true,
		},
		{
			name:    "valid path",
			params:  map[string]any{"path": "/tmp/test"},
			want:    "/tmp/test",
			wantErr: false,
		},
		{
			name:    "empty path",
			params:  map[string]any{"path": ""},
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePath(tt.params, checker)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidatePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestValidatePath_WithMockChecker tests ValidatePath with a mock checker.
func TestValidatePath_WithMockChecker(t *testing.T) {
	// Create a mock checker that checks path prefix
	checker := &mockChecker{allowedPrefix: "/allowed"}

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
	}{
		{
			name:    "allowed path",
			params:  map[string]any{"path": "/allowed/test"},
			wantErr: false,
		},
		{
			name:    "not allowed path",
			params:  map[string]any{"path": "/not-allowed/test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.params, checker)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// mockChecker is a mock implementation of trustedpath.Checker for testing.
type mockChecker struct {
	allowedPrefix string
}

// IsTrustedPath checks if path starts with allowedPrefix.
func (m *mockChecker) IsTrustedPath(path string) bool {
	return strings.HasPrefix(path, m.allowedPrefix)
}

// Ensure mockChecker implements the interface.
var _ trustedpath.Checker = (*mockChecker)(nil)

// TestValidateStringParam tests ValidateStringParam function.
func TestValidateStringParam(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]any
		paramName string
		want      string
		wantErr   bool
	}{
		{
			name:      "missing param",
			params:    map[string]any{},
			paramName: "name",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "invalid type",
			params:    map[string]any{"name": 123},
			paramName: "name",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "valid param",
			params:    map[string]any{"name": "test"},
			paramName: "name",
			want:      "test",
			wantErr:   false,
		},
		{
			name:      "empty string",
			params:    map[string]any{"name": ""},
			paramName: "name",
			want:      "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateStringParam(tt.params, tt.paramName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStringParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateStringParam() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestValidateStringParamOrDefault tests ValidateStringParamOrDefault function.
func TestValidateStringParamOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		params     map[string]any
		paramName  string
		defaultVal string
		want       string
	}{
		{
			name:       "missing param",
			params:     map[string]any{},
			paramName:  "name",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "invalid type",
			params:     map[string]any{"name": 123},
			paramName:  "name",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "empty string uses default",
			params:     map[string]any{"name": ""},
			paramName:  "name",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "valid param",
			params:     map[string]any{"name": "test"},
			paramName:  "name",
			defaultVal: "default",
			want:       "test",
		},
		{
			name:       "empty default",
			params:     map[string]any{"name": ""},
			paramName:  "name",
			defaultVal: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateStringParamOrDefault(tt.params, tt.paramName, tt.defaultVal)
			if got != tt.want {
				t.Errorf("ValidateStringParamOrDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}
