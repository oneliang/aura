package internal

import (
	"os/exec"
	"sync"

	"github.com/oneliang/aura/shared/pkg/constants"
)

// LSPDetector detects available LSP servers on the system.
type LSPDetector struct {
	hasGopls        bool
	hasRustAnalyzer bool
	hasTSServer     bool
	hasPylsp        bool
	once            sync.Once
}

// NewLSPDetector creates a new LSP detector.
func NewLSPDetector() *LSPDetector {
	return &LSPDetector{}
}

// DetectAvailable detects which LSP servers are installed.
func (d *LSPDetector) DetectAvailable() {
	d.once.Do(func() {
		_, err := exec.LookPath(constants.LSPServerGopls)
		d.hasGopls = (err == nil)

		_, err = exec.LookPath(constants.LSPServerRustAnalyzer)
		d.hasRustAnalyzer = (err == nil)

		_, err = exec.LookPath(constants.LSPServerTSServer)
		d.hasTSServer = (err == nil)

		_, err = exec.LookPath(constants.LSPServerPylsp)
		d.hasPylsp = (err == nil)
	})
}

// HasGopls checks if gopls is available.
func (d *LSPDetector) HasGopls() bool {
	d.DetectAvailable()
	return d.hasGopls
}

// HasRustAnalyzer checks if rust-analyzer is available.
func (d *LSPDetector) HasRustAnalyzer() bool {
	d.DetectAvailable()
	return d.hasRustAnalyzer
}

// HasTSServer checks if typescript-language-server is available.
func (d *LSPDetector) HasTSServer() bool {
	d.DetectAvailable()
	return d.hasTSServer
}

// HasPylsp checks if pylsp is available.
func (d *LSPDetector) HasPylsp() bool {
	d.DetectAvailable()
	return d.hasPylsp
}

// AvailableLanguages returns list of languages with available servers.
func (d *LSPDetector) AvailableLanguages() []string {
	d.DetectAvailable()

	var langs []string
	if d.hasGopls {
		langs = append(langs, constants.LanguageGo)
	}
	if d.hasRustAnalyzer {
		langs = append(langs, constants.LanguageRust)
	}
	if d.hasTSServer {
		langs = append(langs, constants.LanguageTypeScript)
	}
	if d.hasPylsp {
		langs = append(langs, constants.LanguagePython)
	}
	return langs
}

// GetServerForLanguage returns the LSP server command for a language.
func (d *LSPDetector) GetServerForLanguage(language string) string {
	server, ok := constants.DefaultLSPServerMap[language]
	if !ok {
		return ""
	}
	return server
}

// IsLanguageAvailable checks if an LSP server is available for the language.
func (d *LSPDetector) IsLanguageAvailable(language string) bool {
	d.DetectAvailable()

	switch language {
	case constants.LanguageGo:
		return d.hasGopls
	case constants.LanguageRust:
		return d.hasRustAnalyzer
	case constants.LanguageTypeScript:
		return d.hasTSServer
	case constants.LanguagePython:
		return d.hasPylsp
	default:
		return false
	}
}

// Default LSP detector instance.
var defaultLSPDetector = NewLSPDetector()

// DetectAvailable detects available LSP servers using default detector.
func DetectAvailable() {
	defaultLSPDetector.DetectAvailable()
}

// HasGopls checks if gopls is available using default detector.
func HasGopls() bool {
	return defaultLSPDetector.HasGopls()
}

// HasRustAnalyzer checks if rust-analyzer is available using default detector.
func HasRustAnalyzer() bool {
	return defaultLSPDetector.HasRustAnalyzer()
}

// HasTSServer checks if typescript-language-server is available using default detector.
func HasTSServer() bool {
	return defaultLSPDetector.HasTSServer()
}

// HasPylsp checks if pylsp is available using default detector.
func HasPylsp() bool {
	return defaultLSPDetector.HasPylsp()
}

// AvailableLanguages returns languages with available servers using default detector.
func AvailableLanguages() []string {
	return defaultLSPDetector.AvailableLanguages()
}

// IsLanguageAvailable checks if language has available server using default detector.
func IsLanguageAvailable(language string) bool {
	return defaultLSPDetector.IsLanguageAvailable(language)
}