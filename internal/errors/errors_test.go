package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "simple error",
			err:      New(CodeSessionNotFound, "session not found"),
			expected: "SESSION_NOT_FOUND: session not found",
		},
		{
			name:     "wrapped error",
			err:      Wrap(CodeSyncFailed, "sync failed", fmt.Errorf("disk full")),
			expected: "SYNC_FAILED: sync failed: disk full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	t.Run("no wrapped error", func(t *testing.T) {
		err := New(CodeSessionNotFound, "not found")
		if err.Unwrap() != nil {
			t.Errorf("Unwrap() = %v, want nil", err.Unwrap())
		}
	})

	t.Run("with wrapped error", func(t *testing.T) {
		underlying := fmt.Errorf("io error")
		err := Wrap(CodeSyncFailed, "sync failed", underlying)

		unwrapped := err.Unwrap()
		if unwrapped == nil {
			t.Fatal("Unwrap() = nil, want error")
		}
		if unwrapped.Error() != "io error" {
			t.Errorf("Unwrap() = %q, want %q", unwrapped.Error(), "io error")
		}
	})

	t.Run("stdlib errors.Is compatibility", func(t *testing.T) {
		underlying := fmt.Errorf("io error")
		err := Wrap(CodeSyncFailed, "sync failed", underlying)

		if !errors.Is(err, underlying) {
			t.Error("errors.Is() = false, want true for wrapped error")
		}
	})

	t.Run("stdlib errors.As compatibility", func(t *testing.T) {
		err := New(CodeSessionNotFound, "not found")

		var zipfsErr *Error
		if !errors.As(err, &zipfsErr) {
			t.Error("errors.As() = false, want true for zipfs error")
		}
		if zipfsErr.Code != CodeSessionNotFound {
			t.Errorf("errors.As() code = %q, want %q", zipfsErr.Code, CodeSessionNotFound)
		}
	})
}

func TestNew(t *testing.T) {
	err := New("TEST_CODE", "test message")

	if err.Code != "TEST_CODE" {
		t.Errorf("Code = %q, want %q", err.Code, "TEST_CODE")
	}
	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}
	if err.wrapped != nil {
		t.Errorf("wrapped = %v, want nil", err.wrapped)
	}
}

func TestWrap(t *testing.T) {
	underlying := fmt.Errorf("underlying error")
	err := Wrap("TEST_CODE", "test message", underlying)

	if err.Code != "TEST_CODE" {
		t.Errorf("Code = %q, want %q", err.Code, "TEST_CODE")
	}
	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}
	if err.wrapped != underlying {
		t.Errorf("wrapped = %v, want %v", err.wrapped, underlying)
	}
}

func TestCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "zipfs error",
			err:      New(CodeSessionNotFound, "not found"),
			expected: CodeSessionNotFound,
		},
		{
			name:     "wrapped zipfs error",
			err:      Wrap(CodeSyncFailed, "sync failed", fmt.Errorf("io error")),
			expected: CodeSyncFailed,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard error"),
			expected: "",
		},
		{
			name:     "wrapped standard error",
			err:      fmt.Errorf("wrapped: %w", New(CodeZipInvalid, "invalid")),
			expected: CodeZipInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Code(tt.err)
			if got != tt.expected {
				t.Errorf("Code() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     string
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			code:     CodeSessionNotFound,
			expected: false,
		},
		{
			name:     "matching code",
			err:      New(CodeSessionNotFound, "not found"),
			code:     CodeSessionNotFound,
			expected: true,
		},
		{
			name:     "non-matching code",
			err:      New(CodeSessionNotFound, "not found"),
			code:     CodeZipInvalid,
			expected: false,
		},
		{
			name:     "wrapped zipfs error",
			err:      Wrap(CodeSyncFailed, "sync failed", fmt.Errorf("io error")),
			code:     CodeSyncFailed,
			expected: true,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard error"),
			code:     CodeSessionNotFound,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Is(tt.err, tt.code)
			if got != tt.expected {
				t.Errorf("Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Test all convenience constructors
func TestSessionNotFound(t *testing.T) {
	err := SessionNotFound("my-session")

	if err.Code != CodeSessionNotFound {
		t.Errorf("Code = %q, want %q", err.Code, CodeSessionNotFound)
	}
	if !strings.Contains(err.Message, "my-session") {
		t.Errorf("Message = %q, should contain %q", err.Message, "my-session")
	}
	if !strings.Contains(err.Message, "not found") {
		t.Errorf("Message = %q, should contain %q", err.Message, "not found")
	}
}

func TestAmbiguousSession(t *testing.T) {
	err := AmbiguousSession(3)

	if err.Code != CodeAmbiguousSession {
		t.Errorf("Code = %q, want %q", err.Code, CodeAmbiguousSession)
	}
	if !strings.Contains(err.Message, "3") {
		t.Errorf("Message = %q, should contain %q", err.Message, "3")
	}
	if !strings.Contains(err.Message, "sessions are open") {
		t.Errorf("Message = %q, should contain session count info", err.Message)
	}
}

func TestNoSessions(t *testing.T) {
	err := NoSessions()

	if err.Code != CodeNoSessions {
		t.Errorf("Code = %q, want %q", err.Code, CodeNoSessions)
	}
	if !strings.Contains(err.Message, "no sessions") {
		t.Errorf("Message = %q, should mention no sessions", err.Message)
	}
}

func TestZipNotFound(t *testing.T) {
	err := ZipNotFound("/tmp/test.zip")

	if err.Code != CodeZipNotFound {
		t.Errorf("Code = %q, want %q", err.Code, CodeZipNotFound)
	}
	if !strings.Contains(err.Message, "/tmp/test.zip") {
		t.Errorf("Message = %q, should contain %q", err.Message, "/tmp/test.zip")
	}
	if !strings.Contains(err.Message, "not found") || !strings.Contains(err.Message, "not readable") {
		t.Errorf("Message = %q, should mention not found or not readable", err.Message)
	}
}

func TestZipInvalid(t *testing.T) {
	err := ZipInvalid("/tmp/test.txt")

	if err.Code != CodeZipInvalid {
		t.Errorf("Code = %q, want %q", err.Code, CodeZipInvalid)
	}
	if !strings.Contains(err.Message, "/tmp/test.txt") {
		t.Errorf("Message = %q, should contain %q", err.Message, "/tmp/test.txt")
	}
	if !strings.Contains(err.Message, "not a valid zip") {
		t.Errorf("Message = %q, should mention invalid zip", err.Message)
	}
}

func TestZipBombDetected(t *testing.T) {
	err := ZipBombDetected("compression ratio exceeds 100:1")

	if err.Code != CodeZipBombDetected {
		t.Errorf("Code = %q, want %q", err.Code, CodeZipBombDetected)
	}
	if !strings.Contains(err.Message, "compression ratio exceeds 100:1") {
		t.Errorf("Message = %q, should contain reason %q", err.Message, "compression ratio exceeds 100:1")
	}
	if !strings.Contains(err.Message, "zip bomb") {
		t.Errorf("Message = %q, should mention zip bomb", err.Message)
	}
}

func TestConflictDetected(t *testing.T) {
	err := ConflictDetected("/tmp/reports.zip")

	if err.Code != CodeConflictDetected {
		t.Errorf("Code = %q, want %q", err.Code, CodeConflictDetected)
	}
	if !strings.Contains(err.Message, "/tmp/reports.zip") {
		t.Errorf("Message = %q, should contain %q", err.Message, "/tmp/reports.zip")
	}
	if !strings.Contains(err.Message, "modified externally") {
		t.Errorf("Message = %q, should mention external modification", err.Message)
	}
}

func TestSyncFailed(t *testing.T) {
	underlying := fmt.Errorf("disk full")
	err := SyncFailed(underlying)

	if err.Code != CodeSyncFailed {
		t.Errorf("Code = %q, want %q", err.Code, CodeSyncFailed)
	}
	if !strings.Contains(err.Message, "sync") {
		t.Errorf("Message = %q, should mention sync", err.Message)
	}
	if err.Unwrap() != underlying {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), underlying)
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("Error() = %q, should include wrapped error", err.Error())
	}
}

func TestPathTraversal(t *testing.T) {
	err := PathTraversal("../../../etc/passwd")

	if err.Code != CodePathTraversal {
		t.Errorf("Code = %q, want %q", err.Code, CodePathTraversal)
	}
	if !strings.Contains(err.Message, "../../../etc/passwd") {
		t.Errorf("Message = %q, should contain %q", err.Message, "../../../etc/passwd")
	}
	if !strings.Contains(err.Message, "escape") {
		t.Errorf("Message = %q, should mention escape/traversal", err.Message)
	}
}

func TestPathNotFound(t *testing.T) {
	err := PathNotFound("data/missing.txt")

	if err.Code != CodePathNotFound {
		t.Errorf("Code = %q, want %q", err.Code, CodePathNotFound)
	}
	if !strings.Contains(err.Message, "data/missing.txt") {
		t.Errorf("Message = %q, should contain %q", err.Message, "data/missing.txt")
	}
	if !strings.Contains(err.Message, "not found") {
		t.Errorf("Message = %q, should mention not found", err.Message)
	}
}

func TestLocked(t *testing.T) {
	err := Locked("abc123")

	if err.Code != CodeLocked {
		t.Errorf("Code = %q, want %q", err.Code, CodeLocked)
	}
	if !strings.Contains(err.Message, "abc123") {
		t.Errorf("Message = %q, should contain %q", err.Message, "abc123")
	}
	if !strings.Contains(err.Message, "locked") {
		t.Errorf("Message = %q, should mention locked", err.Message)
	}
}

func TestLimitExceeded(t *testing.T) {
	err := LimitExceeded("max sessions (10)")

	if err.Code != CodeLimitExceeded {
		t.Errorf("Code = %q, want %q", err.Code, CodeLimitExceeded)
	}
	if !strings.Contains(err.Message, "max sessions (10)") {
		t.Errorf("Message = %q, should contain %q", err.Message, "max sessions (10)")
	}
	if !strings.Contains(err.Message, "limit exceeded") {
		t.Errorf("Message = %q, should mention limit exceeded", err.Message)
	}
}

func TestNameCollision(t *testing.T) {
	err := NameCollision("my-session")

	if err.Code != CodeNameCollision {
		t.Errorf("Code = %q, want %q", err.Code, CodeNameCollision)
	}
	if !strings.Contains(err.Message, "my-session") {
		t.Errorf("Message = %q, should contain %q", err.Message, "my-session")
	}
	if !strings.Contains(err.Message, "already in use") {
		t.Errorf("Message = %q, should mention already in use", err.Message)
	}
}

// Benchmark tests
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New(CodeSessionNotFound, "session not found")
	}
}

func BenchmarkWrap(b *testing.B) {
	underlying := fmt.Errorf("underlying error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Wrap(CodeSyncFailed, "sync failed", underlying)
	}
}

func BenchmarkCode(b *testing.B) {
	err := New(CodeSessionNotFound, "not found")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Code(err)
	}
}

func BenchmarkIs(b *testing.B) {
	err := New(CodeSessionNotFound, "not found")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Is(err, CodeSessionNotFound)
	}
}
