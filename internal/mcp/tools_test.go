package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/Fuabioo/zipfs/internal/errors"
	"github.com/mark3labs/mcp-go/mcp"
)

// newTestRequest creates a CallToolRequest for testing
func newTestRequest(arguments map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: arguments,
		},
	}
}

// getResultText extracts the text from a CallToolResult for testing
func getResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
		return textContent.Text
	}
	return ""
}

func TestHandleOpen_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	files := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}
	createTestZip(t, zipPath, files)

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"path": zipPath,
		"name": "test-session",
	}

	result, err := srv.handleOpen(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleOpen failed: %v", err)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["session_id"] == nil {
		t.Error("expected session_id in response")
	}

	if response["name"] != "test-session" {
		t.Errorf("expected name 'test-session', got %v", response["name"])
	}

	if response["file_count"] != float64(2) {
		t.Errorf("expected file_count 2, got %v", response["file_count"])
	}

	if response["workspace_path"] == nil {
		t.Error("expected workspace_path in response")
	}
}

func TestHandleOpen_MissingPath(t *testing.T) {
	setupTestEnvironment(t)

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{}

	result, err := srv.handleOpen(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return error result
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["error"] == nil {
		t.Error("expected error in response")
	}
}

func TestHandleOpen_ZipNotFound(t *testing.T) {
	setupTestEnvironment(t)

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"path": "/nonexistent/file.zip",
	}

	result, err := srv.handleOpen(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return ZIP_NOT_FOUND error
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	errData := response["error"].(map[string]interface{})
	if errData["code"] != errors.CodeZipNotFound {
		t.Errorf("expected error code %s, got %v", errors.CodeZipNotFound, errData["code"])
	}
}

func TestHandleClose_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "test-close", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session": session.Name,
		"sync":    false,
	}

	result, err := srv.handleClose(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleClose failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["closed"] != true {
		t.Error("expected closed to be true")
	}

	if response["synced"] != false {
		t.Error("expected synced to be false")
	}

	// Verify session is deleted
	_, err = core.GetSession(session.ID)
	if err == nil {
		t.Error("expected session to be deleted")
	}
}

func TestHandleLs_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{
		"file1.txt":     "content1",
		"dir/file2.txt": "content2",
	})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session":   session.ID,
		"path":      "/",
		"recursive": false,
	}

	result, err := srv.handleLs(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleLs failed: %v", err)
	}

	var response map[string]interface{}
	resultText := getResultText(result)
	if err := json.Unmarshal([]byte(resultText), &response); err != nil {
		t.Fatalf("failed to parse response: %v (text: %s)", err, resultText)
	}

	entries, ok := response["entries"].([]interface{})
	if !ok {
		t.Fatalf("entries not found or wrong type in response: %+v", response)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestHandleTree_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{
		"file1.txt":     "content1",
		"dir/file2.txt": "content2",
	})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session": session.ID,
		"path":    "/",
	}

	result, err := srv.handleTree(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleTree failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	tree := response["tree"].(string)
	if tree == "" {
		t.Error("expected non-empty tree")
	}

	if response["file_count"] == nil {
		t.Error("expected file_count in response")
	}

	if response["dir_count"] == nil {
		t.Error("expected dir_count in response")
	}
}

func TestHandleRead_UTF8(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	content := "Hello, World!"
	createTestZip(t, zipPath, map[string]string{"file.txt": content})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session":  session.ID,
		"path":     "file.txt",
		"encoding": "utf-8",
	}

	result, err := srv.handleRead(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleRead failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["content"] != content {
		t.Errorf("expected content %q, got %v", content, response["content"])
	}

	if response["encoding"] != "utf-8" {
		t.Errorf("expected encoding 'utf-8', got %v", response["encoding"])
	}
}

func TestHandleRead_Base64(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	content := "Binary content"
	createTestZip(t, zipPath, map[string]string{"file.bin": content})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session":  session.ID,
		"path":     "file.bin",
		"encoding": "base64",
	}

	result, err := srv.handleRead(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleRead failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Decode and verify
	decoded, err := base64.StdEncoding.DecodeString(response["content"].(string))
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}

	if string(decoded) != content {
		t.Errorf("expected decoded content %q, got %q", content, string(decoded))
	}
}

func TestHandleWrite_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"existing.txt": "old content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	newContent := "new content"
	args := map[string]interface{}{
		"session": session.ID,
		"path":    "newfile.txt",
		"content": newContent,
	}

	result, err := srv.handleWrite(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleWrite failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["written"] != true {
		t.Error("expected written to be true")
	}

	// Verify file was written
	dirName := session.ID
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	data, err := core.ReadFile(contentsDir, "newfile.txt")
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(data) != newContent {
		t.Errorf("expected content %q, got %q", newContent, string(data))
	}
}

func TestHandleDelete_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"todelete.txt": "content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session": session.ID,
		"path":    "todelete.txt",
	}

	result, err := srv.handleDelete(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleDelete failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["deleted"] != true {
		t.Error("expected deleted to be true")
	}

	// Verify file was deleted
	dirName := session.ID
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	_, err = core.ReadFile(contentsDir, "todelete.txt")
	if !errors.Is(err, errors.CodePathNotFound) {
		t.Error("expected file to be deleted")
	}
}

func TestHandleGrep_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{
		"file1.txt": "Hello World\nTest line\n",
		"file2.txt": "Another file\nWith Hello\n",
	})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session": session.ID,
		"pattern": "Hello",
	}

	result, err := srv.handleGrep(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleGrep failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	matches := response["matches"].([]interface{})
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestHandlePath_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "path-test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session": session.Name,
	}

	result, err := srv.handlePath(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handlePath failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	path := response["path"].(string)
	if !strings.Contains(path, "path-test") {
		t.Errorf("expected path to contain 'path-test', got %q", path)
	}

	// Verify path exists
	if _, err := os.Stat(path); err != nil {
		t.Errorf("path does not exist: %v", err)
	}
}

func TestHandleStatus_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Modify workspace
	dirName := session.ID
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	// Add a file
	if err := core.WriteFile(contentsDir, "newfile.txt", []byte("new"), true); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session": session.ID,
	}

	result, err := srv.handleStatus(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleStatus failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	added := response["added"].([]interface{})
	if len(added) != 1 {
		t.Errorf("expected 1 added file, got %d", len(added))
	}
}

func TestHandleSessions_Success(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create multiple sessions
	zipPath1 := filepath.Join(tempDir, "test1.zip")
	createTestZip(t, zipPath1, map[string]string{"file.txt": "content"})

	zipPath2 := filepath.Join(tempDir, "test2.zip")
	createTestZip(t, zipPath2, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	_, err := core.CreateSession(zipPath1, "session1", cfg)
	if err != nil {
		t.Fatalf("failed to create session1: %v", err)
	}

	_, err = core.CreateSession(zipPath2, "session2", cfg)
	if err != nil {
		t.Fatalf("failed to create session2: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{}

	result, err := srv.handleSessions(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleSessions failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	sessions := response["sessions"].([]interface{})
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestHandlePrune_DryRun(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "to-prune", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"all":     true,
		"dry_run": true,
	}

	result, err := srv.handlePrune(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handlePrune failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	pruned := response["pruned"].([]interface{})
	if len(pruned) != 1 {
		t.Errorf("expected 1 session to be pruned, got %d", len(pruned))
	}

	// Verify session still exists (dry run)
	_, err = core.GetSession(session.ID)
	if err != nil {
		t.Error("expected session to still exist after dry run")
	}
}

func TestHandlePrune_All(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "to-prune", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"all": true,
	}

	result, err := srv.handlePrune(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handlePrune failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	pruned := response["pruned"].([]interface{})
	if len(pruned) != 1 {
		t.Errorf("expected 1 session to be pruned, got %d", len(pruned))
	}

	// Verify session is deleted
	_, err = core.GetSession(session.ID)
	if err == nil {
		t.Error("expected session to be deleted")
	}
}

func TestHandleSync_DryRun(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	args := map[string]interface{}{
		"session": session.ID,
		"dry_run": true,
	}

	result, err := srv.handleSync(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleSync failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["synced"] != false {
		t.Error("expected synced to be false for dry run")
	}
}

func TestResolveSession_Auto(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create single session
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	session, err := core.CreateSession(zipPath, "", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Don't specify session - should auto-resolve
	args := map[string]interface{}{
		"path": "file.txt",
	}

	result, err := srv.handleRead(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("handleRead failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should succeed without specifying session
	if response["error"] != nil {
		t.Errorf("expected success, got error: %v", response["error"])
	}

	// Clean up
	core.DeleteSession(session.ID)
}

func TestResolveSession_Ambiguous(t *testing.T) {
	setupTestEnvironment(t)
	tempDir := t.TempDir()

	// Create multiple sessions
	zipPath1 := filepath.Join(tempDir, "test1.zip")
	createTestZip(t, zipPath1, map[string]string{"file.txt": "content"})

	zipPath2 := filepath.Join(tempDir, "test2.zip")
	createTestZip(t, zipPath2, map[string]string{"file.txt": "content"})

	cfg := core.DefaultConfig()
	session1, err := core.CreateSession(zipPath1, "session1", cfg)
	if err != nil {
		t.Fatalf("failed to create session1: %v", err)
	}

	session2, err := core.CreateSession(zipPath2, "session2", cfg)
	if err != nil {
		t.Fatalf("failed to create session2: %v", err)
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Don't specify session - should fail with ambiguous error
	args := map[string]interface{}{
		"path": "file.txt",
	}

	result, err := srv.handleRead(context.Background(), newTestRequest(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should return AMBIGUOUS_SESSION error
	errData := response["error"].(map[string]interface{})
	if errData["code"] != errors.CodeAmbiguousSession {
		t.Errorf("expected error code %s, got %v", errors.CodeAmbiguousSession, errData["code"])
	}

	// Clean up
	core.DeleteSession(session1.ID)
	core.DeleteSession(session2.ID)
}
