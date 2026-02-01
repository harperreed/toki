// ABOUTME: Tests for import/migrate command functionality
// ABOUTME: Covers truncateStr, importProject, importTodo functions

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/storage"
)

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "needs truncation",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "truncate to 3 chars",
			input:    "abcdefghij",
			maxLen:   6,
			expected: "abc...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateStr(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestImportCommand(t *testing.T) {
	t.Run("command is registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"import"})
		if err != nil {
			t.Fatalf("import command not found: %v", err)
		}

		if cmd.Name() != "import" {
			t.Errorf("expected command name 'import', got '%s'", cmd.Name())
		}
	})

	t.Run("has dry-run flag", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"import"})
		if err != nil {
			t.Fatalf("import command not found: %v", err)
		}

		dryRunFlag := cmd.Flags().Lookup("dry-run")
		if dryRunFlag == nil {
			t.Error("expected --dry-run flag on import command")
		}
	})

	t.Run("requires exactly 1 argument", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"import"})
		if err != nil {
			t.Fatalf("import command not found: %v", err)
		}

		if cmd.Args == nil {
			t.Error("expected import command to have Args set")
		}
	})
}

func setupMigrateTestStorage(t *testing.T) (*storage.SQLiteStorage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "toki-migrate-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("failed to create storage: %v", err)
	}

	cleanup := func() {
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

//nolint:funlen,gocognit
func TestImportProject(t *testing.T) {
	store, cleanup := setupMigrateTestStorage(t)
	defer cleanup()

	t.Run("import new project", func(t *testing.T) {
		proj := ExportProject{
			ID:   uuid.New().String(),
			Name: "new-project",
			Path: "/path/to/project",
		}

		imported, err := importProject(store, proj, false)
		if err != nil {
			t.Fatalf("importProject failed: %v", err)
		}

		if !imported {
			t.Error("expected project to be imported")
		}

		// Verify project exists
		id, _ := uuid.Parse(proj.ID)
		project, err := store.GetProject(id)
		if err != nil {
			t.Fatalf("failed to get imported project: %v", err)
		}

		if project.Name != "new-project" {
			t.Errorf("expected name 'new-project', got '%s'", project.Name)
		}
		if project.DirectoryPath != "/path/to/project" {
			t.Errorf("expected path '/path/to/project', got '%s'", project.DirectoryPath)
		}
	})

	t.Run("skip existing project", func(t *testing.T) {
		// Create a project first
		projID := uuid.New()
		existingProj := &storage.Project{
			ID:        projID,
			Name:      "existing-project",
			CreatedAt: time.Now().UTC(),
		}
		if err := store.CreateProject(existingProj); err != nil {
			t.Fatalf("failed to create existing project: %v", err)
		}

		// Try to import with same ID
		proj := ExportProject{
			ID:   projID.String(),
			Name: "existing-project",
		}

		imported, err := importProject(store, proj, false)
		if err != nil {
			t.Fatalf("importProject failed: %v", err)
		}

		if imported {
			t.Error("expected existing project to be skipped")
		}
	})

	t.Run("dry run does not create", func(t *testing.T) {
		proj := ExportProject{
			ID:   uuid.New().String(),
			Name: "dry-run-project",
		}

		imported, err := importProject(store, proj, true)
		if err != nil {
			t.Fatalf("importProject failed: %v", err)
		}

		if !imported {
			t.Error("expected dry run to report would import")
		}

		// Verify project was NOT created
		id, _ := uuid.Parse(proj.ID)
		_, err = store.GetProject(id)
		if err == nil {
			t.Error("expected project to NOT be created in dry run mode")
		}
	})

	t.Run("invalid project ID", func(t *testing.T) {
		proj := ExportProject{
			ID:   "not-a-valid-uuid",
			Name: "invalid-id-project",
		}

		_, err := importProject(store, proj, false)
		if err == nil {
			t.Error("expected error for invalid project ID")
		}
	})
}

//nolint:gocognit,funlen
func TestImportTodo(t *testing.T) {
	store, cleanup := setupMigrateTestStorage(t)
	defer cleanup()

	// Create a project for todos
	projID := uuid.New()
	project := &storage.Project{
		ID:        projID,
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	t.Run("import new todo", func(t *testing.T) {
		now := time.Now().UTC()
		todo := ExportTodo{
			ID:          uuid.New().String(),
			Description: "new todo",
			Done:        false,
			Priority:    "high",
			Notes:       "some notes",
			Tags:        []string{"tag1", "tag2"},
			CreatedAt:   now.Format(time.RFC3339),
			UpdatedAt:   now.Format(time.RFC3339),
		}

		imported, tagCount, err := importTodo(store, projID.String(), todo, false)
		if err != nil {
			t.Fatalf("importTodo failed: %v", err)
		}

		if !imported {
			t.Error("expected todo to be imported")
		}
		if tagCount != 2 {
			t.Errorf("expected 2 tags, got %d", tagCount)
		}

		// Verify todo exists
		id, _ := uuid.Parse(todo.ID)
		storedTodo, err := store.GetTodo(id)
		if err != nil {
			t.Fatalf("failed to get imported todo: %v", err)
		}

		if storedTodo.Description != "new todo" {
			t.Errorf("expected description 'new todo', got '%s'", storedTodo.Description)
		}
		if storedTodo.Priority != "high" {
			t.Errorf("expected priority 'high', got '%s'", storedTodo.Priority)
		}
	})

	t.Run("import completed todo", func(t *testing.T) {
		now := time.Now().UTC()
		completedAt := now.Add(-1 * time.Hour)
		dueDate := now.Add(24 * time.Hour)

		todo := ExportTodo{
			ID:          uuid.New().String(),
			Description: "completed todo",
			Done:        true,
			CompletedAt: completedAt.Format(time.RFC3339),
			DueDate:     dueDate.Format(time.RFC3339),
			CreatedAt:   now.Format(time.RFC3339),
			UpdatedAt:   now.Format(time.RFC3339),
		}

		imported, _, err := importTodo(store, projID.String(), todo, false)
		if err != nil {
			t.Fatalf("importTodo failed: %v", err)
		}

		if !imported {
			t.Error("expected todo to be imported")
		}

		// Verify todo exists with correct fields
		id, _ := uuid.Parse(todo.ID)
		storedTodo, err := store.GetTodo(id)
		if err != nil {
			t.Fatalf("failed to get imported todo: %v", err)
		}

		if !storedTodo.Done {
			t.Error("expected todo to be marked done")
		}
		if storedTodo.DueDate == nil {
			t.Error("expected due date to be set")
		}
	})

	t.Run("skip existing todo", func(t *testing.T) {
		// Create a todo first
		now := time.Now().UTC()
		todoID := uuid.New()
		existingTodo := &storage.Todo{
			ID:          todoID,
			ProjectID:   projID,
			Description: "existing todo",
			Done:        false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := store.CreateTodo(existingTodo); err != nil {
			t.Fatalf("failed to create existing todo: %v", err)
		}

		// Try to import with same ID
		todo := ExportTodo{
			ID:          todoID.String(),
			Description: "existing todo",
			CreatedAt:   now.Format(time.RFC3339),
		}

		imported, _, err := importTodo(store, projID.String(), todo, false)
		if err != nil {
			t.Fatalf("importTodo failed: %v", err)
		}

		if imported {
			t.Error("expected existing todo to be skipped")
		}
	})

	t.Run("dry run does not create", func(t *testing.T) {
		now := time.Now().UTC()
		todo := ExportTodo{
			ID:          uuid.New().String(),
			Description: "dry run todo",
			CreatedAt:   now.Format(time.RFC3339),
		}

		imported, _, err := importTodo(store, projID.String(), todo, true)
		if err != nil {
			t.Fatalf("importTodo failed: %v", err)
		}

		if !imported {
			t.Error("expected dry run to report would import")
		}

		// Verify todo was NOT created
		id, _ := uuid.Parse(todo.ID)
		_, err = store.GetTodo(id)
		if err == nil {
			t.Error("expected todo to NOT be created in dry run mode")
		}
	})

	t.Run("invalid todo ID", func(t *testing.T) {
		now := time.Now().UTC()
		todo := ExportTodo{
			ID:          "not-a-valid-uuid",
			Description: "invalid id todo",
			CreatedAt:   now.Format(time.RFC3339),
		}

		_, _, err := importTodo(store, projID.String(), todo, false)
		if err == nil {
			t.Error("expected error for invalid todo ID")
		}
	})

	t.Run("invalid project ID", func(t *testing.T) {
		now := time.Now().UTC()
		todo := ExportTodo{
			ID:          uuid.New().String(),
			Description: "invalid project todo",
			CreatedAt:   now.Format(time.RFC3339),
		}

		_, _, err := importTodo(store, "not-a-valid-uuid", todo, false)
		if err == nil {
			t.Error("expected error for invalid project ID")
		}
	})

	t.Run("invalid created_at", func(t *testing.T) {
		todo := ExportTodo{
			ID:          uuid.New().String(),
			Description: "invalid timestamp todo",
			CreatedAt:   "not-a-timestamp",
		}

		_, _, err := importTodo(store, projID.String(), todo, false)
		if err == nil {
			t.Error("expected error for invalid created_at")
		}
	})
}
