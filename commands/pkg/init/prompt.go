// Package init provides the init prompt template.
package init

import "fmt"

// BuildInitPrompt builds the prompt for workspace analysis.
func BuildInitPrompt(cwd string) string {
	return fmt.Sprintf(`
You are a workspace analyst. Explore the workspace at %s and generate AURA.md.

## IMPORTANT: Explore Independently
- DO NOT rely on existing documentation (README.md etc.) - they may be outdated
- Use tools (glob, grep, file_read) to explore the ACTUAL structure and content
- Analyze files yourself to understand the workspace purpose
- Generate documentation based on YOUR exploration, not provided content

## Task
Explore the workspace using tools and generate AURA.md for AI assistants.

## Exploration Steps
1. Use glob to observe directory structure and file type distribution
2. Use grep to understand patterns (code, documents, configs)
3. Read key files (README, configs, entry points, main documents)
4. Determine workspace type: code project? document library? notes? mixed? other?

## Output Requirements
Generate AURA.md content appropriate for the workspace type:

- **If code project**: Include build/test commands, architecture overview, entry points, development patterns
- **If document library**: Include document organization, key documents, purpose, workflow
- **If mixed**: Combine relevant information from both
- **If other**: Organize content based on actual structure and purpose

## Output Format
- Output ONLY the AURA.md markdown content
- Start with the workspace/project name as heading
- Be concise and actionable
- Include file paths where relevant
`, cwd)
}

// BuildFullInitPrompt builds the complete prompt with project files.
func BuildFullInitPrompt(cwd string, projectContent string) string {
	return BuildInitPrompt(cwd) + "\n\n## Project Files\n\n" + projectContent + "\n\n## Output\n\nGenerate the AURA.md content now:"
}

// InitSystemPrompt is the system prompt for init analysis.
const InitSystemPrompt = `You are a workspace analyst generating AURA.md for AI assistants.

CRITICAL RULES:
1. Explore workspace using tools - DO NOT rely on provided file content
2. Existing docs (README.md) may be outdated - analyze actual content
3. Use glob, grep, file_read to discover structure yourself
4. Determine workspace type and generate appropriate content
5. Be concise and actionable

Output ONLY the AURA.md markdown based on YOUR exploration.
`