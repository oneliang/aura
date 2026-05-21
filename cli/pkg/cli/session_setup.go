// Package cli provides session setup using the SDK SessionManager.
package cli

import (
	sdkpkg "github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// initSessionManager initializes the SDK SessionManager from the data directory.
// Returns the manager, current session ID, and any error.
func initSessionManager(dataDir string, cfg *config.Config, userID string) (*sdkpkg.SessionManager, string, error) {
	mgr, err := sdkpkg.NewSessionManager(dataDir, userID, cfg)
	if err != nil {
		return nil, "", err
	}

	sessionID, err := mgr.GetOrCreateSession()
	if err != nil {
		return mgr, "", err
	}
	return mgr, sessionID, nil
}

// initSessionManagerForTUI initializes SDK SessionManager for TUI mode.
// Returns the session manager and the current session ID.
func initSessionManagerForTUI() (*sdkpkg.SessionManager, string) {
	dataDir, err := getDataDir()
	if err != nil {
		return nil, ""
	}
	ctx := getCommandContext()
	var cfg *config.Config
	if ctx != nil && ctx.Config != nil {
		cfg = ctx.Config
	}
	userID := ctx.UserID

	mgr, err := sdkpkg.NewSessionManager(dataDir, userID, cfg)
	if err != nil {
		return nil, ""
	}

	sessionID, err := mgr.GetOrCreateSession()
	if err != nil {
		return mgr, ""
	}

	return mgr, sessionID
}

// getDataDir returns the Aura data directory.
func getDataDir() (string, error) {
	return ffp.MustAuraHomePath(constants.DirSessions), nil
}
