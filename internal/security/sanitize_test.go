package security

import (
	"strings"
	"testing"
)

func TestValidateSessionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		errMsg  string
		wantErr bool
	}{
		// Valid cases
		{name: "simple alphanumeric", input: "session123", wantErr: false},
		{name: "with hyphens", input: "my-session", wantErr: false},
		{name: "with underscores", input: "my_session", wantErr: false},
		{name: "mixed valid chars", input: "Session_123-alpha", wantErr: false},
		{name: "all uppercase", input: "SESSION", wantErr: false},
		{name: "all lowercase", input: "session", wantErr: false},
		{name: "numbers only", input: "123456", wantErr: false},
		{name: "single character", input: "a", wantErr: false},
		{name: "max length 64", input: strings.Repeat("a", 64), wantErr: false},

		// Invalid cases - empty
		{name: "empty string", input: "", wantErr: true, errMsg: "cannot be empty"},

		// Invalid cases - length
		{name: "exceeds max length", input: strings.Repeat("a", 65), wantErr: true, errMsg: "exceeds maximum length"},

		// Invalid cases - invalid characters
		{name: "with spaces", input: "my session", wantErr: true, errMsg: "alphanumeric"},
		{name: "with dots", input: "my.session", wantErr: true, errMsg: "alphanumeric"},
		{name: "with slashes", input: "my/session", wantErr: true, errMsg: "alphanumeric"},
		{name: "with backslashes", input: "my\\session", wantErr: true, errMsg: "alphanumeric"},
		{name: "with unicode", input: "session文件", wantErr: true, errMsg: "alphanumeric"},
		{name: "with special chars", input: "session@123", wantErr: true, errMsg: "alphanumeric"},
		{name: "with parentheses", input: "session(1)", wantErr: true, errMsg: "alphanumeric"},
		{name: "with brackets", input: "session[1]", wantErr: true, errMsg: "alphanumeric"},
		{name: "with plus", input: "session+1", wantErr: true, errMsg: "alphanumeric"},
		{name: "with equals", input: "session=1", wantErr: true, errMsg: "alphanumeric"},
		{name: "with colon", input: "session:1", wantErr: true, errMsg: "alphanumeric"},
		{name: "with semicolon", input: "session;1", wantErr: true, errMsg: "alphanumeric"},
		{name: "with null byte", input: "session\x00", wantErr: true, errMsg: "alphanumeric"},
		{name: "with newline", input: "session\n", wantErr: true, errMsg: "alphanumeric"},
		{name: "with tab", input: "session\t", wantErr: true, errMsg: "alphanumeric"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateSessionName(%q) expected error, got nil", tt.input)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateSessionName(%q) error = %v, want error containing %q", tt.input, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateSessionName(%q) unexpected error = %v", tt.input, err)
				}
			}
		})
	}
}

func TestValidateRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		errMsg  string
		wantErr bool
	}{
		// Valid cases
		{name: "simple file", input: "file.txt", wantErr: false},
		{name: "nested path", input: "dir/subdir/file.txt", wantErr: false},
		{name: "current dir prefix", input: "./file.txt", wantErr: false},
		{name: "hidden file", input: ".hidden", wantErr: false},
		{name: "with spaces", input: "my file.txt", wantErr: false},
		{name: "with unicode", input: "文件.txt", wantErr: false},
		{name: "dots in name", input: "my..file.txt", wantErr: false},

		// Invalid cases - empty
		{name: "empty string", input: "", wantErr: true, errMsg: "cannot be empty"},

		// Invalid cases - absolute paths
		{name: "absolute unix path", input: "/etc/passwd", wantErr: true, errMsg: "must be relative"},
		// Note: Windows paths like "C:\Windows" are only detected as absolute on Windows platform

		// Invalid cases - parent traversal
		{name: "parent directory", input: "../file.txt", wantErr: true, errMsg: ".."},
		{name: "double parent", input: "../../file.txt", wantErr: true, errMsg: ".."},
		{name: "parent in middle", input: "dir/../file.txt", wantErr: true, errMsg: ".."},
		{name: "parent at end", input: "dir/..", wantErr: true, errMsg: ".."},

		// Invalid cases - null bytes
		{name: "null byte", input: "file\x00.txt", wantErr: true, errMsg: "null byte"},

		// Invalid cases - control characters
		{name: "newline", input: "file\n.txt", wantErr: true, errMsg: "control character"},
		{name: "carriage return", input: "file\r.txt", wantErr: true, errMsg: "control character"},
		{name: "tab", input: "file\t.txt", wantErr: true, errMsg: "control character"},
		{name: "bell", input: "file\x07.txt", wantErr: true, errMsg: "control character"},
		{name: "escape", input: "file\x1b.txt", wantErr: true, errMsg: "control character"},
		{name: "delete", input: "file\x7f.txt", wantErr: true, errMsg: "control character"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelativePath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRelativePath(%q) expected error, got nil", tt.input)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateRelativePath(%q) error = %v, want error containing %q", tt.input, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRelativePath(%q) unexpected error = %v", tt.input, err)
				}
			}
		})
	}
}

func TestSanitizeGlobPattern(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		errMsg  string
		wantErr bool
	}{
		// Valid cases
		{name: "simple wildcard", input: "*.txt", wantErr: false},
		{name: "recursive wildcard", input: "**/*.go", wantErr: false},
		{name: "question mark", input: "file?.txt", wantErr: false},
		{name: "character class", input: "file[0-9].txt", wantErr: false},
		{name: "multiple wildcards", input: "dir/*/subdir/*.txt", wantErr: false},
		{name: "nested pattern", input: "src/**/*.{go,js}", wantErr: false},

		// Invalid cases - empty
		{name: "empty string", input: "", wantErr: true, errMsg: "cannot be empty"},

		// Invalid cases - absolute paths
		{name: "absolute unix path", input: "/tmp/*.txt", wantErr: true, errMsg: "must be relative"},
		// Note: Windows absolute paths are only detected as absolute on Windows platform

		// Invalid cases - parent traversal
		{name: "parent directory", input: "../*.txt", wantErr: true, errMsg: ".."},
		{name: "double parent", input: "../../*.txt", wantErr: true, errMsg: ".."},
		{name: "parent in middle", input: "dir/../*.txt", wantErr: true, errMsg: ".."},

		// Invalid cases - null bytes
		{name: "null byte", input: "*.txt\x00", wantErr: true, errMsg: "null byte"},

		// Invalid cases - control characters
		{name: "newline", input: "*.txt\n", wantErr: true, errMsg: "control character"},
		{name: "carriage return", input: "*.txt\r", wantErr: true, errMsg: "control character"},
		{name: "bell", input: "*.txt\x07", wantErr: true, errMsg: "control character"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SanitizeGlobPattern(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SanitizeGlobPattern(%q) expected error, got nil", tt.input)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("SanitizeGlobPattern(%q) error = %v, want error containing %q", tt.input, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SanitizeGlobPattern(%q) unexpected error = %v", tt.input, err)
				}
			}
		})
	}
}

func TestValidateSessionName_EdgeCases(t *testing.T) {
	// Test boundary conditions more thoroughly
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "length 63 (valid)", input: strings.Repeat("a", 63), wantErr: false},
		{name: "length 64 (valid)", input: strings.Repeat("a", 64), wantErr: false},
		{name: "length 65 (invalid)", input: strings.Repeat("a", 65), wantErr: true},
		{name: "leading hyphen", input: "-session", wantErr: false},
		{name: "trailing hyphen", input: "session-", wantErr: false},
		{name: "leading underscore", input: "_session", wantErr: false},
		{name: "trailing underscore", input: "session_", wantErr: false},
		{name: "multiple hyphens", input: "my--session", wantErr: false},
		{name: "multiple underscores", input: "my__session", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSessionName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRelativePath_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "just dot", input: ".", wantErr: false},
		{name: "trailing slash", input: "dir/", wantErr: false},
		{name: "multiple slashes", input: "dir//file.txt", wantErr: false},  // normalized by filepath
		{name: "windows backslash", input: "dir\\file.txt", wantErr: false}, // might be valid on windows
		{name: "deeply nested", input: "a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/file.txt", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelativePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRelativePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeGlobPattern_ComplexPatterns(t *testing.T) {
	// Test realistic glob patterns
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "go files", input: "**/*.go", wantErr: false},
		{name: "js and ts", input: "src/**/*.{js,ts}", wantErr: false},
		{name: "test files", input: "*_test.go", wantErr: false},
		{name: "dotfiles", input: ".*", wantErr: false},
		{name: "specific extension", input: "*.{json,yaml,yml}", wantErr: false},
		{name: "exclude pattern", input: "!vendor/**", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SanitizeGlobPattern(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeGlobPattern(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// BenchmarkValidateSessionName measures performance
func BenchmarkValidateSessionName(b *testing.B) {
	name := "valid-session-name-123"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateSessionName(name)
	}
}

// BenchmarkValidateRelativePath measures performance
func BenchmarkValidateRelativePath(b *testing.B) {
	path := "dir/subdir/file.txt"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateRelativePath(path)
	}
}

// BenchmarkSanitizeGlobPattern measures performance
func BenchmarkSanitizeGlobPattern(b *testing.B) {
	pattern := "**/*.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeGlobPattern(pattern)
	}
}

// TestValidateSessionName_SQLInjection tests that SQL injection patterns are rejected
func TestValidateSessionName_SQLInjection(t *testing.T) {
	sqlPatterns := []string{
		"'; DROP TABLE sessions--",
		"1' OR '1'='1",
		"admin'--",
		"' UNION SELECT * FROM users--",
	}

	for _, pattern := range sqlPatterns {
		t.Run(pattern, func(t *testing.T) {
			err := ValidateSessionName(pattern)
			if err == nil {
				t.Errorf("ValidateSessionName(%q) should reject SQL injection pattern", pattern)
			}
		})
	}
}

// TestValidateRelativePath_CommandInjection tests path validation behavior with shell metacharacters
func TestValidateRelativePath_CommandInjection(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		reason  string
		wantErr bool
	}{
		// Note: Characters like ;, &&, | are valid in Unix filenames
		// Command injection protection happens at the shell execution layer, not path validation
		{
			name:    "semicolon in filename",
			path:    "file;name.txt",
			wantErr: false,
			reason:  "semicolons are valid filename characters on Unix",
		},
		{
			name:    "ampersand in filename",
			path:    "file&&name.txt",
			wantErr: false,
			reason:  "ampersands are valid filename characters on Unix",
		},
		{
			name:    "pipe in filename",
			path:    "file|name.txt",
			wantErr: false,
			reason:  "pipes are valid filename characters on Unix",
		},
		{
			name:    "newline in filename",
			path:    "file\nname.txt",
			wantErr: true,
			reason:  "newlines are control characters and should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelativePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRelativePath(%q) error = %v, wantErr %v (%s)", tt.path, err, tt.wantErr, tt.reason)
			}
		})
	}
}
