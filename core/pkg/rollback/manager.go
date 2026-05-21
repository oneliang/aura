// Package rollback provides git-based rollback functionality for plan mode execution.
// It creates git stash snapshots before execution and supports rollback on failure.
package rollback

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/oneliang/aura/shared/pkg/logger"
)

// Manager handles rollback operations using git stash.
type Manager struct {
	workDir string
	logger  *logger.Logger

	mu        sync.Mutex
	snapshots map[string]*Snapshot // Active snapshots by ID
}

// Snapshot represents a rollback snapshot created via git stash.
type Snapshot struct {
	ID        string    // Unique identifier
	StashRef  string    // Git stash reference (e.g., "stash@{0}")
	Message   string    // Stash commit message
	CreatedAt time.Time // Creation timestamp
	Files     []string  // Files that were stashed (for tracking)
}

// RollbackResult represents the result of a rollback operation.
type RollbackResult struct {
	Success bool
	Message string
	Files   []string // Files restored
}

// NewManager creates a new rollback manager.
func NewManager(workDir string, log *logger.Logger) *Manager {
	if log == nil {
		log = logger.Default()
	}
	return &Manager{
		workDir:   workDir,
		logger:    log,
		snapshots: make(map[string]*Snapshot),
	}
}

// CreateSnapshot creates a new rollback snapshot before execution.
// Uses git stash to save current working directory changes.
func (m *Manager) CreateSnapshot(ctx context.Context, id string, files []string) (*Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug().Str("module", "rollback").Str("id", id).Msg("CreateSnapshot: creating snapshot")

	// Check if git is available
	if !m.isGitAvailable(ctx) {
		return nil, fmt.Errorf("git not available in working directory: %s", m.workDir)
	}

	// Create stash with unique message
	message := fmt.Sprintf("aura-plan-snapshot-%s-%d", id, time.Now().Unix())
	stashRef, err := m.gitStash(ctx, message)
	if err != nil {
		// If no changes to stash, create an empty snapshot marker
		if strings.Contains(err.Error(), "no local changes") {
			m.logger.Debug().Str("module", "rollback").Msg("CreateSnapshot: no changes to stash, creating empty snapshot")
			snapshot := &Snapshot{
				ID:        id,
				StashRef:  "",
				Message:   message,
				CreatedAt: time.Now(),
				Files:     files,
			}
			m.snapshots[id] = snapshot
			return snapshot, nil
		}
		return nil, fmt.Errorf("failed to create git stash: %w", err)
	}

	snapshot := &Snapshot{
		ID:        id,
		StashRef:  stashRef,
		Message:   message,
		CreatedAt: time.Now(),
		Files:     files,
	}

	m.snapshots[id] = snapshot
	m.logger.Debug().Str("module", "rollback").Str("id", id).Str("stash_ref", stashRef).Msg("CreateSnapshot: snapshot created")

	return snapshot, nil
}

// Rollback restores the working directory to the snapshot state.
// Uses git stash pop to restore stashed changes.
func (m *Manager) Rollback(ctx context.Context, snapshotID string) (*RollbackResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug().Str("module", "rollback").Str("id", snapshotID).Msg("Rollback: starting rollback")

	snapshot, exists := m.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	// Empty snapshot (no changes were stashed) - nothing to restore
	if snapshot.StashRef == "" {
		m.logger.Debug().Str("module", "rollback").Str("id", snapshotID).Msg("Rollback: empty snapshot, nothing to restore")
		return &RollbackResult{
			Success: true,
			Message: "No changes to restore (empty snapshot)",
			Files:   snapshot.Files,
		}, nil
	}

	// Restore from stash
	err := m.gitStashPop(ctx, snapshot.StashRef)
	if err != nil {
		return nil, fmt.Errorf("failed to restore git stash: %w", err)
	}

	// Remove snapshot after successful rollback
	delete(m.snapshots, snapshotID)

	m.logger.Debug().Str("module", "rollback").Str("id", snapshotID).Msg("Rollback: rollback completed")

	return &RollbackResult{
		Success: true,
		Message: fmt.Sprintf("Restored from %s", snapshot.StashRef),
		Files:   snapshot.Files,
	}, nil
}

// Cleanup removes a snapshot without restoring it.
// Uses git stash drop to remove the stash entry.
func (m *Manager) Cleanup(ctx context.Context, snapshotID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, exists := m.snapshots[snapshotID]
	if !exists {
		return nil // Already cleaned up
	}

	// Empty snapshot - just remove from tracking
	if snapshot.StashRef == "" {
		delete(m.snapshots, snapshotID)
		return nil
	}

	// Drop the stash
	err := m.gitStashDrop(ctx, snapshot.StashRef)
	if err != nil {
		m.logger.Warn().Str("module", "rollback").Str("id", snapshotID).Err(err).Msg("Cleanup: failed to drop stash")
		// Still remove from tracking even if drop fails
	}

	delete(m.snapshots, snapshotID)
	m.logger.Debug().Str("module", "rollback").Str("id", snapshotID).Msg("Cleanup: snapshot cleaned up")

	return nil
}

// GetSnapshot returns a snapshot by ID.
func (m *Manager) GetSnapshot(id string) *Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshots[id]
}

// ListSnapshots returns all active snapshots.
func (m *Manager) ListSnapshots() []*Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*Snapshot, 0, len(m.snapshots))
	for _, s := range m.snapshots {
		result = append(result, s)
	}
	return result
}

// gitStash creates a git stash with the given message.
func (m *Manager) gitStash(ctx context.Context, message string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "stash", "push", "-m", message)
	cmd.Dir = m.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git stash failed: %w, output: %s", err, string(output))
	}

	// Get the stash reference
	cmd = exec.CommandContext(ctx, "git", "stash", "list", "-n1")
	cmd.Dir = m.workDir

	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git stash list failed: %w", err)
	}

	// Parse stash reference from output: "stash@{0}: On branch: message"
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 && strings.Contains(lines[0], "stash@{") {
		// Extract just "stash@{0}" part
		parts := strings.SplitN(lines[0], ":", 2)
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0]), nil
		}
	}

	return "stash@{0}", nil // Default to most recent stash
}

// gitStashPop restores from a specific stash reference.
func (m *Manager) gitStashPop(ctx context.Context, ref string) error {
	cmd := exec.CommandContext(ctx, "git", "stash", "pop", ref)
	cmd.Dir = m.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash pop failed: %w, output: %s", err, string(output))
	}

	return nil
}

// gitStashDrop removes a specific stash entry.
func (m *Manager) gitStashDrop(ctx context.Context, ref string) error {
	cmd := exec.CommandContext(ctx, "git", "stash", "drop", ref)
	cmd.Dir = m.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash drop failed: %w, output: %s", err, string(output))
	}

	return nil
}

// isGitAvailable checks if git is available and we're in a git repository.
func (m *Manager) isGitAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = m.workDir

	err := cmd.Run()
	return err == nil
}