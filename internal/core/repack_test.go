package core

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Fuabioo/zipfs/internal/security"
)

func TestRepack_Basic(t *testing.T) {
	tempDir := t.TempDir()

	// Create source directory with files
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Create test files
	files := map[string]string{
		"file1.txt":     "content1",
		"file2.txt":     "content2",
		"dir/file3.txt": "content3",
	}

	for path, content := range files {
		fullPath := filepath.Join(sourceDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	// Repack into zip
	zipPath := filepath.Join(tempDir, "repacked.zip")
	if err := Repack(sourceDir, zipPath); err != nil {
		t.Fatalf("failed to repack: %v", err)
	}

	// Verify zip exists
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("zip file doesn't exist: %v", err)
	}

	// Extract and verify contents
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("failed to create extract dir: %v", err)
	}

	limits := security.DefaultLimits()
	_, _, err := Extract(zipPath, extractDir, limits)
	if err != nil {
		t.Fatalf("failed to extract repacked zip: %v", err)
	}

	// Verify all files are present with correct content
	for path, expectedContent := range files {
		fullPath := filepath.Join(extractDir, path)
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

func TestRepack_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create empty source directory
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Repack into zip
	zipPath := filepath.Join(tempDir, "empty.zip")
	if err := Repack(sourceDir, zipPath); err != nil {
		t.Fatalf("failed to repack: %v", err)
	}

	// Verify zip exists
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("zip file doesn't exist: %v", err)
	}
}

func TestRepack_NestedDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create source directory with deeply nested files
	sourceDir := filepath.Join(tempDir, "source")
	deepPath := filepath.Join(sourceDir, "a", "b", "c", "d")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatalf("failed to create deep dir: %v", err)
	}

	// Create file in deep directory
	filePath := filepath.Join(deepPath, "deep.txt")
	if err := os.WriteFile(filePath, []byte("deep content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Repack into zip
	zipPath := filepath.Join(tempDir, "nested.zip")
	if err := Repack(sourceDir, zipPath); err != nil {
		t.Fatalf("failed to repack: %v", err)
	}

	// Verify zip contains the deep file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	found := false
	for _, f := range r.File {
		if f.Name == "a/b/c/d/deep.txt" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to find deep file in zip")
	}
}

func TestRepack_PreservesFilePermissions(t *testing.T) {
	tempDir := t.TempDir()

	// Create source directory
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Create executable file
	execPath := filepath.Join(sourceDir, "script.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho hello"), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Repack into zip
	zipPath := filepath.Join(tempDir, "perms.zip")
	if err := Repack(sourceDir, zipPath); err != nil {
		t.Fatalf("failed to repack: %v", err)
	}

	// Check zip entry has correct mode
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "script.sh" {
			mode := f.Mode()
			// Check if executable bit is set
			if mode&0111 == 0 {
				t.Error("expected executable bit to be set")
			}
		}
	}
}

func TestRepack_NonExistentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")

	err := Repack("/nonexistent/directory", zipPath)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestRepack_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()

	// Create original zip
	originalZip := filepath.Join(tempDir, "original.zip")
	files := map[string]string{
		"readme.txt":      "README content",
		"src/main.go":     "package main",
		"docs/manual.txt": "User manual",
	}
	createTestZip(t, originalZip, files)

	// Extract
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("failed to create extract dir: %v", err)
	}

	limits := security.DefaultLimits()
	_, _, err := Extract(originalZip, extractDir, limits)
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	// Modify a file
	modifiedFile := filepath.Join(extractDir, "readme.txt")
	if err := os.WriteFile(modifiedFile, []byte("MODIFIED README"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Repack
	repackedZip := filepath.Join(tempDir, "repacked.zip")
	if err := Repack(extractDir, repackedZip); err != nil {
		t.Fatalf("failed to repack: %v", err)
	}

	// Extract repacked zip
	extractDir2 := filepath.Join(tempDir, "extracted2")
	if err := os.MkdirAll(extractDir2, 0755); err != nil {
		t.Fatalf("failed to create extract dir: %v", err)
	}

	_, _, err = Extract(repackedZip, extractDir2, limits)
	if err != nil {
		t.Fatalf("failed to extract repacked: %v", err)
	}

	// Verify modification persisted
	content, err := os.ReadFile(filepath.Join(extractDir2, "readme.txt"))
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	if string(content) != "MODIFIED README" {
		t.Errorf("expected modified content, got %q", string(content))
	}
}

func TestRepack_InvalidOutputPath(t *testing.T) {
	tempDir := t.TempDir()

	// Create source directory
	sourceDir := filepath.Join(tempDir, "source")
	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("content"), 0644)

	// Try to write to invalid path (directory exists as file)
	invalidZipPath := filepath.Join(tempDir, "notadir", "output.zip")

	err := Repack(sourceDir, invalidZipPath)
	if err == nil {
		t.Fatal("expected error for invalid output path")
	}
}

func TestRepack_WithSymlinks(t *testing.T) {
	tempDir := t.TempDir()

	// Create source directory with symlink
	sourceDir := filepath.Join(tempDir, "source")
	os.MkdirAll(sourceDir, 0755)

	targetFile := filepath.Join(sourceDir, "target.txt")
	os.WriteFile(targetFile, []byte("target content"), 0644)

	linkFile := filepath.Join(sourceDir, "link.txt")
	os.Symlink(targetFile, linkFile)

	// Repack
	zipPath := filepath.Join(tempDir, "output.zip")
	err := Repack(sourceDir, zipPath)
	if err != nil {
		t.Fatalf("failed to repack with symlinks: %v", err)
	}

	// Verify zip was created
	if _, err := os.Stat(zipPath); err != nil {
		t.Error("expected zip file to exist")
	}
}
