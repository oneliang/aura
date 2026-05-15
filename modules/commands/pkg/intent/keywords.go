package intent

import (
	"strings"
	"sync"

	"github.com/oneliang/aura/shared/pkg/i18n"
)

// KeywordLoader loads and caches keywords from i18n.
type KeywordLoader struct {
	keywords map[string][]string
	patterns map[string][]string
	configs  []string
	once     sync.Once
}

// NewKeywordLoader creates a new keyword loader.
func NewKeywordLoader() *KeywordLoader {
	return &KeywordLoader{
		keywords: make(map[string][]string),
		patterns: make(map[string][]string),
	}
}

// Load initializes the keyword cache from i18n.
func (l *KeywordLoader) Load() {
	l.once.Do(func() {
		// Load action keywords
		l.loadKeywordGroup(KeywordCreate)
		l.loadKeywordGroup(KeywordList)
		l.loadKeywordGroup(KeywordDelete)
		l.loadKeywordGroup(KeywordShow)
		l.loadKeywordGroup(KeywordClear)
		l.loadKeywordGroup(KeywordStats)

		// Load entity keywords
		l.loadKeywordGroup(KeywordSession)
		l.loadKeywordGroup(KeywordSkill)
		l.loadKeywordGroup(KeywordAgent)
		l.loadKeywordGroup(KeywordConfig)
		l.loadKeywordGroup(KeywordMemory)

		// Load command keywords
		l.loadKeywordGroup(KeywordExit)
		l.loadKeywordGroup(KeywordHelp)

		// Load patterns
		l.loadPatternGroup(PatternName)

		// Load config keys
		l.loadConfigKeys()
	})
}

// loadKeywordGroup loads a keyword group from i18n.
func (l *KeywordLoader) loadKeywordGroup(group string) {
	key := KeyPrefixKeyword + group
	value := i18n.T(key)
	if value != key {
		l.keywords[group] = strings.Split(value, ",")
	}
}

// loadPatternGroup loads a pattern group from i18n.
func (l *KeywordLoader) loadPatternGroup(group string) {
	key := KeyPrefixPattern + group
	value := i18n.T(key)
	if value != key {
		l.patterns[group] = strings.Split(value, ",")
	}
}

// loadConfigKeys loads config keys.
func (l *KeywordLoader) loadConfigKeys() {
	// Config keys are fixed identifiers, not translated
	l.configs = []string{
		ConfigKeyModel,
		ConfigKeyTemperature,
		ConfigKeyProvider,
	}
}

// GetKeywords returns keywords for a group.
func (l *KeywordLoader) GetKeywords(group string) []string {
	l.Load()
	if kw, ok := l.keywords[group]; ok {
		return kw
	}
	return nil
}

// GetPatterns returns patterns for a group.
func (l *KeywordLoader) GetPatterns(group string) []string {
	l.Load()
	if p, ok := l.patterns[group]; ok {
		return p
	}
	return nil
}

// GetConfigKeys returns available config keys.
func (l *KeywordLoader) GetConfigKeys() []string {
	l.Load()
	return l.configs
}

// ContainsKeyword checks if input contains any keyword from the group.
func (l *KeywordLoader) ContainsKeyword(input, group string) bool {
	keywords := l.GetKeywords(group)
	for _, kw := range keywords {
		if strings.Contains(input, kw) {
			return true
		}
	}
	return false
}

// ExtractName extracts a name from input using patterns.
func (l *KeywordLoader) ExtractName(input string) string {
	patterns := l.GetPatterns(PatternName)
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if idx := strings.Index(input, p); idx != -1 {
			name := strings.TrimSpace(input[idx+len(p):])
			// Take first word as name
			if spaceIdx := strings.Index(name, " "); spaceIdx != -1 {
				name = name[:spaceIdx]
			}
			return name
		}
	}
	return ""
}

// ExtractConfigKey extracts a config key from input.
func (l *KeywordLoader) ExtractConfigKey(input string) string {
	for _, key := range l.GetConfigKeys() {
		if strings.Contains(input, key) {
			return key
		}
	}
	return ""
}

// Global keyword loader instance.
var defaultLoader = NewKeywordLoader()

// GetKeywordLoader returns the default keyword loader.
func GetKeywordLoader() *KeywordLoader {
	return defaultLoader
}
