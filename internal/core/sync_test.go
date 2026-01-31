package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Fuabioo/zipfs/internal/errors"
	"github.com/Fuabioo/zipfs/internal/security"
)

func TestSync_Basic(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	files := map[string]string{
		"file1.txt": "original content 1",
		"file2.txt": "original content 2",
	}
	createTestZip(t, zipPath, files)

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "sync-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Modify a file in the workspace
	contentsDir, err := ContentsDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	modifiedContent := "MODIFIED CONTENT"
	err = os.WriteFile(filepath.Join(contentsDir, "file1.txt"), []byte(modifiedContent), 0644)
	if err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Sync
	result, err := Sync(session, false, cfg)
	if err != nil {
		t.Fatalf("failed to sync: %v", err)
	}

	if result.BackupPath == "" {
		t.Error("expected backup path to be set")
	}

	// Verify backup exists
	if _, err := os.Stat(result.BackupPath); err != nil {
		t.Errorf("backup file doesn't exist: %v", err)
	}

	// Verify source zip was updated
	if _, err := os.Stat(zipPath); err != nil {
		t.Errorf("source zip doesn't exist after sync: %v", err)
	}

	// Extract and verify modification persisted
	extractDir := filepath.Join(tempDir, "verify")
	os.MkdirAll(extractDir, 0755)

	limits := security.DefaultLimits()
	_, _, err = Extract(zipPath, extractDir, limits)
	if err != nil {
		t.Fatalf("failed to extract synced zip: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(extractDir, "file1.txt"))
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	if string(content) != modifiedContent {
		t.Errorf("expected content %q, got %q", modifiedContent, string(content))
	}
}

func TestSync_ConflictDetection(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "conflict-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Externally modify the source zip
	createTestZip(t, zipPath, map[string]string{"file.txt": "EXTERNALLY MODIFIED"})

	// Try to sync without force (should fail)
	_, err = Sync(session, false, cfg)
	if err == nil {
		t.Fatal("expected error for conflict")
	}

	if !errors.Is(err, errors.CodeConflictDetected) {
		t.Errorf("expected CONFLICT_DETECTED error, got: %v", err)
	}

	// Sync with force should succeed
	_, err = Sync(session, true, cfg)
	if err != nil {
		t.Fatalf("failed to sync with force: %v", err)
	}
}

func TestSync_SourceDoesNotExist(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "missing-source", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Delete source zip
	os.Remove(zipPath)

	// Sync should fail
	_, err = Sync(session, false, cfg)
	if err == nil {
		t.Fatal("expected error when source doesn't exist")
	}
}

func TestSync_UpdatesMetadata(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "metadata-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.LastSyncedAt != nil {
		t.Error("expected LastSyncedAt to be nil initially")
	}

	originalHash := session.ZipHashSHA256

	// Sync
	_, err = Sync(session, false, cfg)
	if err != nil {
		t.Fatalf("failed to sync: %v", err)
	}

	// Verify metadata was updated
	if session.LastSyncedAt == nil {
		t.Error("expected LastSyncedAt to be set after sync")
	}

	if session.State != "open" {
		t.Errorf("expected state to be 'open', got %q", session.State)
	}

	// Hash might change due to zip compression
	if session.ZipHashSHA256 == "" {
		t.Error("expected hash to be set")
	}

	// Reload session and verify persistence
	retrieved, err := GetSession("metadata-test")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.LastSyncedAt == nil {
		t.Error("expected persisted LastSyncedAt to be set")
	}

	_ = originalHash
}

func TestRotateBackups_Basic(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	sourcePath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, sourcePath, map[string]string{"file.txt": "v1"})

	// First rotation
	bakPath, err := RotateBackups(sourcePath, 3)
	if err != nil {
		t.Fatalf("failed to rotate backups: %v", err)
	}

	expectedBak := filepath.Join(tempDir, "test.bak.zip")
	if bakPath != expectedBak {
		t.Errorf("expected backup path %s, got %s", expectedBak, bakPath)
	}

	// Verify source was renamed to .bak
	if _, err := os.Stat(expectedBak); err != nil {
		t.Error("expected .bak file to exist")
	}

	// Source should no longer exist
	if _, err := os.Stat(sourcePath); !os.IsNotExist(err) {
		t.Error("expected source to be moved")
	}
}

func TestRotateBackups_MultipleRotations(t *testing.T) {
	tempDir := t.TempDir()

	sourcePath := filepath.Join(tempDir, "test.zip")

	// Create and rotate multiple times
	for i := 1; i <= 4; i++ {
		// Create new version
		createTestZip(t, sourcePath, map[string]string{"file.txt": "version"})

		_, err := RotateBackups(sourcePath, 3)
		if err != nil {
			t.Fatalf("failed to rotate backups iteration %d: %v", i, err)
		}
	}

	// Verify rotation depth is respected
	bak1 := filepath.Join(tempDir, "test.bak.zip")
	bak2 := filepath.Join(tempDir, "test.bak.2.zip")
	bak3 := filepath.Join(tempDir, "test.bak.3.zip")
	bak4 := filepath.Join(tempDir, "test.bak.4.zip")

	if _, err := os.Stat(bak1); err != nil {
		t.Error("expected .bak to exist")
	}

	if _, err := os.Stat(bak2); err != nil {
		t.Error("expected .bak.2 to exist")
	}

	if _, err := os.Stat(bak3); err != nil {
		t.Error("expected .bak.3 to exist")
	}

	// .bak.4 should not exist (depth limit is 3)
	if _, err := os.Stat(bak4); !os.IsNotExist(err) {
		t.Error("expected .bak.4 to not exist (exceeds depth)")
	}
}

func TestRotateBackups_DifferentExtension(t *testing.T) {
	tempDir := t.TempDir()

	sourcePath := filepath.Join(tempDir, "archive.tar.gz")
	os.WriteFile(sourcePath, []byte("data"), 0644)

	bakPath, err := RotateBackups(sourcePath, 3)
	if err != nil {
		t.Fatalf("failed to rotate backups: %v", err)
	}

	expectedBak := filepath.Join(tempDir, "archive.tar.bak.gz")
	if bakPath != expectedBak {
		t.Errorf("expected backup path %s, got %s", expectedBak, bakPath)
	}
}
