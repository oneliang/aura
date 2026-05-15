package manager

import (
	"context"
	"testing"

	"github.com/oneliang/aura/tools/pkg/lsp/client"
)

func TestManager_GetClientForFile(t *testing.T) {
	mgr := NewManager(".")

	tests := []struct {
		name     string
		filePath string
		wantLang string
		wantErr  bool
	}{
		{"go file", "main.go", "go", false},
		{"go file with path", "src/pkg/main.go", "go", false},
		{"rust file", "src/main.rs", "rust", false},
		{"typescript file", "src/index.ts", "typescript", false},
		{"python file", "main.py", "python", false},
		{"unknown extension", "file.xyz", "", true},
		{"no extension", "README", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := mgr.GetClientForFile(tt.filePath)
			if tt.wantErr {
				if err == nil {
					t.Error("GetClientForFile should return error for unknown language")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetClientForFile failed: %v", err)
			}
			if client == nil {
				t.Fatal("GetClientForFile returned nil client")
			}
			if client.Language() != tt.wantLang {
				t.Errorf("Language() = %q, want %q", client.Language(), tt.wantLang)
			}
		})
	}
}

func TestManager_GetClient(t *testing.T) {
	mgr := NewManager(".")

	tests := []struct {
		name     string
		language string
		wantErr  bool
	}{
		{"go", "go", false},
		{"rust", "rust", false},
		{"typescript", "typescript", false},
		{"python", "python", false},
		{"unknown", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := mgr.GetClient(tt.language)
			if tt.wantErr {
				if err == nil {
					t.Error("GetClient should return error for unknown language")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetClient failed: %v", err)
			}
			if client == nil {
				t.Fatal("GetClient returned nil client")
			}
		})
	}
}

func TestManager_AvailableLanguages(t *testing.T) {
	mgr := NewManager(".")

	langs := mgr.AvailableLanguages()
	t.Logf("available languages: %v", langs)

	// Should include go if gopls is available
	client, err := mgr.GetClient("go")
	if err == nil && client.IsAvailable() {
		found := false
		for _, lang := range langs {
			if lang == "go" {
				found = true
				break
			}
		}
		if !found {
			t.Error("gopls available but 'go' not in AvailableLanguages")
		}
	}
}

func TestManager_RegisterClient(t *testing.T) {
	mgr := NewManager(".")

	// Register a mock client for a new language
	mockLang := "mocklang"
	mockClient := &mockClient{language: mockLang}

	mgr.RegisterClient(mockLang, mockClient)

	client, err := mgr.GetClient(mockLang)
	if err != nil {
		t.Fatalf("GetClient failed after RegisterClient: %v", err)
	}
	if client.Language() != mockLang {
		t.Errorf("Language() = %q, want %q", client.Language(), mockLang)
	}
}

// mockClient for testing
type mockClient struct {
	language string
}

func (m *mockClient) Language() string { return m.language }
func (m *mockClient) IsAvailable() bool { return true }
func (m *mockClient) Execute(ctx context.Context, op client.Operation, params client.Params) (*client.Result, error) {
	return &client.Result{}, nil
}