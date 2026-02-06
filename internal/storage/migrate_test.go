// ABOUTME: Tests for storage migration between toki backends
// ABOUTME: Covers sqlite-to-markdown, markdown-to-sqlite, data integrity, and round-trips

package storage

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
)

// seedTokiTestData populates a storage backend with a representative data set
// and returns the entities for verification.
func seedTokiTestData(t *testing.T, src Storage) ([]*Project, []*Todo) {
	t.Helper()

	projects := seedTestProjects(t, src)
	todos := seedTestTodos(t, src, projects)
	return projects, todos
}

func seedTestProjects(t *testing.T, src Storage) []*Project {
	t.Helper()
	project1 := &Project{
		ID: uuid.New(), Name: "work",
		DirectoryPath: "/path/to/work",
		CreatedAt:     time.Now().UTC().Truncate(time.Millisecond),
	}
	project2 := &Project{
		ID: uuid.New(), Name: "personal",
		DirectoryPath: "/path/to/personal",
		CreatedAt:     time.Now().UTC().Truncate(time.Millisecond),
	}
	mustNoError(t, src.CreateProject(project1))
	mustNoError(t, src.CreateProject(project2))
	return []*Project{project1, project2}
}

func seedTestTodos(t *testing.T, src Storage, projects []*Project) []*Todo {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	dueDate := now.Add(24 * time.Hour)
	completedAt := now.Add(-1 * time.Hour)

	todo1 := &Todo{
		ID: uuid.New(), ProjectID: projects[0].ID, ProjectName: projects[0].Name,
		Description: "Implement feature X", Done: false, Priority: "high",
		Notes: "This is a complex task\nwith multiple lines",
		Tags:  []string{"backend", "urgent"}, CreatedAt: now.Add(-3 * time.Minute),
		UpdatedAt: now, DueDate: &dueDate,
	}
	todo2 := &Todo{
		ID: uuid.New(), ProjectID: projects[0].ID, ProjectName: projects[0].Name,
		Description: "Fix bug in login", Done: true, Priority: "medium",
		Tags: []string{"bugfix"}, CreatedAt: now.Add(-2 * time.Minute),
		UpdatedAt: now, CompletedAt: &completedAt,
	}
	todo3 := &Todo{
		ID: uuid.New(), ProjectID: projects[1].ID, ProjectName: projects[1].Name,
		Description: "Buy groceries", Done: false, Priority: "low",
		CreatedAt: now.Add(-1 * time.Minute), UpdatedAt: now,
	}

	mustNoError(t, src.CreateTodo(todo1))
	mustNoError(t, src.CreateTodo(todo2))
	mustNoError(t, src.CreateTodo(todo3))
	return []*Todo{todo1, todo2, todo3}
}

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// verifyMigratedTokiData checks that the destination storage contains all expected data.
func verifyMigratedTokiData(t *testing.T, dst Storage, projects []*Project, todos []*Todo) {
	t.Helper()

	// Verify projects
	for _, orig := range projects {
		got, err := dst.GetProject(orig.ID)
		if err != nil {
			t.Errorf("project %s (%s) not found in destination: %v", orig.Name, orig.ID, err)
			continue
		}
		if got.Name != orig.Name {
			t.Errorf("project name mismatch: want %q, got %q", orig.Name, got.Name)
		}
		if got.DirectoryPath != orig.DirectoryPath {
			t.Errorf("project path mismatch: want %q, got %q", orig.DirectoryPath, got.DirectoryPath)
		}
	}

	// Verify todos
	for _, orig := range todos {
		got, err := dst.GetTodo(orig.ID)
		if err != nil {
			t.Errorf("todo %s (%s) not found in destination: %v", orig.Description, orig.ID, err)
			continue
		}
		if got.Description != orig.Description {
			t.Errorf("todo description mismatch: want %q, got %q", orig.Description, got.Description)
		}
		if got.Done != orig.Done {
			t.Errorf("todo done mismatch for %q: want %v, got %v", orig.Description, orig.Done, got.Done)
		}
		if got.Priority != orig.Priority {
			t.Errorf("todo priority mismatch for %q: want %q, got %q", orig.Description, orig.Priority, got.Priority)
		}
		if got.Notes != orig.Notes {
			t.Errorf("todo notes mismatch for %q: want %q, got %q", orig.Description, orig.Notes, got.Notes)
		}

		// Check tags
		origTags := make([]string, len(orig.Tags))
		copy(origTags, orig.Tags)
		sort.Strings(origTags)

		gotTags := make([]string, len(got.Tags))
		copy(gotTags, got.Tags)
		sort.Strings(gotTags)

		if len(origTags) != len(gotTags) {
			t.Errorf("todo tags count mismatch for %q: want %d, got %d", orig.Description, len(origTags), len(gotTags))
		} else {
			for i := range origTags {
				if origTags[i] != gotTags[i] {
					t.Errorf("todo tags mismatch for %q at index %d: want %q, got %q", orig.Description, i, origTags[i], gotTags[i])
				}
			}
		}

		// Check completed status
		if (orig.CompletedAt == nil) != (got.CompletedAt == nil) {
			t.Errorf("todo completedAt nil mismatch for %q: orig=%v, got=%v", orig.Description, orig.CompletedAt, got.CompletedAt)
		}
		if (orig.DueDate == nil) != (got.DueDate == nil) {
			t.Errorf("todo dueDate nil mismatch for %q: orig=%v, got=%v", orig.Description, orig.DueDate, got.DueDate)
		}
	}
}

func TestMigrateData_SqliteToMarkdown(t *testing.T) {
	// Source (sqlite)
	srcDir := t.TempDir()
	src, err := NewSQLiteStorage(filepath.Join(srcDir, "toki.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer func() { _ = src.Close() }()

	projects, todos := seedTokiTestData(t, src)

	// Destination (markdown)
	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer func() { _ = dst.Close() }()

	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	if summary.Projects != len(projects) {
		t.Errorf("summary projects: want %d, got %d", len(projects), summary.Projects)
	}
	if summary.Todos != len(todos) {
		t.Errorf("summary todos: want %d, got %d", len(todos), summary.Todos)
	}

	verifyMigratedTokiData(t, dst, projects, todos)
}

func TestMigrateData_MarkdownToSqlite(t *testing.T) {
	// Source (markdown)
	srcDir := t.TempDir()
	src, err := NewMarkdownStore(srcDir)
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer func() { _ = src.Close() }()

	projects, todos := seedTokiTestData(t, src)

	// Destination (sqlite)
	dstDir := t.TempDir()
	dst, err := NewSQLiteStorage(filepath.Join(dstDir, "toki.db"))
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer func() { _ = dst.Close() }()

	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	if summary.Projects != len(projects) {
		t.Errorf("summary projects: want %d, got %d", len(projects), summary.Projects)
	}
	if summary.Todos != len(todos) {
		t.Errorf("summary todos: want %d, got %d", len(todos), summary.Todos)
	}

	verifyMigratedTokiData(t, dst, projects, todos)
}

func TestMigrateData_EmptySource(t *testing.T) {
	srcDir := t.TempDir()
	src, err := NewSQLiteStorage(filepath.Join(srcDir, "toki.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer func() { _ = src.Close() }()

	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer func() { _ = dst.Close() }()

	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	if summary.Projects != 0 || summary.Todos != 0 || summary.Tags != 0 {
		t.Errorf("expected all zero counts, got projects=%d todos=%d tags=%d",
			summary.Projects, summary.Todos, summary.Tags)
	}
}

func TestMigrateRoundTrip_SqliteMarkdownSqlite(t *testing.T) {
	// Phase 1: Create data in SQLite
	srcDir := t.TempDir()
	original, err := NewSQLiteStorage(filepath.Join(srcDir, "original.db"))
	if err != nil {
		t.Fatalf("create original store: %v", err)
	}
	defer func() { _ = original.Close() }()

	projects, todos := seedTokiTestData(t, original)

	// Phase 2: Migrate SQLite -> Markdown
	mdDir := t.TempDir()
	mdStore, err := NewMarkdownStore(mdDir)
	if err != nil {
		t.Fatalf("create markdown store: %v", err)
	}
	defer func() { _ = mdStore.Close() }()

	_, err = MigrateData(original, mdStore)
	if err != nil {
		t.Fatalf("MigrateData (sqlite->markdown) failed: %v", err)
	}

	// Phase 3: Migrate Markdown -> SQLite
	dstDir := t.TempDir()
	final, err := NewSQLiteStorage(filepath.Join(dstDir, "final.db"))
	if err != nil {
		t.Fatalf("create final store: %v", err)
	}
	defer func() { _ = final.Close() }()

	_, err = MigrateData(mdStore, final)
	if err != nil {
		t.Fatalf("MigrateData (markdown->sqlite) failed: %v", err)
	}

	// Phase 4: Verify
	verifyMigratedTokiData(t, final, projects, todos)
}

func TestMigrateRoundTrip_MarkdownSqliteMarkdown(t *testing.T) {
	// Phase 1: Create data in Markdown
	srcDir := t.TempDir()
	original, err := NewMarkdownStore(srcDir)
	if err != nil {
		t.Fatalf("create original store: %v", err)
	}
	defer func() { _ = original.Close() }()

	projects, todos := seedTokiTestData(t, original)

	// Phase 2: Migrate Markdown -> SQLite
	sqlDir := t.TempDir()
	sqlStore, err := NewSQLiteStorage(filepath.Join(sqlDir, "mid.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer func() { _ = sqlStore.Close() }()

	_, err = MigrateData(original, sqlStore)
	if err != nil {
		t.Fatalf("MigrateData (markdown->sqlite) failed: %v", err)
	}

	// Phase 3: Migrate SQLite -> Markdown
	dstDir := t.TempDir()
	final, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create final store: %v", err)
	}
	defer func() { _ = final.Close() }()

	_, err = MigrateData(sqlStore, final)
	if err != nil {
		t.Fatalf("MigrateData (sqlite->markdown) failed: %v", err)
	}

	// Phase 4: Verify
	verifyMigratedTokiData(t, final, projects, todos)
}

func TestIsDirNonEmpty(t *testing.T) {
	// Empty directory
	emptyDir := t.TempDir()
	nonEmpty, err := IsDirNonEmpty(emptyDir)
	if err != nil {
		t.Fatalf("IsDirNonEmpty on empty dir: %v", err)
	}
	if nonEmpty {
		t.Error("expected empty dir to be reported as empty")
	}

	// Non-empty directory
	nonEmptyDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("data"), 0600); err != nil {
		t.Fatalf("create file: %v", err)
	}
	nonEmpty, err = IsDirNonEmpty(nonEmptyDir)
	if err != nil {
		t.Fatalf("IsDirNonEmpty on non-empty dir: %v", err)
	}
	if !nonEmpty {
		t.Error("expected non-empty dir to be reported as non-empty")
	}

	// Non-existent directory
	nonEmpty, err = IsDirNonEmpty(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("IsDirNonEmpty on non-existent dir: %v", err)
	}
	if nonEmpty {
		t.Error("expected non-existent dir to be reported as empty")
	}
}
