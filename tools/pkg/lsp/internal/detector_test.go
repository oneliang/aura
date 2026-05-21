package internal

import (
	"testing"
)

func TestLSPDetector_DetectAvailable(t *testing.T) {
	detector := NewLSPDetector()
	detector.DetectAvailable()

	// Test that detection runs without error
	// Actual availability depends on system installation
	t.Run("gopls detection", func(t *testing.T) {
		// Just verify the method exists and runs
		available := detector.HasGopls()
		t.Logf("gopls available: %v", available)
	})

	t.Run("rust-analyzer detection", func(t *testing.T) {
		available := detector.HasRustAnalyzer()
		t.Logf("rust-analyzer available: %v", available)
	})

	t.Run("typescript-language-server detection", func(t *testing.T) {
		available := detector.HasTSServer()
		t.Logf("typescript-language-server available: %v", available)
	})

	t.Run("pylsp detection", func(t *testing.T) {
		available := detector.HasPylsp()
		t.Logf("pylsp available: %v", available)
	})
}

func TestLSPDetector_AvailableLanguages(t *testing.T) {
	detector := NewLSPDetector()
	detector.DetectAvailable()

	langs := detector.AvailableLanguages()
	t.Logf("available languages: %v", langs)

	// If gopls is installed, "go" should be in the list
	if detector.HasGopls() {
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

func TestLSPDetector_GetServerForLanguage(t *testing.T) {
	tests := []struct {
		name     string
		language string
		want     string
	}{
		{"go", "go", "gopls"},
		{"rust", "rust", "rust-analyzer"},
		{"typescript", "typescript", "typescript-language-server"},
		{"python", "python", "pylsp"},
		{"unknown", "unknown", ""},
	}

	detector := NewLSPDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.GetServerForLanguage(tt.language)
			if got != tt.want {
				t.Errorf("GetServerForLanguage(%q) = %q, want %q", tt.language, got, tt.want)
			}
		})
	}
}

func TestDefaultLSPDetector(t *testing.T) {
	// Test global detector
	DetectAvailable()

	// Should work same as instance
	if HasGopls() != defaultLSPDetector.HasGopls() {
		t.Error("HasGopls() should match defaultLSPDetector.HasGopls()")
	}
}