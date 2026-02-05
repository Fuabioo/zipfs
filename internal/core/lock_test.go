package core

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Fuabioo/zipfs/internal/errors"
)

func TestLock_AcquireShared(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock, err := AcquireShared(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire shared lock: %v", err)
	}
	defer lock.Release()

	if lock.file == nil {
		t.Error("expected lock file to be set")
	}

	if !lock.isShared {
		t.Error("expected lock to be shared")
	}
}

func TestLock_AcquireExclusive(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock, err := AcquireExclusive(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire exclusive lock: %v", err)
	}
	defer lock.Release()

	if lock.file == nil {
		t.Error("expected lock file to be set")
	}

	if lock.isShared {
		t.Error("expected lock to be exclusive")
	}
}

func TestLock_MultipleShared(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	// Acquire first shared lock
	lock1, err := AcquireShared(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire first shared lock: %v", err)
	}
	defer lock1.Release()

	// Acquire second shared lock (should succeed)
	lock2, err := AcquireShared(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire second shared lock: %v", err)
	}
	defer lock2.Release()
}

func TestLock_ExclusiveBlocksShared(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	// Acquire exclusive lock
	lock1, err := AcquireExclusive(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire exclusive lock: %v", err)
	}
	defer lock1.Release()

	// Try to acquire shared lock (should timeout)
	_, err = AcquireShared(lockPath, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when acquiring shared lock while exclusive is held")
	}

	if !errors.Is(err, errors.CodeLocked) {
		t.Errorf("expected LOCKED error, got: %v", err)
	}
}

func TestLock_ExclusiveBlocksExclusive(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	// Acquire first exclusive lock
	lock1, err := AcquireExclusive(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire first exclusive lock: %v", err)
	}
	defer lock1.Release()

	// Try to acquire second exclusive lock (should timeout)
	_, err = AcquireExclusive(lockPath, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when acquiring exclusive lock while another exclusive is held")
	}

	if !errors.Is(err, errors.CodeLocked) {
		t.Errorf("expected LOCKED error, got: %v", err)
	}
}

func TestLock_Release(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock, err := AcquireExclusive(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	if err := lock.Release(); err != nil {
		t.Errorf("failed to release lock: %v", err)
	}

	if lock.file != nil {
		t.Error("expected lock file to be nil after release")
	}

	// Should be able to acquire again after release
	lock2, err := AcquireExclusive(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire lock after release: %v", err)
	}
	defer lock2.Release()
}

func TestLock_ReleaseAlreadyReleased(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock, err := AcquireExclusive(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	lock.Release()

	// Try to release again
	err = lock.Release()
	if err == nil {
		t.Error("expected error when releasing already released lock")
	}
}

func TestLock_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	var wg sync.WaitGroup
	var mu sync.Mutex
	counter := 0

	// Launch multiple goroutines that increment a counter
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lock, err := AcquireExclusive(lockPath, 5*time.Second)
			if err != nil {
				t.Errorf("failed to acquire lock: %v", err)
				return
			}
			defer lock.Release()

			// Critical section (using both lock and mutex to avoid race detector)
			mu.Lock()
			temp := counter
			time.Sleep(10 * time.Millisecond)
			counter = temp + 1
			mu.Unlock()
		}()
	}

	wg.Wait()

	mu.Lock()
	finalCounter := counter
	mu.Unlock()

	if finalCounter != 10 {
		t.Errorf("expected counter to be 10, got %d (lock didn't prevent race)", finalCounter)
	}
}

func TestLock_LockFileCreated(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock, err := AcquireExclusive(lockPath, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer lock.Release()

	// Check that lock file exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("expected lock file to exist")
	}
}

func TestLock_InvalidPath(t *testing.T) {
	// Try to acquire lock in non-existent directory
	lockPath := "/nonexistent/directory/test.lock"

	_, err := AcquireExclusive(lockPath, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
