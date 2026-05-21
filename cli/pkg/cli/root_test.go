package cli

import (
	"testing"
)

// TestRootCmdInitialization tests root command initialization.
func TestRootCmdInitialization(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}
	if rootCmd.Use != "aura [message]" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "aura [message]")
	}
}
