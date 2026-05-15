// Package intent provides natural language intent recognition for command execution.
package intent

import (
	"testing"
)

func TestGetKeywordLoader(t *testing.T) {
	loader := GetKeywordLoader()
	if loader == nil {
		t.Fatal("expected non-nil loader")
	}

	// Should return the same instance (singleton)
	loader2 := GetKeywordLoader()
	if loader != loader2 {
		t.Error("expected same loader instance (singleton)")
	}
}

func TestKeywordLoader_GetKeywords(t *testing.T) {
	loader := GetKeywordLoader()

	t.Run("returns keywords for valid group if i18n initialized", func(t *testing.T) {
		keywords := loader.GetKeywords(KeywordExit)
		// Keywords may be empty if i18n is not initialized in test environment
		// This test just verifies the method doesn't panic
		t.Logf("exit keywords: %v", keywords)
	})

	t.Run("returns empty for invalid group", func(t *testing.T) {
		keywords := loader.GetKeywords("invalid_group")
		if len(keywords) != 0 {
			t.Errorf("expected empty keywords for invalid group, got %v", keywords)
		}
	})
}

func TestKeywordLoader_GetPatterns(t *testing.T) {
	loader := GetKeywordLoader()

	t.Run("returns patterns for valid identifier if i18n initialized", func(t *testing.T) {
		patterns := loader.GetPatterns(PatternName)
		// Patterns may be empty if i18n is not initialized in test environment
		t.Logf("name patterns: %v", patterns)
	})

	t.Run("returns empty for invalid identifier", func(t *testing.T) {
		patterns := loader.GetPatterns("invalid_pattern")
		if len(patterns) != 0 {
			t.Errorf("expected empty patterns for invalid identifier, got %v", patterns)
		}
	})
}

func TestKeywordLoader_GetConfigKeys(t *testing.T) {
	loader := GetKeywordLoader()

	t.Run("returns config keys", func(t *testing.T) {
		keys := loader.GetConfigKeys()
		if len(keys) == 0 {
			t.Error("expected config keys")
		}
		// Verify expected keys are present
		expectedKeys := []string{ConfigKeyModel, ConfigKeyTemperature, ConfigKeyProvider}
		for _, expected := range expectedKeys {
			found := false
			for _, key := range keys {
				if key == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected config key '%s' to be present", expected)
			}
		}
	})
}

func TestKeywordLoader_ContainsKeyword(t *testing.T) {
	loader := GetKeywordLoader()

	// These tests verify the logic works when keywords are loaded
	// If i18n is not initialized, keywords will be empty and ContainsKeyword returns false
	t.Run("handles matching keyword if loaded", func(t *testing.T) {
		keywords := loader.GetKeywords(KeywordExit)
		if len(keywords) == 0 {
			t.Skip("skip: exit keywords not loaded (i18n not initialized)")
		}
		if !loader.ContainsKeyword("exit", KeywordExit) {
			t.Error("expected 'exit' to contain exit keyword")
		}
	})

	t.Run("handles partial match if loaded", func(t *testing.T) {
		keywords := loader.GetKeywords(KeywordExit)
		if len(keywords) == 0 {
			t.Skip("skip: exit keywords not loaded (i18n not initialized)")
		}
		if !loader.ContainsKeyword("please exit now", KeywordExit) {
			t.Error("expected 'please exit now' to contain exit keyword")
		}
	})

	t.Run("returns false for non-matching input", func(t *testing.T) {
		if loader.ContainsKeyword("hello world", KeywordExit) {
			t.Error("expected 'hello world' to not contain exit keyword")
		}
	})
}

func TestKeywordLoader_ExtractName(t *testing.T) {
	loader := GetKeywordLoader()

	t.Run("extracts name from pattern if loaded", func(t *testing.T) {
		patterns := loader.GetPatterns(PatternName)
		if len(patterns) == 0 {
			t.Skip("skip: name patterns not loaded (i18n not initialized)")
		}
		name := loader.ExtractName("create session called test")
		if name != "test" {
			t.Errorf("expected name 'test', got '%s'", name)
		}
	})

	t.Run("returns empty for no pattern", func(t *testing.T) {
		name := loader.ExtractName("create session")
		if name != "" {
			t.Errorf("expected empty name, got '%s'", name)
		}
	})
}

func TestKeywordLoader_ExtractConfigKey(t *testing.T) {
	loader := GetKeywordLoader()

	t.Run("extracts model key", func(t *testing.T) {
		key := loader.ExtractConfigKey("set model to llama")
		if key != ConfigKeyModel {
			t.Errorf("expected key '%s', got '%s'", ConfigKeyModel, key)
		}
	})

	t.Run("extracts temperature key", func(t *testing.T) {
		key := loader.ExtractConfigKey("change temperature setting")
		if key != ConfigKeyTemperature {
			t.Errorf("expected key '%s', got '%s'", ConfigKeyTemperature, key)
		}
	})

	t.Run("returns empty for no config key", func(t *testing.T) {
		key := loader.ExtractConfigKey("hello world")
		if key != "" {
			t.Errorf("expected empty key, got '%s'", key)
		}
	})
}
