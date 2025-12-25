// ABOUTME: Charm client wrapper for Toki
// ABOUTME: Handles KV store initialization, configuration, and lifecycle

package charm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
)

// Config holds Charm client configuration.
type Config struct {
	Server         string        `json:"server"`
	AutoSync       bool          `json:"auto_sync"`
	StaleThreshold time.Duration `json:"stale_threshold,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Server:         "charm.2389.dev",
		AutoSync:       true,
		StaleThreshold: 5 * time.Minute, // Default: sync if data is older than 5 minutes
	}
}

// ApplyEnv applies environment variable overrides to the config.
func (c *Config) ApplyEnv() {
	if host := os.Getenv("CHARM_HOST"); host != "" {
		c.Server = host
	}
}

// ConfigPath returns the path to the Charm config file.
func ConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "toki", "charm.json")
}

// LoadConfig loads configuration from disk, falling back to defaults.
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			cfg.ApplyEnv()
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.ApplyEnv()
	return cfg, nil
}

// SaveConfig saves configuration to disk.
func SaveConfig(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Client wraps the Charm KV store for Toki operations.
type Client struct {
	kv     *kv.KV
	config *Config
}

// NewClient creates a new Charm client with the given database name.
func NewClient(dbName string) (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Set CHARM_HOST for the underlying library
	if cfg.Server != "" {
		_ = os.Setenv("CHARM_HOST", cfg.Server)
	}

	db, err := kv.OpenWithDefaultsFallback(dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to open kv store: %w", err)
	}

	// Sync on startup to pull any remote changes (skip in read-only mode)
	if cfg.AutoSync && !db.IsReadOnly() {
		_ = db.Sync() // Best effort - don't fail startup on sync errors
	}

	return &Client{
		kv:     db,
		config: cfg,
	}, nil
}

// Close closes the underlying KV store.
func (c *Client) Close() error {
	if c.kv != nil {
		return c.kv.Close()
	}
	return nil
}

// Sync synchronizes local data with the Charm server.
func (c *Client) Sync() error {
	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot sync: database is locked by another process (MCP server?)")
	}
	return c.kv.Sync()
}

// LastSyncTime returns the last sync timestamp from the KV database.
func (c *Client) LastSyncTime() time.Time {
	return c.kv.LastSyncTime()
}

// IsStale checks if the local data is stale based on the configured threshold.
func (c *Client) IsStale() bool {
	return c.kv.IsStale(c.config.StaleThreshold)
}

// SyncIfStale syncs with the server if local data is stale.
func (c *Client) SyncIfStale() error {
	if c.kv.IsReadOnly() {
		return nil // Skip sync if read-only
	}
	if c.kv.IsStale(c.config.StaleThreshold) {
		return c.kv.Sync()
	}
	return nil
}

// syncIfEnabled syncs if auto-sync is enabled in config.
// Call this after write operations.
func (c *Client) syncIfEnabled() {
	if c.config != nil && c.config.AutoSync && !c.kv.IsReadOnly() {
		_ = c.kv.Sync() // Best effort, don't fail writes on sync errors
	}
}

// KV returns the underlying KV store for direct access.
func (c *Client) KV() *kv.KV {
	return c.kv
}

// Config returns the current configuration.
func (c *Client) Config() *Config {
	return c.config
}

// IsReadOnly returns true if the database is open in read-only mode.
// This happens when another process (like an MCP server) holds the lock.
func (c *Client) IsReadOnly() bool {
	return c.kv.IsReadOnly()
}

// ID returns the Charm user ID for the current account.
func (c *Client) ID() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", fmt.Errorf("failed to create charm client: %w", err)
	}
	return cc.ID()
}
