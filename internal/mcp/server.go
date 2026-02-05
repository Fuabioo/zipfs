package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "zipfs"
	serverVersion = "0.1.0"
)

// Server wraps the MCP server with zipfs-specific state.
type Server struct {
	mcp *server.MCPServer
	cfg *core.Config
}

// NewServer creates and configures the MCP server with all zipfs tools registered.
func NewServer() (*Server, error) {
	// Load configuration
	dataDir, err := core.DataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	cfg, err := core.LoadConfig(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	s := &Server{
		cfg: cfg,
	}

	// Create MCP server
	s.mcp = server.NewMCPServer(serverName, serverVersion)

	// Register all tools
	if err := s.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return s, nil
}

// registerTools registers all 13 MCP tools defined in ADR-005.
func (s *Server) registerTools() error {
	// zipfs_open
	s.mcp.AddTool(mcp.NewTool("zipfs_open",
		mcp.WithDescription("Opens a zip file and creates a workspace session"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Absolute path to the zip file")),
		mcp.WithString("name",
			mcp.Description("Human-readable session name")),
	), s.handleOpen)

	// zipfs_close
	s.mcp.AddTool(mcp.NewTool("zipfs_close",
		mcp.WithDescription("Closes a session and removes its workspace"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
		mcp.WithBoolean("sync",
			mcp.Description("Sync before closing (default: false)")),
	), s.handleClose)

	// zipfs_ls
	s.mcp.AddTool(mcp.NewTool("zipfs_ls",
		mcp.WithDescription("Lists files and directories in the workspace"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
		mcp.WithString("path",
			mcp.Description("Relative path within workspace (default: \"/\")")),
		mcp.WithBoolean("recursive",
			mcp.Description("Include subdirectories (default: false)")),
	), s.handleLs)

	// zipfs_tree
	s.mcp.AddTool(mcp.NewTool("zipfs_tree",
		mcp.WithDescription("Returns a tree representation of the workspace contents"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
		mcp.WithString("path",
			mcp.Description("Root path for the tree (default: \"/\")")),
		mcp.WithNumber("max_depth",
			mcp.Description("Maximum depth to traverse")),
	), s.handleTree)

	// zipfs_read
	s.mcp.AddTool(mcp.NewTool("zipfs_read",
		mcp.WithDescription("Reads a file from the workspace"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Relative path to file")),
		mcp.WithString("encoding",
			mcp.Description("\"utf-8\" (default) or \"base64\" for binary")),
		mcp.WithNumber("offset",
			mcp.Description("Byte offset to start reading")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum bytes to read")),
	), s.handleRead)

	// zipfs_write
	s.mcp.AddTool(mcp.NewTool("zipfs_write",
		mcp.WithDescription("Writes or updates a file in the workspace"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Relative path to file")),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("File content")),
		mcp.WithString("encoding",
			mcp.Description("\"utf-8\" (default) or \"base64\"")),
		mcp.WithBoolean("create_dirs",
			mcp.Description("Create parent directories (default: true)")),
	), s.handleWrite)

	// zipfs_delete
	s.mcp.AddTool(mcp.NewTool("zipfs_delete",
		mcp.WithDescription("Deletes a file or directory from the workspace"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Relative path within workspace")),
		mcp.WithBoolean("recursive",
			mcp.Description("For directories (default: false)")),
	), s.handleDelete)

	// zipfs_grep
	s.mcp.AddTool(mcp.NewTool("zipfs_grep",
		mcp.WithDescription("Searches file contents in the workspace"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Search pattern (regex)")),
		mcp.WithString("path",
			mcp.Description("Root path to search from (default: \"/\")")),
		mcp.WithString("glob",
			mcp.Description("File glob filter (e.g., \"*.txt\")")),
		mcp.WithBoolean("ignore_case",
			mcp.Description("Case-insensitive search (default: false)")),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum matches to return (default: 100)")),
	), s.handleGrep)

	// zipfs_path
	s.mcp.AddTool(mcp.NewTool("zipfs_path",
		mcp.WithDescription("Returns the filesystem path to the workspace contents directory"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
	), s.handlePath)

	// zipfs_sync
	s.mcp.AddTool(mcp.NewTool("zipfs_sync",
		mcp.WithDescription("Syncs workspace changes back to the original zip file"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
		mcp.WithBoolean("force",
			mcp.Description("Ignore external modification conflict (default: false)")),
		mcp.WithBoolean("dry_run",
			mcp.Description("Preview changes without syncing (default: false)")),
	), s.handleSync)

	// zipfs_status
	s.mcp.AddTool(mcp.NewTool("zipfs_status",
		mcp.WithDescription("Shows what changed in the workspace since extraction"),
		mcp.WithString("session",
			mcp.Description("Session name or ID")),
	), s.handleStatus)

	// zipfs_sessions
	s.mcp.AddTool(mcp.NewTool("zipfs_sessions",
		mcp.WithDescription("Lists all open sessions"),
	), s.handleSessions)

	// zipfs_prune
	s.mcp.AddTool(mcp.NewTool("zipfs_prune",
		mcp.WithDescription("Removes stale or all workspaces"),
		mcp.WithBoolean("all",
			mcp.Description("Remove all sessions (default: false)")),
		mcp.WithString("stale",
			mcp.Description("Duration like \"24h\", \"7d\"")),
		mcp.WithBoolean("dry_run",
			mcp.Description("Preview without removing (default: false)")),
	), s.handlePrune)

	return nil
}

// Serve starts the MCP server on stdio transport.
func (s *Server) Serve() error {
	stdioServer := server.NewStdioServer(s.mcp)
	ctx := context.Background()
	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("failed to serve MCP: %w", err)
	}
	return nil
}

// Serve creates a new MCP server and starts serving on stdio.
func Serve() error {
	srv, err := NewServer()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if err := srv.Serve(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
