package internal

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{"go file", "main.go", "go"},
		{"go file with path", "src/pkg/main.go", "go"},
		{"rust file", "src/main.rs", "rust"},
		{"typescript file", "src/index.ts", "typescript"},
		{"tsx file", "src/component.tsx", "typescript"},
		{"javascript file", "src/app.js", "typescript"},
		{"jsx file", "src/Button.jsx", "typescript"},
		{"python file", "main.py", "python"},
		{"c file", "main.c", "c"},
		{"cpp file", "main.cpp", "cpp"},
		{"header file", "header.h", "c"},
		{"cpp header", "header.hpp", "cpp"},
		{"unknown extension", "file.xyz", ""},
		{"no extension", "README", ""},
		{"empty path", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filePath)
			if got != tt.want {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestLanguageDetector_WithCustomMapping(t *testing.T) {
	customMap := map[string]string{
		".custom": "customlang",
	}

	detector := NewLanguageDetector(customMap)

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{"custom extension", "file.custom", "customlang"},
		{"fallback to default", "main.go", "go"},
		{"unknown", "file.xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.Detect(tt.filePath)
			if got != tt.want {
				t.Errorf("Detect(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}