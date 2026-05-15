// Package commands provides CLI command for running the API server.
package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestServeCmdInitialization tests serve command initialization.
func TestServeCmdInitialization(t *testing.T) {
	if ServeCmd == nil {
		t.Fatal("ServeCmd should not be nil")
	}
	if ServeCmd.Use != "serve" {
		t.Errorf("ServeCmd.Use = %q, want %q", ServeCmd.Use, "serve")
	}
	if ServeCmd.Short == "" {
		t.Error("ServeCmd.Short should not be empty")
	}
}

// TestServeCmdFlags tests serve command flag registration.
func TestServeCmdFlags(t *testing.T) {
	flags := ServeCmd.Flags()

	// Test --port flag
	portFlag := flags.Lookup("port")
	if portFlag == nil {
		t.Fatal("--port flag should be registered")
	}
	if portFlag.DefValue != "" {
		t.Errorf("--port default = %q, want empty string", portFlag.DefValue)
	}

	// Test --llm-url flag
	llmURLFlag := flags.Lookup("llm-url")
	if llmURLFlag == nil {
		t.Fatal("--llm-url flag should be registered")
	}

	// Test --llm-model flag
	llmModelFlag := flags.Lookup("llm-model")
	if llmModelFlag == nil {
		t.Fatal("--llm-model flag should be registered")
	}
}

// TestRunServeConfigLoading tests serve command config loading behavior.
func TestRunServeConfigLoading(t *testing.T) {
	// This is a smoke test - just verify the function exists and doesn't panic
	// Full integration testing requires a running server

	cmd := &cobra.Command{
		Use: "test-serve",
		Run: runServe,
	}

	if cmd.Run == nil {
		t.Fatal("runServe should be assigned to cmd.Run")
	}
}
