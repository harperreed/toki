// ABOUTME: Tests for Charm client wrapper
// ABOUTME: Verifies client creation, configuration, and basic operations

package charm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Use temp directory for test data
	tmpDir, err := os.MkdirTemp("", "toki-charm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Set CHARM_DATA_DIR to isolate test
	_ = os.Setenv("CHARM_DATA_DIR", tmpDir)
	defer func() { _ = os.Unsetenv("CHARM_DATA_DIR") }()

	client, err := NewClient("toki-test")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	if client.kv == nil {
		t.Error("kv should not be nil")
	}
}

func TestClientConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server != "charm.2389.dev" {
		t.Errorf("default server should be charm.2389.dev, got %s", cfg.Server)
	}
	if !cfg.AutoSync {
		t.Error("AutoSync should be true by default")
	}
}

func TestConfigFromEnv(t *testing.T) {
	_ = os.Setenv("CHARM_HOST", "custom.server.com")
	defer func() { _ = os.Unsetenv("CHARM_HOST") }()

	cfg := DefaultConfig()
	cfg.ApplyEnv()

	if cfg.Server != "custom.server.com" {
		t.Errorf("server should be custom.server.com, got %s", cfg.Server)
	}
}

func TestConfigPath(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	tmpDir, _ := os.MkdirTemp("", "toki-config-test-*")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	path := ConfigPath()
	expected := filepath.Join(tmpDir, "toki", "charm.json")
	if path != expected {
		t.Errorf("ConfigPath() = %s, want %s", path, expected)
	}
}
