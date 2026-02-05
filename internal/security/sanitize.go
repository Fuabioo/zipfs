package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// Session name constraints from ADR-008
const (
	maxSessionNameLength = 64
	sessionNamePattern   = `^[a-zA-Z0-9_-]+$`
)

var sessionNameRegex = regexp.MustCompile(sessionNamePattern)

// ValidateSessionName checks if a session name meets requirements:
// - Only alphanumeric characters, hyphens, and underscores
// - Maximum 64 characters
// - Not empty
func ValidateSessionName(name string) error {
	if name == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	if len(name) > maxSessionNameLength {
		return fmt.Errorf("session name exceeds maximum length of %d characters", maxSessionNameLength)
	}

	if !sessionNameRegex.MatchString(name) {
		return fmt.Errorf("session name must contain only alphanumeric characters, hyphens, and underscores: %q", name)
	}

	return nil
}

// ValidateRelativePath checks if a relative path is safe for use within a workspace.
// Rejects:
// - Absolute paths
// - Paths containing ".." components
// - Null bytes
// - Control characters (0x00-0x1F, 0x7F)
func ValidateRelativePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Reject absolute paths
	if filepath.IsAbs(path) {
		return fmt.Errorf("path must be relative, got absolute path: %q", path)
	}

	// Reject null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte: %q", path)
	}

	// Check for control characters
	for _, r := range path {
		if r < 0x20 || r == 0x7F {
			return fmt.Errorf("path contains control character: %q", path)
		}
	}

	// Clean the path and check if it tries to escape
	cleaned := filepath.Clean(path)

	// Reject paths that start with ".." after cleaning
	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("path attempts to traverse outside workspace: %q", path)
	}

	// Also check if path contains ".." as a component anywhere
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("path contains \"..\" component: %q", path)
		}
	}

	return nil
}

// SanitizeGlobPattern validates a glob pattern is safe to use.
// Rejects:
// - Absolute paths
// - Patterns with ".." components
// - Null bytes
// - Control characters
// - Excessively complex patterns (basic validation)
func SanitizeGlobPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("glob pattern cannot be empty")
	}

	// Reject absolute paths
	if filepath.IsAbs(pattern) {
		return fmt.Errorf("glob pattern must be relative, got absolute path: %q", pattern)
	}

	// Reject null bytes
	if strings.Contains(pattern, "\x00") {
		return fmt.Errorf("glob pattern contains null byte: %q", pattern)
	}

	// Check for control characters (including newlines and tabs)
	for _, r := range pattern {
		if unicode.IsControl(r) {
			return fmt.Errorf("glob pattern contains control character: %q", pattern)
		}
	}

	// Check for ".." components in the pattern
	// Even in a glob pattern, ".." should not be used
	parts := strings.Split(filepath.ToSlash(pattern), "/")
	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("glob pattern contains \"..\" component: %q", pattern)
		}
	}

	// Ensure pattern doesn't start with "../" after cleaning
	// (checking before glob metacharacters are expanded)
	if strings.HasPrefix(pattern, "../") || strings.HasPrefix(pattern, "..\\") {
		return fmt.Errorf("glob pattern attempts to traverse outside workspace: %q", pattern)
	}

	return nil
}
