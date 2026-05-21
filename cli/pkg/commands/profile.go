// Package commands provides CLI commands for profile management.
package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/oneliang/aura/personality/pkg/importer"
	"github.com/oneliang/aura/personality/pkg/profile"
	"github.com/oneliang/aura/shared/pkg/constants"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	"github.com/spf13/cobra"
)

// ProfileCmd is the root command for profile management.
var ProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage your personal profile",
}

var profileShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the current profile",
	Run:   runProfileShow,
}

var profileInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default profile at ~/.aura/profile.yaml",
	Run:   runProfileInit,
}

var profileImportCmd = &cobra.Command{
	Use:   "import <path>",
	Short: "Import profile data from a text or markdown file",
	Args:  cobra.ExactArgs(1),
	Run:   runProfileImport,
}

func init() {
	ProfileCmd.AddCommand(profileShowCmd)
	ProfileCmd.AddCommand(profileInitCmd)
	ProfileCmd.AddCommand(profileImportCmd)
}

func profilePath() string {
	return ffp.MustAuraHomePath(constants.DefaultProfileFile)
}

func loadOrDefaultProfile() *profile.Profile {
	p, err := profile.Load(profilePath())
	if err != nil {
		return profile.DefaultProfile()
	}
	return p
}

func runProfileShow(cmd *cobra.Command, args []string) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Use CommandProvider for profile show
	result, err := cmdCtx.CommandProvider.Execute(cmd.Context(), "command_profile_show", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runProfileInit(cmd *cobra.Command, args []string) {
	path := profilePath()
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Profile already exists at %s\n", path)
		fmt.Println("Use 'aura profile show' to view it.")
		return
	}

	p := profile.DefaultProfile()
	if err := p.Save(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving profile: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created default profile at %s\n", path)
	fmt.Println("Edit it to personalise Aura!")
}

func runProfileImport(cmd *cobra.Command, args []string) {
	path := args[0]

	// Profile import requires file I/O and merging logic, handle at CLI layer
	imp := importer.New()

	extracted, err := imp.ImportFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Merge with existing profile
	existing := loadOrDefaultProfile()
	mergeProfile(existing, extracted)

	if err := existing.Save(profilePath()); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Profile updated from %s\n", path)
	if existing.BasicInfo.Name != "" && existing.BasicInfo.Name != "User" {
		fmt.Printf("  Name: %s\n", existing.BasicInfo.Name)
	}
	if len(extracted.Skills) > 0 {
		names := make([]string, 0, len(extracted.Skills))
		for _, s := range extracted.Skills {
			names = append(names, s.Name)
		}
		fmt.Printf("  Skills added: %s\n", strings.Join(names, ", "))
	}
}

// mergeProfile merges src into dst (non-destructive).
func mergeProfile(dst, src *profile.Profile) {
	if src.BasicInfo.Name != "" && src.BasicInfo.Name != "User" {
		dst.BasicInfo.Name = src.BasicInfo.Name
	}
	if src.BasicInfo.Occupation != "" {
		dst.BasicInfo.Occupation = src.BasicInfo.Occupation
	}
	if src.BasicInfo.Location != "" {
		dst.BasicInfo.Location = src.BasicInfo.Location
	}
	if src.Background != "" {
		dst.Background = src.Background
	}
	dst.Skills = append(dst.Skills, src.Skills...)
	dst.Experiences = append(dst.Experiences, src.Experiences...)
	dst.Preferences = append(dst.Preferences, src.Preferences...)
}
