// Package init provides the init command handler for generating AURA.md.
package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneliang/aura/shared/pkg/config"
)

// Handler handles the init command for generating AURA.md.
type Handler struct {
	Config       *config.Config
	ConfigLoader ConfigLoader
}

// ConfigLoader interface for loading configuration.
type ConfigLoader interface {
	Load() (*config.Config, error)
}

// GetCommands returns the available init commands.
func (h *Handler) GetCommands() []CommandInfo {
	return []CommandInfo{
		{
			Name:        "command_init",
			DisplayName: "Init",
			Description: "Initialize AURA.md with codebase documentation",
			Params: []ParamInfo{
				{
					Name:     "force",
					Type:     "bool",
					Required: false,
					Desc:     "Force regeneration even if AURA.md exists",
				},
			},
		},
	}
}

// Execute executes the init command.
func (h *Handler) Execute(ctx context.Context, cmd string, params map[string]any) (string, error) {
	// Get working directory
	cwd, ok := params["cwd"].(string)
	if !ok || cwd == "" {
		cwd, _ = os.Getwd()
	}

	// Check force flag
	force := params["force"] == true

	claudeMdPath := filepath.Join(cwd, "AURA.md")

	// Check if AURA.md already exists
	if !force {
		if _, err := os.Stat(claudeMdPath); err == nil {
			return fmt.Sprintf("AURA.md already exists at: %s\nUse force=true to regenerate.", claudeMdPath), nil
		}
	}

	// Build init prompt
	prompt := BuildInitPrompt(cwd)

	// For CLI mode, we return the prompt to be executed by the runtime
	// For internal command mode, we need to execute through the runtime
	result, err := h.executeAnalysis(ctx, cwd, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to analyze codebase: %w", err)
	}

	// Write AURA.md
	if err := h.writeClaudeMd(claudeMdPath, result); err != nil {
		return "", fmt.Errorf("failed to write AURA.md: %w", err)
	}

	return fmt.Sprintf("Generated AURA.md at: %s\n\nPreview:\n%s", claudeMdPath, truncatePreview(result, 500)), nil
}

// executeAnalysis performs the codebase analysis using file scanning.
func (h *Handler) executeAnalysis(ctx context.Context, cwd, prompt string) (string, error) {
	// Phase 1: Detect project type
	projectType := h.detectProjectType(cwd)

	// Phase 2: Scan project structure
	structure := h.scanStructure(cwd)

	// Phase 3: Analyze build commands
	buildCommands := h.analyzeBuildCommands(cwd, projectType)

	// Phase 4: Analyze architecture
	architecture := h.analyzeArchitecture(cwd, projectType)

	// Phase 5: Generate AURA.md content
	content := h.generateClaudeMdContent(projectType, structure, buildCommands, architecture)

	return content, nil
}

// detectProjectType identifies the project type based on marker files.
func (h *Handler) detectProjectType(cwd string) string {
	markers := map[string]string{
		"go.mod":          "Go",
		"package.json":    "Node/TypeScript",
		"Cargo.toml":      "Rust",
		"pyproject.toml":  "Python",
		"requirements.txt": "Python",
		"Makefile":        "Make",
		"CMakeLists.txt":  "C/C++",
	}

	for marker, projectType := range markers {
		if _, err := os.Stat(filepath.Join(cwd, marker)); err == nil {
			return projectType
		}
	}

	return "Unknown"
}

// scanStructure scans the project directory structure.
func (h *Handler) scanStructure(cwd string) *ProjectStructure {
	structure := &ProjectStructure{
		EntryPoints:   []string{},
		ConfigFiles:   []string{},
		DocFiles:      []string{},
		ModuleDirs:    []string{},
	}

	// Common entry point patterns
	entryPatterns := []string{
		"main.go", "cmd/", "index.ts", "index.js", "app.py", "main.py",
		"src/main.rs", "lib/main.rs",
	}

	for _, pattern := range entryPatterns {
		path := filepath.Join(cwd, pattern)
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				structure.ModuleDirs = append(structure.ModuleDirs, pattern)
			} else {
				structure.EntryPoints = append(structure.EntryPoints, pattern)
			}
		}
	}

	// Common config files
	configPatterns := []string{
		"config.yaml", ".env", "settings.json", "config.json",
		".claude/config.yaml", "configs/config.yaml",
	}

	for _, pattern := range configPatterns {
		path := filepath.Join(cwd, pattern)
		if _, err := os.Stat(path); err == nil {
			structure.ConfigFiles = append(structure.ConfigFiles, pattern)
		}
	}

	// Documentation files
	docPatterns := []string{"README.md", "docs/", "AURA.md"}
	for _, pattern := range docPatterns {
		path := filepath.Join(cwd, pattern)
		if _, err := os.Stat(path); err == nil {
			structure.DocFiles = append(structure.DocFiles, pattern)
		}
	}

	return structure
}

// analyzeBuildCommands extracts build commands from project files.
func (h *Handler) analyzeBuildCommands(cwd, projectType string) *BuildCommands {
	buildCmds := &BuildCommands{}

	switch projectType {
	case "Go":
		buildCmds.Build = "go build"
		buildCmds.Test = "go test ./..."
		buildCmds.Run = "go run main.go"
		buildCmds.Lint = "golangci-lint run"

		// Check for Makefile
		makefilePath := filepath.Join(cwd, "Makefile")
		if content, err := os.ReadFile(makefilePath); err == nil {
			buildCmds = h.extractMakefileTargets(string(content), buildCmds)
		}

	case "Node/TypeScript":
		// Check package.json for scripts
		packageJsonPath := filepath.Join(cwd, "package.json")
		if content, err := os.ReadFile(packageJsonPath); err == nil {
			buildCmds = h.extractPackageJsonScripts(string(content), buildCmds)
		}

	case "Rust":
		buildCmds.Build = "cargo build"
		buildCmds.Test = "cargo test"
		buildCmds.Run = "cargo run"
		buildCmds.Lint = "cargo clippy"

	case "Python":
		buildCmds.Test = "pytest"
		buildCmds.Run = "python main.py"
		buildCmds.Lint = "ruff check"

		// Check for pyproject.toml
		pyprojectPath := filepath.Join(cwd, "pyproject.toml")
		if _, err := os.Stat(pyprojectPath); err == nil {
			buildCmds.Test = "pytest"
		}
	}

	return buildCmds
}

// extractMakefileTargets extracts build targets from Makefile content.
func (h *Handler) extractMakefileTargets(content string, buildCmds *BuildCommands) *BuildCommands {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "build:") {
			buildCmds.Build = "make build"
		}
		if strings.HasPrefix(line, "test:") {
			buildCmds.Test = "make test"
		}
		if strings.HasPrefix(line, "run:") {
			buildCmds.Run = "make run"
		}
		if strings.HasPrefix(line, "lint:") {
			buildCmds.Lint = "make lint"
		}
	}
	return buildCmds
}

// extractPackageJsonScripts extracts scripts from package.json content.
func (h *Handler) extractPackageJsonScripts(content string, buildCmds *BuildCommands) *BuildCommands {
	// Simple extraction - look for common script names
	if strings.Contains(content, `"build"`) {
		buildCmds.Build = "npm run build"
	}
	if strings.Contains(content, `"test"`) {
		buildCmds.Test = "npm test"
	}
	if strings.Contains(content, `"start"`) {
		buildCmds.Run = "npm start"
	}
	if strings.Contains(content, `"lint"`) {
		buildCmds.Lint = "npm run lint"
	}
	return buildCmds
}

// analyzeArchitecture analyzes the project architecture.
func (h *Handler) analyzeArchitecture(cwd, projectType string) *Architecture {
	arch := &Architecture{
		Type:        "Unknown",
		Description: "Project architecture not yet analyzed",
	}

	switch projectType {
	case "Go":
		// Check for layered architecture
		if h.hasLayeredGoStructure(cwd) {
			arch.Type = "Layered"
			arch.Description = "4-layer architecture (Base → Service → Core → Application)"
			arch.Layers = []string{"Base (shared, tools, storage)", "Service (session, commands)", "Core (engine, runtime)", "Application (cli, api, adapters)"}
		} else {
			arch.Type = "Standard Go"
			arch.Description = "Standard Go project structure"
		}
	}

	return arch
}

// hasLayeredGoStructure checks if the project has a layered architecture.
func (h *Handler) hasLayeredGoStructure(cwd string) bool {
	layerDirs := []string{"modules/core", "modules/cli", "modules/api", "modules/shared"}
	foundLayers := 0
	for _, dir := range layerDirs {
		if _, err := os.Stat(filepath.Join(cwd, dir)); err == nil {
			foundLayers++
		}
	}
	return foundLayers >= 2
}

// generateClaudeMdContent generates the AURA.md content.
func (h *Handler) generateClaudeMdContent(projectType string, structure *ProjectStructure, buildCmds *BuildCommands, arch *Architecture) string {
	var content strings.Builder

	content.WriteString("# AURA.md\n\n")
	content.WriteString("This file provides guidance to Aura when working with code in this repository.\n\n")
	content.WriteString(fmt.Sprintf("**Project Type**: %s\n\n", projectType))

	// Build & Development Commands
	content.WriteString("## Build & Development Commands\n\n")
	if buildCmds.Build != "" {
		content.WriteString(fmt.Sprintf("**Build**: `%s`\n", buildCmds.Build))
	}
	if buildCmds.Test != "" {
		content.WriteString(fmt.Sprintf("**Test**: `%s`\n", buildCmds.Test))
	}
	if buildCmds.Run != "" {
		content.WriteString(fmt.Sprintf("**Run**: `%s`\n", buildCmds.Run))
	}
	if buildCmds.Lint != "" {
		content.WriteString(fmt.Sprintf("**Lint**: `%s`\n", buildCmds.Lint))
	}
	content.WriteString("\n")

	// Architecture Overview
	content.WriteString("## Architecture Overview\n\n")
	content.WriteString(fmt.Sprintf("**Type**: %s\n", arch.Type))
	content.WriteString(fmt.Sprintf("**Description**: %s\n\n", arch.Description))
	if len(arch.Layers) > 0 {
		content.WriteString("**Layers**:\n")
		for _, layer := range arch.Layers {
			content.WriteString(fmt.Sprintf("- %s\n", layer))
		}
		content.WriteString("\n")
	}

	// Entry Points
	if len(structure.EntryPoints) > 0 {
		content.WriteString("## Entry Points\n\n")
		for _, ep := range structure.EntryPoints {
			content.WriteString(fmt.Sprintf("- `%s`\n", ep))
		}
		content.WriteString("\n")
	}

	// Configuration
	if len(structure.ConfigFiles) > 0 {
		content.WriteString("## Configuration\n\n")
		for _, cfg := range structure.ConfigFiles {
			content.WriteString(fmt.Sprintf("- `%s`\n", cfg))
		}
		content.WriteString("\n")
	}

	// Notes
	content.WriteString("## Notes\n\n")
	content.WriteString("- This file was auto-generated by `aura init`\n")
	content.WriteString("- Review and customize the content for your project\n")
	content.WriteString("- Add project-specific patterns and conventions\n")

	return content.String()
}

// writeClaudeMd writes the content to AURA.md file.
func (h *Handler) writeClaudeMd(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// truncatePreview truncates content for preview display.
func truncatePreview(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// ProjectStructure holds the scanned project structure.
type ProjectStructure struct {
	EntryPoints   []string
	ConfigFiles   []string
	DocFiles      []string
	ModuleDirs    []string
}

// BuildCommands holds the extracted build commands.
type BuildCommands struct {
	Build string
	Test  string
	Run   string
	Lint  string
}

// Architecture holds the project architecture information.
type Architecture struct {
	Type        string
	Description string
	Layers      []string
}