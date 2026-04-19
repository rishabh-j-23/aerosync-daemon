package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"log"
)

type SyncPath struct {
	Path  string `toml:"path"`
	Label string `toml:"label"`
}

type Config struct {
	SyncPaths    []SyncPath `toml:"sync_paths"`
	Provider     string     `toml:"provider"`      // e.g., "gdrive"
	SyncInterval string     `toml:"sync_interval"` // e.g., "1h", "30m"
	Versioning   bool       `toml:"versioning"`    // Enable versioning (default true)
	MaxVersions  int        `toml:"max_versions"`  // Number of versions to keep (default 3)
	Ignore       []string   `toml:"ignore"`        // Patterns to ignore (e.g., [".git", "*.log"])
}

func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "aerosync")
	os.MkdirAll(path, 0755)
	return path
}

func GetLogsDir() string {
	path := filepath.Join(GetConfigDir(), "logs")
	os.MkdirAll(path, 0755)
	return path
}

func LoadConfig() (*Config, error) {
	configPath := filepath.Join(GetConfigDir(), "service.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try local if global doesn't exist (migration support)
		localPath := filepath.Join(".", "service.toml")
		if _, err := os.Stat(localPath); err == nil {
			log.Printf("Migrating local config to global: %s", configPath)
			data, _ := os.ReadFile(localPath)
			os.WriteFile(configPath, data, 0644)
		} else {
			// Generate default config
			log.Printf("Creating default config at %s", configPath)
			defaultCfg := &Config{
				SyncPaths:    []SyncPath{},
				Provider:     "gdrive",
				SyncInterval: "1h",
				Versioning:   true,
				MaxVersions:  3,
				Ignore:       []string{".git", "node_modules", "*.log", "tmp", "temp"},
			}
			if err := defaultCfg.Save(); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return defaultCfg, nil
		}
	}

	log.Printf("Using config file: %s", configPath)

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

func (c *Config) Save() error {
	configPath := filepath.Join(GetConfigDir(), "service.toml")
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	return nil
}

func (c *Config) GetSyncPath(label string) string {
	for _, p := range c.SyncPaths {
		if p.Label == label {
			return p.Path
		}
	}
	return ""
}
