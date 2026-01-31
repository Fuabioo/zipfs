package core

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Fuabioo/zipfs/internal/errors"
	"github.com/Fuabioo/zipfs/internal/security"
)

// Extract extracts a zip file to the destination directory.
// Returns the number of files extracted and the total size in bytes.
// Uses fail-closed security validation - any single invalid path aborts the entire extraction.
func Extract(zipPath, destDir string, limits security.Limits) (int, uint64, error) {
	// Pre-scan for zip bomb
	bombCheck, err := security.CheckZipBomb(zipPath, limits)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to check for zip bomb: %w", err)
	}
	if !bombCheck.IsSafe {
		return 0, 0, errors.ZipBombDetected(bombCheck.Reason)
	}

	// Open the zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	// Validate all paths first (fail-closed)
	var paths []string
	for _, f := range r.File {
		paths = append(paths, f.Name)
	}
	if err := security.ValidateAllPaths(destDir, paths); err != nil {
		return 0, 0, fmt.Errorf("path validation failed: %w", err)
	}

	// Extract all files
	var fileCount int
	var totalSize uint64

	for _, f := range r.File {
		if err := extractFile(f, destDir, &fileCount, &totalSize); err != nil {
			return fileCount, totalSize, fmt.Errorf("failed to extract %q: %w", f.Name, err)
		}
	}

	return fileCount, totalSize, nil
}

// extractFile extracts a single file from the zip archive.
func extractFile(f *zip.File, destDir string, fileCount *int, totalSize *uint64) error {
	// Construct the destination path
	destPath := filepath.Join(destDir, f.Name)

	// Handle directories
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(destPath, f.Mode()); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		return nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Open the file in the archive
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in archive: %w", err)
	}
	defer rc.Close()

	// Create the destination file
	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy the data and track size
	written, err := io.Copy(outFile, rc)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	*fileCount++
	*totalSize += uint64(written)

	return nil
}
