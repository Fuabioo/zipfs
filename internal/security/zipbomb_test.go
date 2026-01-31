package security

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultLimits(t *testing.T) {
	limits := DefaultLimits()

	if limits.MaxExtractedSize != 1*1024*1024*1024 {
		t.Errorf("DefaultLimits().MaxExtractedSize = %d, want %d", limits.MaxExtractedSize, 1*1024*1024*1024)
	}
	if limits.MaxFileCount != 100000 {
		t.Errorf("DefaultLimits().MaxFileCount = %d, want %d", limits.MaxFileCount, 100000)
	}
	if limits.MaxCompressionRatio != 100.0 {
		t.Errorf("DefaultLimits().MaxCompressionRatio = %f, want %f", limits.MaxCompressionRatio, 100.0)
	}
}

// createTestZip creates a zip file with the specified characteristics
func createTestZip(t *testing.T, files []testFile) string {
	t.Helper()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create temp zip: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for _, tf := range files {
		// Set compression method
		header := &zip.FileHeader{
			Name:   tf.name,
			Method: tf.method,
		}

		// For stored (uncompressed) method, we need to set CRC32
		if tf.method == zip.Store {
			header.UncompressedSize64 = uint64(len(tf.content))
		}

		fw, err := w.CreateHeader(header)
		if err != nil {
			t.Fatalf("failed to create file in zip: %v", err)
		}

		_, err = fw.Write([]byte(tf.content))
		if err != nil {
			t.Fatalf("failed to write file content: %v", err)
		}
	}

	return zipPath
}

type testFile struct {
	name    string
	content string
	method  uint16
}

func TestCheckZipBomb(t *testing.T) {
	tests := []struct {
		name      string
		errSubstr string
		files     []testFile
		limits    Limits
		wantSafe  bool
	}{
		{
			name: "normal zip passes",
			files: []testFile{
				{name: "file1.txt", content: "hello world", method: zip.Deflate},
				{name: "file2.txt", content: "another file", method: zip.Deflate},
			},
			limits:   DefaultLimits(),
			wantSafe: true,
		},
		{
			name: "exceeds size limit",
			files: []testFile{
				{name: "large.txt", content: strings.Repeat("x", 1000), method: zip.Store},
			},
			limits: Limits{
				MaxExtractedSize:    500, // 500 bytes
				MaxFileCount:        10,
				MaxCompressionRatio: 100.0,
			},
			wantSafe:  false,
			errSubstr: "exceeds limit",
		},
		{
			name: "exceeds file count",
			files: func() []testFile {
				files := make([]testFile, 15)
				for i := 0; i < 15; i++ {
					files[i] = testFile{
						name:    filepath.Join("file", string(rune(i))+"txt"),
						content: "small",
						method:  zip.Deflate,
					}
				}
				return files
			}(),
			limits: Limits{
				MaxExtractedSize:    1024 * 1024,
				MaxFileCount:        10,
				MaxCompressionRatio: 100.0,
			},
			wantSafe:  false,
			errSubstr: "file count",
		},
		{
			name: "high compression ratio",
			files: []testFile{
				// Create highly compressible content (same character repeated)
				{name: "compressed.txt", content: strings.Repeat("a", 10000), method: zip.Deflate},
			},
			limits: Limits{
				MaxExtractedSize:    1024 * 1024,
				MaxFileCount:        1000,
				MaxCompressionRatio: 10.0, // Low ratio threshold
			},
			wantSafe:  false,
			errSubstr: "compression ratio",
		},
		{
			name: "empty zip",
			files: []testFile{
				{name: "dir/", content: "", method: zip.Store},
			},
			limits:   DefaultLimits(),
			wantSafe: true,
		},
		{
			name: "mix of compressed and stored",
			files: []testFile{
				{name: "file1.txt", content: "compressed content", method: zip.Deflate},
				{name: "file2.txt", content: "stored content", method: zip.Store},
			},
			limits:   DefaultLimits(),
			wantSafe: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipPath := createTestZip(t, tt.files)

			result, err := CheckZipBomb(zipPath, tt.limits)
			if err != nil {
				t.Fatalf("CheckZipBomb() unexpected error: %v", err)
			}

			if result.IsSafe != tt.wantSafe {
				t.Errorf("CheckZipBomb() IsSafe = %v, want %v (reason: %s)", result.IsSafe, tt.wantSafe, result.Reason)
			}

			if !tt.wantSafe && tt.errSubstr != "" {
				if !strings.Contains(result.Reason, tt.errSubstr) {
					t.Errorf("CheckZipBomb() reason = %q, want substring %q", result.Reason, tt.errSubstr)
				}
			}
		})
	}
}

func TestCheckZipBomb_InvalidFile(t *testing.T) {
	tests := []struct {
		name    string
		zipPath string
		wantErr bool
	}{
		{
			name:    "nonexistent file",
			zipPath: "/tmp/nonexistent-zip-file.zip",
			wantErr: true,
		},
		{
			name:    "empty path",
			zipPath: "",
			wantErr: true,
		},
	}

	limits := DefaultLimits()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CheckZipBomb(tt.zipPath, limits)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckZipBomb() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckZipBomb_CorruptedZip(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "corrupted.zip")

	// Create a file that's not a valid zip
	err := os.WriteFile(zipPath, []byte("not a zip file"), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	limits := DefaultLimits()
	_, err = CheckZipBomb(zipPath, limits)
	if err == nil {
		t.Error("CheckZipBomb() expected error for corrupted zip, got nil")
	}
}

func TestCheckZipBombFromReader(t *testing.T) {
	// Create a test zip in memory
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add test files
	files := []struct {
		name    string
		content string
	}{
		{"file1.txt", "hello"},
		{"file2.txt", "world"},
	}

	for _, f := range files {
		fw, err := w.Create(f.name)
		if err != nil {
			t.Fatalf("failed to create file in zip: %v", err)
		}
		_, err = fw.Write([]byte(f.content))
		if err != nil {
			t.Fatalf("failed to write file content: %v", err)
		}
	}

	err := w.Close()
	if err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	// Read the zip
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to open zip reader: %v", err)
	}

	// Test the reader function
	limits := DefaultLimits()
	result := CheckZipBombFromReader(r, limits)

	if !result.IsSafe {
		t.Errorf("CheckZipBombFromReader() IsSafe = false, want true (reason: %s)", result.Reason)
	}

	if result.FileCount != len(files) {
		t.Errorf("CheckZipBombFromReader() FileCount = %d, want %d", result.FileCount, len(files))
	}
}

func TestCheckZipBombFromReader_ExceedsLimits(t *testing.T) {
	tests := []struct {
		name      string
		fileCount int
		limits    Limits
		wantSafe  bool
	}{
		{
			name:      "within limits",
			fileCount: 5,
			limits: Limits{
				MaxExtractedSize:    1024,
				MaxFileCount:        10,
				MaxCompressionRatio: 100.0,
			},
			wantSafe: true,
		},
		{
			name:      "exceeds file count",
			fileCount: 15,
			limits: Limits{
				MaxExtractedSize:    1024 * 1024,
				MaxFileCount:        10,
				MaxCompressionRatio: 100.0,
			},
			wantSafe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create zip in memory
			var buf bytes.Buffer
			w := zip.NewWriter(&buf)

			for i := 0; i < tt.fileCount; i++ {
				fw, err := w.Create(filepath.Join("file", string(rune(i))+".txt"))
				if err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				_, err = fw.Write([]byte("content"))
				if err != nil {
					t.Fatalf("failed to write content: %v", err)
				}
			}

			err := w.Close()
			if err != nil {
				t.Fatalf("failed to close writer: %v", err)
			}

			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			if err != nil {
				t.Fatalf("failed to create reader: %v", err)
			}

			result := CheckZipBombFromReader(r, tt.limits)
			if result.IsSafe != tt.wantSafe {
				t.Errorf("CheckZipBombFromReader() IsSafe = %v, want %v", result.IsSafe, tt.wantSafe)
			}
		})
	}
}

func TestBombCheckResult_Fields(t *testing.T) {
	// Test that all fields are properly populated
	files := []testFile{
		{name: "file.txt", content: strings.Repeat("a", 1000), method: zip.Deflate},
	}

	zipPath := createTestZip(t, files)
	limits := DefaultLimits()

	result, err := CheckZipBomb(zipPath, limits)
	if err != nil {
		t.Fatalf("CheckZipBomb() error: %v", err)
	}

	if result.TotalUncompressedSize == 0 {
		t.Error("TotalUncompressedSize should not be zero")
	}

	if result.FileCount == 0 {
		t.Error("FileCount should not be zero")
	}

	if result.MaxCompressionRatio == 0 {
		t.Error("MaxCompressionRatio should not be zero for compressed file")
	}
}

// TestCheckZipBomb_DirectoryHandling verifies directories don't count toward size limits
func TestCheckZipBomb_DirectoryHandling(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add directories
	for i := 0; i < 5; i++ {
		_, err := w.Create(filepath.Join("dir", string(rune(i)), "") + "/")
		if err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	// Add one small file
	fw, err := w.Create("file.txt")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	_, err = io.WriteString(fw, "content")
	if err != nil {
		t.Fatalf("failed to write content: %v", err)
	}

	err = w.Close()
	if err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}

	result := CheckZipBombFromReader(r, DefaultLimits())

	// Verify directories didn't inflate the uncompressed size significantly
	if result.TotalUncompressedSize > 100 {
		t.Errorf("TotalUncompressedSize = %d, directories should not count toward size", result.TotalUncompressedSize)
	}
}

// BenchmarkCheckZipBomb measures performance of bomb detection
func BenchmarkCheckZipBomb(b *testing.B) {
	// Create a realistic test zip
	files := []testFile{
		{name: "file1.txt", content: strings.Repeat("test content\n", 100), method: zip.Deflate},
		{name: "file2.txt", content: strings.Repeat("more content\n", 100), method: zip.Deflate},
		{name: "file3.txt", content: strings.Repeat("even more\n", 100), method: zip.Deflate},
	}

	// Create temp file for benchmark
	tmpDir := b.TempDir()
	zipPath := filepath.Join(tmpDir, "bench.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		b.Fatalf("failed to create zip: %v", err)
	}

	w := zip.NewWriter(f)
	for _, tf := range files {
		fw, err := w.Create(tf.name)
		if err != nil {
			b.Fatalf("failed to create file: %v", err)
		}
		_, err = fw.Write([]byte(tf.content))
		if err != nil {
			b.Fatalf("failed to write content: %v", err)
		}
	}
	w.Close()
	f.Close()

	limits := DefaultLimits()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CheckZipBomb(zipPath, limits)
	}
}
