package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Security.MaxExtractedSizeBytes != 1*1024*1024*1024 {
		t.Errorf("expected max extracted size 1GB, got %d", cfg.Security.MaxExtractedSizeBytes)
	}

	if cfg.Security.MaxFileCount != 100000 {
		t.Errorf("expected max file count 100000, got %d", cfg.Security.MaxFileCount)
	}

	if cfg.Security.MaxCompressionRatio != 100.0 {
		t.Errorf("expected max compression ratio 100.0, got %f", cfg.Security.MaxCompressionRatio)
	}

	if cfg.Security.MaxSessions != 32 {
		t.Errorf("expected max sessions 32, got %d", cfg.Security.MaxSessions)
	}

	if cfg.Defaults.BackupRotationDepth != 3 {
		t.Errorf("expected backup rotation depth 3, got %d", cfg.Defaults.BackupRotationDepth)
	}
}

func TestLoadConfig_DefaultsWhenFileDoesntExist(t *testing.T) {
	tempDir := t.TempDir()

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return defaults when file doesn't exist
	if cfg.Security.MaxExtractedSizeBytes != 1*1024*1024*1024 {
		t.Errorf("expected default max extracted size, got %d", cfg.Security.MaxExtractedSizeBytes)
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a config file with custom values
	customConfig := &Config{
		Security: SecurityConfig{
			MaxExtractedSizeBytes: 2 * 1024 * 1024 * 1024,
			MaxFileCount:          200000,
			MaxCompressionRatio:   200.0,
			MaxTotalDiskBytes:     20 * 1024 * 1024 * 1024,
			MaxSessions:           64,
			AllowSymlinks:         true,
			RegexTimeoutMS:        10000,
		},
		Defaults: DefaultsConfig{
			BackupRotationDepth: 5,
		},
	}

	configData, err := json.MarshalIndent(customConfig, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load the config
	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Security.MaxExtractedSizeBytes != 2*1024*1024*1024 {
		t.Errorf("expected max extracted size 2GB, got %d", cfg.Security.MaxExtractedSizeBytes)
	}

	if cfg.Security.MaxFileCount != 200000 {
		t.Errorf("expected max file count 200000, got %d", cfg.Security.MaxFileCount)
	}

	if cfg.Defaults.BackupRotationDepth != 5 {
		t.Errorf("expected backup rotation depth 5, got %d", cfg.Defaults.BackupRotationDepth)
	}
}

func TestLoadConfig_EnvVarOverrides(t *testing.T) {
	tempDir := t.TempDir()

	// Set environment variables
	os.Setenv("ZIPFS_MAX_EXTRACTED_SIZE", "3221225472") // 3GB
	defer os.Unsetenv("ZIPFS_MAX_EXTRACTED_SIZE")

	os.Setenv("ZIPFS_MAX_SESSIONS", "128")
	defer os.Unsetenv("ZIPFS_MAX_SESSIONS")

	os.Setenv("ZIPFS_MAX_FILE_COUNT", "500000")
	defer os.Unsetenv("ZIPFS_MAX_FILE_COUNT")

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Security.MaxExtractedSizeBytes != 3221225472 {
		t.Errorf("expected max extracted size 3GB, got %d", cfg.Security.MaxExtractedSizeBytes)
	}

	if cfg.Security.MaxSessions != 128 {
		t.Errorf("expected max sessions 128, got %d", cfg.Security.MaxSessions)
	}

	if cfg.Security.MaxFileCount != 500000 {
		t.Errorf("expected max file count 500000, got %d", cfg.Security.MaxFileCount)
	}
}

func TestLoadConfig_InvalidEnvVar(t *testing.T) {
	tempDir := t.TempDir()

	os.Setenv("ZIPFS_MAX_EXTRACTED_SIZE", "not-a-number")
	defer os.Unsetenv("ZIPFS_MAX_EXTRACTED_SIZE")

	_, err := LoadConfig(tempDir)
	if err == nil {
		t.Fatal("expected error for invalid env var, got nil")
	}
}

func TestLoadConfig_InvalidMaxSessions(t *testing.T) {
	tempDir := t.TempDir()

	os.Setenv("ZIPFS_MAX_SESSIONS", "not-a-number")
	defer os.Unsetenv("ZIPFS_MAX_SESSIONS")

	_, err := LoadConfig(tempDir)
	if err == nil {
		t.Fatal("expected error for invalid ZIPFS_MAX_SESSIONS")
	}
}

func TestLoadConfig_InvalidMaxFileCount(t *testing.T) {
	tempDir := t.TempDir()

	os.Setenv("ZIPFS_MAX_FILE_COUNT", "not-a-number")
	defer os.Unsetenv("ZIPFS_MAX_FILE_COUNT")

	_, err := LoadConfig(tempDir)
	if err == nil {
		t.Fatal("expected error for invalid ZIPFS_MAX_FILE_COUNT")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Write invalid JSON to config file
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte("{invalid json}"), 0600); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, err := LoadConfig(tempDir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadConfig_ReadError(t *testing.T) {
	tempDir := t.TempDir()

	// Create config.json as a directory (will cause read error)
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	_, err := LoadConfig(tempDir)
	if err == nil {
		t.Fatal("expected error when config.json is a directory")
	}
}

func TestToSecurityLimits(t *testing.T) {
	cfg := DefaultConfig()
	limits := cfg.ToSecurityLimits()

	if limits.MaxExtractedSize != cfg.Security.MaxExtractedSizeBytes {
		t.Errorf("expected max extracted size %d, got %d", cfg.Security.MaxExtractedSizeBytes, limits.MaxExtractedSize)
	}

	if limits.MaxFileCount != cfg.Security.MaxFileCount {
		t.Errorf("expected max file count %d, got %d", cfg.Security.MaxFileCount, limits.MaxFileCount)
	}

	if limits.MaxCompressionRatio != cfg.Security.MaxCompressionRatio {
		t.Errorf("expected max compression ratio %f, got %f", cfg.Security.MaxCompressionRatio, limits.MaxCompressionRatio)
	}
}
