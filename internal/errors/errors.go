package errors

import (
	"errors"
	"fmt"
)

// Error code constants matching ADR-005 error codes
const (
	CodeSessionNotFound  = "SESSION_NOT_FOUND"
	CodeAmbiguousSession = "AMBIGUOUS_SESSION"
	CodeNoSessions       = "NO_SESSIONS"
	CodeZipNotFound      = "ZIP_NOT_FOUND"
	CodeZipInvalid       = "ZIP_INVALID"
	CodeZipBombDetected  = "ZIP_BOMB_DETECTED"
	CodeConflictDetected = "CONFLICT_DETECTED"
	CodeSyncFailed       = "SYNC_FAILED"
	CodePathTraversal    = "PATH_TRAVERSAL"
	CodePathNotFound     = "PATH_NOT_FOUND"
	CodeLocked           = "LOCKED"
	CodeLimitExceeded    = "LIMIT_EXCEEDED"
	CodeNameCollision    = "NAME_COLLISION"
)

// Error represents a zipfs error with a code and message.
// It implements the error interface and supports error wrapping.
type Error struct {
	wrapped error
	Code    string
	Message string
}

// Error returns the error message, implementing the error interface.
func (e *Error) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.wrapped)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error, supporting errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.wrapped
}

// New creates a new zipfs error with the given code and message.
func New(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap creates a new zipfs error that wraps an underlying error.
func Wrap(code string, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		wrapped: err,
	}
}

// Code extracts the error code from an error.
// Returns an empty string if the error is not a zipfs error.
func Code(err error) string {
	if err == nil {
		return ""
	}
	var zipfsErr *Error
	if errors.As(err, &zipfsErr) {
		return zipfsErr.Code
	}
	return ""
}

// Is checks if an error has a specific error code.
func Is(err error, code string) bool {
	return Code(err) == code
}

// Convenience constructors for each error code

// SessionNotFound creates a SESSION_NOT_FOUND error.
func SessionNotFound(name string) *Error {
	return New(CodeSessionNotFound, fmt.Sprintf("session %q not found", name))
}

// AmbiguousSession creates an AMBIGUOUS_SESSION error.
func AmbiguousSession(count int) *Error {
	return New(CodeAmbiguousSession, fmt.Sprintf("%d sessions are open, please specify which one", count))
}

// NoSessions creates a NO_SESSIONS error.
func NoSessions() *Error {
	return New(CodeNoSessions, "no sessions are open")
}

// ZipNotFound creates a ZIP_NOT_FOUND error.
func ZipNotFound(path string) *Error {
	return New(CodeZipNotFound, fmt.Sprintf("zip file %q not found or not readable", path))
}

// ZipInvalid creates a ZIP_INVALID error.
func ZipInvalid(path string) *Error {
	return New(CodeZipInvalid, fmt.Sprintf("file %q is not a valid zip archive", path))
}

// ZipBombDetected creates a ZIP_BOMB_DETECTED error.
func ZipBombDetected(reason string) *Error {
	return New(CodeZipBombDetected, fmt.Sprintf("zip bomb detected: %s", reason))
}

// ConflictDetected creates a CONFLICT_DETECTED error.
func ConflictDetected(path string) *Error {
	return New(CodeConflictDetected, fmt.Sprintf("source zip %q has been modified externally since it was opened", path))
}

// SyncFailed creates a SYNC_FAILED error wrapping the underlying cause.
func SyncFailed(err error) *Error {
	return Wrap(CodeSyncFailed, "failed to sync workspace to zip", err)
}

// PathTraversal creates a PATH_TRAVERSAL error.
func PathTraversal(path string) *Error {
	return New(CodePathTraversal, fmt.Sprintf("path %q attempts to escape workspace", path))
}

// PathNotFound creates a PATH_NOT_FOUND error.
func PathNotFound(path string) *Error {
	return New(CodePathNotFound, fmt.Sprintf("path %q not found in workspace", path))
}

// Locked creates a LOCKED error.
func Locked(sessionID string) *Error {
	return New(CodeLocked, fmt.Sprintf("session %q is locked by another operation", sessionID))
}

// LimitExceeded creates a LIMIT_EXCEEDED error.
func LimitExceeded(limit string) *Error {
	return New(CodeLimitExceeded, fmt.Sprintf("limit exceeded: %s", limit))
}

// NameCollision creates a NAME_COLLISION error.
func NameCollision(name string) *Error {
	return New(CodeNameCollision, fmt.Sprintf("session name %q is already in use", name))
}
