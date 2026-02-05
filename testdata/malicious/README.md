# Malicious Test Fixtures

## Overview

This directory is for documenting the test strategy for malicious zip files used in the `internal/security` package tests.

## Strategy: Programmatic Generation

**We do NOT store actual malicious zip files in this repository.** Instead, all malicious test cases are generated programmatically within the test code itself.

### Rationale

1. **Security**: Storing actual malicious files (even as test fixtures) can trigger security scanners, antivirus software, and GitHub's security scanning
2. **Transparency**: Generating test cases in code makes it clear exactly what is being tested
3. **Flexibility**: Easy to modify test cases without managing binary files
4. **Version Control**: Code is easier to review and diff than binary zip files
5. **Reproducibility**: Anyone can run the tests without downloading potentially dangerous files

## Test Coverage

The security tests programmatically generate zip files to test:

### Zip Bomb Detection (`zipbomb_test.go`)

- **Normal zips**: Small files with realistic compression
- **Size bombs**: Files exceeding `MaxExtractedSize` limit
- **File count bombs**: Zips exceeding `MaxFileCount` limit
- **Compression ratio bombs**: Highly compressible content (repeated characters) exceeding `MaxCompressionRatio`
- **Edge cases**: Empty zips, directories, mixed compression methods

Generated using Go's `archive/zip` package:
```go
func createTestZip(t *testing.T, files []testFile) string {
    // Creates zip in temporary directory
    // Returns path to generated zip file
}
```

### Zip Slip Detection (`zipslip_test.go`)

Path traversal attacks are tested with string patterns, not actual zip files:

- `../etc/passwd` - Simple parent traversal
- `../../../../../../etc/passwd` - Deep traversal
- `dir/../../../etc/passwd` - Traversal after valid prefix
- `/etc/passwd` - Absolute paths
- `file\x00.txt` - Null bytes
- Windows-specific patterns (`C:\\Windows\\System32`, `..\\..\\windows`)

### Input Sanitization (`sanitize_test.go`)

String-based tests for:
- Session name validation
- Relative path validation
- Glob pattern validation
- SQL injection patterns
- Command injection patterns

## Running Tests

All tests use temporary directories created with `t.TempDir()` which are automatically cleaned up:

```bash
# Run all security tests
go test ./internal/security/ -v

# Run with race detection
go test ./internal/security/ -race -count=1 -v

# Run specific test
go test ./internal/security/ -run TestCheckZipBomb -v

# Run benchmarks
go test ./internal/security/ -bench=. -benchmem
```

## Adding New Malicious Test Cases

To add a new test case:

1. **Do NOT** create an actual malicious zip file
2. **Do** add a test function that generates the malicious pattern programmatically
3. Document the attack vector in the test name and comments
4. Ensure the test verifies the security control properly rejects the malicious input

Example:
```go
func TestCheckZipBomb_NewAttackVector(t *testing.T) {
    // Create malicious zip programmatically
    files := []testFile{
        {name: "bomb.txt", content: createMaliciousContent(), method: zip.Deflate},
    }
    zipPath := createTestZip(t, files)

    // Verify it's rejected
    result, err := CheckZipBomb(zipPath, DefaultLimits())
    if result.IsSafe {
        t.Error("Expected malicious zip to be rejected")
    }
}
```

## References

- [ADR-008: Security Model](/docs/ADR/008-security.md)
- [Zip Slip Vulnerability](https://security.snyk.io/research/zip-slip-vulnerability)
- [Zip Bomb Detection](https://en.wikipedia.org/wiki/Zip_bomb)
