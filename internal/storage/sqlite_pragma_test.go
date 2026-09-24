// ABOUTME: Regression tests for SQLite connection pragmas
// ABOUTME: Verifies WAL, foreign keys, busy timeout, and pooling behavior

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// pragmaValue reads a single-value PRAGMA as a string.
func pragmaValue(t *testing.T, db *sql.DB, name string) string {
	t.Helper()

	var value string
	//nolint:gosec // PRAGMA name is a test-controlled constant.
	if err := db.QueryRow("PRAGMA " + name).Scan(&value); err != nil {
		t.Fatalf("failed to read PRAGMA %s: %v", name, err)
	}
	return value
}

// desiredPragmas lists the connection settings NewSQLiteStorage promises.
var desiredPragmas = []struct {
	name  string
	value string
}{
	{"journal_mode", "wal"},
	{"busy_timeout", "5000"},
	{"foreign_keys", "1"},
}

// checkConnPragmas verifies a single pooled connection carries every pragma.
func checkConnPragmas(ctx context.Context, conn *sql.Conn) error {
	for _, p := range desiredPragmas {
		var got string
		//nolint:gosec // PRAGMA name is a test-controlled constant.
		if err := conn.QueryRowContext(ctx, "PRAGMA "+p.name).Scan(&got); err != nil {
			return fmt.Errorf("failed to read PRAGMA %s: %w", p.name, err)
		}
		if got != p.value {
			return fmt.Errorf("pragma %s = %q, want %q", p.name, got, p.value)
		}
	}
	return nil
}

// TestSQLiteJournalModeIsWAL verifies the database runs in WAL mode.
func TestSQLiteJournalModeIsWAL(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	if got := pragmaValue(t, storage.db, "journal_mode"); got != "wal" {
		t.Errorf("journal_mode = %q, want %q", got, "wal")
	}
}

// TestSQLiteForeignKeysEnforced verifies foreign key enforcement is on.
func TestSQLiteForeignKeysEnforced(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	if got := pragmaValue(t, storage.db, "foreign_keys"); got != "1" {
		t.Errorf("foreign_keys = %q, want %q", got, "1")
	}

	// A todo referencing a nonexistent project must be rejected.
	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		Description: "orphan",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := storage.CreateTodo(todo); err == nil {
		t.Fatal("expected foreign key violation creating todo for nonexistent project")
	}
}

// TestSQLiteBusyTimeoutConfigured verifies the 5-second busy timeout.
func TestSQLiteBusyTimeoutConfigured(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	if got := pragmaValue(t, storage.db, "busy_timeout"); got != "5000" {
		t.Errorf("busy_timeout = %q, want %q", got, "5000")
	}
}

// TestSQLiteForeignKeysCascadeDelete verifies project deletion removes todo rows.
func TestSQLiteForeignKeysCascadeDelete(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "cascade",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		Description: "cascades",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := storage.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := storage.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	// Query the raw table so the GetTodo project JOIN cannot hide orphans.
	var count int
	if err := storage.db.QueryRow(`SELECT COUNT(*) FROM todos WHERE id = ?`, todo.ID.String()).Scan(&count); err != nil {
		t.Fatalf("failed to count todos: %v", err)
	}
	if count != 0 {
		t.Errorf("todo rows remaining after project delete = %d, want 0", count)
	}
}

// TestSQLitePragmasApplyToAllPooledConnections verifies every connection in the
// pool receives the configured pragmas, not just the first one.
func TestSQLitePragmasApplyToAllPooledConnections(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	const poolSize = 4
	storage.db.SetMaxOpenConns(poolSize)
	storage.db.SetMaxIdleConns(poolSize)

	ready := make(chan struct{}, poolSize)
	release := make(chan struct{})
	errs := make(chan error, poolSize)

	var wg sync.WaitGroup
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Hold this connection open so the pool must open several.
			conn, err := storage.db.Conn(t.Context())
			if err != nil {
				errs <- err
				ready <- struct{}{}
				return
			}
			defer func() { _ = conn.Close() }()

			ready <- struct{}{}
			<-release

			if err := checkConnPragmas(t.Context(), conn); err != nil {
				errs <- err
			}
		}()
	}

	// Wait until every worker holds its own pooled connection.
	for i := 0; i < poolSize; i++ {
		<-ready
	}
	open := storage.db.Stats().OpenConnections
	close(release)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
	if open < 2 {
		t.Errorf("open pooled connections = %d, want at least 2", open)
	}
}
