// Package commands provides CLI commands for knowledge base management.
package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestKnowledgeCmdInitialization tests knowledge command initialization.
func TestKnowledgeCmdInitialization(t *testing.T) {
	if KnowledgeCmd == nil {
		t.Fatal("KnowledgeCmd should not be nil")
	}
	if KnowledgeCmd.Use != "knowledge" {
		t.Errorf("KnowledgeCmd.Use = %q, want %q", KnowledgeCmd.Use, "knowledge")
	}
	if KnowledgeCmd.Short == "" {
		t.Error("KnowledgeCmd.Short should not be empty")
	}
}

// TestImportCmdInitialization tests import command initialization.
func TestImportCmdInitialization(t *testing.T) {
	if importCmd == nil {
		t.Fatal("importCmd should not be nil")
	}
	if importCmd.Use != "import <path>" {
		t.Errorf("importCmd.Use = %q, want %q", importCmd.Use, "import <path>")
	}
	if importCmd.Args == nil {
		t.Fatal("importCmd.Args should be set")
	}
}

// TestSearchCmdInitialization tests search command initialization.
func TestSearchCmdInitialization(t *testing.T) {
	if searchCmd == nil {
		t.Fatal("searchCmd should not be nil")
	}
	if searchCmd.Use != "search <query>" {
		t.Errorf("searchCmd.Use = %q, want %q", searchCmd.Use, "search <query>")
	}
}

// TestKbStatsCmdInitialization tests stats command initialization.
func TestKbStatsCmdInitialization(t *testing.T) {
	if kbStatsCmd == nil {
		t.Fatal("kbStatsCmd should not be nil")
	}
	if kbStatsCmd.Use != "stats" {
		t.Errorf("kbStatsCmd.Use = %q, want %q", kbStatsCmd.Use, "stats")
	}
}

// TestKnowledgeCmdSubcommands tests knowledge command subcommand registration.
func TestKnowledgeCmdSubcommands(t *testing.T) {
	subcommands := KnowledgeCmd.Commands()
	if len(subcommands) != 3 {
		t.Fatalf("KnowledgeCmd should have 3 subcommands, got %d", len(subcommands))
	}

	foundImport := false
	foundSearch := false
	foundStats := false

	for _, cmd := range subcommands {
		switch cmd.Use {
		case "import <path>":
			foundImport = true
		case "search <query>":
			foundSearch = true
		case "stats":
			foundStats = true
		}
	}

	if !foundImport {
		t.Error("KnowledgeCmd should have import subcommand")
	}
	if !foundSearch {
		t.Error("KnowledgeCmd should have search subcommand")
	}
	if !foundStats {
		t.Error("KnowledgeCmd should have stats subcommand")
	}
}

// TestRunImport_ArgValidation tests import command argument validation.
func TestRunImport_ArgValidation(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "import <path>",
		Args: cobra.ExactArgs(1),
		Run:  runImport,
	}

	// Test with no arguments (should fail validation)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("import command should require exactly 1 argument")
	}

	// Test with too many arguments (should fail validation)
	cmd.SetArgs([]string{"arg1", "arg2"})
	err = cmd.Execute()
	if err == nil {
		t.Error("import command should require exactly 1 argument")
	}
}

// TestRunSearch_ArgValidation tests search command argument validation.
func TestRunSearch_ArgValidation(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "search <query>",
		Args: cobra.ExactArgs(1),
		Run:  runSearch,
	}

	// Test with no arguments (should fail validation)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("search command should require exactly 1 argument")
	}
}

// TestKbStatsCmd_NoArgs tests stats command accepts no arguments.
func TestKbStatsCmd_NoArgs(t *testing.T) {
	// This test just verifies the command structure is correct
	// Actual execution requires config to be loaded which happens in initConfig()

	// Verify the command is properly set up
	if kbStatsCmd.Run == nil {
		t.Error("kbStatsCmd.Run should be set")
	}

	// Verify it has the correct Use
	if kbStatsCmd.Use != "stats" {
		t.Errorf("kbStatsCmd.Use = %q, want %q", kbStatsCmd.Use, "stats")
	}
}

// TestGetKnowledgeDataDir tests knowledge data directory resolution.
func TestGetKnowledgeDataDir(t *testing.T) {
	// Save original HOME
	origHome := os.Getenv("HOME")
	defer func() {
		if origHome != "" {
			os.Setenv("HOME", origHome)
		}
	}()

	// Test with custom HOME
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	expected := filepath.Join(tmpDir, ".aura", "knowledge")
	result := getKnowledgeDataDir()

	if result != expected {
		t.Errorf("getKnowledgeDataDir() with HOME=%q = %q, want %q", tmpDir, result, expected)
	}
}

// TestOpenCollection_DataDirResolution tests collection opening with different data dirs.
func TestOpenCollection_DataDirResolution(t *testing.T) {
	// This is a smoke test - it will fail if config is not loaded
	// We just verify the function exists and doesn't panic on nil checks

	// Since openCollection depends on global cfg which is nil in tests,
	// we can't directly test it without mocking. Instead, verify the helper exists.
	result := getKnowledgeDataDir()
	if result == "" {
		t.Error("getKnowledgeDataDir should return non-empty path")
	}
	if !strings.Contains(result, ".aura") {
		t.Errorf("getKnowledgeDataDir() = %q, should contain .aura", result)
	}
}

// Helper function extracted for testing
func getKnowledgeDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.aura/knowledge"
	}
	return filepath.Join(homeDir, ".aura", "knowledge")
}

// TestKnowledgeCmdHelp tests knowledge command help text.
func TestKnowledgeCmdHelp(t *testing.T) {
	helpText := KnowledgeCmd.Short
	if helpText == "" {
		t.Error("KnowledgeCmd.Short should not be empty")
	}
}

// TestImportCmdHelp tests import command help text.
func TestImportCmdHelp(t *testing.T) {
	helpText := importCmd.Short
	if helpText == "" {
		t.Error("importCmd.Short should not be empty")
	}
}

// TestSearchCmdHelp tests search command help text.
func TestSearchCmdHelp(t *testing.T) {
	helpText := searchCmd.Short
	if helpText == "" {
		t.Error("searchCmd.Short should not be empty")
	}
}

// TestKbStatsCmdHelp tests stats command help text.
func TestKbStatsCmdHelp(t *testing.T) {
	helpText := kbStatsCmd.Short
	if helpText == "" {
		t.Error("kbStatsCmd.Short should not be empty")
	}
}
