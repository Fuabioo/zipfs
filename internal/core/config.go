package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Fuabioo/zipfs/internal/security"
)

// Config holds global configuration for zipfs.
type Config struct {
	Security SecurityConfig `json:"security"`
	Defaults DefaultsConfig `json:"defaults"`
}

// SecurityConfig holds security limits and constraints.
type SecurityConfig struct {
	MaxExtractedSizeBytes uint64  `json:"max_extracted_size_bytes"`
	MaxFileCount          int     `json:"max_file_count"`
	MaxCompressionRatio   float64 `json:"max_compression_ratio"`
	MaxTotalDiskBytes     uint64  `json:"max_total_disk_bytes"`
	MaxSessions           int     `json:"max_sessions"`
	AllowSymlinks         bool    `json:"allow_symlinks"`
	RegexTimeoutMS        int     `json:"regex_timeout_ms"`
}

// DefaultsConfig holds default values for operations.
type DefaultsConfig struct {
	BackupRotationDepth int `json:"backup_rotation_depth"`
}

// DefaultConfig returns the default configuration as specified in ADR-002.
func DefaultConfig() *Config {
	return &Config{
		Security: SecurityConfig{
			MaxExtractedSizeBytes: 1 * 1024 * 1024 * 1024, // 1GB
			MaxFileCount:          100000,
			MaxCompressionRatio:   100.0,
			MaxTotalDiskBytes:     10 * 1024 * 1024 * 1024, // 10GB
			MaxSessions:           32,
			AllowSymlinks:         false,
			RegexTimeoutMS:        5000,
		},
		Defaults: DefaultsConfig{
			BackupRotationDepth: 3,
		},
	}
}

// LoadConfig loads configuration from config.json in the data directory.
// Falls back to default configuration if config.json doesn't exist.
// Environment variables override both file and default values.
func LoadConfig(dataDir string) (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config.json
	configPath := filepath.Join(dataDir, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config.json: %w", err)
	}
	// If file doesn't exist, we continue with defaults

	// Apply environment variable overrides
	if err := applyEnvOverrides(cfg); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the config.
func applyEnvOverrides(cfg *Config) error {
	if val, ok := os.LookupEnv("ZIPFS_MAX_EXTRACTED_SIZE"); ok {
		parsed, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ZIPFS_MAX_EXTRACTED_SIZE: %w", err)
		}
		cfg.Security.MaxExtractedSizeBytes = parsed
	}

	if val, ok := os.LookupEnv("ZIPFS_MAX_SESSIONS"); ok {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid ZIPFS_MAX_SESSIONS: %w", err)
		}
		cfg.Security.MaxSessions = parsed
	}

	if val, ok := os.LookupEnv("ZIPFS_MAX_FILE_COUNT"); ok {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid ZIPFS_MAX_FILE_COUNT: %w", err)
		}
		cfg.Security.MaxFileCount = parsed
	}

	return nil
}

// ToSecurityLimits converts the config to security.Limits for use with security package.
func (c *Config) ToSecurityLimits() security.Limits {
	return security.Limits{
		MaxExtractedSize:    c.Security.MaxExtractedSizeBytes,
		MaxFileCount:        c.Security.MaxFileCount,
		MaxCompressionRatio: c.Security.MaxCompressionRatio,
	}
}
