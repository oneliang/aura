// Package init provides the init prompt template.
package init

import "fmt"

// BuildInitPrompt builds the prompt for codebase analysis.
func BuildInitPrompt(cwd string) string {
	return fmt.Sprintf(`
You are a codebase analyst. Explore the project at %s and generate AURA.md.

## IMPORTANT: Explore Independently
- DO NOT rely on existing documentation (README.md etc.) - they may be outdated
- Use tools (glob, grep, file_read) to explore the ACTUAL code structure
- Analyze files yourself to understand current architecture
- Generate documentation based on YOUR exploration, not provided content

## Task
Explore the codebase using tools and generate AURA.md for AI assistants.

## Exploration Steps
1. Use glob to find key files (Makefile, go.mod, package.json, configs)
2. Use grep to understand code patterns and architecture
3. Read main entry points and config files
4. Discover architecture by exploring directory structure

## What AI Assistants Need
1. Exact build/test/lint commands (from Makefile, configs)
2. Architecture patterns (from actual code exploration)
3. Key file locations and their purposes
4. Code style and naming conventions
5. Important details for working with this codebase

## Required Sections
1. Build & Development Commands - use ACTUAL commands from files
2. Architecture Overview - describe what YOU discovered
3. Configuration - file locations and key settings
4. Development Patterns - conventions YOU observed
5. Notes - important details YOU found

## Output Format
- Output ONLY the AURA.md markdown content
- Start directly with a project name heading
- Be concise and actionable
- Include file paths where relevant
`, cwd)
}

// BuildFullInitPrompt builds the complete prompt with project files.
func BuildFullInitPrompt(cwd string, projectContent string) string {
	return BuildInitPrompt(cwd) + "\n\n## Project Files\n\n" + projectContent + "\n\n## Output\n\nGenerate the AURA.md content now:"
}

// InitSystemPrompt is the system prompt for init analysis.
const InitSystemPrompt = `You are a codebase analyst generating AURA.md for AI assistants.

CRITICAL RULES:
1. Explore codebase using tools - DO NOT rely on provided file content
2. Existing docs (README.md) may be outdated - analyze actual code
3. Use glob, grep, file_read to discover architecture yourself
4. Focus on what AI needs: commands, patterns, file locations
5. Be concise and actionable

Output ONLY the AURA.md markdown based on YOUR exploration.
`