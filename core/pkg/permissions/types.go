// Package permissions provides multi-level permission control for tools.
package permissions

// PermissionLevel represents the level of permission required for a tool.
type PermissionLevel string

const (
	// PermissionReadOnly - Read-only operations that don't modify state.
	// These tools can be executed without user confirmation.
	// Examples: file_read, file_list, file_search, datetime, calculator
	PermissionReadOnly PermissionLevel = "read"

	// PermissionWrite - Write operations that modify files or data.
	// These tools require user confirmation before execution.
	// Examples: file_write, knowledge_import
	PermissionWrite PermissionLevel = "write"

	// PermissionExecute - Command execution operations that can run arbitrary code.
	// These tools require user confirmation and may have command restrictions.
	// Examples: bash, ssh_exec
	PermissionExecute PermissionLevel = "execute"

	// PermissionAdmin - Administrative operations that require explicit authorization.
	// These tools require explicit session-based authorization.
	// Examples: system configuration, privilege escalation
	PermissionAdmin PermissionLevel = "admin"
)

// PermissionControlLevel represents the control strategy for a permission level.
// This is used in configuration to specify how to handle each permission level.
type PermissionControlLevel string

const (
	// ControlAllow - Auto-approve without confirmation.
	ControlAllow PermissionControlLevel = "allow"
	// ControlAsk - Require user confirmation.
	ControlAsk PermissionControlLevel = "ask"
	// ControlDeny - Block execution.
	ControlDeny PermissionControlLevel = "deny"
)

// ParsePermissionLevel parses a string to PermissionLevel.
// Returns empty string if invalid.
func ParsePermissionLevel(s string) PermissionLevel {
	switch s {
	case "read", "readonly":
		return PermissionReadOnly
	case "write":
		return PermissionWrite
	case "execute", "exec":
		return PermissionExecute
	case "admin":
		return PermissionAdmin
	default:
		return ""
	}
}

// ParsePermissionControlLevel parses a string to PermissionControlLevel.
// Returns ControlAsk as default for invalid values.
func ParsePermissionControlLevel(s string) PermissionControlLevel {
	switch s {
	case "allow":
		return ControlAllow
	case "deny":
		return ControlDeny
	case "ask", "confirm":
		return ControlAsk
	default:
		return ControlAsk // Default to ask for safety
	}
}

// PermissionInheritanceStrategy defines how sub-agents inherit permissions from parent.
type PermissionInheritanceStrategy string

const (
	// PermissionInherit - Fully inherit parent agent's permission manager.
	PermissionInherit PermissionInheritanceStrategy = "inherit"
	// PermissionInheritDowngrade - Inherit but downgrade to specified control level.
	PermissionInheritDowngrade PermissionInheritanceStrategy = "inherit_downgrade"
	// PermissionIndependent - Use independent permission configuration.
	PermissionIndependent PermissionInheritanceStrategy = "independent"
	// PermissionReadonly - Auto readonly mode, only read-only tools allowed.
	// Equivalent to inherit_downgrade with ControlDeny.
	PermissionReadonly PermissionInheritanceStrategy = "readonly"
)

// ParsePermissionInheritanceStrategy parses a string to PermissionInheritanceStrategy.
// Returns PermissionInherit as default for invalid values.
func ParsePermissionInheritanceStrategy(s string) PermissionInheritanceStrategy {
	switch s {
	case "inherit":
		return PermissionInherit
	case "inherit_downgrade", "downgrade":
		return PermissionInheritDowngrade
	case "independent":
		return PermissionIndependent
	case "readonly":
		return PermissionReadonly
	default:
		return PermissionInherit // Default to inherit
	}
}

// String returns the string representation.
func (s PermissionInheritanceStrategy) String() string {
	return string(s)
}

// PermissionDecision represents the decision for a permission request.
type PermissionDecision int

const (
	// DecisionUnknown - No decision made yet.
	DecisionUnknown PermissionDecision = iota
	// DecisionAllow - Permission granted.
	DecisionAllow
	// DecisionDeny - Permission denied.
	DecisionDeny
)

// String returns the string representation of the permission level.
func (p PermissionLevel) String() string {
	return string(p)
}

// RequiresConfirmation returns true if this permission level requires user confirmation.
func (p PermissionLevel) RequiresConfirmation() bool {
	switch p {
	case PermissionWrite, PermissionExecute, PermissionAdmin:
		return true
	default:
		return false
	}
}

// IsHigherThan returns true if this permission level is higher than another.
func (p PermissionLevel) IsHigherThan(other PermissionLevel) bool {
	order := map[PermissionLevel]int{
		PermissionReadOnly: 0,
		PermissionWrite:    1,
		PermissionExecute:  2,
		PermissionAdmin:    3,
	}
	return order[p] > order[other]
}
