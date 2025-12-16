// ABOUTME: Tests for sync configuration management
// ABOUTME: Validates config loading, saving, environment overrides, and error handling

package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	assert.Contains(t, path, "sync.json")
	assert.Contains(t, path, "toki")
}

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	assert.Contains(t, dir, "toki")
	assert.Equal(t, dir, filepath.Dir(ConfigPath()))
}

func TestEnsureConfigDir(t *testing.T) {
	// Use temp dir for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "toki", "sync.json")
	configDir := filepath.Dir(configPath)

	// Test creating directory
	err := os.MkdirAll(configDir, 0o750)
	require.NoError(t, err)

	// Verify directory was created with correct permissions
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0o750), info.Mode().Perm())
}

func TestLoadConfig_NoFile(t *testing.T) {
	// Create temp directory for config
	tmpDir := t.TempDir()

	// Set environment to use temp directory
	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should return default config
	assert.Contains(t, cfg.VaultDB, "vault.db")
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "toki")
	configPath := filepath.Join(configDir, "sync.json")

	err := os.MkdirAll(configDir, 0o750)
	require.NoError(t, err)

	testCfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "user123",
		Token:      "token123",
		DerivedKey: "key123",
		DeviceID:   "device123",
		VaultDB:    filepath.Join(configDir, "vault.db"),
		AutoSync:   true,
	}

	data, err := json.MarshalIndent(testCfg, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(configPath, data, 0o600)
	require.NoError(t, err)

	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, testCfg.Server, cfg.Server)
	assert.Equal(t, testCfg.UserID, cfg.UserID)
	assert.Equal(t, testCfg.Token, cfg.Token)
	assert.Equal(t, testCfg.DerivedKey, cfg.DerivedKey)
	assert.Equal(t, testCfg.DeviceID, cfg.DeviceID)
	assert.Equal(t, testCfg.AutoSync, cfg.AutoSync)
}

func TestLoadConfig_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "toki")
	configPath := filepath.Join(configDir, "sync.json")

	err := os.MkdirAll(configDir, 0o750)
	require.NoError(t, err)

	// Write invalid JSON
	err = os.WriteFile(configPath, []byte("{invalid json}"), 0o600)
	require.NoError(t, err)

	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "corrupted")

	// Check that backup was created
	backupFiles, err := filepath.Glob(configPath + ".corrupt.*")
	require.NoError(t, err)
	assert.NotEmpty(t, backupFiles)
}

func TestLoadConfig_DirectoryInsteadOfFile(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "toki")
	configPath := filepath.Join(configDir, "sync.json")

	// Create directory where file should be
	err := os.MkdirAll(configPath, 0o750)
	require.NoError(t, err)

	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "directory")
}

func TestApplyEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("HOME", tmpDir)
	t.Setenv("TOKI_SERVER", "https://env.example.com")
	t.Setenv("TOKI_TOKEN", "env-token")
	t.Setenv("TOKI_AUTO_SYNC", "true")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "https://env.example.com", cfg.Server)
	assert.Equal(t, "env-token", cfg.Token)
	assert.True(t, cfg.AutoSync)
}

func TestApplyEnvOverrides_AutoSyncVariants(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true", "true", true},
		{"1", "1", true},
		{"false", "false", false},
		{"0", "0", false},
		{"empty", "", true}, // default is now true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			if tt.envValue != "" {
				t.Setenv("TOKI_AUTO_SYNC", tt.envValue)
			}

			cfg, err := LoadConfig()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.AutoSync)
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".config", "toki", "sync.json")

	t.Setenv("HOME", tmpDir)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "user123",
		Token:      "token123",
		DerivedKey: "key123",
		DeviceID:   "device123",
		AutoSync:   true,
	}

	err := SaveConfig(cfg)
	require.NoError(t, err)

	// Verify file was created with correct permissions
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	// Verify content
	data, err := os.ReadFile(configPath) //nolint:gosec // test file path is safe
	require.NoError(t, err)

	var loaded Config
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, cfg.Server, loaded.Server)
	assert.Equal(t, cfg.UserID, loaded.UserID)
	assert.Equal(t, cfg.Token, loaded.Token)
	assert.Equal(t, cfg.DerivedKey, loaded.DerivedKey)
	assert.Equal(t, cfg.DeviceID, loaded.DeviceID)
	assert.Equal(t, cfg.AutoSync, loaded.AutoSync)
}

func TestInitConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := InitConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify device ID is a valid ULID
	assert.Len(t, cfg.DeviceID, 26) // ULID length

	// Verify VaultDB path is set
	assert.Contains(t, cfg.VaultDB, "vault.db")

	// Verify config file was created
	assert.True(t, ConfigExists())
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Should not exist initially
	assert.False(t, ConfigExists())

	// Create config
	_, err := InitConfig()
	require.NoError(t, err)

	// Should exist now
	assert.True(t, ConfigExists())
}

func TestIsConfigured(t *testing.T) { //nolint:funlen // test table is necessarily long
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "fully configured",
			config: &Config{
				Server:     "https://test.example.com",
				Token:      "token123",
				UserID:     "user123",
				DerivedKey: "key123",
			},
			expected: true,
		},
		{
			name: "missing server",
			config: &Config{
				Token:      "token123",
				UserID:     "user123",
				DerivedKey: "key123",
			},
			expected: false,
		},
		{
			name: "missing token",
			config: &Config{
				Server:     "https://test.example.com",
				UserID:     "user123",
				DerivedKey: "key123",
			},
			expected: false,
		},
		{
			name: "missing user ID",
			config: &Config{
				Server:     "https://test.example.com",
				Token:      "token123",
				DerivedKey: "key123",
			},
			expected: false,
		},
		{
			name: "missing derived key",
			config: &Config{
				Server: "https://test.example.com",
				Token:  "token123",
				UserID: "user123",
			},
			expected: false,
		},
		{
			name:     "empty config",
			config:   &Config{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsConfigured())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	// Default server should be set
	expectedServer := "https://api.storeusa.org"
	assert.Equal(t, expectedServer, cfg.Server)

	// VaultDB should be set to default location
	assert.Contains(t, cfg.VaultDB, "vault.db")

	// AutoSync should be enabled by default
	assert.True(t, cfg.AutoSync)
}
