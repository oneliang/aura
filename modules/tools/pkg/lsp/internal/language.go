package internal

import (
	"path/filepath"
	"sync"

	"github.com/oneliang/aura/shared/pkg/constants"
)

// LanguageDetector detects programming language from file extension.
type LanguageDetector struct {
	extensionMap map[string]string
	once         sync.Once
}

// NewLanguageDetector creates a language detector with optional custom mapping.
// If customMap is nil, uses default extension mapping from constants.
func NewLanguageDetector(customMap map[string]string) *LanguageDetector {
	return &LanguageDetector{
		extensionMap: customMap,
	}
}

// Detect returns language ID for the given file path.
// If custom mapping is set, first checks custom, then falls back to default.
func (d *LanguageDetector) Detect(filePath string) string {
	d.once.Do(func() {
		// Ensure extensionMap is initialized
		if d.extensionMap == nil {
			d.extensionMap = constants.DefaultExtensionMap
		}
	})

	ext := filepath.Ext(filePath)
	if ext == "" {
		return ""
	}

	// Check custom/primary mapping
	if lang, ok := d.extensionMap[ext]; ok {
		return lang
	}

	// Fallback to default mapping (for custom detectors)
	if lang, ok := constants.DefaultExtensionMap[ext]; ok {
		return lang
	}

	return ""
}

// Default language detector instance.
var defaultDetector = NewLanguageDetector(nil)

// DetectLanguage returns language ID for the given file path using default mapping.
func DetectLanguage(filePath string) string {
	return defaultDetector.Detect(filePath)
}