// Package commands provides CLI commands for habit management.
package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/oneliang/aura/habit/pkg/manager"
	"github.com/oneliang/aura/habit/pkg/model"
	"github.com/spf13/cobra"
)

// HabitCmd is the root command for habit management.
var HabitCmd = &cobra.Command{
	Use:   "habit",
	Short: "Manage user behavior habits",
	Long:  `View and manage learned user behavior habits and preferences.`,
}

var habitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all habits for the current user",
	Run:   runHabitList,
}

var habitShowCmd = &cobra.Command{
	Use:   "show <habit-id>",
	Short: "Show habit details",
	Args:  cobra.ExactArgs(1),
	Run:   runHabitShow,
}

var habitDeleteCmd = &cobra.Command{
	Use:   "delete <habit-id>",
	Short: "Delete a habit",
	Args:  cobra.ExactArgs(1),
	Run:   runHabitDelete,
}

var habitRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Re-analyze actions and refresh habits",
	Run:   runHabitRefresh,
}

var preferenceCmd = &cobra.Command{
	Use:   "preference",
	Short: "Manage user preferences",
}

var preferenceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all preferences for the current user",
	Run:   runPreferenceList,
}

func init() {
	HabitCmd.AddCommand(habitListCmd)
	HabitCmd.AddCommand(habitShowCmd)
	HabitCmd.AddCommand(habitDeleteCmd)
	HabitCmd.AddCommand(habitRefreshCmd)
	HabitCmd.AddCommand(preferenceCmd)
	preferenceCmd.AddCommand(preferenceListCmd)
}

func getHabitManager() (*manager.Manager, string, error) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		return nil, "", fmt.Errorf("command context not initialized")
	}

	userID := cmdCtx.UserID
	if userID == "" {
		return nil, "", fmt.Errorf("multi-user mode not enabled, habits are not available")
	}

	habitMgr, err := manager.New(manager.DefaultConfig())
	if err != nil {
		return nil, "", fmt.Errorf("failed to create habit manager: %w", err)
	}

	return habitMgr, userID, nil
}

func runHabitList(cmd *cobra.Command, args []string) {
	habitMgr, userID, err := getHabitManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx := cmd.Context()
	habits, err := habitMgr.GetHabits(ctx, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(habits) == 0 {
		fmt.Println("No habits learned yet.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tCATEGORY\tNAME\tCONFIDENCE\tCOUNT\n")
	fmt.Fprintf(w, "----\t--------\t----\t----------\t-----\n")
	for _, h := range habits {
		fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%d\n", h.ID, h.Category, h.Name, h.Confidence, h.Frequency.TotalCount)
	}
	w.Flush()
}

func runHabitShow(cmd *cobra.Command, args []string) {
	habitMgr, userID, err := getHabitManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	habitID := args[0]
	habits, err := habitMgr.GetHabits(cmd.Context(), userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var found *model.Habit
	for _, h := range habits {
		if h.ID == habitID {
			found = h
			break
		}
	}

	if found == nil {
		fmt.Fprintf(os.Stderr, "Habit not found: %s\n", habitID)
		os.Exit(1)
	}

	fmt.Printf("=== Habit Details ===\n")
	fmt.Printf("ID:         %s\n", found.ID)
	fmt.Printf("Name:       %s\n", found.Name)
	fmt.Printf("Category:   %s\n", found.Category)
	fmt.Printf("Confidence: %.2f\n", found.Confidence)
	fmt.Printf("Count:      %d\n", found.Frequency.TotalCount)
	fmt.Printf("Trend:      %s\n", found.Frequency.Trend)
	fmt.Printf("Last Seen:  %s\n", found.LastSeen.Format("2006-01-02 15:04:05"))
	if len(found.Pattern.ToolUsage) > 0 {
		fmt.Printf("Tools:      %v\n", found.Pattern.ToolUsage)
	}
	if len(found.Pattern.Keywords) > 0 {
		fmt.Printf("Keywords:   %v\n", found.Pattern.Keywords)
	}
	if len(found.Pattern.CommandSeq) > 0 {
		fmt.Printf("Sequences:  %v\n", found.Pattern.CommandSeq)
	}
}

func runHabitDelete(cmd *cobra.Command, args []string) {
	habitMgr, userID, err := getHabitManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	habitID := args[0]
	err = habitMgr.DeleteHabit(cmd.Context(), userID, habitID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Habit deleted: %s\n", habitID)
}

func runHabitRefresh(cmd *cobra.Command, args []string) {
	habitMgr, userID, err := getHabitManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	habits, err := habitMgr.RefreshHabits(cmd.Context(), userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(habits) == 0 {
		fmt.Println("No habits found after refresh.")
		return
	}

	fmt.Printf("Refreshed %d habits:\n", len(habits))
	for _, h := range habits {
		fmt.Printf("  - %s (confidence: %.2f)\n", h.Name, h.Confidence)
	}
}

func runPreferenceList(cmd *cobra.Command, args []string) {
	habitMgr, userID, err := getHabitManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	prefs, err := habitMgr.GetPreferences(cmd.Context(), userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(prefs) == 0 {
		fmt.Println("No preferences learned yet.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "CATEGORY\tNAME\tVALUE\tSOURCE\n")
	fmt.Fprintf(w, "--------\t----\t-----\t------\n")
	for _, p := range prefs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Category, p.Name, p.Value, p.Source)
	}
	w.Flush()
}
