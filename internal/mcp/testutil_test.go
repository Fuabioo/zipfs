package mcp

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// createTestZip creates a simple test zip file with the given files.
// files is a map of path -> content.
func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	for path, content := range files {
		// Create directories if needed
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			header := &zip.FileHeader{
				Name:   dir + "/",
				Method: zip.Store,
			}
			header.SetMode(0755 | os.ModeDir)
			if _, err := w.CreateHeader(header); err != nil {
				t.Fatalf("failed to create directory %s: %v", dir, err)
			}
		}

		// Create file
		f, err := w.Create(path)
		if err != nil {
			t.Fatalf("failed to create file %s in zip: %v", path, err)
		}

		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write content to %s: %v", path, err)
		}
	}
}

// setupTestEnvironment sets up a clean test environment with custom data dir.
func setupTestEnvironment(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	os.Setenv("ZIPFS_DATA_DIR", tempDir)
	t.Cleanup(func() {
		os.Unsetenv("ZIPFS_DATA_DIR")
	})

	return tempDir
}
