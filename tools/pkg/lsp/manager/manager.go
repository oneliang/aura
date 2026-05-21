package manager

import (
	"fmt"
	"sync"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/tools/pkg/lsp/client"
	"github.com/oneliang/aura/tools/pkg/lsp/internal"
)

// Manager manages LSP clients for multiple languages.
type Manager struct {
	clients  map[string]client.Client
	detector *internal.LSPDetector
	rootPath string
	mu       sync.RWMutex
}

// NewManager creates a new LSP client manager.
func NewManager(rootPath string) *Manager {
	mgr := &Manager{
		clients:  make(map[string]client.Client),
		detector: internal.NewLSPDetector(),
		rootPath: rootPath,
	}

	// Register built-in clients
	mgr.registerBuiltInClients()

	return mgr
}

// registerBuiltInClients registers all built-in LSP clients.
func (m *Manager) registerBuiltInClients() {
	// Go client
	m.clients[constants.LanguageGo] = client.NewGoplsClient(m.rootPath)

	// Rust client (placeholder - will be fully implemented later)
	m.clients[constants.LanguageRust] = client.NewRustAnalyzerClient(m.rootPath)

	// TypeScript client (placeholder - will be fully implemented later)
	m.clients[constants.LanguageTypeScript] = client.NewTSServerClient(m.rootPath)

	// Python client (placeholder - will be fully implemented later)
	m.clients[constants.LanguagePython] = client.NewPylspClient(m.rootPath)
}

// RegisterClient registers a custom LSP client for a language.
func (m *Manager) RegisterClient(language string, c client.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[language] = c
}

// GetClientForFile returns the appropriate client for the given file path.
func (m *Manager) GetClientForFile(filePath string) (client.Client, error) {
	lang := internal.DetectLanguage(filePath)
	if lang == "" {
		return nil, fmt.Errorf("unsupported file type: %s", filePath)
	}
	return m.GetClient(lang)
}

// GetClient returns the client for a specific language.
func (m *Manager) GetClient(language string) (client.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.clients[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
	return c, nil
}

// AvailableLanguages returns list of languages with registered and available clients.
func (m *Manager) AvailableLanguages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var langs []string
	for lang, c := range m.clients {
		if c.IsAvailable() {
			langs = append(langs, lang)
		}
	}
	return langs
}

// Detector returns the LSP detector.
func (m *Manager) Detector() *internal.LSPDetector {
	return m.detector
}