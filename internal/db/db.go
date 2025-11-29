// ABOUTME: Database connection management and initialization
// ABOUTME: Handles SQLite connection and migration execution

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitDB initializes the database connection and runs migrations
func InitDB(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// GetDefaultDBPath returns the default database path following XDG standards
func GetDefaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(dataDir, "toki", "toki.db")
}
