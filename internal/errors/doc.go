// Package errors provides typed error handling for zipfs operations.
//
// All error codes are defined in ADR-005 and match the MCP protocol specification.
//
// Example usage:
//
//	// Creating errors
//	err := errors.SessionNotFound("my-session")
//	err := errors.ZipBombDetected("compression ratio exceeds 100:1")
//
//	// Wrapping errors
//	err := errors.SyncFailed(ioErr)
//
//	// Checking error codes
//	if errors.Is(err, errors.CodeSessionNotFound) {
//	    // handle session not found
//	}
//
//	// Extracting codes
//	code := errors.Code(err)
//	if code == errors.CodeZipInvalid {
//	    // handle invalid zip
//	}
//
//	// Stdlib compatibility
//	var zipfsErr *errors.Error
//	if errors.As(err, &zipfsErr) {
//	    fmt.Println(zipfsErr.Code, zipfsErr.Message)
//	}
package errors
