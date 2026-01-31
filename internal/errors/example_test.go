package errors_test

import (
	"fmt"
	"io/fs"

	"github.com/Fuabioo/zipfs/internal/errors"
)

// Example_basic demonstrates basic error creation and checking.
func Example_basic() {
	// Create a simple error
	err := errors.SessionNotFound("my-session")
	fmt.Println(err)

	// Check the error code
	if errors.Is(err, errors.CodeSessionNotFound) {
		fmt.Println("Session not found")
	}

	// Output:
	// SESSION_NOT_FOUND: session "my-session" not found
	// Session not found
}

// Example_wrapping demonstrates error wrapping.
func Example_wrapping() {
	// Simulate an I/O error
	ioErr := fs.ErrNotExist

	// Wrap it with a zipfs error
	err := errors.SyncFailed(ioErr)
	fmt.Println(err)

	// Extract the code
	code := errors.Code(err)
	fmt.Println("Error code:", code)

	// Output:
	// SYNC_FAILED: failed to sync workspace to zip: file does not exist
	// Error code: SYNC_FAILED
}

// Example_checking demonstrates different ways to check errors.
func Example_checking() {
	err := errors.ZipInvalid("/tmp/test.txt")

	// Method 1: Use the Is helper
	if errors.Is(err, errors.CodeZipInvalid) {
		fmt.Println("Invalid zip file")
	}

	// Method 2: Extract and compare code
	if errors.Code(err) == errors.CodeZipInvalid {
		fmt.Println("Still invalid")
	}

	// Method 3: Use errors.As for full access
	var zipfsErr *errors.Error
	if e := err; e != nil {
		zipfsErr = e
		fmt.Printf("Code: %s, Message: %s\n", zipfsErr.Code, zipfsErr.Message)
	}

	// Output:
	// Invalid zip file
	// Still invalid
	// Code: ZIP_INVALID, Message: file "/tmp/test.txt" is not a valid zip archive
}
