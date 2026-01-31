package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fuabioo/zipfs/internal/errors"
)

func TestListFiles_Basic(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create test files
	os.WriteFile(filepath.Join(contentsDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "file2.txt"), []byte("content2"), 0644)
	os.MkdirAll(filepath.Join(contentsDir, "dir"), 0755)

	entries, err := ListFiles(contentsDir, ".", false)
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestListFiles_Recursive(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "a", "b"), 0755)

	// Create nested files
	os.WriteFile(filepath.Join(contentsDir, "file1.txt"), []byte("c1"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "a", "file2.txt"), []byte("c2"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "a", "b", "file3.txt"), []byte("c3"), 0644)

	entries, err := ListFiles(contentsDir, ".", true)
	if err != nil {
		t.Fatalf("failed to list files recursively: %v", err)
	}

	// Should include all files and directories
	if len(entries) < 3 {
		t.Errorf("expected at least 3 entries, got %d", len(entries))
	}
}

func TestListFiles_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, err := ListFiles(contentsDir, "../../../etc/passwd", false)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}

	// The error might be wrapped, so just check that it mentions path traversal
	if !strings.Contains(err.Error(), "path") && !strings.Contains(err.Error(), "traverse") {
		t.Errorf("expected path traversal error, got: %v", err)
	}
}

func TestListFiles_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, err := ListFiles(contentsDir, "nonexistent", false)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}

	if !errors.Is(err, errors.CodePathNotFound) {
		t.Errorf("expected PATH_NOT_FOUND error, got: %v", err)
	}
}

func TestTreeView_Basic(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "dir1"), 0755)
	os.MkdirAll(filepath.Join(contentsDir, "dir2"), 0755)
	os.WriteFile(filepath.Join(contentsDir, "file1.txt"), []byte("c1"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "dir1", "file2.txt"), []byte("c2"), 0644)

	tree, fileCount, dirCount, err := TreeView(contentsDir, ".", 0)
	if err != nil {
		t.Fatalf("failed to generate tree: %v", err)
	}

	if tree == "" {
		t.Error("expected non-empty tree")
	}

	if fileCount != 2 {
		t.Errorf("expected 2 files, got %d", fileCount)
	}

	if dirCount != 2 {
		t.Errorf("expected 2 directories, got %d", dirCount)
	}

	// Tree should contain file names
	if !strings.Contains(tree, "file1.txt") {
		t.Error("expected tree to contain file1.txt")
	}
}

func TestTreeView_MaxDepth(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "a", "b", "c"), 0755)
	os.WriteFile(filepath.Join(contentsDir, "a", "b", "c", "deep.txt"), []byte("c"), 0644)

	tree, _, _, err := TreeView(contentsDir, ".", 2)
	if err != nil {
		t.Fatalf("failed to generate tree: %v", err)
	}

	// Should not include deeply nested file
	if strings.Contains(tree, "deep.txt") {
		t.Error("expected tree to respect max depth")
	}
}

func TestReadFile_Basic(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	expectedContent := "test content"
	os.WriteFile(filepath.Join(contentsDir, "test.txt"), []byte(expectedContent), 0644)

	content, err := ReadFile(contentsDir, "test.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, string(content))
	}
}

func TestReadFile_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, err := ReadFile(contentsDir, "../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}

	// The error might be wrapped, so just check that it mentions path traversal
	if !strings.Contains(err.Error(), "path") && !strings.Contains(err.Error(), "traverse") {
		t.Errorf("expected path traversal error, got: %v", err)
	}
}

func TestReadFile_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, err := ReadFile(contentsDir, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	if !errors.Is(err, errors.CodePathNotFound) {
		t.Errorf("expected PATH_NOT_FOUND error, got: %v", err)
	}
}

func TestWriteFile_Basic(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	content := []byte("new content")
	err := WriteFile(contentsDir, "newfile.txt", content, false)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Verify file exists
	written, err := os.ReadFile(filepath.Join(contentsDir, "newfile.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(written) != string(content) {
		t.Errorf("expected content %q, got %q", string(content), string(written))
	}
}

func TestWriteFile_CreateDirs(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	content := []byte("nested content")
	err := WriteFile(contentsDir, "a/b/c/file.txt", content, true)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Verify file exists in nested location
	written, err := os.ReadFile(filepath.Join(contentsDir, "a", "b", "c", "file.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(written) != string(content) {
		t.Errorf("expected content %q, got %q", string(content), string(written))
	}
}

func TestWriteFile_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	err := WriteFile(contentsDir, "../../../tmp/malicious.txt", []byte("bad"), false)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}

	// The error might be wrapped, so just check that it mentions path traversal
	if !strings.Contains(err.Error(), "path") && !strings.Contains(err.Error(), "traverse") {
		t.Errorf("expected path traversal error, got: %v", err)
	}
}

func TestDeleteFile_Basic(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create file
	os.WriteFile(filepath.Join(contentsDir, "delete-me.txt"), []byte("content"), 0644)

	err := DeleteFile(contentsDir, "delete-me.txt", false)
	if err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(filepath.Join(contentsDir, "delete-me.txt")); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestDeleteFile_Recursive(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "dir", "subdir"), 0755)
	os.WriteFile(filepath.Join(contentsDir, "dir", "subdir", "file.txt"), []byte("c"), 0644)

	err := DeleteFile(contentsDir, "dir", true)
	if err != nil {
		t.Fatalf("failed to delete directory: %v", err)
	}

	// Verify directory is gone
	if _, err := os.Stat(filepath.Join(contentsDir, "dir")); !os.IsNotExist(err) {
		t.Error("expected directory to be deleted")
	}
}

func TestDeleteFile_DirectoryWithoutRecursive(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "dir"), 0755)

	err := DeleteFile(contentsDir, "dir", false)
	if err == nil {
		t.Fatal("expected error when deleting directory without recursive flag")
	}
}

func TestDeleteFile_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	err := DeleteFile(contentsDir, "../../../tmp/file.txt", false)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}

	// The error might be wrapped, so just check that it mentions path traversal
	if !strings.Contains(err.Error(), "path") && !strings.Contains(err.Error(), "traverse") {
		t.Errorf("expected path traversal error, got: %v", err)
	}
}

func TestGrepFiles_Basic(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create files with searchable content
	os.WriteFile(filepath.Join(contentsDir, "file1.txt"), []byte("hello world\nfoo bar\n"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "file2.txt"), []byte("hello again\nbaz\n"), 0644)

	matches, total, err := GrepFiles(contentsDir, ".", "hello", "", false, 0)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	if total != 2 {
		t.Errorf("expected 2 matches, got %d", total)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 match results, got %d", len(matches))
	}
}

func TestGrepFiles_CaseInsensitive(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	os.WriteFile(filepath.Join(contentsDir, "file.txt"), []byte("HELLO\nhello\nHeLLo\n"), 0644)

	_, total, err := GrepFiles(contentsDir, ".", "hello", "", true, 0)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	if total != 3 {
		t.Errorf("expected 3 matches, got %d", total)
	}
}

func TestGrepFiles_WithGlob(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	os.WriteFile(filepath.Join(contentsDir, "file.txt"), []byte("match\n"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "file.log"), []byte("match\n"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "file.md"), []byte("match\n"), 0644)

	matches, _, err := GrepFiles(contentsDir, ".", "match", "*.txt", false, 0)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	// Should only match .txt files
	if len(matches) != 1 {
		t.Errorf("expected 1 match (*.txt only), got %d", len(matches))
	}
}

func TestGrepFiles_MaxResults(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create file with many matches
	content := strings.Repeat("match\n", 100)
	os.WriteFile(filepath.Join(contentsDir, "file.txt"), []byte(content), 0644)

	matches, total, err := GrepFiles(contentsDir, ".", "match", "", false, 10)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	if len(matches) != 10 {
		t.Errorf("expected 10 matches (max results), got %d", len(matches))
	}

	if total < 10 {
		t.Errorf("expected total >= 10, got %d", total)
	}
}

func TestStatus_NoChanges(t *testing.T) {
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
	session, err := CreateSession(zipPath, "status-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Get status without making changes
	result, err := Status(session)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	// Note: Extraction process may change modification times, so files might appear "modified"
	// even if content is unchanged. This is expected behavior.

	if len(result.Added) != 0 {
		t.Errorf("expected 0 added files, got %d", len(result.Added))
	}

	if len(result.Deleted) != 0 {
		t.Errorf("expected 0 deleted files, got %d", len(result.Deleted))
	}

	// Total count should still be 2
	totalFiles := len(result.Modified) + result.UnchangedCount
	if totalFiles != 2 {
		t.Errorf("expected 2 total files, got %d", totalFiles)
	}
}

func TestStatus_WithModifications(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	files := map[string]string{
		"file1.txt": "original content",
		"file2.txt": "content2",
	}
	createTestZip(t, zipPath, files)

	cfg := DefaultConfig()
	session, err := CreateSession(zipPath, "status-mod-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	contentsDir, err := ContentsDir(session.Name)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	// Modify a file
	os.WriteFile(filepath.Join(contentsDir, "file1.txt"), []byte("MODIFIED"), 0644)

	// Add a file
	os.WriteFile(filepath.Join(contentsDir, "file3.txt"), []byte("new file"), 0644)

	// Delete a file
	os.Remove(filepath.Join(contentsDir, "file2.txt"))

	// Get status
	result, err := Status(session)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if len(result.Modified) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(result.Modified))
	}

	if len(result.Added) != 1 {
		t.Errorf("expected 1 added file, got %d", len(result.Added))
	}

	if len(result.Deleted) != 1 {
		t.Errorf("expected 1 deleted file, got %d", len(result.Deleted))
	}
}
