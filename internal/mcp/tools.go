package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/Fuabioo/zipfs/internal/errors"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleOpen implements zipfs_open: Opens a zip file and creates a workspace session.
func (s *Server) handleOpen(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	path, err := request.RequireString("path")
	if err != nil {
		return errorResult("INVALID_PARAMS", "path is required"), nil
	}

	name := request.GetString("name", "")

	// Create session
	session, err := core.CreateSession(path, name, s.cfg)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get workspace path
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// Build response
	response := map[string]interface{}{
		"session_id":           session.ID,
		"name":                 session.Name,
		"workspace_path":       contentsDir,
		"file_count":           session.FileCount,
		"extracted_size_bytes": session.ExtractedSizeBytes,
	}

	return jsonResult(response), nil
}

// handleClose implements zipfs_close: Closes a session and removes its workspace.
func (s *Server) handleClose(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")
	doSync := request.GetBool("sync", false)

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Sync if requested
	synced := false
	if doSync {
		_, err := core.Sync(session, false, s.cfg)
		if err != nil {
			return mcpErrorResult(err), nil
		}
		synced = true
	}

	// Delete session
	if err := core.DeleteSession(session.ID); err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	response := map[string]interface{}{
		"closed": true,
		"synced": synced,
	}

	return jsonResult(response), nil
}

// handleLs implements zipfs_ls: Lists files and directories in the workspace.
func (s *Server) handleLs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")
	path := request.GetString("path", ".")
	// Convert "/" to "." for root path
	if path == "/" || path == "" {
		path = "."
	}
	recursive := request.GetBool("recursive", false)

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get contents directory
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// List files
	entries, err := core.ListFiles(contentsDir, path, recursive)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Convert to response format
	var responseEntries []map[string]interface{}
	for _, entry := range entries {
		responseEntries = append(responseEntries, map[string]interface{}{
			"name":        entry.Name,
			"type":        entry.Type,
			"size_bytes":  entry.SizeBytes,
			"modified_at": time.Unix(entry.ModifiedAt, 0).Format(time.RFC3339),
		})
	}

	response := map[string]interface{}{
		"entries": responseEntries,
	}

	// Touch session (non-fatal)
	_ = core.TouchSession(session)

	return jsonResult(response), nil
}

// handleTree implements zipfs_tree: Returns a tree representation of the workspace contents.
func (s *Server) handleTree(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")
	path := request.GetString("path", ".")
	// Convert "/" to "." for root path
	if path == "/" || path == "" {
		path = "."
	}
	maxDepth := request.GetInt("max_depth", 0)

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get contents directory
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// Generate tree
	tree, fileCount, dirCount, err := core.TreeView(contentsDir, path, maxDepth)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	response := map[string]interface{}{
		"tree":       tree,
		"file_count": fileCount,
		"dir_count":  dirCount,
	}

	// Touch session (non-fatal)
	_ = core.TouchSession(session)

	return jsonResult(response), nil
}

// handleRead implements zipfs_read: Reads a file from the workspace.
func (s *Server) handleRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")
	path, err := request.RequireString("path")
	if err != nil {
		return errorResult("INVALID_PARAMS", "path is required"), nil
	}
	encoding := request.GetString("encoding", "utf-8")
	offset := request.GetInt("offset", 0)
	limit := request.GetInt("limit", 0)

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get contents directory
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// Read file
	data, err := core.ReadFile(contentsDir, path)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Apply offset and limit
	if offset > 0 {
		if offset >= len(data) {
			data = []byte{}
		} else {
			data = data[offset:]
		}
	}
	if limit > 0 && len(data) > limit {
		data = data[:limit]
	}

	// Encode based on encoding parameter
	var content string
	if encoding == "base64" {
		content = base64.StdEncoding.EncodeToString(data)
	} else {
		content = string(data)
	}

	response := map[string]interface{}{
		"content":    content,
		"size_bytes": len(data),
		"encoding":   encoding,
	}

	// Touch session (non-fatal)
	_ = core.TouchSession(session)

	return jsonResult(response), nil
}

// handleWrite implements zipfs_write: Writes or updates a file in the workspace.
func (s *Server) handleWrite(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")
	path, err := request.RequireString("path")
	if err != nil {
		return errorResult("INVALID_PARAMS", "path is required"), nil
	}
	content, err := request.RequireString("content")
	if err != nil {
		return errorResult("INVALID_PARAMS", "content is required"), nil
	}
	encoding := request.GetString("encoding", "utf-8")
	createDirs := request.GetBool("create_dirs", true)

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get contents directory
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// Decode content based on encoding
	var data []byte
	if encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return errorResult("INVALID_PARAMS", "invalid base64 encoding"), nil
		}
		data = decoded
	} else {
		data = []byte(content)
	}

	// Write file
	if err := core.WriteFile(contentsDir, path, data, createDirs); err != nil {
		return mcpErrorResult(err), nil
	}

	response := map[string]interface{}{
		"written":    true,
		"size_bytes": len(data),
	}

	// Touch session (non-fatal)
	_ = core.TouchSession(session)

	return jsonResult(response), nil
}

// handleDelete implements zipfs_delete: Deletes a file or directory from the workspace.
func (s *Server) handleDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")
	path, err := request.RequireString("path")
	if err != nil {
		return errorResult("INVALID_PARAMS", "path is required"), nil
	}
	recursive := request.GetBool("recursive", false)

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get contents directory
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// Delete file
	if err := core.DeleteFile(contentsDir, path, recursive); err != nil {
		return mcpErrorResult(err), nil
	}

	response := map[string]interface{}{
		"deleted": true,
		"path":    path,
	}

	// Touch session (non-fatal)
	_ = core.TouchSession(session)

	return jsonResult(response), nil
}

// handleGrep implements zipfs_grep: Searches file contents in the workspace.
func (s *Server) handleGrep(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")
	pattern, err := request.RequireString("pattern")
	if err != nil {
		return errorResult("INVALID_PARAMS", "pattern is required"), nil
	}
	path := request.GetString("path", ".")
	// Convert "/" to "." for root path
	if path == "/" || path == "" {
		path = "."
	}
	glob := request.GetString("glob", "")
	ignoreCase := request.GetBool("ignore_case", false)
	maxResults := request.GetInt("max_results", 100)

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get contents directory
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// Search files
	matches, totalMatches, err := core.GrepFiles(contentsDir, path, pattern, glob, ignoreCase, maxResults)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Convert to response format
	var responseMatches []map[string]interface{}
	for _, match := range matches {
		responseMatches = append(responseMatches, map[string]interface{}{
			"file":         match.File,
			"line_number":  match.LineNumber,
			"line_content": match.LineContent,
		})
	}

	response := map[string]interface{}{
		"matches":       responseMatches,
		"total_matches": totalMatches,
		"truncated":     totalMatches > len(matches),
	}

	// Touch session (non-fatal)
	_ = core.TouchSession(session)

	return jsonResult(response), nil
}

// handlePath implements zipfs_path: Returns the filesystem path to the workspace contents directory.
func (s *Server) handlePath(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get contents directory
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	response := map[string]interface{}{
		"path": contentsDir,
	}

	// Touch session (non-fatal)
	_ = core.TouchSession(session)

	return jsonResult(response), nil
}

// handleSync implements zipfs_sync: Syncs workspace changes back to the original zip file.
func (s *Server) handleSync(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")
	force := request.GetBool("force", false)
	dryRun := request.GetBool("dry_run", false)

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// For dry run, use status instead
	if dryRun {
		status, err := core.Status(session)
		if err != nil {
			return mcpErrorResult(err), nil
		}

		response := map[string]interface{}{
			"synced":         false,
			"backup_path":    "",
			"files_modified": len(status.Modified),
			"files_added":    len(status.Added),
			"files_deleted":  len(status.Deleted),
		}

		return jsonResult(response), nil
	}

	// Perform sync
	result, err := core.Sync(session, force, s.cfg)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	response := map[string]interface{}{
		"synced":         true,
		"backup_path":    result.BackupPath,
		"files_modified": result.FilesModified,
		"files_added":    result.FilesAdded,
		"files_deleted":  result.FilesDeleted,
	}

	return jsonResult(response), nil
}

// handleStatus implements zipfs_status: Shows what changed in the workspace since extraction.
func (s *Server) handleStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	sessionID := request.GetString("session", "")

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	// Get status
	status, err := core.Status(session)
	if err != nil {
		return mcpErrorResult(err), nil
	}

	response := map[string]interface{}{
		"modified":        status.Modified,
		"added":           status.Added,
		"deleted":         status.Deleted,
		"unchanged_count": status.UnchangedCount,
	}

	// Touch session (non-fatal)
	_ = core.TouchSession(session)

	return jsonResult(response), nil
}

// handleSessions implements zipfs_sessions: Lists all open sessions.
func (s *Server) handleSessions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// List all sessions
	sessions, err := core.ListSessions()
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// Convert to response format
	var responseSessions []map[string]interface{}
	for _, session := range sessions {
		lastSyncedAt := ""
		if session.LastSyncedAt != nil {
			lastSyncedAt = session.LastSyncedAt.Format(time.RFC3339)
		}

		responseSessions = append(responseSessions, map[string]interface{}{
			"id":                   session.ID,
			"name":                 session.Name,
			"source_path":          session.SourcePath,
			"state":                session.State,
			"created_at":           session.CreatedAt.Format(time.RFC3339),
			"last_accessed_at":     session.LastAccessedAt.Format(time.RFC3339),
			"last_synced_at":       lastSyncedAt,
			"file_count":           session.FileCount,
			"extracted_size_bytes": session.ExtractedSizeBytes,
		})
	}

	response := map[string]interface{}{
		"sessions": responseSessions,
	}

	return jsonResult(response), nil
}

// handlePrune implements zipfs_prune: Removes stale or all workspaces.
func (s *Server) handlePrune(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	all := request.GetBool("all", false)
	staleStr := request.GetString("stale", "")
	dryRun := request.GetBool("dry_run", false)

	// List all sessions
	sessions, err := core.ListSessions()
	if err != nil {
		return errorResult("INTERNAL_ERROR", err.Error()), nil
	}

	// Parse stale duration
	var staleDuration time.Duration
	if staleStr != "" {
		parsed, err := time.ParseDuration(staleStr)
		if err != nil {
			return errorResult("INVALID_PARAMS", fmt.Sprintf("invalid duration: %s", err)), nil
		}
		staleDuration = parsed
	}

	// Determine which sessions to prune
	var toPrune []*core.Session
	for _, session := range sessions {
		shouldPrune := false

		if all {
			shouldPrune = true
		} else if staleDuration > 0 {
			age := time.Since(session.LastAccessedAt)
			if age > staleDuration {
				shouldPrune = true
			}
		}

		if shouldPrune {
			toPrune = append(toPrune, session)
		}
	}

	// Calculate freed space
	var freedBytes uint64
	for _, session := range toPrune {
		freedBytes += session.ExtractedSizeBytes
	}

	// Build result list
	var prunedList []map[string]interface{}
	for _, session := range toPrune {
		age := time.Since(session.LastAccessedAt)
		reason := ""
		if all {
			reason = "all sessions"
		} else {
			reason = fmt.Sprintf("stale (%s)", age.Round(time.Hour))
		}

		prunedList = append(prunedList, map[string]interface{}{
			"id":     session.ID,
			"name":   session.Name,
			"reason": reason,
		})
	}

	// Actually delete if not dry run
	if !dryRun {
		for _, session := range toPrune {
			if err := core.DeleteSession(session.ID); err != nil {
				// Continue on error, but could log here
				continue
			}
		}
	}

	response := map[string]interface{}{
		"pruned":      prunedList,
		"freed_bytes": freedBytes,
	}

	return jsonResult(response), nil
}

// Helper functions

// mcpErrorResult converts a zipfs error to an MCP error result.
func mcpErrorResult(err error) *mcp.CallToolResult {
	code := errors.Code(err)
	if code == "" {
		code = "INTERNAL_ERROR"
	}

	return errorResult(code, err.Error())
}

// errorResult creates an MCP error result.
func errorResult(code, message string) *mcp.CallToolResult {
	errorData := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	jsonBytes, err := json.Marshal(errorData)
	if err != nil {
		// Fallback to simple text
		return mcp.NewToolResultText(fmt.Sprintf("Error: %s - %s", code, message))
	}

	return mcp.NewToolResultText(string(jsonBytes))
}

// jsonResult creates an MCP success result from a JSON-serializable object.
func jsonResult(data interface{}) *mcp.CallToolResult {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return errorResult("INTERNAL_ERROR", fmt.Sprintf("failed to marshal response: %s", err))
	}

	return mcp.NewToolResultText(string(jsonBytes))
}
