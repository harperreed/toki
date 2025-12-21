// ABOUTME: Tests for Charm client wrapper
// ABOUTME: Verifies client creation, configuration, and basic operations

package charm

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Skip in CI - charm cloud connectivity required
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("skipping charm test in CI - no charm cloud connectivity")
	}

	// Use temp directory for test data (automatically cleaned up)
	tmpDir := t.TempDir()

	// Set CHARM_DATA_DIR to isolate test (test-scoped, avoids race conditions)
	t.Setenv("CHARM_DATA_DIR", tmpDir)

	client, err := NewClient("toki-test")
	if err != nil {
		t.Skipf("skipping test - charm KV not available: %v", err)
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

func TestWALConcurrentConnections(t *testing.T) {
	// Skip in CI - charm cloud connectivity required
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("skipping charm test in CI - no charm cloud connectivity")
	}

	// Test that multiple clients can open the same database concurrently.
	// This verifies the WAL mode fix prevents SQLITE_BUSY errors.
	tmpDir := t.TempDir()
	t.Setenv("CHARM_DATA_DIR", tmpDir)

	// First, initialize the database and Charm keys with a single client.
	// This avoids race conditions on key generation.
	initClient, err := NewClient("toki-wal-test")
	if err != nil {
		t.Skipf("skipping WAL test - charm KV not available: %v", err)
	}
	_ = initClient.Close()

	const numClients = 3
	const writesPerClient = 5

	var wg sync.WaitGroup
	errors := make(chan error, numClients*writesPerClient)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Each goroutine opens its own client (simulates separate processes)
			client, err := NewClient("toki-wal-test")
			if err != nil {
				errors <- err
				return
			}
			defer func() { _ = client.Close() }()

			// Perform writes
			for j := 0; j < writesPerClient; j++ {
				project := &Project{
					Name: "test-project",
				}
				if err := client.CreateProject(project); err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Collect any errors
	errs := make([]error, 0, numClients*writesPerClient)
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Errorf("concurrent connections produced %d errors, first: %v", len(errs), errs[0])
	}
}
