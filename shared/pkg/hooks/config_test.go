package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestHooksConfigPreResponseParsing(t *testing.T) {
	// Test that PreResponse hook can be parsed from YAML
	yamlConfig := `
enabled: true
PreResponse:
  - matcher: ".*"
    hooks:
      - type: command
        command: "./scripts/verify.sh"
        timeout: 30000
`

	var config HooksConfig
	err := yaml.Unmarshal([]byte(yamlConfig), &config)

	assert.NoError(t, err, "PreResponse YAML should parse without error")
	assert.True(t, config.Enabled, "Enabled should be true")
	assert.Len(t, config.PreResponse, 1, "PreResponse should have 1 entry")
	assert.Equal(t, ".*", config.PreResponse[0].Matcher, "Matcher should match")
	assert.Len(t, config.PreResponse[0].Hooks, 1, "Should have 1 hook")
	assert.Equal(t, "command", config.PreResponse[0].Hooks[0].Type, "Hook type should be command")
	assert.Equal(t, "./scripts/verify.sh", config.PreResponse[0].Hooks[0].Command, "Command should match")
	assert.Equal(t, 30000, config.PreResponse[0].Hooks[0].Timeout, "Timeout should match")
}

func TestHooksConfigPreResponseEmpty(t *testing.T) {
	// Test that empty PreResponse config is valid
	yamlConfig := `
enabled: true
`

	var config HooksConfig
	err := yaml.Unmarshal([]byte(yamlConfig), &config)

	assert.NoError(t, err, "Empty PreResponse YAML should parse without error")
	assert.Nil(t, config.PreResponse, "PreResponse should be nil when not configured")
}