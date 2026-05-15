package constants

// Language IDs for LSP support
const (
	LanguageGo         = "go"
	LanguageRust       = "rust"
	LanguageTypeScript = "typescript"
	LanguagePython     = "python"
	LanguageC          = "c"
	LanguageCPP        = "cpp"
)

// LSP server command names
const (
	LSPServerGopls          = "gopls"
	LSPServerRustAnalyzer   = "rust-analyzer"
	LSPServerTSServer       = "typescript-language-server"
	LSPServerPylsp          = "pylsp"
	LSPServerClangd         = "clangd"
)

// Default extension to language mapping
// Map key is file extension (with dot), value is language ID
var DefaultExtensionMap = map[string]string{
	".go":  LanguageGo,
	".rs":  LanguageRust,
	".ts":  LanguageTypeScript,
	".tsx": LanguageTypeScript,
	".js":  LanguageTypeScript,
	".jsx": LanguageTypeScript,
	".py":  LanguagePython,
	".c":   LanguageC,
	".cpp": LanguageCPP,
	".h":   LanguageC,
	".hpp": LanguageCPP,
}

// Default language to LSP server mapping
var DefaultLSPServerMap = map[string]string{
	LanguageGo:         LSPServerGopls,
	LanguageRust:       LSPServerRustAnalyzer,
	LanguageTypeScript: LSPServerTSServer,
	LanguagePython:     LSPServerPylsp,
	LanguageC:          LSPServerClangd,
	LanguageCPP:        LSPServerClangd,
}