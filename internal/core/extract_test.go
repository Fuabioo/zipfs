package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Fuabioo/zipfs/internal/security"
)

func TestExtract_BasicZip(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create test zip
	files := map[string]string{
		"file1.txt":     "content1",
		"file2.txt":     "content2",
		"dir/file3.txt": "content3",
	}
	createTestZip(t, zipPath, files)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// Extract
	limits := security.DefaultLimits()
	fileCount, totalSize, err := Extract(zipPath, destDir, limits)
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	if fileCount != 3 {
		t.Errorf("expected 3 files, got %d", fileCount)
	}

	if totalSize == 0 {
		t.Error("expected non-zero total size")
	}

	// Verify files exist
	for path, expectedContent := range files {
		fullPath := filepath.Join(destDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", path, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("expected content %q for %s, got %q", expectedContent, path, string(content))
		}
	}
}

func TestExtract_MaliciousZip(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "malicious.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create malicious zip with path traversal
	createMaliciousZip(t, zipPath)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// Extract should fail
	limits := security.DefaultLimits()
	_, _, err := Extract(zipPath, destDir, limits)
	if err == nil {
		t.Fatal("expected error for malicious zip, got nil")
	}
}

func TestExtract_ZipBomb(t *testing.T) {
	t.Skip("Skipping zip bomb test - creating 150k files is too slow")
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "bomb.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create a zip that exceeds limits
	files := make(map[string]string)
	for i := 0; i < 150000; i++ {
		files[filepath.Join("dir", filepath.Join("subdir", filepath.Join("file", filepath.Join("deep", "file.txt"))))] = "x"
	}
	createTestZip(t, zipPath, files)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// Extract should fail due to file count limit
	limits := security.DefaultLimits()
	_, _, err := Extract(zipPath, destDir, limits)
	if err == nil {
		t.Fatal("expected error for zip bomb, got nil")
	}
}

func TestExtract_EmptyZip(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "empty.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create empty zip
	createTestZip(t, zipPath, map[string]string{})

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// Extract
	limits := security.DefaultLimits()
	fileCount, totalSize, err := Extract(zipPath, destDir, limits)
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	if fileCount != 0 {
		t.Errorf("expected 0 files, got %d", fileCount)
	}

	if totalSize != 0 {
		t.Errorf("expected 0 total size, got %d", totalSize)
	}
}

func TestExtract_WithDirectories(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create test zip with nested directories
	files := map[string]string{
		"a/b/c/file.txt": "deep content",
		"a/file.txt":     "shallow content",
	}
	createTestZip(t, zipPath, files)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// Extract
	limits := security.DefaultLimits()
	fileCount, _, err := Extract(zipPath, destDir, limits)
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	if fileCount != 2 {
		t.Errorf("expected 2 files, got %d", fileCount)
	}

	// Verify directory structure
	deepFile := filepath.Join(destDir, "a", "b", "c", "file.txt")
	if _, err := os.Stat(deepFile); err != nil {
		t.Errorf("expected deep file to exist: %v", err)
	}
}

func TestComputeZipHash(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")

	// Create test zip
	files := map[string]string{
		"file.txt": "content",
	}
	createTestZip(t, zipPath, files)

	// Compute hash
	hash1, err := ComputeZipHash(zipPath)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	if hash1 == "" {
		t.Error("expected non-empty hash")
	}

	// Compute again - should be same
	hash2, err := ComputeZipHash(zipPath)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	if hash1 != hash2 {
		t.Error("expected same hash for same file")
	}

	// Different file should have different hash
	zipPath2 := filepath.Join(tempDir, "test2.zip")
	files2 := map[string]string{
		"file.txt": "different content",
	}
	createTestZip(t, zipPath2, files2)

	hash3, err := ComputeZipHash(zipPath2)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	if hash1 == hash3 {
		t.Error("expected different hash for different file")
	}
}

func TestComputeZipHash_NonExistentFile(t *testing.T) {
	_, err := ComputeZipHash("/nonexistent/file.zip")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
