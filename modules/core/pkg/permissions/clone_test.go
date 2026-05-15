package permissions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionInheritanceStrategy_String(t *testing.T) {
	tests := []struct {
		name     string
		strategy PermissionInheritanceStrategy
		expected string
	}{
		{"inherit", PermissionInherit, "inherit"},
		{"inherit_downgrade", PermissionInheritDowngrade, "inherit_downgrade"},
		{"independent", PermissionIndependent, "independent"},
		{"readonly", PermissionReadonly, "readonly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.strategy.String())
		})
	}
}

func TestParsePermissionInheritanceStrategy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected PermissionInheritanceStrategy
	}{
		{"inherit", "inherit", PermissionInherit},
		{"inherit_downgrade", "inherit_downgrade", PermissionInheritDowngrade},
		{"independent", "independent", PermissionIndependent},
		{"readonly", "readonly", PermissionReadonly},
		{"empty", "", PermissionInherit},       // default
		{"invalid", "unknown", PermissionInherit}, // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePermissionInheritanceStrategy(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManager_CloneWithDowngrade(t *testing.T) {
	// Create parent manager with ControlAsk default
	defaultCfg := DefaultPermissionConfig()
	parentCfg := &PermissionConfig{
		DefaultLevel: ControlAsk,
		Tools: map[string]PermissionControlLevel{
			"file_read":   ControlAllow,
			"file_write":  ControlAsk,
			"bash":        ControlAsk,
			"ssh_exec":    ControlDeny,
		},
		ShellRestrictions: defaultCfg.ShellRestrictions,
		SSHRestrictions:   defaultCfg.SSHRestrictions,
	}

	parentMgr, err := NewManager(parentCfg)
	require.NoError(t, err)

	// Test 1: Clone with downgrade to ControlDeny for all tools
	t.Run("downgrade_to_deny", func(t *testing.T) {
		clonedMgr, err := parentMgr.CloneWithDowngrade(ControlDeny)
		require.NoError(t, err)

		// Cloned manager should have ControlDeny default
		assert.Equal(t, ControlDeny, clonedMgr.GetConfig().DefaultLevel)

		// All tools should be denied except read-only
		allowed, requiresConfirm, _ := clonedMgr.CheckPermission(nil, "file_read", nil)
		assert.True(t, allowed) // read-only should still be allowed
		assert.False(t, requiresConfirm)

		allowed, requiresConfirm, _ = clonedMgr.CheckPermission(nil, "file_write", nil)
		assert.False(t, allowed) // write should be denied
		assert.False(t, requiresConfirm)

		allowed, requiresConfirm, _ = clonedMgr.CheckPermission(nil, "bash", nil)
		assert.False(t, allowed) // execute should be denied
		assert.False(t, requiresConfirm)
	})

	// Test 2: Clone with downgrade to ControlAsk (more restrictive than parent's ControlAsk)
	t.Run("downgrade_to_ask_from_ask", func(t *testing.T) {
		// Parent already has ControlAsk, downgrade should keep ControlAsk
		clonedMgr, err := parentMgr.CloneWithDowngrade(ControlAsk)
		require.NoError(t, err)

		assert.Equal(t, ControlAsk, clonedMgr.GetConfig().DefaultLevel)

		// file_read should still be allowed (parent override)
		allowed, requiresConfirm, _ := clonedMgr.CheckPermission(nil, "file_read", nil)
		assert.True(t, allowed)
		assert.False(t, requiresConfirm)
	})

	// Test 3: Clone should not modify parent
	t.Run("clone_isolation", func(t *testing.T) {
		clonedMgr, err := parentMgr.CloneWithDowngrade(ControlDeny)
		require.NoError(t, err)

		// Modify cloned manager's config
		clonedCfg := clonedMgr.GetConfig()
		clonedCfg.Tools["new_tool"] = ControlAllow

		// Parent should not have new_tool
		parentCfg := parentMgr.GetConfig()
		_, exists := parentCfg.Tools["new_tool"]
		assert.False(t, exists) // Key should not exist in parent
	})

	// Test 4: Clone should preserve trusted dirs
	t.Run("preserve_trusted_dirs", func(t *testing.T) {
		err := parentMgr.AddTrustedDir("/tmp/test")
		require.NoError(t, err)

		clonedMgr, err := parentMgr.CloneWithDowngrade(ControlDeny)
		require.NoError(t, err)

		// Cloned should have same trusted dirs
		parentDirs := parentMgr.GetTrustedDirs()
		clonedDirs := clonedMgr.GetTrustedDirs()
		assert.Equal(t, parentDirs, clonedDirs)
	})
}

func TestManager_CloneWithDowngrade_EdgeCases(t *testing.T) {
	t.Run("nil_config", func(t *testing.T) {
		mgr, err := NewManager(nil)
		require.NoError(t, err)

		clonedMgr, err := mgr.CloneWithDowngrade(ControlDeny)
		require.NoError(t, err)
		assert.Equal(t, ControlDeny, clonedMgr.GetConfig().DefaultLevel)
	})

	t.Run("downgrade_from_allow", func(t *testing.T) {
		// Parent has ControlAllow for everything
		defaultCfg := DefaultPermissionConfig()
		parentCfg := &PermissionConfig{
			DefaultLevel:      ControlAllow,
			Tools:             map[string]PermissionControlLevel{},
			ShellRestrictions: defaultCfg.ShellRestrictions,
			SSHRestrictions:   defaultCfg.SSHRestrictions,
		}

		parentMgr, err := NewManager(parentCfg)
		require.NoError(t, err)

		// Downgrade to ControlAsk
		clonedMgr, err := parentMgr.CloneWithDowngrade(ControlAsk)
		require.NoError(t, err)

		// Cloned should have ControlAsk
		assert.Equal(t, ControlAsk, clonedMgr.GetConfig().DefaultLevel)

		// All tools should require confirmation
		allowed, requiresConfirm, _ := clonedMgr.CheckPermission(nil, "file_write", nil)
		assert.True(t, allowed)
		assert.True(t, requiresConfirm)
	})
}

// TestManager_CloneWithDowngrade_ReadOnlyToolPreservation verifies that read-only tools
// that were explicitly set as ControlAllow in parent config remain ControlAllow
// even after downgrade to ControlDeny.
func TestManager_CloneWithDowngrade_ReadOnlyToolPreservation(t *testing.T) {
	defaultCfg := DefaultPermissionConfig()
	parentCfg := &PermissionConfig{
		DefaultLevel: ControlAllow,
		Tools: map[string]PermissionControlLevel{
			// Read-only tools explicitly allowed in parent
			"file_read":     ControlAllow,
			"web_fetch":     ControlAllow,
			"code_navigate": ControlAllow,
			"datetime":      ControlAllow,
			// Write tools explicitly allowed in parent
			"file_write": ControlAllow,
			"bash":       ControlAllow,
		},
		ShellRestrictions: defaultCfg.ShellRestrictions,
		SSHRestrictions:   defaultCfg.SSHRestrictions,
	}

	parentMgr, err := NewManager(parentCfg)
	require.NoError(t, err)

	// Downgrade to most restrictive level
	clonedMgr, err := parentMgr.CloneWithDowngrade(ControlDeny)
	require.NoError(t, err)

	// Read-only tools that were ControlAllow in parent should remain allowed
	readOnlyAllowed := []string{"file_read", "web_fetch", "code_navigate", "datetime"}
	for _, tool := range readOnlyAllowed {
		allowed, requiresConfirm, _ := clonedMgr.CheckPermission(nil, tool, nil)
		assert.True(t, allowed, "read-only tool %s should remain allowed (was ControlAllow in parent)", tool)
		assert.False(t, requiresConfirm, "read-only tool %s should not require confirmation", tool)
	}

	// Write operations should be denied (not read-only, downgraded)
	writeTools := []string{"file_write", "bash"}
	for _, tool := range writeTools {
		allowed, requiresConfirm, _ := clonedMgr.CheckPermission(nil, tool, nil)
		assert.False(t, allowed, "write tool %s should be denied after downgrade", tool)
		assert.False(t, requiresConfirm, "write tool %s should not require confirmation when denied", tool)
	}

	// Tools NOT in parent's Tools map use DefaultLevel (ControlDeny after downgrade)
	// This is expected behavior - only explicitly-set tools are preserved
	notSetTools := []string{"file_list", "glob", "grep", "knowledge_search"}
	for _, tool := range notSetTools {
		allowed, _, _ := clonedMgr.CheckPermission(nil, tool, nil)
		assert.False(t, allowed, "tool %s not in parent config should use DefaultLevel (ControlDeny)", tool)
	}
}