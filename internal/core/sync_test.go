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

func TestRotateBackups_NoExtension(t *testing.T) {
	tempDir := t.TempDir()

	sourcePath := filepath.Join(tempDir, "noextension")
	os.WriteFile(sourcePath, []byte("data"), 0644)

	bakPath, err := RotateBackups(sourcePath, 3)
	if err != nil {
		t.Fatalf("failed to rotate backups: %v", err)
	}

	expectedBak := filepath.Join(tempDir, "noextension.bak")
	if bakPath != expectedBak {
		t.Errorf("expected backup path %s, got %s", expectedBak, bakPath)
	}
}

func TestRotateBackups_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()

	sourcePath := filepath.Join(tempDir, "nonexistent.zip")

	_, err := RotateBackups(sourcePath, 3)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestSync_AddFileToWorkspace(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file1.txt": "content1"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "add-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	contentsDir, err := ContentsDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	// Add a new file
	os.WriteFile(filepath.Join(contentsDir, "newfile.txt"), []byte("new content"), 0644)

	// Sync
	_, err = Sync(session, false, cfg)
	if err != nil {
		t.Fatalf("failed to sync: %v", err)
	}

	// Verify new file is in the zip
	extractDir := filepath.Join(tempDir, "verify")
	os.MkdirAll(extractDir, 0755)

	limits := security.DefaultLimits()
	_, _, err = Extract(zipPath, extractDir, limits)
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	newFilePath := filepath.Join(extractDir, "newfile.txt")
	if _, err := os.Stat(newFilePath); err != nil {
		t.Error("expected new file to exist in synced zip")
	}
}

func TestSync_DeletedFile(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip with multiple files
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
		"file3.txt": "content3",
	})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "delete-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	contentsDir, err := ContentsDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	// Delete a file
	os.Remove(filepath.Join(contentsDir, "file2.txt"))

	// Sync
	_, err = Sync(session, false, cfg)
	if err != nil {
		t.Fatalf("failed to sync: %v", err)
	}

	// Verify file is gone from the zip
	extractDir := filepath.Join(tempDir, "verify")
	os.MkdirAll(extractDir, 0755)

	limits := security.DefaultLimits()
	_, _, err = Extract(zipPath, extractDir, limits)
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	deletedFilePath := filepath.Join(extractDir, "file2.txt")
	if _, err := os.Stat(deletedFilePath); !os.IsNotExist(err) {
		t.Error("expected deleted file to not exist in synced zip")
	}
}

func TestSync_FullRoundTrip(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create a zip with nested directory structure (mimics multi-level layout)
	zipPath := filepath.Join(tempDir, "roundtrip.zip")
	originalCSV := "id,name,value\n1,alpha,100\n2,beta,200\n"
	originalBin := string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x89, 0x50, 0x4E, 0x47})
	files := map[string]string{
		"data/records.csv":     originalCSV,
		"data/docs/report.bin": originalBin,
	}
	createTestZip(t, zipPath, files)

	cfg := DefaultConfig()

	// Phase 1: Open session, verify original content
	session, err := CreateSession(zipPath, "roundtrip-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	contentsDir, err := ContentsDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	gotCSV, err := os.ReadFile(filepath.Join(contentsDir, "data", "records.csv"))
	if err != nil {
		t.Fatalf("failed to read original csv: %v", err)
	}
	if string(gotCSV) != originalCSV {
		t.Fatalf("original csv mismatch: got %q, want %q", string(gotCSV), originalCSV)
	}

	gotBin, err := os.ReadFile(filepath.Join(contentsDir, "data", "docs", "report.bin"))
	if err != nil {
		t.Fatalf("failed to read original bin: %v", err)
	}
	if string(gotBin) != originalBin {
		t.Fatalf("original bin mismatch: got %x, want %x", gotBin, []byte(originalBin))
	}

	// Phase 2: Modify files (simulates xlq cell edits or any external tool)
	modifiedCSV := "id,name,value\n1,alpha,999\n2,beta,200\n3,gamma,300\n"
	modifiedBin := string([]byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE})

	err = os.WriteFile(filepath.Join(contentsDir, "data", "records.csv"), []byte(modifiedCSV), 0644)
	if err != nil {
		t.Fatalf("failed to write modified csv: %v", err)
	}

	err = os.WriteFile(filepath.Join(contentsDir, "data", "docs", "report.bin"), []byte(modifiedBin), 0644)
	if err != nil {
		t.Fatalf("failed to write modified bin: %v", err)
	}

	// Phase 3: Sync — write modifications back to the source zip
	result, err := Sync(session, false, cfg)
	if err != nil {
		t.Fatalf("failed to sync: %v", err)
	}
	if result.BackupPath == "" {
		t.Error("expected backup path to be set after sync")
	}

	// Phase 4: Close session (delete workspace)
	err = DeleteSession("roundtrip-test")
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Verify session is gone
	_, err = GetSession("roundtrip-test")
	if err == nil {
		t.Fatal("expected session to be deleted")
	}

	// Phase 5: Reopen from the same (now-synced) zip
	session2, err := CreateSession(zipPath, "roundtrip-verify", cfg)
	if err != nil {
		t.Fatalf("failed to create verification session: %v", err)
	}

	contentsDir2, err := ContentsDir(session2.Name)
	if err != nil {
		t.Fatalf("failed to get verification contents dir: %v", err)
	}

	// Phase 6: Verify modifications persisted through the full cycle
	gotCSV2, err := os.ReadFile(filepath.Join(contentsDir2, "data", "records.csv"))
	if err != nil {
		t.Fatalf("failed to read csv after round-trip: %v", err)
	}
	if string(gotCSV2) != modifiedCSV {
		t.Errorf("csv round-trip failed: got %q, want %q", string(gotCSV2), modifiedCSV)
	}

	gotBin2, err := os.ReadFile(filepath.Join(contentsDir2, "data", "docs", "report.bin"))
	if err != nil {
		t.Fatalf("failed to read bin after round-trip: %v", err)
	}
	if string(gotBin2) != modifiedBin {
		t.Errorf("bin round-trip failed: got %x, want %x", gotBin2, []byte(modifiedBin))
	}

	// Cleanup
	err = DeleteSession("roundtrip-verify")
	if err != nil {
		t.Fatalf("failed to delete verification session: %v", err)
	}
}

func TestRotateBackups_ZeroDepth(t *testing.T) {
	tempDir := t.TempDir()

	sourcePath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, sourcePath, map[string]string{"file.txt": "content"})

	// Rotate with depth 0 (should still create one backup)
	bakPath, err := RotateBackups(sourcePath, 0)
	if err != nil {
		t.Fatalf("failed to rotate: %v", err)
	}

	if bakPath == "" {
		t.Error("expected backup path to be returned")
	}
}
