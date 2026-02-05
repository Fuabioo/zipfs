package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateWorkspace creates the directory structure for a session workspace.
func CreateWorkspace(session *Session, dirName string) error {
	workspaceDir, err := WorkspaceDir(dirName)
	if err != nil {
		return fmt.Errorf("failed to get workspace directory: %w", err)
	}

	// Create workspace directory with user-only permissions
	if err := os.MkdirAll(workspaceDir, 0700); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Create contents subdirectory
	contentsDir := filepath.Join(workspaceDir, "contents")
	if err := os.MkdirAll(contentsDir, 0700); err != nil {
		return fmt.Errorf("failed to create contents directory: %w", err)
	}

	return nil
}

// RemoveWorkspace removes the entire workspace directory for a session.
func RemoveWorkspace(session *Session, dirName string) error {
	workspaceDir, err := WorkspaceDir(dirName)
	if err != nil {
		return fmt.Errorf("failed to get workspace directory: %w", err)
	}

	if err := os.RemoveAll(workspaceDir); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}

	return nil
}
