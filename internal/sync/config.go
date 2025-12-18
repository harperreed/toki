// ABOUTME: Sync configuration management for vault integration
// ABOUTME: Handles loading, saving, and environment overrides for sync settings

package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// Config represents the sync configuration.
type Config struct {
	Server       string `json:"server"`
	UserID       string `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenExpires string `json:"token_expires,omitempty"`
	DerivedKey   string `json:"derived_key"`
	DeviceID     string `json:"device_id"`
	VaultDB      string `json:"vault_db"`
}

// ConfigPath returns the path to the sync config file.
func ConfigPath() string {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "toki", "sync.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".toki", "sync.json")
	}
	return filepath.Join(home, ".config", "toki", "sync.json")
}

// ConfigDir returns the directory containing the config file.
func ConfigDir() string {
	return filepath.Dir(ConfigPath())
}

// EnsureConfigDir creates the config directory if it doesn't exist.
//
//nolint:nestif // Complex nested blocks needed to handle various filesystem states.
func EnsureConfigDir() error {
	dir := ConfigDir()
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			backup := dir + ".backup." + time.Now().Format("20060102-150405")
			if err := os.Rename(dir, backup); err != nil {
				return fmt.Errorf("config path %s is a file, failed to backup: %w", dir, err)
			}
		} else {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check config dir: %w", err)
	}
	return os.MkdirAll(dir, 0o750)
}

// LoadConfig loads config from file and applies environment variable overrides.
func LoadConfig() (*Config, error) {
	cfg := defaultConfig()

	configPath := ConfigPath()

	info, statErr := os.Stat(configPath)
	if statErr == nil && info.IsDir() {
		return nil, fmt.Errorf("config path %s is a directory, not a file", configPath)
	}

	//#nosec G304 -- configPath is derived from user's home directory
	data, err := os.ReadFile(configPath)
	if err == nil {
		if jsonErr := json.Unmarshal(data, cfg); jsonErr != nil {
			backup := configPath + ".corrupt." + time.Now().Format("20060102-150405")
			if renameErr := os.Rename(configPath, backup); renameErr == nil {
				fmt.Fprintf(os.Stderr, "Warning: corrupted config backed up to %s\n", backup)
			}
			return nil, fmt.Errorf("config file corrupted: %w", jsonErr)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config: %w", err)
	}

	applyEnvOverrides(cfg)

	if cfg.VaultDB == "" {
		cfg.VaultDB = filepath.Join(ConfigDir(), "vault.db")
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server:  "https://api.storeusa.org",
		VaultDB: filepath.Join(ConfigDir(), "vault.db"),
	}
}

func applyEnvOverrides(cfg *Config) {
	if server := os.Getenv("TOKI_SERVER"); server != "" {
		cfg.Server = server
	}
	if token := os.Getenv("TOKI_TOKEN"); token != "" {
		cfg.Token = token
	}
	if userID := os.Getenv("TOKI_USER_ID"); userID != "" {
		cfg.UserID = userID
	}
	if vaultDB := os.Getenv("TOKI_VAULT_DB"); vaultDB != "" {
		cfg.VaultDB = expandPath(vaultDB)
	}
	if deviceID := os.Getenv("TOKI_DEVICE_ID"); deviceID != "" {
		cfg.DeviceID = deviceID
	}
}

// SaveConfig writes config to file.
func SaveConfig(cfg *Config) error {
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(), data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// InitConfig creates a new config with device ID.
func InitConfig() (*Config, error) {
	deviceID := ulid.Make().String()

	cfg := &Config{
		Server:   "https://api.storeusa.org",
		DeviceID: deviceID,
		VaultDB:  filepath.Join(ConfigDir(), "vault.db"),
	}

	if err := SaveConfig(cfg); err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "Config created at %s\n", ConfigPath())
	fmt.Fprintf(os.Stderr, "Device ID: %s\n", deviceID)

	return cfg, nil
}

// ConfigExists returns true if config file exists.
func ConfigExists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

// IsConfigured returns true if sync is fully configured.
func (c *Config) IsConfigured() bool {
	return c.Server != "" && c.Token != "" && c.UserID != "" && c.DerivedKey != ""
}

func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
