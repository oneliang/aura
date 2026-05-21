package tui

import (
	"os"
	"path/filepath"

	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/constants"
)

// sessionInfoToItem converts an sdk.SessionInfo to a TUI SessionItem.
func sessionInfoToItem(info sdk.SessionInfo) SessionItem {
	return SessionItem{
		id:      info.ID,
		name:    info.Name,
		created: info.Created,
		updated: info.Updated,
		subs:    info.Subs,
		role:    info.Role,
	}
}

// sessionInfosToItems converts a slice of sdk.SessionInfo to TUI SessionItems.
func sessionInfosToItems(items []sdk.SessionInfo) []SessionItem {
	result := make([]SessionItem, len(items))
	for i, item := range items {
		result[i] = sessionInfoToItem(item)
	}
	return result
}

// subscriptionInfoToItem converts an sdk.SubscriptionInfo to a TUI SubscriptionItem.
func subscriptionInfoToItem(info sdk.SubscriptionInfo) SubscriptionItem {
	return SubscriptionItem{
		id:      info.ID,
		trigger: info.Trigger,
		source:  info.Source,
		active:  info.Active,
	}
}

// subscriptionInfosToItems converts a slice of sdk.SubscriptionInfo to TUI SubscriptionItems.
func subscriptionInfosToItems(items []sdk.SubscriptionInfo) []SubscriptionItem {
	result := make([]SubscriptionItem, len(items))
	for i, item := range items {
		result[i] = subscriptionInfoToItem(item)
	}
	return result
}

// ListRoles returns all available roles.
func ListRoles() ([]string, error) {
	rolesDir, err := getRolesDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var roles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			role := entry.Name()[:len(entry.Name())-3] // Remove .md extension
			roles = append(roles, role)
		}
	}
	return roles, nil
}

// getRolesDir returns the roles directory path.
func getRolesDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, constants.DefaultHomeDir, constants.DirRoles), nil
}
