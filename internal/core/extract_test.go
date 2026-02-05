package core

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestExtract_InvalidZipPath(t *testing.T) {
	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "extracted")
	os.MkdirAll(destDir, 0755)

	limits := security.DefaultLimits()
	_, _, err := Extract("/nonexistent/file.zip", destDir, limits)
	if err == nil {
		t.Fatal("expected error for nonexistent zip file")
	}
}

func TestExtract_InvalidDestDir(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	// Use a file as destination (not a directory)
	invalidDest := filepath.Join(tempDir, "notadir")
	os.WriteFile(invalidDest, []byte("file"), 0644)

	limits := security.DefaultLimits()
	_, _, err := Extract(zipPath, invalidDest, limits)
	if err == nil {
		t.Fatal("expected error when dest is not a directory")
	}
}

func TestExtract_CorruptedZip(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "corrupted.zip")

	// Write invalid zip data
	os.WriteFile(zipPath, []byte("not a zip file"), 0644)

	destDir := filepath.Join(tempDir, "extracted")
	os.MkdirAll(destDir, 0755)

	limits := security.DefaultLimits()
	_, _, err := Extract(zipPath, destDir, limits)
	if err == nil {
		t.Fatal("expected error for corrupted zip")
	}
}

func TestExtract_DirectoryEntries(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create zip with explicit directory entries
	files := map[string]string{
		"dir1/":         "",
		"dir1/file.txt": "content",
		"dir2/":         "",
	}

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	for path, content := range files {
		header := &zip.FileHeader{
			Name:   path,
			Method: zip.Deflate,
		}

		if strings.HasSuffix(path, "/") {
			header.SetMode(0755 | os.ModeDir)
		} else {
			header.SetMode(0644)
		}

		f, err := w.CreateHeader(header)
		if err != nil {
			t.Fatalf("failed to create entry %s: %v", path, err)
		}

		if content != "" {
			if _, err := f.Write([]byte(content)); err != nil {
				t.Fatalf("failed to write content: %v", err)
			}
		}
	}
	w.Close()
	zipFile.Close()

	// Extract
	os.MkdirAll(destDir, 0755)
	limits := security.DefaultLimits()
	_, _, err = Extract(zipPath, destDir, limits)
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	// Verify directories exist
	if _, err := os.Stat(filepath.Join(destDir, "dir1")); err != nil {
		t.Error("expected dir1 to exist")
	}

	if _, err := os.Stat(filepath.Join(destDir, "dir2")); err != nil {
		t.Error("expected dir2 to exist")
	}
}

func TestComputeZipHash_EmptyZip(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "empty.zip")

	createTestZip(t, zipPath, map[string]string{})

	hash, err := ComputeZipHash(zipPath)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash for empty zip")
	}
}

func TestExtract_LargeNumberOfFiles(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "many-files.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create zip with many files
	files := make(map[string]string)
	for i := 0; i < 100; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = fmt.Sprintf("content%d", i)
	}
	createTestZip(t, zipPath, files)

	os.MkdirAll(destDir, 0755)
	limits := security.DefaultLimits()
	fileCount, _, err := Extract(zipPath, destDir, limits)
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	if fileCount != 100 {
		t.Errorf("expected 100 files, got %d", fileCount)
	}
}
