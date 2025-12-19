// ABOUTME: Tests for Charm client wrapper
// ABOUTME: Verifies client creation, configuration, and basic operations

package charm

import (
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Use temp directory for test data (automatically cleaned up)
	tmpDir := t.TempDir()

	// Set CHARM_DATA_DIR to isolate test (test-scoped, avoids race conditions)
	t.Setenv("CHARM_DATA_DIR", tmpDir)

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
	t.Setenv("CHARM_HOST", "custom.server.com")

	cfg := DefaultConfig()
	cfg.ApplyEnv()

	if cfg.Server != "custom.server.com" {
		t.Errorf("server should be custom.server.com, got %s", cfg.Server)
	}
}

func TestConfigPath(t *testing.T) {
	// Test with XDG_CONFIG_HOME set (test-scoped, avoids race conditions)
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path := ConfigPath()
	expected := filepath.Join(tmpDir, "toki", "charm.json")
	if path != expected {
		t.Errorf("ConfigPath() = %s, want %s", path, expected)
	}
}
