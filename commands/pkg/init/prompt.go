// Package init provides the init prompt template.
package init

import "fmt"

// ============================================================================
// Role and System Prompts
// ============================================================================

// InitSystemPrompt is the system prompt for init analysis.
const InitSystemPrompt = `You are a workspace analyst generating AURA.md with YAML frontmatter for AI assistants.

CRITICAL RULES:
1. Explore workspace using tools - DO NOT rely on provided file content
2. Determine project type FIRST: code, documents, mixed, or other
3. Choose appropriate sections based on type
4. Use file_write tool to save AURA.md - DO NOT output conversational text
5. DO NOT say "I've generated" or "Here is the file" - just use file_write

OUTPUT STRUCTURE:
- YAML frontmatter starting with ---
- Universal sections: Project Summary, Structure Overview
- Type-specific sections based on project type

Use file_write tool with path "AURA.md" to save the generated content.
`

// rolePrompt defines the analyst role.
const rolePrompt = `You are a workspace analyst. Generate AURA.md with YAML frontmatter for AI assistants.`

// ============================================================================
// Project Types
// ============================================================================

// projectTypeTable defines project type classification indicators.
const projectTypeTable = `
| Type | Indicators |
|------|------------|
| go-project | *.go files, go.mod |
| js-project | *.js/*.ts files, package.json |
| py-project | *.py files, pyproject.toml/requirements.txt |
| rust-project | *.rs files, Cargo.toml |
| documents | Mostly *.md files, no source code |
| mixed | Multiple languages or code + docs hybrid |
| other | Cannot classify into above |
`

// ============================================================================
// Exploration Steps
// ============================================================================

// step1DetermineType instructs to classify the workspace.
const step1DetermineType = `## Step 1: Determine Project Type

First, explore and classify the workspace into one of these types:`

// step2UniversalExploration defines common exploration for all project types.
const step2UniversalExploration = `## Step 2: Universal Exploration (ALL types)

Use glob and file_read to:
- [ ] List top-level directories and understand purpose
- [ ] Identify file type distribution
- [ ] Read README.md or main documentation (VERIFY accuracy)
- [ ] Understand workspace purpose and audience`

// step3CodeExploration defines exploration for code projects.
const step3CodeExploration = `### For Code Projects (go/js/py/rust/mixed with code)
Use grep and file_read to:
- [ ] Find build commands (Makefile, package.json scripts, etc.)
- [ ] Find entry points (main.go, index.js, main.py)
- [ ] Identify core modules/packages
- [ ] Analyze design patterns`

// step3DocumentExploration defines exploration for document projects.
const step3DocumentExploration = `### For Document Projects
Use glob and file_read to:
- [ ] Identify document categories/topics
- [ ] Find key documents (guides, references)
- [ ] Understand document organization
- [ ] Note writing conventions`

// step3MixedExploration defines exploration for mixed/other projects.
const step3MixedExploration = `### For Mixed/Other Projects
- [ ] Combine relevant exploration from above
- [ ] Focus on most important aspects`

// ============================================================================
// Output Format
// ============================================================================

// yamlFrontmatterTemplate defines the required YAML structure.
const yamlFrontmatterTemplate = `## YAML Frontmatter (Required)

---
name: [identifier]
type: [go-project|js-project|py-project|rust-project|documents|mixed|other]
language: [primary-language-or-mixed]
---`

// universalSections defines sections required for all project types.
const universalSections = `## Universal Sections (Required)

### Project Summary
One paragraph describing purpose (for whom, what problem it solves).

### Structure Overview
High-level organization: key directories and their roles.`

// codeProjectSections defines sections for code projects.
const codeProjectSections = `### For Code Projects: Add These Sections

#### Commands
Build/test/lint commands if they exist.

#### Key Files
Entry points and core files.

#### Patterns
Design patterns and conventions.`

// documentProjectSections defines sections for document projects.
const documentProjectSections = `### For Document Projects: Add These Sections

#### Document Categories
How documents are organized (by topic, by audience, etc.)

#### Key Documents
Important documents to read first.

#### Writing Conventions
Style guidelines if observable.`

// mixedProjectSections defines sections for mixed/other projects.
const mixedProjectSections = `### For Mixed/Other Projects

Add sections based on what's most relevant from the above.`

// ============================================================================
// Output Rules
// ============================================================================

// outputRules defines the final output requirements.
const outputRules = `## Output Rules

1. Use file_write tool with path "AURA.md" to save the generated content
2. DO NOT output conversational text like "I've generated" or "Here is the file"
3. Start content with YAML frontmatter: ---
4. Determine type FIRST, then choose appropriate sections
5. Be concise - each section 2-5 items max
6. Paths relative to project root
7. No placeholder text - use real discovered content`

// ============================================================================
// Prompt Builder
// ============================================================================

// BuildInitPrompt builds the prompt for workspace analysis.
func BuildInitPrompt(cwd string) string {
	return fmt.Sprintf(`
%s

## Output Format Structure

Your output MUST follow this structure:

1. **YAML frontmatter** (required) - 3 fields enclosed by ---
2. **Universal sections** (required for ALL types) - Project Summary, Structure Overview
3. **Type-specific sections** (based on detected project type) - adapt to actual content

%s
%s

## Step 3: Type-Specific Exploration

%s

%s

%s

%s

%s

%s

%s

%s

%s

%s

%s
`, cwd,
		rolePrompt,
		step1DetermineType, projectTypeTable,
		step2UniversalExploration,
		step3CodeExploration, step3DocumentExploration, step3MixedExploration,
		yamlFrontmatterTemplate,
		universalSections,
		codeProjectSections, documentProjectSections, mixedProjectSections,
		outputRules)
}

// BuildFullInitPrompt builds the complete prompt with project files.
func BuildFullInitPrompt(cwd string, projectContent string) string {
	return BuildInitPrompt(cwd) + "\n\n## Project Files\n\n" + projectContent + "\n\n## Output\n\nGenerate the AURA.md content now:"
}