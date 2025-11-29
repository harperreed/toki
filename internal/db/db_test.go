// ABOUTME: Tests for database connection and migrations
// ABOUTME: Uses temporary test databases for isolation

package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitDB(t *testing.T) {
	// Create temp dir for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify tables exist
	tables := []string{"projects", "todos", "tags", "todo_tags"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("Error checking table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

func TestGetDefaultDBPath(t *testing.T) {
	path := GetDefaultDBPath()

	if path == "" {
		t.Error("Default path should not be empty")
	}

	// Should contain .local/share/toki
	if !filepath.IsAbs(path) {
		t.Error("Path should be absolute")
	}
}
