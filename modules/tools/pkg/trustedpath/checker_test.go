package trustedpath

import "testing"

func TestNopChecker(t *testing.T) {
	checker := NopChecker()

	// nopChecker should always return true
	if !checker.IsTrustedPath("/some/path") {
		t.Error("NopChecker should always return true")
	}

	if !checker.IsTrustedPath("~/.ssh/id_rsa") {
		t.Error("NopChecker should always return true for any path")
	}

	if !checker.IsTrustedPath("/etc/passwd") {
		t.Error("NopChecker should always return true for any path")
	}
}
