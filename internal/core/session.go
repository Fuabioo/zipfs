package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Fuabioo/zipfs/internal/errors"
	"github.com/Fuabioo/zipfs/internal/security"
	"github.com/google/uuid"
)

// Session represents a zipfs session with metadata.
type Session struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	SourcePath         string     `json:"source_path"`
	CreatedAt          time.Time  `json:"created_at"`
	LastSyncedAt       *time.Time `json:"last_synced_at"`
	LastAccessedAt     time.Time  `json:"last_accessed_at"`
	State              string     `json:"state"` // "open", "syncing"
	ZipHashSHA256      string     `json:"zip_hash_sha256"`
	ExtractedSizeBytes uint64     `json:"extracted_size_bytes"`
	FileCount          int        `json:"file_count"`
}

// CreateSession creates a new session for the given zip file.
// This implements the "open" workflow from ADR-003.
func CreateSession(sourcePath, name string, cfg *Config) (*Session, error) {
	// Validate source path exists and is a zip file
	if _, err := os.Stat(sourcePath); err != nil {
		if os.IsNotExist(err) {
			return nil, errors.ZipNotFound(sourcePath)
		}
		return nil, fmt.Errorf("failed to stat source zip: %w", err)
	}

	// Make source path absolute
	absSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Validate session name if provided
	if name != "" {
		if err := security.ValidateSessionName(name); err != nil {
			return nil, fmt.Errorf("invalid session name: %w", err)
		}

		// Check if name looks like a UUID (to avoid ambiguity)
		if _, err := uuid.Parse(name); err == nil {
			return nil, fmt.Errorf("session name cannot be a valid UUID")
		}

		// Check for name collision
		existing, err := GetSession(name)
		if err == nil && existing != nil {
			return nil, errors.NameCollision(name)
		}
	}

	// Check global limits
	sessions, err := ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	if len(sessions) >= cfg.Security.MaxSessions {
		return nil, errors.LimitExceeded(fmt.Sprintf("max sessions (%d)", cfg.Security.MaxSessions))
	}

	// Pre-scan zip for security checks
	bombCheck, err := security.CheckZipBomb(absSourcePath, cfg.ToSecurityLimits())
	if err != nil {
		return nil, errors.ZipInvalid(absSourcePath)
	}
	if !bombCheck.IsSafe {
		return nil, errors.ZipBombDetected(bombCheck.Reason)
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Use name as directory name if provided, otherwise use UUID
	dirName := sessionID
	if name != "" {
		dirName = name
	}

	// Create workspace directory structure
	session := &Session{
		ID:             sessionID,
		Name:           name,
		SourcePath:     absSourcePath,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		State:          "open",
	}

	if err := CreateWorkspace(session, dirName); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Compute hash of source zip
	hash, err := ComputeZipHash(absSourcePath)
	if err != nil {
		_ = RemoveWorkspace(session, dirName)
		return nil, fmt.Errorf("failed to compute zip hash: %w", err)
	}
	session.ZipHashSHA256 = hash

	// Copy source zip to workspace
	originalZipPath, err := OriginalZipPath(dirName)
	if err != nil {
		_ = RemoveWorkspace(session, dirName)
		return nil, fmt.Errorf("failed to get original zip path: %w", err)
	}

	if err := copyFile(absSourcePath, originalZipPath); err != nil {
		_ = RemoveWorkspace(session, dirName)
		return nil, fmt.Errorf("failed to copy source zip: %w", err)
	}

	// Extract contents
	contentsDir, err := ContentsDir(dirName)
	if err != nil {
		_ = RemoveWorkspace(session, dirName)
		return nil, fmt.Errorf("failed to get contents directory: %w", err)
	}

	fileCount, totalSize, err := Extract(absSourcePath, contentsDir, cfg.ToSecurityLimits())
	if err != nil {
		_ = RemoveWorkspace(session, dirName)
		return nil, fmt.Errorf("failed to extract zip: %w", err)
	}

	session.FileCount = fileCount
	session.ExtractedSizeBytes = totalSize

	// Write metadata
	if err := UpdateSession(session, dirName); err != nil {
		_ = RemoveWorkspace(session, dirName)
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by name, UUID, or UUID prefix.
func GetSession(identifier string) (*Session, error) {
	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	workspacesDir, err := WorkspacesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get workspaces directory: %w", err)
	}

	// Ensure workspaces directory exists
	if _, err := os.Stat(workspacesDir); os.IsNotExist(err) {
		return nil, errors.SessionNotFound(identifier)
	}

	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspaces directory: %w", err)
	}

	var matches []*Session

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()

		// Try exact match first (name or UUID)
		if dirName == identifier {
			session, err := loadSession(dirName)
			if err != nil {
				continue
			}
			return session, nil
		}

		// Try UUID match
		session, err := loadSession(dirName)
		if err != nil {
			continue
		}

		if session.ID == identifier {
			return session, nil
		}

		// Try UUID prefix match (minimum 4 characters)
		if len(identifier) >= 4 && strings.HasPrefix(session.ID, identifier) {
			matches = append(matches, session)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	if len(matches) > 1 {
		return nil, errors.AmbiguousSession(len(matches))
	}

	return nil, errors.SessionNotFound(identifier)
}

// ListSessions returns all active sessions.
func ListSessions() ([]*Session, error) {
	workspacesDir, err := WorkspacesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get workspaces directory: %w", err)
	}

	// If workspaces directory doesn't exist, return empty list
	if _, err := os.Stat(workspacesDir); os.IsNotExist(err) {
		return []*Session{}, nil
	}

	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspaces directory: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		session, err := loadSession(entry.Name())
		if err != nil {
			// Skip invalid sessions
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// ResolveSession implements auto-resolution logic from ADR-003.
// Returns the session if exactly one exists, otherwise returns an error.
func ResolveSession(identifier string) (*Session, error) {
	// If identifier is provided, use it directly
	if identifier != "" {
		return GetSession(identifier)
	}

	// Auto-resolve: check how many sessions exist
	sessions, err := ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	switch len(sessions) {
	case 0:
		return nil, errors.NoSessions()
	case 1:
		return sessions[0], nil
	default:
		return nil, errors.AmbiguousSession(len(sessions))
	}
}

// DeleteSession removes a session workspace.
func DeleteSession(id string) error {
	session, err := GetSession(id)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}

	return RemoveWorkspace(session, dirName)
}

// UpdateSession writes the session metadata to disk.
func UpdateSession(session *Session, dirName string) error {
	if dirName == "" {
		dirName = session.ID
		if session.Name != "" {
			dirName = session.Name
		}
	}

	metadataPath, err := MetadataPath(dirName)
	if err != nil {
		return fmt.Errorf("failed to get metadata path: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// TouchSession updates the last_accessed_at timestamp.
func TouchSession(session *Session) error {
	session.LastAccessedAt = time.Now()

	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}

	return UpdateSession(session, dirName)
}

// loadSession loads a session from its workspace directory.
func loadSession(dirName string) (*Session, error) {
	metadataPath, err := MetadataPath(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata path: %w", err)
	}

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &session, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}

// ComputeZipHash computes the SHA-256 hash of a zip file.
func ComputeZipHash(zipPath string) (string, error) {
	file, err := os.Open(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
