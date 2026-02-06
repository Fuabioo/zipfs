package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Fuabioo/zipfs/internal/errors"
)

// SyncResult contains the results of a sync operation.
type SyncResult struct {
	StatusError     error
	BackupPath      string
	FilesModified   int
	FilesAdded      int
	FilesDeleted    int
	NewZipSizeBytes uint64
}

// Sync synchronizes the workspace contents back to the source zip file.
// This implements the sync workflow from ADR-004.
func Sync(session *Session, force bool, cfg *Config) (*SyncResult, error) {
	dirName := session.DirName()

	// 1. Acquire exclusive lock
	lockPath, err := LockPath(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to get lock path: %w", err)
	}

	lock, err := AcquireExclusive(lockPath, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() { _ = lock.Release() }()

	// 2. Verify session state is "open"
	if session.State != "open" {
		return nil, fmt.Errorf("session state is %q, expected \"open\"", session.State)
	}

	// 3. Set state to "syncing"
	session.State = "syncing"
	if err := UpdateSession(session, dirName); err != nil {
		return nil, fmt.Errorf("failed to update session state: %w", err)
	}

	// Defer restoring state to "open" on error
	restoreState := true
	defer func() {
		if restoreState {
			session.State = "open"
			_ = UpdateSession(session, dirName)
		}
	}()

	// 4. Verify source path exists and parent is writable
	if _, err := os.Stat(session.SourcePath); err != nil {
		return nil, fmt.Errorf("source zip no longer exists: %w", err)
	}

	sourceDir := filepath.Dir(session.SourcePath)
	if err := checkWritable(sourceDir); err != nil {
		return nil, fmt.Errorf("source directory not writable: %w", err)
	}

	// 5. Compute SHA-256 of current source zip
	currentHash, err := ComputeZipHash(session.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute current hash: %w", err)
	}

	// 6. Compare hashes
	if currentHash != session.ZipHashSHA256 && !force {
		return nil, errors.ConflictDetected(session.SourcePath)
	}

	// 7. Build new zip from contents into temp file
	contentsDir, err := ContentsDir(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents directory: %w", err)
	}

	// Create temp file in the same directory as source (for atomic rename)
	tempFile, err := os.CreateTemp(sourceDir, fmt.Sprintf(".%s.zipfs-tmp-*", filepath.Base(session.SourcePath)))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	// Ensure temp file is cleaned up on error
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			os.Remove(tempPath)
		}
	}()

	// Capture status before repack to compute file changes
	statusResult, statusErr := Status(session)

	// Repack the contents
	if err := Repack(contentsDir, tempPath); err != nil {
		return nil, errors.SyncFailed(err)
	}

	// Get temp file size
	tempInfo, err := os.Stat(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	// 8-9. Rotate existing backups
	backupPath, err := RotateBackups(session.SourcePath, cfg.Defaults.BackupRotationDepth)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate backups: %w", err)
	}

	// 10. Rename temp file to source.zip
	if err := os.Rename(tempPath, session.SourcePath); err != nil {
		return nil, fmt.Errorf("failed to rename temp file to source: %w", err)
	}
	cleanupTemp = false // Successfully renamed, don't clean up

	// 11. Update metadata
	now := time.Now()
	session.LastSyncedAt = &now

	newHash, err := ComputeZipHash(session.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute new hash: %w", err)
	}
	session.ZipHashSHA256 = newHash

	// 12. Set state back to "open"
	session.State = "open"
	restoreState = false // Don't restore in defer

	if err := UpdateSession(session, dirName); err != nil {
		return nil, fmt.Errorf("failed to update session metadata: %w", err)
	}

	result := &SyncResult{
		BackupPath:      backupPath,
		NewZipSizeBytes: uint64(tempInfo.Size()),
	}

	// Populate change counts if status was computed successfully
	if statusErr != nil {
		result.StatusError = fmt.Errorf("change tracking unavailable: %w", statusErr)
	} else {
		result.FilesModified = len(statusResult.Modified)
		result.FilesAdded = len(statusResult.Added)
		result.FilesDeleted = len(statusResult.Deleted)
	}

	return result, nil
}

// RotateBackups rotates backup files for a source zip.
// Returns the path to the new backup file.
func RotateBackups(sourcePath string, maxDepth int) (string, error) {
	ext := filepath.Ext(sourcePath)
	base := sourcePath[:len(sourcePath)-len(ext)]

	// Rotate existing backups
	for i := maxDepth; i >= 2; i-- {
		oldPath := fmt.Sprintf("%s.bak.%d%s", base, i-1, ext)
		newPath := fmt.Sprintf("%s.bak.%d%s", base, i, ext)

		// Remove the destination if it exists
		os.Remove(newPath)

		// Rename if old path exists
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				return "", fmt.Errorf("failed to rotate backup %d: %w", i, err)
			}
		}
	}

	// Rename source.bak to source.bak.2 if it exists
	bakPath := fmt.Sprintf("%s.bak%s", base, ext)
	bak2Path := fmt.Sprintf("%s.bak.2%s", base, ext)

	if _, err := os.Stat(bakPath); err == nil {
		os.Remove(bak2Path)
		if err := os.Rename(bakPath, bak2Path); err != nil {
			return "", fmt.Errorf("failed to rotate .bak to .bak.2: %w", err)
		}
	}

	// Rename source to source.bak
	if err := os.Rename(sourcePath, bakPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return bakPath, nil
}

// checkWritable checks if a directory is writable.
func checkWritable(dir string) error {
	tempFile, err := os.CreateTemp(dir, ".zipfs-write-test-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	os.Remove(tempPath)
	return nil
}
