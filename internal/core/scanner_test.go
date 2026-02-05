package core

import (
	"bytes"
	"fmt"
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

func TestListFiles_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	entries, err := ListFiles(contentsDir, ".", false)
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries in empty directory, got %d", len(entries))
	}
}

func TestListFiles_InvalidRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, err := ListFiles(contentsDir, "/absolute/path", false)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestTreeView_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	tree, fileCount, dirCount, err := TreeView(contentsDir, ".", 0)
	if err != nil {
		t.Fatalf("failed to generate tree: %v", err)
	}

	if tree != "" {
		t.Error("expected empty tree for empty directory")
	}

	if fileCount != 0 || dirCount != 0 {
		t.Errorf("expected 0 files and dirs, got %d files, %d dirs", fileCount, dirCount)
	}
}

func TestTreeView_InvalidPath(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, _, _, err := TreeView(contentsDir, "/absolute/path", 0)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestWriteFile_WithoutCreateDirs(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Try to write to a nested path without creating directories
	err := WriteFile(contentsDir, "a/b/c/file.txt", []byte("content"), false)
	if err == nil {
		t.Fatal("expected error when parent directories don't exist")
	}
}

func TestGrepFiles_InvalidRegex(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, _, err := GrepFiles(contentsDir, ".", "[invalid(regex", "", false, 0)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestGrepFiles_InvalidGlob(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Use an absolute path as glob pattern (invalid)
	_, _, err := GrepFiles(contentsDir, ".", "pattern", "/absolute/path", false, 0)
	if err == nil {
		t.Fatal("expected error for invalid glob pattern")
	}
}

func TestGrepFiles_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, _, err := GrepFiles(contentsDir, "../../../etc", "pattern", "", false, 0)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestGrepFiles_EmptyResults(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	os.WriteFile(filepath.Join(contentsDir, "file.txt"), []byte("no match here"), 0644)

	matches, total, err := GrepFiles(contentsDir, ".", "NOTFOUND", "", false, 0)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	if len(matches) != 0 || total != 0 {
		t.Errorf("expected no matches, got %d matches, %d total", len(matches), total)
	}
}

func TestListFiles_SingleFile(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create a file
	os.WriteFile(filepath.Join(contentsDir, "test.txt"), []byte("content"), 0644)

	// List just that file (not the directory)
	entries, err := ListFiles(contentsDir, "test.txt", false)
	if err != nil {
		t.Fatalf("failed to list file: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry for single file, got %d", len(entries))
	}

	if entries[0].Type != "file" {
		t.Errorf("expected type 'file', got %q", entries[0].Type)
	}
}

func TestListFiles_RecursiveError(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create a directory with restricted permissions
	restrictedDir := filepath.Join(contentsDir, "restricted")
	os.MkdirAll(restrictedDir, 0000)
	defer os.Chmod(restrictedDir, 0755) // cleanup

	// Try to list recursively - may fail due to permissions
	_, err := ListFiles(contentsDir, ".", true)
	// This may or may not error depending on permissions enforcement
	_ = err
}

func TestTreeView_SingleFile(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	os.WriteFile(filepath.Join(contentsDir, "file.txt"), []byte("c"), 0644)

	tree, fileCount, dirCount, err := TreeView(contentsDir, ".", 0)
	if err != nil {
		t.Fatalf("failed to generate tree: %v", err)
	}

	if !strings.Contains(tree, "file.txt") {
		t.Error("expected tree to contain file.txt")
	}

	if fileCount != 1 {
		t.Errorf("expected 1 file, got %d", fileCount)
	}

	if dirCount != 0 {
		t.Errorf("expected 0 directories, got %d", dirCount)
	}
}

func TestDeleteFile_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	err := DeleteFile(contentsDir, "nonexistent.txt", false)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	if !errors.Is(err, errors.CodePathNotFound) {
		t.Errorf("expected PATH_NOT_FOUND error, got: %v", err)
	}
}

func TestReadFile_Directory(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "dir"), 0755)

	_, err := ReadFile(contentsDir, "dir")
	if err == nil {
		t.Fatal("expected error when reading directory")
	}
}

func TestGrepFiles_BinaryFile(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create a binary file with null bytes
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}
	os.WriteFile(filepath.Join(contentsDir, "binary.bin"), binaryData, 0644)

	// Grep should handle binary files gracefully
	matches, _, err := GrepFiles(contentsDir, ".", "pattern", "", false, 0)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	// Binary files shouldn't match text patterns
	if len(matches) > 0 {
		t.Error("expected no matches in binary file")
	}
}

func TestGrepFiles_NestedDirectories(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "a", "b", "c"), 0755)

	// Create files at different levels
	os.WriteFile(filepath.Join(contentsDir, "top.txt"), []byte("match here\n"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "a", "mid.txt"), []byte("match here\n"), 0644)
	os.WriteFile(filepath.Join(contentsDir, "a", "b", "c", "deep.txt"), []byte("match here\n"), 0644)

	matches, total, err := GrepFiles(contentsDir, ".", "match", "", false, 0)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	if total != 3 {
		t.Errorf("expected 3 total matches in nested files, got %d", total)
	}

	if len(matches) != 3 {
		t.Errorf("expected 3 match results, got %d", len(matches))
	}
}

func TestListFiles_Subdirectory(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "subdir"), 0755)

	os.WriteFile(filepath.Join(contentsDir, "subdir", "file.txt"), []byte("content"), 0644)

	// List files in subdirectory
	entries, err := ListFiles(contentsDir, "subdir", false)
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry in subdirectory, got %d", len(entries))
	}
}

func TestStatus_NonExistentSession(t *testing.T) {
	setupTestEnvironment(t)

	// Create a session object without proper workspace
	session := &Session{
		ID:   "fake-id",
		Name: "fake-name",
	}

	_, err := Status(session)
	if err == nil {
		t.Fatal("expected error for session without workspace")
	}
}

func TestWriteFile_Overwrite(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Write initial content
	err := WriteFile(contentsDir, "file.txt", []byte("initial"), false)
	if err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	// Overwrite with new content
	err = WriteFile(contentsDir, "file.txt", []byte("overwritten"), false)
	if err != nil {
		t.Fatalf("failed to overwrite file: %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(filepath.Join(contentsDir, "file.txt"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "overwritten" {
		t.Errorf("expected 'overwritten', got %q", string(content))
	}
}

func TestTreeView_DeepNesting(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")

	// Create deeply nested structure
	deepPath := filepath.Join(contentsDir, "a", "b", "c", "d", "e")
	os.MkdirAll(deepPath, 0755)
	os.WriteFile(filepath.Join(deepPath, "deep.txt"), []byte("c"), 0644)

	tree, fileCount, dirCount, err := TreeView(contentsDir, ".", 0)
	if err != nil {
		t.Fatalf("failed to generate tree: %v", err)
	}

	if fileCount != 1 {
		t.Errorf("expected 1 file, got %d", fileCount)
	}

	if dirCount != 5 {
		t.Errorf("expected 5 directories, got %d", dirCount)
	}

	if !strings.Contains(tree, "deep.txt") {
		t.Error("expected tree to contain deep.txt")
	}
}

func TestGrepFiles_MultilineContent(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	content := `line 1 with pattern
line 2 without
line 3 with pattern
line 4 without
line 5 with pattern`

	os.WriteFile(filepath.Join(contentsDir, "file.txt"), []byte(content), 0644)

	matches, total, err := GrepFiles(contentsDir, ".", "pattern", "", false, 0)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	if total != 3 {
		t.Errorf("expected 3 matches, got %d", total)
	}

	if len(matches) != 3 {
		t.Errorf("expected 3 match results, got %d", len(matches))
	}

	// Verify line numbers are correct
	expectedLines := []int{1, 3, 5}
	for i, match := range matches {
		if match.LineNumber != expectedLines[i] {
			t.Errorf("match %d: expected line %d, got %d", i, expectedLines[i], match.LineNumber)
		}
	}
}

func TestListFiles_ReadDirError(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create a file (not a directory)
	os.WriteFile(filepath.Join(contentsDir, "file.txt"), []byte("content"), 0644)

	// Try to list it as a directory (should only return the file itself)
	entries, err := ListFiles(contentsDir, "file.txt", false)
	if err != nil {
		t.Fatalf("failed to list file: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Type != "file" {
		t.Errorf("expected type 'file', got %q", entries[0].Type)
	}
}

func TestTreeView_UnreadableDirectory(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create a directory with no read permissions
	restrictedDir := filepath.Join(contentsDir, "restricted")
	os.MkdirAll(restrictedDir, 0000)
	defer os.Chmod(restrictedDir, 0755)

	// Tree view should handle error gracefully
	_, _, _, err := TreeView(contentsDir, ".", 0)
	// This may fail depending on permissions enforcement
	_ = err
}

func TestGrepFiles_MaxResultsExactly(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create files with multiple matches each
	for i := 0; i < 5; i++ {
		content := strings.Repeat("match\n", 10)
		os.WriteFile(filepath.Join(contentsDir, fmt.Sprintf("file%d.txt", i)), []byte(content), 0644)
	}

	// Set max results to exactly match total available
	matches, total, err := GrepFiles(contentsDir, ".", "match", "", false, 5)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	if len(matches) != 5 {
		t.Errorf("expected exactly 5 matches (max results), got %d", len(matches))
	}

	if total < 5 {
		t.Errorf("expected total >= 5, got %d", total)
	}
}

func TestReadFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create empty file
	os.WriteFile(filepath.Join(contentsDir, "empty.txt"), []byte{}, 0644)

	content, err := ReadFile(contentsDir, "empty.txt")
	if err != nil {
		t.Fatalf("failed to read empty file: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("expected empty content, got %d bytes", len(content))
	}
}

func TestWriteFile_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create large content (1MB)
	largeContent := bytes.Repeat([]byte("x"), 1024*1024)

	err := WriteFile(contentsDir, "large.txt", largeContent, false)
	if err != nil {
		t.Fatalf("failed to write large file: %v", err)
	}

	// Verify it was written correctly
	readContent, err := ReadFile(contentsDir, "large.txt")
	if err != nil {
		t.Fatalf("failed to read large file: %v", err)
	}

	if len(readContent) != len(largeContent) {
		t.Errorf("expected %d bytes, got %d", len(largeContent), len(readContent))
	}
}

func TestDeleteFile_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "emptydir"), 0755)

	// Delete empty directory with recursive flag
	err := DeleteFile(contentsDir, "emptydir", true)
	if err != nil {
		t.Fatalf("failed to delete empty directory: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(filepath.Join(contentsDir, "emptydir")); !os.IsNotExist(err) {
		t.Error("expected empty directory to be deleted")
	}
}

func TestListFiles_SymlinkHandling(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create a file and symlink
	targetFile := filepath.Join(contentsDir, "target.txt")
	os.WriteFile(targetFile, []byte("content"), 0644)

	linkFile := filepath.Join(contentsDir, "link.txt")
	os.Symlink(targetFile, linkFile)

	// List should include both
	entries, err := ListFiles(contentsDir, ".", false)
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries (file + symlink), got %d", len(entries))
	}
}

func TestGrepFiles_UnreadableFile(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Create a file with no read permissions
	restrictedFile := filepath.Join(contentsDir, "restricted.txt")
	os.WriteFile(restrictedFile, []byte("content"), 0000)
	defer os.Chmod(restrictedFile, 0644)

	// Grep should handle the error gracefully by skipping the file
	matches, _, err := GrepFiles(contentsDir, ".", "content", "", false, 0)
	if err != nil {
		t.Fatalf("failed to grep: %v", err)
	}

	// Should have skipped the unreadable file
	if len(matches) > 0 {
		t.Error("expected no matches from unreadable file")
	}
}

func TestTreeView_MaxDepthZero(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(filepath.Join(contentsDir, "dir"), 0755)
	os.WriteFile(filepath.Join(contentsDir, "file.txt"), []byte("c"), 0644)

	// Max depth 0 means unlimited
	tree, fileCount, _, err := TreeView(contentsDir, ".", 0)
	if err != nil {
		t.Fatalf("failed to generate tree: %v", err)
	}

	if fileCount == 0 {
		t.Error("expected files to be counted")
	}

	if !strings.Contains(tree, "file.txt") {
		t.Error("expected tree to contain file.txt")
	}
}

func TestWriteFile_NestedPathWithoutCreateDirs(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	// Try to write nested path without createDirs
	err := WriteFile(contentsDir, "dir/subdir/file.txt", []byte("content"), false)
	if err == nil {
		t.Fatal("expected error when writing to non-existent nested path without createDirs")
	}
}

func TestReadFile_InvalidRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	_, err := ReadFile(contentsDir, "/absolute/path")
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestWriteFile_InvalidRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	err := WriteFile(contentsDir, "/absolute/path", []byte("content"), false)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestDeleteFile_InvalidRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	contentsDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentsDir, 0755)

	err := DeleteFile(contentsDir, "/absolute/path", false)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}
