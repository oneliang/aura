// Package commands provides CLI commands for user management.
package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/oneliang/aura/personality/pkg/profile"
	"github.com/oneliang/aura/shared/pkg/user"
	"github.com/spf13/cobra"
)

// UserCmd is the root command for user management.
var UserCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage multi-user profiles",
	Long:  `Manage multiple user profiles, each with independent profile, knowledge base, and permissions.`,
}

var userCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(1),
	Run:   runUserCreate,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Run:   runUserList,
}

var userSwitchCmd = &cobra.Command{
	Use:   "switch <user-id>",
	Short: "Switch to a different user",
	Args:  cobra.ExactArgs(1),
	Run:   runUserSwitch,
}

var userCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current user",
	Run:   runUserCurrent,
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <user-id>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserDelete,
}

var userTokenCmd = &cobra.Command{
	Use:   "token <user-id>",
	Short: "Show user's API token",
	Args:  cobra.ExactArgs(1),
	Run:   runUserToken,
}

var userInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize multi-user system with first user",
	Args:  cobra.ExactArgs(1),
	Run:   runUserInit,
}

func init() {
	UserCmd.AddCommand(userCreateCmd)
	UserCmd.AddCommand(userListCmd)
	UserCmd.AddCommand(userSwitchCmd)
	UserCmd.AddCommand(userCurrentCmd)
	UserCmd.AddCommand(userDeleteCmd)
	UserCmd.AddCommand(userTokenCmd)
	UserCmd.AddCommand(userInitCmd)
}

func runUserInit(cmd *cobra.Command, args []string) {
	name := args[0]

	// Check if users.yaml already exists
	usersCfg, err := user.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(usersCfg.Definitions) > 0 {
		fmt.Println("Multi-user system already initialized.")
		fmt.Println("Use 'aura user create <name>' to add more users.")
		return
	}

	// Create user manager
	mgr, err := user.NewManager(usersCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user manager: %v\n", err)
		os.Exit(1)
	}

	// Create user
	newUser, err := mgr.CreateUser(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user: %v\n", err)
		os.Exit(1)
	}

	// Create default profile for user
	profilePath := newUser.ProfilePath
	defaultProfile := profile.DefaultProfile()
	if err := defaultProfile.Save(profilePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create profile: %v\n", err)
	}

	// Update manager and save
	mgr.SwitchUser(newUser.ID)
	if err := mgr.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving users: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized multi-user system with user: %s (%s)\n", newUser.Name, newUser.ID)
	fmt.Printf("User directory: %s\n", mgr.GetUserDir(newUser.ID))
	fmt.Printf("Profile: %s\n", newUser.ProfilePath)
	fmt.Printf("Knowledge dirs: %v\n", newUser.KnowledgeDirs)
	fmt.Printf("API Token: %s\n", newUser.APIToken)
	fmt.Println()
	fmt.Println("Switch to this user with: aura user switch " + newUser.ID)
}

func runUserCreate(cmd *cobra.Command, args []string) {
	name := args[0]

	usersCfg, err := user.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(usersCfg.Definitions) == 0 {
		fmt.Println("Multi-user system not initialized.")
		fmt.Println("Run 'aura user init <name>' first.")
		return
	}

	mgr, err := user.NewManager(usersCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user manager: %v\n", err)
		os.Exit(1)
	}

	newUser, err := mgr.CreateUser(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user: %v\n", err)
		os.Exit(1)
	}

	// Create default profile
	defaultProfile := profile.DefaultProfile()
	if err := defaultProfile.Save(newUser.ProfilePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create profile: %v\n", err)
	}

	// Save users config
	if err := mgr.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving users: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created user: %s (%s)\n", newUser.Name, newUser.ID)
	fmt.Printf("API Token: %s\n", newUser.APIToken)
	fmt.Println()
	fmt.Println("Switch to this user with: aura user switch " + newUser.ID)
}

func runUserList(cmd *cobra.Command, args []string) {
	usersCfg, err := user.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(usersCfg.Definitions) == 0 {
		fmt.Println("No users configured.")
		fmt.Println("Run 'aura user init <name>' to create the first user.")
		return
	}

	mgr, err := user.NewManager(usersCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user manager: %v\n", err)
		os.Exit(1)
	}

	users := mgr.ListUsers()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tAPI TOKEN\tPROFILE PATH\n")
	fmt.Fprintf(w, "----\t----\t----------\t------------\n")
	for _, u := range users {
		marker := " "
		if u.ID == usersCfg.Default {
			marker = "*"
		}
		fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\n", marker, u.ID, u.Name, maskAPIToken(u.APIToken), u.ProfilePath)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("* = current user")
}

func runUserSwitch(cmd *cobra.Command, args []string) {
	userID := args[0]

	usersCfg, err := user.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mgr, err := user.NewManager(usersCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user manager: %v\n", err)
		os.Exit(1)
	}

	// Verify user exists
	user := mgr.GetUserByID(userID)
	if user == nil {
		fmt.Fprintf(os.Stderr, "User not found: %s\n", userID)
		os.Exit(1)
	}

	if err := mgr.SwitchUser(userID); err != nil {
		fmt.Fprintf(os.Stderr, "Error switching user: %v\n", err)
		os.Exit(1)
	}

	if err := mgr.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving users: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Switched to user: %s (%s)\n", user.Name, userID)
}

func runUserCurrent(cmd *cobra.Command, args []string) {
	usersCfg, err := user.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mgr, err := user.NewManager(usersCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user manager: %v\n", err)
		os.Exit(1)
	}

	currentUser := mgr.GetDefaultUser()
	if currentUser == nil {
		fmt.Println("No default user set.")
		return
	}

	fmt.Printf("Current user: %s\n", currentUser.Name)
	fmt.Printf("User ID: %s\n", currentUser.ID)
	fmt.Printf("Profile: %s\n", currentUser.ProfilePath)
	fmt.Printf("Knowledge dirs: %v\n", currentUser.KnowledgeDirs)
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	userID := args[0]

	usersCfg, err := user.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading users config: %w", err)
	}

	mgr, err := user.NewManager(usersCfg)
	if err != nil {
		return fmt.Errorf("error creating user manager: %w", err)
	}

	// Verify user exists
	user := mgr.GetUserByID(userID)
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Prevent deleting default user
	if userID == usersCfg.Default {
		return fmt.Errorf("cannot delete default user, switch to another user first")
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete user '%s' (%s)?\n", user.Name, userID)
	fmt.Println("This will permanently delete:")
	fmt.Printf("  - User directory: %s\n", mgr.GetUserDir(userID))
	fmt.Printf("  - Profile: %s\n", user.ProfilePath)
	fmt.Printf("  - Knowledge base: %v\n", user.KnowledgeDirs)
	fmt.Print("Type 'yes' to confirm: ")

	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("Deletion cancelled.")
		return nil
	}

	if err := mgr.DeleteUser(userID); err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	if err := mgr.Save(); err != nil {
		return fmt.Errorf("error saving users: %w", err)
	}

	fmt.Printf("User '%s' (%s) has been deleted.\n", user.Name, userID)
	return nil
}

func runUserToken(cmd *cobra.Command, args []string) {
	userID := args[0]

	usersCfg, err := user.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mgr, err := user.NewManager(usersCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user manager: %v\n", err)
		os.Exit(1)
	}

	// Verify user exists
	u := mgr.GetUserByID(userID)
	if u == nil {
		fmt.Fprintf(os.Stderr, "User not found: %s\n", userID)
		os.Exit(1)
	}

	fmt.Printf("User: %s (%s)\n", u.Name, userID)
	fmt.Printf("API Token: %s\n", u.APIToken)
	fmt.Println()
	fmt.Println("Use this token for:")
	fmt.Println("  - WebUI login (enter User ID: " + userID + ")")
	fmt.Println("  - API authentication (Authorization: Bearer " + u.APIToken + ")")
}

// maskAPIToken masks the API token for display (show first and last 4 chars).
func maskAPIToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
