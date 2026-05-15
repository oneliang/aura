// Package alias provides command alias management for natural language command matching.
package alias

import (
	"sync"

	"github.com/oneliang/aura/commands/pkg"
)

// Manager manages command aliases and synonyms.
type Manager struct {
	aliases map[string]string // alias -> command
	mutex   sync.RWMutex
}

// NewManager creates a new alias manager with built-in aliases.
func NewManager() *Manager {
	m := &Manager{
		aliases: make(map[string]string),
	}
	m.initBuiltInAliases()
	return m
}

// initBuiltInAliases initializes the built-in command aliases.
func (m *Manager) initBuiltInAliases() {
	// Session aliases
	m.aliases["create session"] = commands.CmdNameSessionCreate
	m.aliases["new session"] = commands.CmdNameSessionCreate
	m.aliases["make session"] = commands.CmdNameSessionCreate
	m.aliases["list sessions"] = commands.CmdNameSessions
	m.aliases["show sessions"] = commands.CmdNameSessions
	m.aliases["delete session"] = commands.CmdNameSessionDelete
	m.aliases["remove session"] = commands.CmdNameSessionDelete

	// Profile aliases
	m.aliases["show profile"] = commands.CmdNameProfileShow
	m.aliases["my profile"] = commands.CmdNameProfileShow

	// Config aliases
	m.aliases["show config"] = commands.CmdNameConfigShow
	m.aliases["current config"] = commands.CmdNameConfigShow
	m.aliases["config path"] = commands.CmdNameConfigPath
	m.aliases["get config"] = commands.CmdNameConfigGet
	m.aliases["what model"] = commands.CmdNameConfigGet
	m.aliases["current model"] = commands.CmdNameConfigGet

	// Memory aliases
	m.aliases["clear memory"] = commands.CmdNameClear
	m.aliases["reset memory"] = commands.CmdNameClear
	m.aliases["clear conversation"] = commands.CmdNameClear
	m.aliases["clear"] = commands.CmdNameClear // standalone clear command
	m.aliases["memory stats"] = commands.CmdNameMemory
	m.aliases["memory status"] = commands.CmdNameMemory
	m.aliases["compact memory"] = commands.CmdNameCompact
	// Chinese aliases
	m.aliases["清空会话"] = commands.CmdNameClear
	m.aliases["清除对话"] = commands.CmdNameClear
	m.aliases["清空记忆"] = commands.CmdNameClear
	m.aliases["清除记忆"] = commands.CmdNameClear

	// Knowledge aliases
	m.aliases["search knowledge"] = commands.CmdNameKnowledgeSearch
	m.aliases["import knowledge"] = commands.CmdNameKnowledgeImport
	m.aliases["knowledge stats"] = commands.CmdNameKnowledgeStats

	// Skills aliases
	m.aliases["list skills"] = commands.CmdNameSkillList
	m.aliases["show skills"] = commands.CmdNameSkillList
	m.aliases["create skill"] = commands.CmdNameSkillCreate
	m.aliases["new skill"] = commands.CmdNameSkillCreate
	m.aliases["delete skill"] = commands.CmdNameSkillDelete
	m.aliases["remove skill"] = commands.CmdNameSkillDelete

	// Agents aliases
	m.aliases["list agents"] = commands.CmdNameAgentList
	m.aliases["show agents"] = commands.CmdNameAgentList
	m.aliases["create agent"] = commands.CmdNameAgentCreate
	m.aliases["new agent"] = commands.CmdNameAgentCreate
	m.aliases["delete agent"] = commands.CmdNameAgentDelete
	m.aliases["remove agent"] = commands.CmdNameAgentDelete

	// Help aliases
	m.aliases["help"] = commands.CmdNameHelp
	m.aliases["what can you do"] = commands.CmdNameHelp

	// Tools aliases
	m.aliases["list tools"] = commands.CmdNameTools
	m.aliases["show tools"] = commands.CmdNameTools

	// Status aliases
	m.aliases["status"] = commands.CmdNameStatus
	m.aliases["system status"] = commands.CmdNameStatus

	// Exit aliases
	m.aliases["quit"] = commands.CmdNameQuit
	m.aliases["exit"] = commands.CmdNameExit
	m.aliases["bye"] = commands.CmdNameQuit
}

// Register registers a new alias.
func (m *Manager) Register(aliasStr, command string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.aliases[aliasStr] = command
	return nil
}

// Unregister removes an alias.
func (m *Manager) Unregister(aliasStr string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.aliases, aliasStr)
	return nil
}

// Resolve resolves an alias to its command.
// Returns the command and true if found, empty string and false otherwise.
func (m *Manager) Resolve(aliasStr string) (string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cmd, ok := m.aliases[aliasStr]
	return cmd, ok
}

// List lists all registered aliases.
func (m *Manager) List() map[string]string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]string, len(m.aliases))
	for k, v := range m.aliases {
		result[k] = v
	}
	return result
}

// ResolveWithPrefix tries to find an alias that starts with the given prefix.
// This is useful for partial matching in natural language input.
// Uses longest-prefix matching to ensure deterministic results when multiple aliases match.
func (m *Manager) ResolveWithPrefix(input string) (string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Try exact match first
	if cmd, ok := m.aliases[input]; ok {
		return cmd, true
	}

	// Find the longest matching prefix (deterministic regardless of map iteration order)
	var bestAlias string
	var bestLen int
	for aliasStr, cmd := range m.aliases {
		if len(input) >= len(aliasStr) && input[:len(aliasStr)] == aliasStr {
			if len(aliasStr) > bestLen {
				bestLen = len(aliasStr)
				bestAlias = cmd
			}
		}
	}
	if bestLen > 0 {
		return bestAlias, true
	}

	return "", false
}
