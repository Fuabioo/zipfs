package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidatePath checks if an entry path is safe to extract within the given base directory.
// Returns an error if the path attempts to escape the base directory.
//
// Security checks:
// - Rejects absolute paths
// - Rejects paths with null bytes
// - Rejects paths with ".." sequences that escape base
// - Ensures cleaned path resolves within base directory
func ValidatePath(base, entryPath string) error {
	if entryPath == "" {
		return fmt.Errorf("entry path cannot be empty")
	}

	// Reject null bytes (potential for filesystem attacks)
	if strings.Contains(entryPath, "\x00") {
		return fmt.Errorf("entry path contains null byte: %q", entryPath)
	}

	// Reject absolute paths
	if filepath.IsAbs(entryPath) {
		return fmt.Errorf("entry path must be relative, got absolute path: %q", entryPath)
	}

	// Clean the base path to ensure consistent comparison
	cleanBase := filepath.Clean(base)

	// Clean the entry path to normalize it
	cleanEntry := filepath.Clean(entryPath)

	// After cleaning, if path starts with "..", it's trying to escape
	// This catches cases like "../file" or "../../file"
	if strings.HasPrefix(cleanEntry, "..") {
		return fmt.Errorf("path traversal detected: %q attempts to escape base directory", entryPath)
	}

	// Join base with cleaned entry path
	target := filepath.Join(cleanBase, cleanEntry)

	// Compute relative path from base to target
	// If target escapes base, rel will start with ".."
	rel, err := filepath.Rel(cleanBase, target)
	if err != nil {
		return fmt.Errorf("path resolution failed for %q: %w", entryPath, err)
	}

	// Check if the relative path tries to escape (starts with "..")
	if strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path traversal detected: %q resolves outside base directory", entryPath)
	}

	// Additional verification: ensure target has base as prefix
	// This is redundant with the filepath.Rel check above but provides defense in depth
	if !strings.HasPrefix(target, cleanBase) {
		return fmt.Errorf("path %q resolves outside base directory", entryPath)
	}

	return nil
}

// ValidateAllPaths validates a slice of entry paths, returning an error on the first invalid path.
// Fail-closed: if any path is invalid, all are rejected.
//
// This implements the ADR requirement: "If any single entry fails validation,
// the entire extraction aborts (fail-closed)."
func ValidateAllPaths(base string, paths []string) error {
	for _, path := range paths {
		if err := ValidatePath(base, path); err != nil {
			return fmt.Errorf("validation failed for paths: %w", err)
		}
	}
	return nil
}
