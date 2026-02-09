// ABOUTME: Toki configuration management with backend selection
// ABOUTME: Handles settings, preferences, and storage backend factory

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harper/suite/mdstore"
	"github.com/harper/toki/internal/storage"
)

// Config stores Toki configuration.
type Config struct {
	// Backend selects the storage backend: "sqlite" (default) or "markdown".
	Backend string `json:"backend,omitempty"`

	// DataDir is the root directory for data storage.
	// SQLite puts toki.db here. Markdown puts _projects.yaml and project folders here.
	// Supports ~ expansion for home directory. Defaults to ~/.local/share/toki.
	DataDir string `json:"data_dir,omitempty"`
}

// defaultDBFilename is the SQLite database filename used for existing-user detection.
const defaultDBFilename = "toki.db"

// GetBackend returns the configured backend, defaulting to "sqlite".
func (c *Config) GetBackend() string {
	if c.Backend == "" {
		return "sqlite"
	}
	return c.Backend
}

// GetDataDir returns the configured data directory with ~ expanded,
// defaulting to the standard XDG data directory.
func (c *Config) GetDataDir() string {
	if c.DataDir == "" {
		return defaultDataDir()
	}
	return ExpandPath(c.DataDir)
}

// defaultDataDir returns the default data directory for Toki.
func defaultDataDir() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "toki")
}

// defaultFirstRunConfig returns the appropriate default config for first-time runs.
// If an existing SQLite database is found, it preserves SQLite as the backend.
// Otherwise, it defaults to markdown for new users.
func defaultFirstRunConfig() *Config {
	dbPath := filepath.Join(defaultDataDir(), defaultDBFilename)
	_, err := os.Stat(dbPath)
	switch {
	case err == nil:
		return &Config{Backend: "sqlite"}
	case !os.IsNotExist(err):
		fmt.Fprintf(os.Stderr, "warning: could not check for existing database: %v\n", err)
	}
	return &Config{Backend: "markdown"}
}

// ExpandPath expands a leading ~ to the user's home directory.
func ExpandPath(path string) string {
	if path == "" {
		return ""
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// OpenStorage creates a Storage implementation based on the configured backend.
func (c *Config) OpenStorage() (storage.Storage, error) {
	backend := c.GetBackend()
	dataDir := c.GetDataDir()

	switch backend {
	case "sqlite":
		dbPath := filepath.Join(dataDir, "toki.db")
		return storage.NewSQLiteStorage(dbPath)
	case "markdown":
		return storage.NewMarkdownStore(dataDir)
	default:
		return nil, fmt.Errorf("unknown backend: %q", backend)
	}
}

// GetConfigPath returns the config file path.
func GetConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "toki", "config.json")
}

// Load reads config from disk.
func Load() (*Config, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path) // #nosec G304 - config path from XDG standard
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultFirstRunConfig()
			if saveErr := cfg.Save(); saveErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save default config: %v\n", saveErr)
			}
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes config to disk.
func (c *Config) Save() error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return mdstore.AtomicWrite(path, data)
}
