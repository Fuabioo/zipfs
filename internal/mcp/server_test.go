package mcp

import (
	"testing"
)

func TestNewServer(t *testing.T) {
	// Create temp environment for testing
	setupTestEnvironment(t)

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if srv == nil {
		t.Fatal("expected non-nil server")
	}

	if srv.mcp == nil {
		t.Error("expected MCP server to be initialized")
	}

	if srv.cfg == nil {
		t.Error("expected config to be initialized")
	}
}

func TestNewServer_WithConfig(t *testing.T) {
	// Create temp environment for testing
	setupTestEnvironment(t)

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Verify config has expected defaults
	if srv.cfg.Security.MaxSessions == 0 {
		t.Error("expected max sessions to be set")
	}

	if srv.cfg.Security.MaxExtractedSizeBytes == 0 {
		t.Error("expected max extracted size to be set")
	}
}
