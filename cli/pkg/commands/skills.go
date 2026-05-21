// Package commands provides CLI commands for skills management.
package commands

import (
	"fmt"
	"os"

	"github.com/oneliang/aura/skill/pkg/loader"
	"github.com/oneliang/aura/skill/pkg/skill"
	"github.com/spf13/cobra"
)

// SkillsCmd displays loaded skills.
var SkillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "List loaded skills",
	Long:  `Display all skills loaded from ~/.aura/skills/ directory.`,
	Run:   runSkills,
}

func runSkills(cmd *cobra.Command, args []string) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Load config if not already loaded
	cfg := cmdCtx.Config
	if cfg == nil {
		var err error
		cfg, err = cmdCtx.ConfigLoader.Load("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	// Create skills loader and load skills
	var skillsList []skill.Skill
	if cfg.Skills.Enabled && len(cfg.Skills.Directories) > 0 {
		skillLoader := loader.NewLoader(cfg.Skills.Directories)
		var err error
		skillsList, err = skillLoader.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading skills: %v\n", err)
			os.Exit(1)
		}
	}

	// Format and display skills
	if len(skillsList) == 0 {
		fmt.Println("No skills loaded. Add skills to ~/.aura/skills/ directory.")
		return
	}

	fmt.Println("Loaded Skills:")
	for _, s := range skillsList {
		fmt.Printf("  - %s: %s\n", s.Name, s.Description)
	}
}
