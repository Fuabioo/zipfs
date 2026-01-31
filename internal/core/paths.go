package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// DataDir returns the base data directory for zipfs.
// It follows the XDG Base Directory Specification:
// - $ZIPFS_DATA_DIR (full override)
// - $XDG_DATA_HOME/zipfs
// - ~/.local/share/zipfs (fallback)
func DataDir() (string, error) {
	// Check for full override
	if dir := os.Getenv("ZIPFS_DATA_DIR"); dir != "" {
		return dir, nil
	}

	// Check XDG_DATA_HOME
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "zipfs"), nil
	}

	// Fallback to ~/.local/share/zipfs
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(home, ".local", "share", "zipfs"), nil
}

// WorkspacesDir returns the directory containing all session workspaces.
func WorkspacesDir() (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}
	return filepath.Join(dataDir, "workspaces"), nil
}

// WorkspaceDir returns the directory for a specific session workspace.
func WorkspaceDir(sessionID string) (string, error) {
	workspacesDir, err := WorkspacesDir()
	if err != nil {
		return "", fmt.Errorf("failed to get workspaces directory: %w", err)
	}
	return filepath.Join(workspacesDir, sessionID), nil
}

// ContentsDir returns the contents/ directory within a session workspace.
func ContentsDir(sessionID string) (string, error) {
	workspaceDir, err := WorkspaceDir(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace directory: %w", err)
	}
	return filepath.Join(workspaceDir, "contents"), nil
}

// MetadataPath returns the path to the metadata.json file for a session.
func MetadataPath(sessionID string) (string, error) {
	workspaceDir, err := WorkspaceDir(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace directory: %w", err)
	}
	return filepath.Join(workspaceDir, "metadata.json"), nil
}

// OriginalZipPath returns the path to the original.zip file for a session.
func OriginalZipPath(sessionID string) (string, error) {
	workspaceDir, err := WorkspaceDir(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace directory: %w", err)
	}
	return filepath.Join(workspaceDir, "original.zip"), nil
}

// LockPath returns the path to the lock file for a session.
func LockPath(sessionID string) (string, error) {
	workspaceDir, err := WorkspaceDir(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace directory: %w", err)
	}
	return filepath.Join(workspaceDir, "metadata.json.lock"), nil
}
