package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	SyncPaths    []string `toml:"sync_paths"`
	Provider     string   `toml:"provider"`      // e.g., "gdrive"
	SyncInterval string   `toml:"sync_interval"` // e.g., "1h", "30m"
	Versioning   bool     `toml:"versioning"`    // Enable versioning (default true)
	MaxVersions  int      `toml:"max_versions"`  // Number of versions to keep (default 3)
	Ignore       []string `toml:"ignore"`        // Patterns to ignore (e.g., [".git", "*.log"])
}

func LoadConfig() (*Config, error) {
	configPath := filepath.Join(".", "service.toml") // For testing, use local file

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s", configPath)
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}
