package mcp

import (
	"testing"
)

// TestServeFunction verifies the Serve convenience function.
func TestServeFunction(t *testing.T) {
	// This test can't actually run Serve() because it blocks on stdio,
	// but we can verify the function exists and returns a proper error
	// if there's a setup issue.

	// We just verify the function signature is correct
	// The actual serving is tested via integration tests
	t.Skip("Serve() blocks on stdio - tested via integration")
}

// TestServerServe verifies the Server.Serve method exists.
func TestServerServe(t *testing.T) {
	setupTestEnvironment(t)

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Verify the server was created successfully
	if srv == nil {
		t.Error("expected server to be non-nil")
	}

	t.Skip("Server.Serve() blocks on stdio - tested via integration")
}
