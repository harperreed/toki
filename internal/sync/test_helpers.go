// ABOUTME: Shared test helpers for sync package tests
// ABOUTME: Provides database setup and syncer creation utilities

package sync

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"suitesync/vault"

	"github.com/harper/toki/internal/db"
	"github.com/stretchr/testify/require"
)

// setupTestSyncerWithDB creates a test syncer and returns both the syncer and appDB.
func setupTestSyncerWithDB(t *testing.T) (*Syncer, *sql.DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	database, err := db.InitDB(filepath.Join(tmpDir, "test.db"))
	require.NoError(t, err)

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
		AutoSync:   false,
	}

	syncer, err := NewSyncer(cfg, database)
	require.NoError(t, err)

	cleanup := func() {
		_ = syncer.Close()
		_ = database.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return syncer, database, cleanup
}

// setupTestSyncer creates a test syncer (legacy helper for existing tests).
func setupTestSyncer(t *testing.T) *Syncer {
	t.Helper()
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	t.Cleanup(func() { _ = appDB.Close() })

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
		AutoSync:   false,
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)

	return syncer
}

// setupTestDB creates a minimal test database with toki schema.
func setupTestDB(t *testing.T, tmpDir string) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.InitDB(dbPath)
	require.NoError(t, err)

	return database
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}
