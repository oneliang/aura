// Package intent provides natural language intent recognition for command execution.
package intent

// Source types for recognition results.
const (
	SourceAlias   = "alias"
	SourceCommand = "command"
	SourceKeyword = "keyword"
)

// Confidence levels for recognition results.
const (
	ConfidenceHigh   = 1.0
	ConfidenceMedium = 0.8
	ConfidenceLow    = 0.7
)

// i18n key prefixes for intent keywords and patterns.
const (
	KeyPrefixKeyword = "intent.keyword."
	KeyPrefixPattern = "intent.pattern."
	KeyPrefixConfig  = "intent.config."
)

// Keyword group identifiers (used with KeyPrefixKeyword).
const (
	KeywordCreate  = "create"
	KeywordList    = "list"
	KeywordDelete  = "delete"
	KeywordShow    = "show"
	KeywordClear   = "clear"
	KeywordStats   = "stats"
	KeywordSession = "session"
	KeywordSkill   = "skill"
	KeywordAgent   = "agent"
	KeywordConfig  = "config"
	KeywordMemory  = "memory"
	KeywordExit    = "exit"
	KeywordHelp    = "help"
)

// Pattern identifiers (used with KeyPrefixPattern).
const (
	PatternName = "name"
)

// Config key identifiers (used with KeyPrefixConfig).
const (
	ConfigKeyModel       = "model"
	ConfigKeyTemperature = "temperature"
	ConfigKeyProvider    = "provider"
)
