package core

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Repack creates a zip file from the contents of a directory.
// Does NOT follow symlinks for security.
func Repack(contentsDir, destZipPath string) error {
	// Create the destination zip file
	zipFile, err := os.Create(destZipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk the contents directory and add all files
	err = filepath.Walk(contentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error: %w", err)
		}

		// Get relative path from contents directory
		relPath, err := filepath.Rel(contentsDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Skip symlinks (security requirement)
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Create header from file info
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header: %w", err)
		}

		// Use forward slashes for zip paths (cross-platform compatibility)
		header.Name = filepath.ToSlash(relPath)

		// Handle directories
		if info.IsDir() {
			header.Name += "/"
			header.Method = zip.Store
		} else {
			header.Method = zip.Deflate
		}

		// Write header
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create zip entry: %w", err)
		}

		// If it's a directory, we're done
		if info.IsDir() {
			return nil
		}

		// Open and copy the file contents
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		if _, err := io.Copy(writer, file); err != nil {
			return fmt.Errorf("failed to write file to zip: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk contents directory: %w", err)
	}

	return nil
}
