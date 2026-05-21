package i18n

import (
	"sync"
	"testing"
)

func TestTranslate(t *testing.T) {
	// Reset for clean test
	defaultBundle = nil
	once = sync.Once{}

	result := T("test.key")
	if result != "test.key" {
		t.Errorf("Expected 'test.key', got '%s'", result)
	}
}

func TestTranslateWithFallback(t *testing.T) {
	// Test fallback behavior
	result := T("nonexistent.key")
	if result != "nonexistent.key" {
		t.Errorf("Expected 'nonexistent.key', got '%s'", result)
	}
}

func TestInit(t *testing.T) {
	// Reset for clean test
	defaultBundle = nil
	once = sync.Once{}

	// Test initialization - embedded locales should load automatically
	err := Init("", "en")
	if err != nil {
		t.Errorf("Init should not error, got %v", err)
	}

	// Test that bundle is created
	if defaultBundle == nil {
		t.Error("Expected defaultBundle to be initialized")
	}
	if defaultBundle.locale != "en" {
		t.Errorf("Expected locale 'en', got '%s'", defaultBundle.locale)
	}
	if defaultBundle.fallback != "en" {
		t.Errorf("Expected fallback 'en', got '%s'", defaultBundle.fallback)
	}

	// Verify embedded locales loaded
	if _, ok := defaultBundle.messages["en"]; !ok {
		t.Error("Expected English embedded messages to be loaded")
	}
	if _, ok := defaultBundle.messages["zh-CN"]; !ok {
		t.Error("Expected Chinese embedded messages to be loaded")
	}
}

func TestEmbeddedTranslation(t *testing.T) {
	// Reset and initialize
	defaultBundle = nil
	once = sync.Once{}
	err := Init("", "en")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test actual translation from embedded en.yaml
	result := T("tui.thinking")
	if result != "Thinking..." {
		t.Errorf("Expected 'Thinking...', got '%s'", result)
	}

	// Switch to Chinese
	SetLocale("zh-CN")
	result = T("tui.thinking")
	if result != "思考中..." {
		t.Errorf("Expected '思考中...', got '%s'", result)
	}

	// Reset to English
	SetLocale("en")
}

func TestSetLocale(t *testing.T) {
	// Ensure initialized
	if defaultBundle == nil {
		defaultBundle = &Bundle{
			locale:   "en",
			messages: make(map[string]map[string]string),
			fallback: "en",
		}
	}

	// Test setting locale
	SetLocale("zh-CN")
	locale := GetLocale()
	if locale != "zh-CN" {
		t.Errorf("Expected 'zh-CN', got '%s'", locale)
	}

	// Reset to default
	SetLocale("en")
}

func TestGetLocale(t *testing.T) {
	// Test getting current locale
	locale := GetLocale()
	if locale == "" {
		t.Error("Expected non-empty locale")
	}
}

func TestBundle_T_WithArgs(t *testing.T) {
	bundle := &Bundle{
		locale:   "en",
		messages: make(map[string]map[string]string),
		fallback: "en",
	}

	// Add a message with format placeholder
	bundle.messages["en"] = map[string]string{
		"greeting": "Hello, %s!",
	}

	// Test translation with argument
	result := bundle.T("greeting", "World")
	if result != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", result)
	}

	// Test translation with multiple arguments
	bundle.messages["en"]["info"] = "Count: %d, Name: %s"
	result = bundle.T("info", 42, "Test")
	if result != "Count: 42, Name: Test" {
		t.Errorf("Expected 'Count: 42, Name: Test', got '%s'", result)
	}
}

func TestBundle_T_Fallback(t *testing.T) {
	bundle := &Bundle{
		locale:   "zh-CN",
		messages: make(map[string]map[string]string),
		fallback: "en",
	}

	// Only add English translation
	bundle.messages["en"] = map[string]string{
		"english_only": "English text",
	}

	// Should fallback to English when Chinese not available
	result := bundle.T("english_only")
	if result != "English text" {
		t.Errorf("Expected 'English text' (fallback), got '%s'", result)
	}
}

func TestT_WithArgs(t *testing.T) {
	// Setup bundle for testing
	defaultBundle = &Bundle{
		locale:   "en",
		messages: make(map[string]map[string]string),
		fallback: "en",
	}
	defaultBundle.messages["en"] = map[string]string{
		"message": "Value: %d",
	}

	result := T("message", 123)
	if result != "Value: 123" {
		t.Errorf("Expected 'Value: 123', got '%s'", result)
	}
}
