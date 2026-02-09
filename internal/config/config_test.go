// ABOUTME: Tests for config package
// ABOUTME: Covers backend selection, data directory, config persistence, and path expansion

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetBackend(t *testing.T) {
	t.Run("default is sqlite", func(t *testing.T) {
		cfg := &Config{}
		if got := cfg.GetBackend(); got != "sqlite" {
			t.Errorf("expected sqlite, got %q", got)
		}
	})

	t.Run("explicit sqlite", func(t *testing.T) {
		cfg := &Config{Backend: "sqlite"}
		if got := cfg.GetBackend(); got != "sqlite" {
			t.Errorf("expected sqlite, got %q", got)
		}
	})

	t.Run("markdown", func(t *testing.T) {
		cfg := &Config{Backend: "markdown"}
		if got := cfg.GetBackend(); got != "markdown" {
			t.Errorf("expected markdown, got %q", got)
		}
	})
}

func TestGetDataDir(t *testing.T) {
	t.Run("default uses XDG", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/custom/data")
		cfg := &Config{}
		if got := cfg.GetDataDir(); got != "/custom/data/toki" {
			t.Errorf("expected /custom/data/toki, got %q", got)
		}
	})

	t.Run("explicit path", func(t *testing.T) {
		cfg := &Config{DataDir: "/my/data"}
		if got := cfg.GetDataDir(); got != "/my/data" {
			t.Errorf("expected /my/data, got %q", got)
		}
	})

	t.Run("tilde expansion", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		cfg := &Config{DataDir: "~/toki-data"}
		expected := filepath.Join(home, "toki-data")
		if got := cfg.GetDataDir(); got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"~", home},
		{"~/foo", filepath.Join(home, "foo")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestOpenStorage(t *testing.T) {
	t.Run("sqlite backend", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{Backend: "sqlite", DataDir: tmpDir}

		store, err := cfg.OpenStorage()
		if err != nil {
			t.Fatalf("OpenStorage (sqlite) failed: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Verify the db file was created
		dbPath := filepath.Join(tmpDir, "toki.db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("expected toki.db to be created")
		}
	})

	t.Run("markdown backend", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{Backend: "markdown", DataDir: tmpDir}

		store, err := cfg.OpenStorage()
		if err != nil {
			t.Fatalf("OpenStorage (markdown) failed: %v", err)
		}
		defer func() { _ = store.Close() }()
	})

	t.Run("unknown backend", func(t *testing.T) {
		cfg := &Config{Backend: "badger"}
		_, err := cfg.OpenStorage()
		if err == nil {
			t.Error("expected error for unknown backend")
		}
	})
}

func TestLoadSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := &Config{Backend: "markdown", DataDir: "~/my-todos"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Backend != "markdown" {
		t.Errorf("expected backend 'markdown', got %q", loaded.Backend)
	}
	if loaded.DataDir != "~/my-todos" {
		t.Errorf("expected data_dir '~/my-todos', got %q", loaded.DataDir)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	// Also set XDG_DATA_HOME so defaultFirstRunConfig checks the right place
	t.Setenv("XDG_DATA_HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should not fail for missing file: %v", err)
	}
	if cfg.Backend != "markdown" {
		t.Errorf("expected markdown backend for new user, got %q", cfg.Backend)
	}

	// Verify config file was auto-created
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config file to be auto-created")
	}
}

func TestLoadMissingConfig_ExistingSQLiteUser(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Create a fake .db file to simulate existing SQLite user
	dbDir := filepath.Join(tmpDir, "toki")
	if err := os.MkdirAll(dbDir, 0750); err != nil {
		t.Fatalf("failed to create db dir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "toki.db")
	if err := os.WriteFile(dbPath, []byte("fake-db"), 0600); err != nil {
		t.Fatalf("failed to create fake db: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should not fail: %v", err)
	}
	if cfg.Backend != "sqlite" {
		t.Errorf("expected sqlite backend for existing user, got %q", cfg.Backend)
	}
}

func TestLoadAutoCreatesValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	_, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Read back the auto-created file and verify it's valid JSON
	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath) // #nosec G304 - test path from t.TempDir()
	if err != nil {
		t.Fatalf("failed to read auto-created config: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("auto-created config is not valid JSON: %v", err)
	}
	if raw["backend"] != "markdown" {
		t.Errorf("expected backend 'markdown' in config file, got %v", raw["backend"])
	}
}
