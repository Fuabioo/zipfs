package core

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Fuabioo/zipfs/internal/errors"
	"github.com/Fuabioo/zipfs/internal/security"
)

// FileEntry represents a file or directory entry.
type FileEntry struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // "file" or "dir"
	SizeBytes  uint64 `json:"size_bytes"`
	ModifiedAt int64  `json:"modified_at"` // Unix timestamp
}

// GrepMatch represents a grep search result.
type GrepMatch struct {
	File        string `json:"file"`
	LineContent string `json:"line_content"`
	LineNumber  int    `json:"line_number"`
}

// StatusResult represents the result of a status check.
type StatusResult struct {
	Modified       []string `json:"modified"`
	Added          []string `json:"added"`
	Deleted        []string `json:"deleted"`
	UnchangedCount int      `json:"unchanged_count"`
}

// ListFiles lists files and directories in the workspace.
func ListFiles(contentsDir, relativePath string, recursive bool) ([]FileEntry, error) {
	// Validate relative path
	if relativePath != "" && relativePath != "." {
		if err := security.ValidateRelativePath(relativePath); err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}
	}

	// Construct absolute path
	targetPath := filepath.Join(contentsDir, relativePath)

	// Validate the resolved path is within contents directory
	if err := security.ValidatePath(contentsDir, relativePath); err != nil {
		return nil, errors.PathTraversal(relativePath)
	}

	// Check if path exists
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.PathNotFound(relativePath)
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var entries []FileEntry

	if !recursive {
		// List only immediate children
		if !info.IsDir() {
			// If it's a file, return just that file
			return []FileEntry{
				{
					Name:       filepath.Base(targetPath),
					Type:       "file",
					SizeBytes:  uint64(info.Size()),
					ModifiedAt: info.ModTime().Unix(),
				},
			}, nil
		}

		// List directory contents
		dirEntries, err := os.ReadDir(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range dirEntries {
			entryInfo, err := entry.Info()
			if err != nil {
				continue
			}

			entryType := "file"
			if entry.IsDir() {
				entryType = "dir"
			}

			entries = append(entries, FileEntry{
				Name:       entry.Name(),
				Type:       entryType,
				SizeBytes:  uint64(entryInfo.Size()),
				ModifiedAt: entryInfo.ModTime().Unix(),
			})
		}
	} else {
		// Recursive listing
		err := filepath.Walk(targetPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip the root directory itself
			if path == targetPath {
				return nil
			}

			// Get relative path from target
			relPath, err := filepath.Rel(targetPath, path)
			if err != nil {
				return err
			}

			entryType := "file"
			if info.IsDir() {
				entryType = "dir"
			}

			entries = append(entries, FileEntry{
				Name:       relPath,
				Type:       entryType,
				SizeBytes:  uint64(info.Size()),
				ModifiedAt: info.ModTime().Unix(),
			})

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	return entries, nil
}

// TreeView generates a tree view of the directory structure.
func TreeView(contentsDir, relativePath string, maxDepth int) (string, int, int, error) {
	// Validate relative path
	if relativePath != "" && relativePath != "." {
		if err := security.ValidateRelativePath(relativePath); err != nil {
			return "", 0, 0, fmt.Errorf("invalid path: %w", err)
		}
	}

	// Construct absolute path
	targetPath := filepath.Join(contentsDir, relativePath)

	// Validate the resolved path is within contents directory
	if err := security.ValidatePath(contentsDir, relativePath); err != nil {
		return "", 0, 0, errors.PathTraversal(relativePath)
	}

	// Check if path exists
	if _, err := os.Stat(targetPath); err != nil {
		if os.IsNotExist(err) {
			return "", 0, 0, errors.PathNotFound(relativePath)
		}
		return "", 0, 0, fmt.Errorf("failed to stat path: %w", err)
	}

	var sb strings.Builder
	var fileCount, dirCount int

	err := buildTree(&sb, targetPath, "", 0, maxDepth, &fileCount, &dirCount)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to build tree: %w", err)
	}

	return sb.String(), fileCount, dirCount, nil
}

// buildTree recursively builds the tree structure.
func buildTree(sb *strings.Builder, path, prefix string, depth, maxDepth int, fileCount, dirCount *int) error {
	if maxDepth > 0 && depth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		isLast := i == len(entries)-1

		// Determine the tree characters
		var connector, childPrefix string
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		} else {
			connector = "├── "
			childPrefix = prefix + "│   "
		}

		// Write the entry
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
			*dirCount++
		} else {
			*fileCount++
		}

		sb.WriteString(prefix)
		sb.WriteString(connector)
		sb.WriteString(name)
		sb.WriteString("\n")

		// Recurse into directories
		if entry.IsDir() {
			childPath := filepath.Join(path, entry.Name())
			if err := buildTree(sb, childPath, childPrefix, depth+1, maxDepth, fileCount, dirCount); err != nil {
				return err
			}
		}
	}

	return nil
}

// ReadFile reads a file from the workspace.
func ReadFile(contentsDir, relativePath string) ([]byte, error) {
	// Validate relative path
	if err := security.ValidateRelativePath(relativePath); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Validate the resolved path is within contents directory
	if err := security.ValidatePath(contentsDir, relativePath); err != nil {
		return nil, errors.PathTraversal(relativePath)
	}

	// Construct absolute path
	targetPath := filepath.Join(contentsDir, relativePath)

	// Read the file
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.PathNotFound(relativePath)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// WriteFile writes data to a file in the workspace.
func WriteFile(contentsDir, relativePath string, content []byte, createDirs bool) error {
	// Validate relative path
	if err := security.ValidateRelativePath(relativePath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Validate the resolved path is within contents directory
	if err := security.ValidatePath(contentsDir, relativePath); err != nil {
		return errors.PathTraversal(relativePath)
	}

	// Construct absolute path
	targetPath := filepath.Join(contentsDir, relativePath)

	// Create parent directories if requested
	if createDirs {
		parentDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Write the file
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DeleteFile deletes a file or directory from the workspace.
func DeleteFile(contentsDir, relativePath string, recursive bool) error {
	// Validate relative path
	if err := security.ValidateRelativePath(relativePath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Validate the resolved path is within contents directory
	if err := security.ValidatePath(contentsDir, relativePath); err != nil {
		return errors.PathTraversal(relativePath)
	}

	// Construct absolute path
	targetPath := filepath.Join(contentsDir, relativePath)

	// Check if path exists
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.PathNotFound(relativePath)
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	// If it's a directory and recursive is not set, error
	if info.IsDir() && !recursive {
		return fmt.Errorf("path is a directory, use recursive=true to delete")
	}

	// Delete the file or directory
	if recursive {
		if err := os.RemoveAll(targetPath); err != nil {
			return fmt.Errorf("failed to remove path: %w", err)
		}
	} else {
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}
	}

	return nil
}

// GrepFiles searches for a pattern in files within the workspace.
func GrepFiles(contentsDir, relativePath, pattern, glob string, ignoreCase bool, maxResults int) ([]GrepMatch, int, error) {
	// Validate relative path
	if relativePath != "" && relativePath != "." {
		if err := security.ValidateRelativePath(relativePath); err != nil {
			return nil, 0, fmt.Errorf("invalid path: %w", err)
		}
	}

	// Validate glob pattern
	if glob != "" {
		if err := security.SanitizeGlobPattern(glob); err != nil {
			return nil, 0, fmt.Errorf("invalid glob pattern: %w", err)
		}
	}

	// Construct absolute path
	targetPath := filepath.Join(contentsDir, relativePath)

	// Validate the resolved path is within contents directory
	if err := security.ValidatePath(contentsDir, relativePath); err != nil {
		return nil, 0, errors.PathTraversal(relativePath)
	}

	// Compile regex pattern
	var re *regexp.Regexp
	var err error
	if ignoreCase {
		re, err = regexp.Compile("(?i)" + pattern)
	} else {
		re, err = regexp.Compile(pattern)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var matches []GrepMatch
	var totalMatches int

	// Walk the directory tree
	err = filepath.Walk(targetPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Apply glob filter if specified
		if glob != "" {
			matched, err := filepath.Match(glob, filepath.Base(path))
			if err != nil {
				return fmt.Errorf("glob match error: %w", err)
			}
			if !matched {
				return nil
			}
		}

		// Get relative path from contents directory
		relPath, err := filepath.Rel(contentsDir, path)
		if err != nil {
			return err
		}

		// Search the file
		fileMatches, err := grepFile(path, relPath, re, maxResults-len(matches))
		if err != nil {
			// Skip files that can't be read
			return nil
		}

		totalMatches += len(fileMatches)
		matches = append(matches, fileMatches...)

		// Stop if we've reached max results
		if maxResults > 0 && len(matches) >= maxResults {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, 0, fmt.Errorf("failed to search files: %w", err)
	}

	// Trim matches to max results
	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches, totalMatches, nil
}

// grepFile searches for a pattern in a single file.
func grepFile(path, relPath string, re *regexp.Regexp, maxMatches int) ([]GrepMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []GrepMatch
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if re.MatchString(line) {
			matches = append(matches, GrepMatch{
				File:        relPath,
				LineNumber:  lineNum,
				LineContent: line,
			})

			if maxMatches > 0 && len(matches) >= maxMatches {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}

// Status compares the current workspace contents with the original zip.
func Status(session *Session) (*StatusResult, error) {
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}

	contentsDir, err := ContentsDir(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents directory: %w", err)
	}

	originalZipPath, err := OriginalZipPath(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to get original zip path: %w", err)
	}

	// Read original zip
	zipReader, err := zip.OpenReader(originalZipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open original zip: %w", err)
	}
	defer zipReader.Close()

	// Build map of original files
	originalFiles := make(map[string]*zip.File)
	for _, f := range zipReader.File {
		if !f.FileInfo().IsDir() {
			originalFiles[f.Name] = f
		}
	}

	// Build map of current files
	currentFiles := make(map[string]bool)
	err = filepath.Walk(contentsDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(contentsDir, path)
		if err != nil {
			return err
		}

		// Normalize to forward slashes
		relPath = filepath.ToSlash(relPath)
		currentFiles[relPath] = true

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk contents directory: %w", err)
	}

	result := &StatusResult{
		Modified: []string{},
		Added:    []string{},
		Deleted:  []string{},
	}

	// Find modified and added files
	for currentPath := range currentFiles {
		if originalFile, exists := originalFiles[currentPath]; exists {
			// File exists in both - check if modified
			currentFullPath := filepath.Join(contentsDir, filepath.FromSlash(currentPath))
			currentInfo, err := os.Stat(currentFullPath)
			if err != nil {
				continue
			}

			// Compare size and modification time
			if uint64(currentInfo.Size()) != originalFile.UncompressedSize64 ||
				!currentInfo.ModTime().Equal(originalFile.Modified) {
				result.Modified = append(result.Modified, currentPath)
			} else {
				result.UnchangedCount++
			}
		} else {
			// File exists only in current - it was added
			result.Added = append(result.Added, currentPath)
		}
	}

	// Find deleted files
	for originalPath := range originalFiles {
		if !currentFiles[originalPath] {
			result.Deleted = append(result.Deleted, originalPath)
		}
	}

	return result, nil
}
