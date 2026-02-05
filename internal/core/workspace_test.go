package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateWorkspace(t *testing.T) {
	setupTestEnvironment(t)

	session := &Session{
		ID:   "test-session-id",
		Name: "test-session",
	}

	err := CreateWorkspace(session, session.Name)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Verify workspace directory exists
	workspaceDir, err := WorkspaceDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get workspace dir: %v", err)
	}

	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		t.Error("expected workspace directory to exist")
	}

	// Verify contents directory exists
	contentsDir := filepath.Join(workspaceDir, "contents")
	if _, err := os.Stat(contentsDir); os.IsNotExist(err) {
		t.Error("expected contents directory to exist")
	}

	// Verify permissions
	info, err := os.Stat(workspaceDir)
	if err != nil {
		t.Fatalf("failed to stat workspace dir: %v", err)
	}

	// Check that it's user-only (0700)
	mode := info.Mode().Perm()
	if mode != 0700 {
		t.Errorf("expected permissions 0700, got %o", mode)
	}
}

func TestRemoveWorkspace(t *testing.T) {
	setupTestEnvironment(t)

	session := &Session{
		ID:   "test-session-id",
		Name: "test-session",
	}

	// Create workspace
	err := CreateWorkspace(session, session.Name)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	workspaceDir, err := WorkspaceDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get workspace dir: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		t.Fatal("workspace should exist before removal")
	}

	// Remove workspace
	err = RemoveWorkspace(session, session.Name)
	if err != nil {
		t.Fatalf("failed to remove workspace: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(workspaceDir); !os.IsNotExist(err) {
		t.Error("expected workspace to be removed")
	}
}

func TestCreateWorkspace_WithUUID(t *testing.T) {
	setupTestEnvironment(t)

	session := &Session{
		ID: "550e8400-e29b-41d4-a716-446655440000",
	}

	err := CreateWorkspace(session, session.ID)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Verify workspace directory exists with UUID name
	workspaceDir, err := WorkspaceDir(session.ID)
	if err != nil {
		t.Fatalf("failed to get workspace dir: %v", err)
	}

	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		t.Error("expected workspace directory to exist")
	}
}

func TestRemoveWorkspace_NonExistent(t *testing.T) {
	setupTestEnvironment(t)

	session := &Session{
		ID:   "nonexistent-id",
		Name: "nonexistent",
	}

	// Should not error when removing non-existent workspace
	err := RemoveWorkspace(session, session.Name)
	if err != nil {
		t.Errorf("unexpected error removing non-existent workspace: %v", err)
	}
}

func TestCreateWorkspace_NestedDirectories(t *testing.T) {
	setupTestEnvironment(t)

	session := &Session{
		ID:   "test-id",
		Name: "test",
	}

	err := CreateWorkspace(session, session.Name)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create nested content
	contentsDir, err := ContentsDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	nestedPath := filepath.Join(contentsDir, "a", "b", "c", "file.txt")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0755); err != nil {
		t.Fatalf("failed to create nested dirs: %v", err)
	}

	if err := os.WriteFile(nestedPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Remove workspace - should remove everything
	err = RemoveWorkspace(session, session.Name)
	if err != nil {
		t.Fatalf("failed to remove workspace: %v", err)
	}

	// Verify nested content is gone
	workspaceDir, err := WorkspaceDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get workspace dir: %v", err)
	}

	if _, err := os.Stat(workspaceDir); !os.IsNotExist(err) {
		t.Error("expected workspace to be completely removed")
	}
}

func TestCreateWorkspace_AlreadyExists(t *testing.T) {
	setupTestEnvironment(t)

	session := &Session{
		ID:   "test-id",
		Name: "test",
	}

	// Create workspace
	err := CreateWorkspace(session, session.Name)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Try to create again - should succeed (idempotent)
	err = CreateWorkspace(session, session.Name)
	if err != nil {
		t.Errorf("expected workspace creation to be idempotent, got error: %v", err)
	}
}
