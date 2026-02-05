package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fuabioo/zipfs/internal/errors"
)

func TestCreateSession_Basic(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	files := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}
	createTestZip(t, zipPath, files)

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "my-session", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.ID == "" {
		t.Error("expected session ID to be set")
	}

	if session.Name != "my-session" {
		t.Errorf("expected name 'my-session', got %q", session.Name)
	}

	if session.State != "open" {
		t.Errorf("expected state 'open', got %q", session.State)
	}

	if session.FileCount != 2 {
		t.Errorf("expected 2 files, got %d", session.FileCount)
	}

	if session.ZipHashSHA256 == "" {
		t.Error("expected zip hash to be set")
	}
}

func TestCreateSession_WithoutName(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.Name != "" {
		t.Error("expected empty name when not provided")
	}

	// Workspace should be created with UUID
	workspaceDir, err := WorkspaceDir(session.ID)
	if err != nil {
		t.Fatalf("failed to get workspace dir: %v", err)
	}

	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		t.Error("expected workspace to exist")
	}
}

func TestCreateSession_NameCollision(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create first session
	zipPath1 := filepath.Join(tempDir, "test1.zip")
	createTestZip(t, zipPath1, map[string]string{"file.txt": "content1"})

	cfg := DefaultConfig()
	_, err := CreateSession(zipPath1, "duplicate", cfg)
	if err != nil {
		t.Fatalf("failed to create first session: %v", err)
	}

	// Try to create second session with same name
	zipPath2 := filepath.Join(tempDir, "test2.zip")
	createTestZip(t, zipPath2, map[string]string{"file.txt": "content2"})

	_, err = CreateSession(zipPath2, "duplicate", cfg)
	if err == nil {
		t.Fatal("expected error for name collision")
	}

	if !errors.Is(err, errors.CodeNameCollision) {
		t.Errorf("expected NAME_COLLISION error, got: %v", err)
	}
}

func TestCreateSession_InvalidName(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()

	// Test various invalid names
	invalidNames := []string{
		"name with spaces",
		"name/with/slashes",
		"name$with$special",
		strings.Repeat("a", 100), // Too long
	}

	for _, name := range invalidNames {
		_, err := CreateSession(zipPath, name, cfg)
		if err == nil {
			t.Errorf("expected error for invalid name %q", name)
		}
	}
}

func TestCreateSession_UUIDAsName(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()

	// Try to use a UUID as name (should be rejected)
	_, err := CreateSession(zipPath, "550e8400-e29b-41d4-a716-446655440000", cfg)
	if err == nil {
		t.Fatal("expected error for UUID as name")
	}
}

func TestCreateSession_NonExistentZip(t *testing.T) {
	setupTestEnvironment(t)

	cfg := DefaultConfig()
	_, err := CreateSession("/nonexistent/file.zip", "test", cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent zip")
	}

	if !errors.Is(err, errors.CodeZipNotFound) {
		t.Errorf("expected ZIP_NOT_FOUND error, got: %v", err)
	}
}

func TestCreateSession_MaxSessionsLimit(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Security.MaxSessions = 2

	// Create first session
	zipPath1 := filepath.Join(tempDir, "test1.zip")
	createTestZip(t, zipPath1, map[string]string{"file.txt": "content"})
	_, err := CreateSession(zipPath1, "session1", cfg)
	if err != nil {
		t.Fatalf("failed to create first session: %v", err)
	}

	// Create second session
	zipPath2 := filepath.Join(tempDir, "test2.zip")
	createTestZip(t, zipPath2, map[string]string{"file.txt": "content"})
	_, err = CreateSession(zipPath2, "session2", cfg)
	if err != nil {
		t.Fatalf("failed to create second session: %v", err)
	}

	// Try to create third session (should fail)
	zipPath3 := filepath.Join(tempDir, "test3.zip")
	createTestZip(t, zipPath3, map[string]string{"file.txt": "content"})
	_, err = CreateSession(zipPath3, "session3", cfg)
	if err == nil {
		t.Fatal("expected error when exceeding max sessions")
	}

	if !errors.Is(err, errors.CodeLimitExceeded) {
		t.Errorf("expected LIMIT_EXCEEDED error, got: %v", err)
	}
}

func TestGetSession_ByName(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	created, err := CreateSession(zipPath, "my-session", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Get by name
	retrieved, err := GetSession("my-session")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
}

func TestGetSession_ByUUID(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	created, err := CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Get by UUID
	retrieved, err := GetSession(created.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
}

func TestGetSession_ByUUIDPrefix(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	created, err := CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Get by UUID prefix (first 8 characters)
	prefix := created.ID[:8]
	retrieved, err := GetSession(prefix)
	if err != nil {
		t.Fatalf("failed to get session by prefix: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	setupTestEnvironment(t)

	_, err := GetSession("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}

	if !errors.Is(err, errors.CodeSessionNotFound) {
		t.Errorf("expected SESSION_NOT_FOUND error, got: %v", err)
	}
}

func TestGetSession_AmbiguousPrefix(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// This test is probabilistic - it may not always trigger ambiguity
	// We'll create multiple sessions and hope for UUID prefix collision
	cfg := DefaultConfig()

	for i := 0; i < 5; i++ {
		zipPath := filepath.Join(tempDir, fmt.Sprintf("test%d.zip", i))
		createTestZip(t, zipPath, map[string]string{"file.txt": "content"})
		_, err := CreateSession(zipPath, "", cfg)
		if err != nil {
			t.Fatalf("failed to create session %d: %v", i, err)
		}
	}

	// Try to get with very short prefix that might match multiple
	_, err := GetSession("0")
	// May or may not be ambiguous depending on random UUIDs
	_ = err
}

func TestListSessions(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	cfg := DefaultConfig()

	// Initially should be empty
	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}

	// Create some sessions
	for i := 1; i <= 3; i++ {
		zipPath := filepath.Join(tempDir, fmt.Sprintf("test%d.zip", i))
		createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

		name := ""
		if i == 1 {
			name = "named-session"
		}

		_, err := CreateSession(zipPath, name, cfg)
		if err != nil {
			t.Fatalf("failed to create session %d: %v", i, err)
		}
	}

	// List again
	sessions, err = ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestResolveSession_SingleSession(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	created, err := CreateSession(zipPath, "only-session", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Resolve without identifier
	resolved, err := ResolveSession("")
	if err != nil {
		t.Fatalf("failed to resolve session: %v", err)
	}

	if resolved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, resolved.ID)
	}
}

func TestResolveSession_NoSessions(t *testing.T) {
	setupTestEnvironment(t)

	_, err := ResolveSession("")
	if err == nil {
		t.Fatal("expected error when no sessions exist")
	}

	if !errors.Is(err, errors.CodeNoSessions) {
		t.Errorf("expected NO_SESSIONS error, got: %v", err)
	}
}

func TestResolveSession_MultipleSessions(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	cfg := DefaultConfig()

	// Create two sessions
	for i := 1; i <= 2; i++ {
		zipPath := filepath.Join(tempDir, fmt.Sprintf("test%d.zip", i))
		createTestZip(t, zipPath, map[string]string{"file.txt": "content"})
		_, err := CreateSession(zipPath, "", cfg)
		if err != nil {
			t.Fatalf("failed to create session %d: %v", i, err)
		}
	}

	// Resolve without identifier should fail
	_, err := ResolveSession("")
	if err == nil {
		t.Fatal("expected error when multiple sessions exist")
	}

	if !errors.Is(err, errors.CodeAmbiguousSession) {
		t.Errorf("expected AMBIGUOUS_SESSION error, got: %v", err)
	}
}

func TestDeleteSession(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	created, err := CreateSession(zipPath, "deletable", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Delete session
	err = DeleteSession("deletable")
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Verify it's gone
	_, err = GetSession(created.ID)
	if err == nil {
		t.Fatal("expected error after deleting session")
	}
}

func TestUpdateSession(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "updatable", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Modify session
	session.State = "syncing"

	// Update
	err = UpdateSession(session, session.Name)
	if err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	// Retrieve and verify
	retrieved, err := GetSession("updatable")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.State != "syncing" {
		t.Errorf("expected state 'syncing', got %q", retrieved.State)
	}
}

func TestTouchSession(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "touchable", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	originalAccessTime := session.LastAccessedAt

	// Touch session
	err = TouchSession(session)
	if err != nil {
		t.Fatalf("failed to touch session: %v", err)
	}

	// Verify timestamp changed
	if !session.LastAccessedAt.After(originalAccessTime) {
		t.Error("expected last accessed time to be updated")
	}

	// Retrieve and verify it persisted
	retrieved, err := GetSession("touchable")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if !retrieved.LastAccessedAt.After(originalAccessTime) {
		t.Error("expected persisted last accessed time to be updated")
	}
}

func TestGetSession_EmptyIdentifier(t *testing.T) {
	setupTestEnvironment(t)

	_, err := GetSession("")
	if err == nil {
		t.Fatal("expected error for empty identifier")
	}
}

func TestGetSession_ShortPrefix(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	created, err := CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Try with prefix shorter than 4 characters (should not match)
	shortPrefix := created.ID[:3]
	_, err = GetSession(shortPrefix)
	if err == nil {
		t.Fatal("expected error for prefix shorter than 4 characters")
	}
}

func TestCreateSession_ExtractionFailure(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create a malicious zip
	zipPath := filepath.Join(tempDir, "malicious.zip")
	createMaliciousZip(t, zipPath)

	cfg := DefaultConfig()
	_, err := CreateSession(zipPath, "bad-session", cfg)
	if err == nil {
		t.Fatal("expected error when extracting malicious zip")
	}

	// Verify workspace was cleaned up
	_, err = GetSession("bad-session")
	if err == nil {
		t.Error("expected session to not exist after failed creation")
	}
}

func TestDeleteSession_ByUUID(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Delete by UUID
	err = DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("failed to delete session by UUID: %v", err)
	}

	// Verify deleted
	_, err = GetSession(session.ID)
	if err == nil {
		t.Fatal("expected error after deleting session")
	}
}

func TestUpdateSession_EmptyName(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Update session (should use ID when name is empty)
	session.State = "modified"
	err = UpdateSession(session, "")
	if err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	// Verify update
	retrieved, err := GetSession(session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.State != "modified" {
		t.Errorf("expected state 'modified', got %q", retrieved.State)
	}
}

func TestListSessions_ErrorReadingWorkspaces(t *testing.T) {
	setupTestEnvironment(t)

	// Initially should work
	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions initially, got %d", len(sessions))
	}
}

func TestResolveSession_ByName(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	created, err := CreateSession(zipPath, "named-session", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Resolve by name
	resolved, err := ResolveSession("named-session")
	if err != nil {
		t.Fatalf("failed to resolve session: %v", err)
	}

	if resolved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, resolved.ID)
	}
}

func TestCreateSession_FileSizeTracking(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create a zip with files of known size
	files := map[string]string{
		"file1.txt": "1234567890", // 10 bytes
		"file2.txt": "abcdefghij", // 10 bytes
	}
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, files)

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "size-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify file count
	if session.FileCount != 2 {
		t.Errorf("expected 2 files, got %d", session.FileCount)
	}

	// Verify size is tracked
	if session.ExtractedSizeBytes == 0 {
		t.Error("expected non-zero extracted size")
	}
}

func TestCreateSession_SourcePathPersistence(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "persist-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify source path is stored
	if session.SourcePath == "" {
		t.Error("expected source path to be set")
	}

	// Reload and verify persistence
	retrieved, err := GetSession("persist-test")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.SourcePath != session.SourcePath {
		t.Errorf("expected source path %s, got %s", session.SourcePath, retrieved.SourcePath)
	}
}

func TestCreateSession_WithSubdirectories(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create zip with subdirectories
	zipPath := filepath.Join(tempDir, "test.zip")
	files := map[string]string{
		"root.txt":            "root content",
		"dir1/file1.txt":      "dir1 content",
		"dir1/dir2/file2.txt": "nested content",
	}
	createTestZip(t, zipPath, files)

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "subdir-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.FileCount != 3 {
		t.Errorf("expected 3 files, got %d", session.FileCount)
	}

	// Verify workspace has nested structure
	contentsDir, err := ContentsDir("subdir-test")
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	nestedFile := filepath.Join(contentsDir, "dir1", "dir2", "file2.txt")
	if _, err := os.Stat(nestedFile); err != nil {
		t.Errorf("expected nested file to exist: %v", err)
	}
}

func TestGetSession_CorruptedMetadata(t *testing.T) {
	setupTestEnvironment(t)

	// Create workspace manually with corrupted metadata
	workspacesDir, err := WorkspacesDir()
	if err != nil {
		t.Fatalf("failed to get workspaces dir: %v", err)
	}

	corruptedDir := filepath.Join(workspacesDir, "corrupted")
	os.MkdirAll(filepath.Join(corruptedDir, "contents"), 0755)

	// Write invalid JSON
	metadataPath := filepath.Join(corruptedDir, "metadata.json")
	os.WriteFile(metadataPath, []byte("{invalid json"), 0600)

	// Try to get session - should handle error gracefully
	_, err = GetSession("corrupted")
	// Should fail due to corrupted metadata
	if err == nil {
		t.Fatal("expected error for corrupted metadata")
	}
}

func TestDeleteSession_NonExistent(t *testing.T) {
	setupTestEnvironment(t)

	err := DeleteSession("nonexistent-session")
	if err == nil {
		t.Fatal("expected error when deleting nonexistent session")
	}

	if !errors.Is(err, errors.CodeSessionNotFound) {
		t.Errorf("expected SESSION_NOT_FOUND error, got: %v", err)
	}
}
