package constants

// Tool names - centralized constants for all tool identifiers
const (
	// Shell and system tools
	ToolShellExec  = "bash"
	ToolSSHExec    = "ssh_exec"
	ToolSystemInfo = "system_info"

	// File system tools
	ToolFileRead   = "file_read"
	ToolFileWrite  = "file_write"
	ToolFileSearch = "file_search"
	ToolFileList   = "file_list"

	// Search tools (Claude Code style)
	ToolGlob = "glob"
	ToolGrep = "grep"

	// Code navigation tools (LSP)
	ToolCodeNavigate = "code_navigate"

	// Web tools
	ToolWebFetch  = "web_fetch"
	ToolWebSearch = "web_search"

	// Utility tools
	ToolDateTime        = "datetime"
	ToolText            = "text"
	ToolCalculator      = "calculator"
	ToolLocation        = "location"
	ToolAskUserQuestion = "ask_user_question"

	// Task tracking
	ToolTask = "task"
)
