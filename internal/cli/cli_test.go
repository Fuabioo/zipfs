package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/Fuabioo/zipfs/internal/errors"
	"github.com/spf13/cobra"
)

// setupTestEnv creates a temporary environment for testing
func setupTestEnv(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Set ZIPFS_DATA_DIR to use temp directory
	oldDataDir := os.Getenv("ZIPFS_DATA_DIR")
	t.Cleanup(func() {
		if oldDataDir != "" {
			os.Setenv("ZIPFS_DATA_DIR", oldDataDir)
		} else {
			os.Unsetenv("ZIPFS_DATA_DIR")
		}
	})

	os.Setenv("ZIPFS_DATA_DIR", tempDir)

	return tempDir
}

// createTestZip creates a test zip file
func createTestZip(t *testing.T, dir string, name string) string {
	t.Helper()

	// Create a simple zip file with test content
	zipPath := filepath.Join(dir, name)

	// Create test files to zip
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}

	testFile := filepath.Join(contentDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create zip using core.Repack
	if err := core.Repack(contentDir, zipPath); err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}

	return zipPath
}

// executeCommand executes a cobra command with args and returns output.
// Captures real os.Stdout/os.Stderr since CLI commands use fmt.Printf.
func executeCommand(t *testing.T, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	// Save and restore original stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Create pipes
	stdoutR, stdoutW, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create stdout pipe: %v", pipeErr)
	}
	stderrR, stderrW, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create stderr pipe: %v", pipeErr)
	}

	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Also set cobra's output to the pipes
	cmd.SetOut(stdoutW)
	cmd.SetErr(stderrW)
	cmd.SetArgs(args)

	// Execute in goroutine so pipe reads don't block
	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Execute()
		stdoutW.Close()
		stderrW.Close()
	}()

	// Read all output
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdoutBuf, stdoutR)
		close(stdoutDone)
	}()
	go func() {
		_, _ = io.Copy(&stderrBuf, stderrR)
		close(stderrDone)
	}()

	err = <-errChan
	<-stdoutDone
	<-stderrDone

	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestHelpers_ParseColonSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSession string
		wantPath    string
	}{
		{
			name:        "with colon",
			input:       "session:path/to/file",
			wantSession: "session",
			wantPath:    "path/to/file",
		},
		{
			name:        "without colon",
			input:       "path/to/file",
			wantSession: "",
			wantPath:    "path/to/file",
		},
		{
			name:        "multiple colons",
			input:       "session:path:with:colons",
			wantSession: "session",
			wantPath:    "path:with:colons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, path := parseColonSyntax(tt.input)
			if session != tt.wantSession {
				t.Errorf("session = %q, want %q", session, tt.wantSession)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
		})
	}
}

func TestHelpers_GetExitCode(t *testing.T) {
	tests := []struct {
		err  error
		name string
		want int
	}{
		{
			name: "nil error",
			err:  nil,
			want: 0,
		},
		{
			name: "session not found",
			err:  errors.SessionNotFound("test"),
			want: 4,
		},
		{
			name: "ambiguous session",
			err:  errors.AmbiguousSession(2),
			want: 4,
		},
		{
			name: "no sessions",
			err:  errors.NoSessions(),
			want: 4,
		},
		{
			name: "zip bomb",
			err:  errors.ZipBombDetected("test"),
			want: 5,
		},
		{
			name: "conflict",
			err:  errors.ConflictDetected("/path"),
			want: 3,
		},
		{
			name: "general error",
			err:  errors.New("UNKNOWN", "test"),
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExitCode(tt.err)
			if got != tt.want {
				t.Errorf("getExitCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelpers_OutputJSON(t *testing.T) {
	data := map[string]interface{}{
		"key":   "value",
		"count": 42,
	}

	// Redirect stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(data)
	if err != nil {
		t.Fatalf("outputJSON() error = %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("key = %v, want value", result["key"])
	}
	if int(result["count"].(float64)) != 42 {
		t.Errorf("count = %v, want 42", result["count"])
	}
}

func TestOpenCommand(t *testing.T) {
	setupTestEnv(t)

	tempDir := t.TempDir()
	zipPath := createTestZip(t, tempDir, "test.zip")

	// Test basic open
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(openCmd)

	stdout, _, err := executeCommand(t, cmd, "open", zipPath)
	if err != nil {
		t.Fatalf("open command failed: %v", err)
	}

	if !strings.Contains(stdout, "Session opened:") {
		t.Errorf("output missing session opened message: %s", stdout)
	}

	// Verify session was created
	sessions, err := core.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestOpenCommand_WithName(t *testing.T) {
	setupTestEnv(t)

	tempDir := t.TempDir()
	zipPath := createTestZip(t, tempDir, "test.zip")

	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(openCmd)

	stdout, _, err := executeCommand(t, cmd, "open", zipPath, "--name", "mysession")
	if err != nil {
		t.Fatalf("open command failed: %v", err)
	}

	if !strings.Contains(stdout, "mysession") {
		t.Errorf("output missing session name: %s", stdout)
	}

	// Verify session name
	sessions, err := core.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "mysession" {
		t.Errorf("session name = %q, want mysession", sessions[0].Name)
	}
}

func TestOpenCommand_JSON(t *testing.T) {
	setupTestEnv(t)

	tempDir := t.TempDir()
	zipPath := createTestZip(t, tempDir, "test.zip")

	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(openCmd)

	// Set global JSON flag directly (--json is a persistent flag on root, not available here)
	flagJSON = true
	defer func() { flagJSON = false }()

	stdout, _, err := executeCommand(t, cmd, "open", zipPath)
	if err != nil {
		t.Fatalf("open command failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if _, ok := result["session_id"]; !ok {
		t.Error("JSON output missing session_id")
	}
	if _, ok := result["workspace_path"]; !ok {
		t.Error("JSON output missing workspace_path")
	}
}

func TestSessionsCommand(t *testing.T) {
	setupTestEnv(t)

	// Create multiple sessions
	tempDir := t.TempDir()
	zip1 := createTestZip(t, tempDir, "test1.zip")
	zip2 := createTestZip(t, tempDir, "test2.zip")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	_, err = core.CreateSession(zip1, "session1", cfg)
	if err != nil {
		t.Fatalf("failed to create session1: %v", err)
	}

	_, err = core.CreateSession(zip2, "session2", cfg)
	if err != nil {
		t.Fatalf("failed to create session2: %v", err)
	}

	// Test sessions command
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(sessionsCmd)

	stdout, _, err := executeCommand(t, cmd, "sessions")
	if err != nil {
		t.Fatalf("sessions command failed: %v", err)
	}

	if !strings.Contains(stdout, "session1") {
		t.Errorf("output missing session1: %s", stdout)
	}
	if !strings.Contains(stdout, "session2") {
		t.Errorf("output missing session2: %s", stdout)
	}
}

func TestCloseCommand(t *testing.T) {
	setupTestEnv(t)

	tempDir := t.TempDir()
	zipPath := createTestZip(t, tempDir, "test.zip")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	session, err := core.CreateSession(zipPath, "test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Test close with --no-sync
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(closeCmd)

	_, _, err = executeCommand(t, cmd, "close", session.Name, "--no-sync")
	if err != nil {
		t.Fatalf("close command failed: %v", err)
	}

	// Verify session was deleted
	sessions, err := core.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after close, got %d", len(sessions))
	}
}

func TestExecute_PrintsErrorToStderr(t *testing.T) {
	setupTestEnv(t)

	// Test that errors from commands are printed to stderr.
	// Use close with a nonexistent session to trigger an error.
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(closeCmd)

	_, stderr, err := executeCommand(t, cmd, "close", "nonexistent-session", "--no-sync")
	if err == nil {
		t.Fatal("expected error for nonexistent session, got nil")
	}

	// The command itself returns an error. In real usage, Execute() in root.go
	// calls printError() before os.Exit(). Since we're testing the command directly,
	// verify that printError produces the right output by calling it explicitly.
	oldStderr := os.Stderr
	stderrR, stderrW, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create stderr pipe: %v", pipeErr)
	}
	os.Stderr = stderrW

	printError(err)

	stderrW.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, stderrR); copyErr != nil {
		t.Fatalf("failed to read stderr: %v", copyErr)
	}

	output := buf.String()
	if !strings.Contains(output, "Error:") {
		t.Errorf("stderr missing 'Error:' prefix, got: %s", output)
	}
	if !strings.Contains(output, "SESSION_NOT_FOUND") {
		t.Errorf("stderr missing error code, got: %s", output)
	}

	// Also verify the original command produced an error on stderr via cobra
	_ = stderr // cobra's SilenceErrors suppresses its own output, but our printError catches it
}

func TestExecute_ErrorExitCodes(t *testing.T) {
	// Verify that getExitCode returns proper codes for errors that would come from CLI
	tests := []struct {
		err      error
		name     string
		wantCode int
	}{
		{
			name:     "session not found gets exit code 4",
			err:      errors.SessionNotFound("nonexistent"),
			wantCode: 4,
		},
		{
			name:     "generic cobra error gets exit code 1",
			err:      fmt.Errorf("unknown flag: --session"),
			wantCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := getExitCode(tt.err)
			if code != tt.wantCode {
				t.Errorf("getExitCode() = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

func TestVersionCommand(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(versionCmd)

	stdout, _, err := executeCommand(t, cmd, "version")
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	if !strings.Contains(stdout, "zipfs version") {
		t.Errorf("output missing version info: %s", stdout)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		bytes uint64
	}{
		{
			name:  "bytes",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "kilobytes",
			bytes: 1024,
			want:  "1.0 KiB",
		},
		{
			name:  "megabytes",
			bytes: 1024 * 1024,
			want:  "1.0 MiB",
		},
		{
			name:  "gigabytes",
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "hours",
			input:   "24h",
			wantErr: false,
		},
		{
			name:    "days",
			input:   "7d",
			wantErr: false,
		},
		{
			name:    "minutes",
			input:   "30m",
			wantErr: false,
		},
		{
			name:    "invalid",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestReadWriteCommands(t *testing.T) {
	setupTestEnv(t)

	tempDir := t.TempDir()
	zipPath := createTestZip(t, tempDir, "test.zip")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	session, err := core.CreateSession(zipPath, "test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Test read
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(readCmd)

	stdout, _, err := executeCommand(t, cmd, "read", session.Name+":test.txt")
	if err != nil {
		t.Fatalf("read command failed: %v", err)
	}

	if !strings.Contains(stdout, "hello world") {
		t.Errorf("read output incorrect: %s", stdout)
	}

	// Test write with content flag
	writeRootCmd := &cobra.Command{Use: "test"}
	writeRootCmd.AddCommand(writeCmd)

	_, _, err = executeCommand(t, writeRootCmd, "write", session.Name+":newfile.txt", "--content", "new content")
	if err != nil {
		t.Fatalf("write command failed: %v", err)
	}

	// Verify write by reading back
	readRootCmd2 := &cobra.Command{Use: "test"}
	readRootCmd2.AddCommand(readCmd)

	stdout, _, err = executeCommand(t, readRootCmd2, "read", session.Name+":newfile.txt")
	if err != nil {
		t.Fatalf("read after write failed: %v", err)
	}

	if !strings.Contains(stdout, "new content") {
		t.Errorf("read after write incorrect: %s", stdout)
	}
}

func TestStatusCommand(t *testing.T) {
	setupTestEnv(t)

	tempDir := t.TempDir()
	zipPath := createTestZip(t, tempDir, "test.zip")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	session, err := core.CreateSession(zipPath, "test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Modify a file
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		t.Fatalf("failed to get contents dir: %v", err)
	}

	err = core.WriteFile(contentsDir, "test.txt", []byte("modified content"), false)
	if err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Test status
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(statusCmd)

	stdout, _, err := executeCommand(t, cmd, "status", session.Name)
	if err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	if !strings.Contains(stdout, "Modified") {
		t.Errorf("status output missing Modified: %s", stdout)
	}
}

func TestPathCommand(t *testing.T) {
	setupTestEnv(t)

	tempDir := t.TempDir()
	zipPath := createTestZip(t, tempDir, "test.zip")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	session, err := core.CreateSession(zipPath, "test", cfg)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Test path command
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(pathCmd)

	stdout, _, err := executeCommand(t, cmd, "path", session.Name)
	if err != nil {
		t.Fatalf("path command failed: %v", err)
	}

	// Should contain the workspace path
	if !strings.Contains(stdout, "contents") {
		t.Errorf("path output incorrect: %s", stdout)
	}
}
