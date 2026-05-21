// Package init provides the init prompt template.
package init

import "fmt"

// BuildInitPrompt builds the prompt for codebase analysis.
func BuildInitPrompt(cwd string) string {
	return fmt.Sprintf(`
You are initializing a CLAUDE.md file for this project.

## Current Directory
%s

## Task
Scan the codebase and generate a comprehensive CLAUDE.md file.

## Workflow

### Phase 1: Project Detection
Identify the project type by checking for:
- go.mod → Go project
- package.json → Node/TypeScript
- Cargo.toml → Rust
- pyproject.toml → Python
- Makefile → Build system

### Phase 2: Structure Discovery
Use available tools to discover:
- Entry points (main.go, cmd/, index.ts, app.py)
- Module/package structure
- Configuration files
- Documentation (README.md, docs/)

### Phase 3: Build Commands Analysis
Find build/test/lint commands:
- From Makefile targets
- From package.json scripts
- From CI/CD files (.github/workflows/, Jenkinsfile)
- From go.mod directives

### Phase 4: Architecture Analysis
Map the architecture:
- Module dependencies
- Layer structure (if layered)
- Key interfaces
- Core components

## Output Format

Generate CLAUDE.md with these sections (Markdown format):

# CLAUDE.md

## Build & Development Commands
[Exact commands from build system]

## CLI Usage
[Command examples]

## Architecture Overview
[ASCII diagram or description]

## Module Dependency Matrix
[Table of modules and dependencies]

## Key Interfaces
[List with file paths]

## Configuration
[Config file locations and key settings]

## Development Patterns
[Project-specific patterns]

## Notes
[Important implementation details]

## Instructions
- Use specific commands (not placeholders like "npm run build")
- Use absolute file paths
- Include actual values from config files
- Focus on project-specific details, not generic advice
`, cwd)
}

// InitPromptTemplate is the full prompt template for LLM-based init.
const InitPromptTemplate = `
You are initializing a CLAUDE.md file for this project.

## Task
Scan the codebase and generate a comprehensive CLAUDE.md file.

## Workflow

### Phase 1: Project Detection
Identify the project type by checking for:
- go.mod → Go project
- package.json → Node/TypeScript
- Cargo.toml → Rust
- pyproject.toml → Python
- Makefile → Build system

### Phase 2: Structure Discovery
Use available tools to discover:
- Entry points (main.go, cmd/, index.ts, app.py)
- Module/package structure
- Configuration files
- Documentation (README.md, docs/)

### Phase 3: Build Commands Analysis
Find build/test/lint commands:
- From Makefile targets
- From package.json scripts
- From CI/CD files (.github/workflows/, Jenkinsfile)
- From go.mod directives

### Phase 4: Architecture Analysis
Map the architecture:
- Module dependencies
- Layer structure (if layered)
- Key interfaces
- Core components

## Output Format

Generate CLAUDE.md with these sections (Markdown format):

# CLAUDE.md

## Build & Development Commands
[Exact commands from build system]

## CLI Usage
[Command examples]

## Architecture Overview
[ASCII diagram or description]

## Module Dependency Matrix
[Table of modules and dependencies]

## Key Interfaces
[List with file paths]

## Configuration
[Config file locations and key settings]

## Development Patterns
[Project-specific patterns]

## Notes
[Important implementation details]

## Instructions
- Use specific commands (not placeholders like "npm run build")
- Use absolute file paths
- Include actual values from config files
- Focus on project-specific details, not generic advice
`