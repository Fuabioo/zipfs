package security

import (
	"archive/zip"
	"fmt"
)

// BombCheckResult contains the results of a zip bomb pre-scan.
type BombCheckResult struct {
	Reason                string
	TotalUncompressedSize uint64
	FileCount             int
	MaxCompressionRatio   float64
	IsSafe                bool
}

// Limits configures the zip bomb detection thresholds.
type Limits struct {
	MaxExtractedSize    uint64  // bytes, default 1GB
	MaxFileCount        int     // default 100000
	MaxCompressionRatio float64 // default 100.0
}

// DefaultLimits returns the default security limits from ADR-008.
func DefaultLimits() Limits {
	return Limits{
		MaxExtractedSize:    1 * 1024 * 1024 * 1024, // 1 GB
		MaxFileCount:        100000,
		MaxCompressionRatio: 100.0,
	}
}

// CheckZipBomb pre-scans a zip file's central directory for zip bomb indicators.
// Does NOT extract any content - only reads metadata.
//
// Returns an error if the file cannot be opened/read.
// Returns a BombCheckResult with IsSafe=false if any limit is exceeded.
func CheckZipBomb(zipPath string, limits Limits) (*BombCheckResult, error) {
	// Open the zip file for reading
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	return CheckZipBombFromReader(&r.Reader, limits), nil
}

// CheckZipBombFromReader scans an already-opened zip reader.
// Does NOT extract any content - only reads central directory metadata.
func CheckZipBombFromReader(r *zip.Reader, limits Limits) *BombCheckResult {
	result := &BombCheckResult{
		IsSafe: true,
	}

	var totalUncompressedSize uint64
	var maxCompressionRatio float64

	for _, f := range r.File {
		// Skip directories (they don't contribute to size)
		if f.FileInfo().IsDir() {
			continue
		}

		totalUncompressedSize += f.UncompressedSize64

		// Calculate compression ratio for this file
		// Handle zero compressed size to avoid division by zero
		if f.CompressedSize64 > 0 {
			ratio := float64(f.UncompressedSize64) / float64(f.CompressedSize64)
			if ratio > maxCompressionRatio {
				maxCompressionRatio = ratio
			}
		}
	}

	result.TotalUncompressedSize = totalUncompressedSize
	result.FileCount = len(r.File)
	result.MaxCompressionRatio = maxCompressionRatio

	// Check total uncompressed size limit
	if totalUncompressedSize > limits.MaxExtractedSize {
		result.IsSafe = false
		result.Reason = fmt.Sprintf(
			"total uncompressed size (%d bytes) exceeds limit (%d bytes)",
			totalUncompressedSize,
			limits.MaxExtractedSize,
		)
		return result
	}

	// Check file count limit
	if len(r.File) > limits.MaxFileCount {
		result.IsSafe = false
		result.Reason = fmt.Sprintf(
			"file count (%d) exceeds limit (%d)",
			len(r.File),
			limits.MaxFileCount,
		)
		return result
	}

	// Check compression ratio limit
	if maxCompressionRatio > limits.MaxCompressionRatio {
		result.IsSafe = false
		result.Reason = fmt.Sprintf(
			"compression ratio (%.2f:1) exceeds limit (%.2f:1)",
			maxCompressionRatio,
			limits.MaxCompressionRatio,
		)
		return result
	}

	return result
}
