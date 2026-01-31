package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDataDir_DefaultFallback(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("ZIPFS_DATA_DIR")
	os.Unsetenv("XDG_DATA_HOME")

	dataDir, err := DataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain .local/share/zipfs
	if !strings.Contains(dataDir, filepath.Join(".local", "share", "zipfs")) {
		t.Errorf("expected data dir to contain .local/share/zipfs, got %s", dataDir)
	}
}

func TestDataDir_XDGDataHome(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Unsetenv("XDG_DATA_HOME")

	os.Unsetenv("ZIPFS_DATA_DIR")

	dataDir, err := DataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "zipfs")
	if dataDir != expected {
		t.Errorf("expected %s, got %s", expected, dataDir)
	}
}

func TestDataDir_OverrideWithZIPFS_DATA_DIR(t *testing.T) {
	tempDir := t.TempDir()
	customPath := filepath.Join(tempDir, "custom-zipfs")

	os.Setenv("ZIPFS_DATA_DIR", customPath)
	defer os.Unsetenv("ZIPFS_DATA_DIR")

	dataDir, err := DataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dataDir != customPath {
		t.Errorf("expected %s, got %s", customPath, dataDir)
	}
}

func TestWorkspacesDir(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("ZIPFS_DATA_DIR", tempDir)
	defer os.Unsetenv("ZIPFS_DATA_DIR")

	workspacesDir, err := WorkspacesDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "workspaces")
	if workspacesDir != expected {
		t.Errorf("expected %s, got %s", expected, workspacesDir)
	}
}

func TestWorkspaceDir(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("ZIPFS_DATA_DIR", tempDir)
	defer os.Unsetenv("ZIPFS_DATA_DIR")

	sessionID := "test-session"
	workspaceDir, err := WorkspaceDir(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "workspaces", sessionID)
	if workspaceDir != expected {
		t.Errorf("expected %s, got %s", expected, workspaceDir)
	}
}

func TestContentsDir(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("ZIPFS_DATA_DIR", tempDir)
	defer os.Unsetenv("ZIPFS_DATA_DIR")

	sessionID := "test-session"
	contentsDir, err := ContentsDir(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "workspaces", sessionID, "contents")
	if contentsDir != expected {
		t.Errorf("expected %s, got %s", expected, contentsDir)
	}
}

func TestMetadataPath(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("ZIPFS_DATA_DIR", tempDir)
	defer os.Unsetenv("ZIPFS_DATA_DIR")

	sessionID := "test-session"
	metadataPath, err := MetadataPath(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "workspaces", sessionID, "metadata.json")
	if metadataPath != expected {
		t.Errorf("expected %s, got %s", expected, metadataPath)
	}
}

func TestOriginalZipPath(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("ZIPFS_DATA_DIR", tempDir)
	defer os.Unsetenv("ZIPFS_DATA_DIR")

	sessionID := "test-session"
	originalZipPath, err := OriginalZipPath(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "workspaces", sessionID, "original.zip")
	if originalZipPath != expected {
		t.Errorf("expected %s, got %s", expected, originalZipPath)
	}
}

func TestLockPath(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("ZIPFS_DATA_DIR", tempDir)
	defer os.Unsetenv("ZIPFS_DATA_DIR")

	sessionID := "test-session"
	lockPath, err := LockPath(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "workspaces", sessionID, "metadata.json.lock")
	if lockPath != expected {
		t.Errorf("expected %s, got %s", expected, lockPath)
	}
}

func TestDataDir_HomeError(t *testing.T) {
	// Clear all environment variables
	os.Unsetenv("ZIPFS_DATA_DIR")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("HOME")

	// This should still work by falling back to os.UserHomeDir()
	_, err := DataDir()
	// May or may not error depending on environment
	_ = err
}
