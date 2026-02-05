package core

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/Fuabioo/zipfs/internal/errors"
)

// Lock represents a file-based lock using flock.
type Lock struct {
	file     *os.File
	path     string
	isShared bool
}

// AcquireShared acquires a shared lock on the given path.
// Multiple shared locks can be held simultaneously.
// Blocks until the lock is acquired or timeout is reached.
func AcquireShared(path string, timeout time.Duration) (*Lock, error) {
	return acquire(path, timeout, true)
}

// AcquireExclusive acquires an exclusive lock on the given path.
// Only one exclusive lock can be held, and it blocks all shared locks.
// Blocks until the lock is acquired or timeout is reached.
func AcquireExclusive(path string, timeout time.Duration) (*Lock, error) {
	return acquire(path, timeout, false)
}

// acquire is the internal implementation for acquiring locks.
func acquire(path string, timeout time.Duration, shared bool) (*Lock, error) {
	// Open or create the lock file
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Determine lock operation
	lockOp := syscall.LOCK_EX // exclusive by default
	if shared {
		lockOp = syscall.LOCK_SH
	}

	// Try to acquire the lock with timeout
	deadline := time.Now().Add(timeout)
	for {
		// Try non-blocking lock first
		err = syscall.Flock(int(file.Fd()), lockOp|syscall.LOCK_NB)
		if err == nil {
			// Lock acquired successfully
			return &Lock{
				file:     file,
				path:     path,
				isShared: shared,
			}, nil
		}

		// Check if we've timed out
		if time.Now().After(deadline) {
			file.Close()
			return nil, errors.Locked(path)
		}

		// Wait a bit before retrying
		time.Sleep(100 * time.Millisecond)
	}
}

// Release releases the lock and closes the file.
func (l *Lock) Release() error {
	if l.file == nil {
		return fmt.Errorf("lock already released")
	}

	// Unlock the file
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN); err != nil {
		l.file.Close()
		return fmt.Errorf("failed to unlock file: %w", err)
	}

	// Close the file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	l.file = nil
	return nil
}
