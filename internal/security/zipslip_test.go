package security

import (
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		entryPath string
		errMsg    string
		wantErr   bool
	}{
		// Valid cases
		{
			name:      "simple file",
			base:      "/tmp/workspace",
			entryPath: "file.txt",
			wantErr:   false,
		},
		{
			name:      "nested directory",
			base:      "/tmp/workspace",
			entryPath: "dir/subdir/file.txt",
			wantErr:   false,
		},
		{
			name:      "dot directory (current)",
			base:      "/tmp/workspace",
			entryPath: "./file.txt",
			wantErr:   false,
		},
		{
			name:      "dot alone",
			base:      "/tmp/workspace",
			entryPath: ".",
			wantErr:   false,
		},
		{
			name:      "directory name with dots",
			base:      "/tmp/workspace",
			entryPath: "my..dir/file.txt",
			wantErr:   false,
		},
		{
			name:      "file name with dots",
			base:      "/tmp/workspace",
			entryPath: "my.file..txt",
			wantErr:   false,
		},
		{
			name:      "hidden file",
			base:      "/tmp/workspace",
			entryPath: ".hidden",
			wantErr:   false,
		},

		// Invalid cases - path traversal
		{
			name:      "simple parent traversal",
			base:      "/tmp/workspace",
			entryPath: "../etc/passwd",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "double parent traversal",
			base:      "/tmp/workspace",
			entryPath: "../../etc/passwd",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "parent traversal in middle",
			base:      "/tmp/workspace",
			entryPath: "dir/../../../etc/passwd",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "parent traversal with forward slashes",
			base:      "/tmp/workspace",
			entryPath: "dir/../../etc/passwd",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},

		// Invalid cases - absolute paths
		{
			name:      "absolute path unix",
			base:      "/tmp/workspace",
			entryPath: "/etc/passwd",
			wantErr:   true,
			errMsg:    "must be relative",
		},

		// Invalid cases - null bytes
		{
			name:      "null byte in path",
			base:      "/tmp/workspace",
			entryPath: "file\x00.txt",
			wantErr:   true,
			errMsg:    "null byte",
		},

		// Invalid cases - empty path
		{
			name:      "empty path",
			base:      "/tmp/workspace",
			entryPath: "",
			wantErr:   true,
			errMsg:    "cannot be empty",
		},

		// Edge cases
		{
			name:      "path with spaces",
			base:      "/tmp/workspace",
			entryPath: "my file.txt",
			wantErr:   false,
		},
		{
			name:      "path with unicode",
			base:      "/tmp/workspace",
			entryPath: "文件.txt",
			wantErr:   false,
		},
		{
			name:      "deeply nested",
			base:      "/tmp/workspace",
			entryPath: "a/b/c/d/e/f/g/h/i/j/k/file.txt",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.base, tt.entryPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePath() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePath() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePath() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidatePath_BasePathVariations(t *testing.T) {
	// Test that validation works with various base path formats
	tests := []struct {
		name      string
		base      string
		entryPath string
		wantErr   bool
	}{
		{
			name:      "base with trailing slash",
			base:      "/tmp/workspace/",
			entryPath: "file.txt",
			wantErr:   false,
		},
		{
			name:      "base without trailing slash",
			base:      "/tmp/workspace",
			entryPath: "file.txt",
			wantErr:   false,
		},
		{
			name:      "relative base path",
			base:      "workspace",
			entryPath: "file.txt",
			wantErr:   false,
		},
		{
			name:      "base with dots in name",
			base:      "/tmp/work..space",
			entryPath: "file.txt",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.base, tt.entryPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAllPaths(t *testing.T) {
	base := "/tmp/workspace"

	tests := []struct {
		name    string
		errMsg  string
		paths   []string
		wantErr bool
	}{
		{
			name: "all valid paths",
			paths: []string{
				"file1.txt",
				"dir/file2.txt",
				"dir/subdir/file3.txt",
			},
			wantErr: false,
		},
		{
			name:    "empty slice",
			paths:   []string{},
			wantErr: false,
		},
		{
			name: "one invalid path at start",
			paths: []string{
				"../etc/passwd",
				"file1.txt",
				"file2.txt",
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "one invalid path in middle",
			paths: []string{
				"file1.txt",
				"../../etc/passwd",
				"file2.txt",
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "one invalid path at end",
			paths: []string{
				"file1.txt",
				"file2.txt",
				"/etc/passwd",
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "multiple invalid paths",
			paths: []string{
				"../etc/passwd",
				"/etc/shadow",
				"file.txt",
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "path with null byte",
			paths: []string{
				"file1.txt",
				"file\x00.txt",
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAllPaths(base, tt.paths)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAllPaths() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateAllPaths() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAllPaths() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidatePath_WindowsPaths(t *testing.T) {
	// Test Windows-specific path patterns
	tests := []struct {
		name      string
		entryPath string
		wantErr   bool
	}{
		{
			name:      "windows backslash traversal",
			entryPath: "..\\..\\windows\\system32",
			wantErr:   true, // filepath.Clean normalizes backslashes and detects ".."
		},
		// Note: Windows drive letters like "C:\windows" are only detected as absolute on Windows
		// On Unix, they're treated as relative paths (which is acceptable for our use case)
	}

	base := "/tmp/workspace"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(base, tt.entryPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// BenchmarkValidatePath measures performance of path validation
func BenchmarkValidatePath(b *testing.B) {
	base := "/tmp/workspace"
	entryPath := "dir/subdir/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePath(base, entryPath)
	}
}

// BenchmarkValidateAllPaths measures performance of batch validation
func BenchmarkValidateAllPaths(b *testing.B) {
	base := "/tmp/workspace"
	paths := []string{
		"file1.txt",
		"dir/file2.txt",
		"dir/subdir/file3.txt",
		"another/deep/nested/path/file4.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateAllPaths(base, paths)
	}
}

// TestValidatePath_RealWorldPatterns tests patterns seen in actual zip files
func TestValidatePath_RealWorldPatterns(t *testing.T) {
	base := "/tmp/workspace"

	tests := []struct {
		name      string
		entryPath string
		wantErr   bool
	}{
		// Common legitimate patterns
		{name: "node_modules nested", entryPath: "node_modules/package/dist/file.js", wantErr: false},
		{name: "git directory", entryPath: ".git/config", wantErr: false},
		{name: "hidden directory", entryPath: ".config/app/settings.json", wantErr: false},
		{name: "maven structure", entryPath: "src/main/java/com/example/App.java", wantErr: false},

		// Known attack patterns
		{name: "classic zip slip", entryPath: "../../../../etc/passwd", wantErr: true},
		{name: "zip slip with normal prefix", entryPath: "normal/../../../../../../etc/passwd", wantErr: true},
		{name: "absolute path linux", entryPath: "/etc/passwd", wantErr: true},
		// Note: Windows absolute paths are only detected as absolute on Windows platform
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(base, tt.entryPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.entryPath, err, tt.wantErr)
			}
		})
	}
}
