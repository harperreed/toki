// ABOUTME: Tests for MCP tool handlers
// ABOUTME: Covers all CRUD operations and helper functions

package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "toki-mcp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("failed to create storage: %v", err)
	}

	server, err := NewServer(store)
	if err != nil {
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("failed to create server: %v", err)
	}

	cleanup := func() {
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return server, cleanup
}

func createTestProject(t *testing.T, store storage.Storage, name string) *storage.Project {
	t.Helper()

	project := &storage.Project{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	return project
}

func createTestTodo(t *testing.T, store storage.Storage, projectID uuid.UUID, description string) *storage.Todo {
	t.Helper()

	now := time.Now().UTC()
	todo := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   projectID,
		ProjectName: "test-project",
		Description: description,
		Done:        false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}
	return todo
}

func TestNewServer(t *testing.T) {
	t.Run("creates server with valid storage", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		if server == nil {
			t.Fatal("expected non-nil server")
		}
		if server.mcp == nil {
			t.Error("expected mcp server to be initialized")
		}
		if server.storage == nil {
			t.Error("expected storage to be set")
		}
	})

	t.Run("fails with nil storage", func(t *testing.T) {
		_, err := NewServer(nil)
		if err == nil {
			t.Error("expected error with nil storage")
		}
	})
}

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name      string
		priority  *string
		wantError bool
	}{
		{"nil priority is valid", nil, false},
		{"low is valid", strPtr("low"), false},
		{"medium is valid", strPtr("medium"), false},
		{"high is valid", strPtr("high"), false},
		{"invalid priority errors", strPtr("urgent"), true},
		{"empty string errors", strPtr(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePriority(tt.priority)
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseDueDate(t *testing.T) {
	tests := []struct {
		name      string
		dateStr   *string
		wantError bool
		wantNil   bool
	}{
		{"nil date returns nil", nil, false, true},
		{"empty string returns nil", strPtr(""), false, true},
		{"valid RFC3339 date", strPtr("2025-12-01T15:04:05Z"), false, false},
		{"invalid date format errors", strPtr("2025-12-01"), true, false},
		{"invalid date errors", strPtr("not-a-date"), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDueDate(tt.dateStr)
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && result != nil {
				t.Error("expected nil result")
			}
		})
	}
}

//nolint:funlen
func TestHandleAddTodo(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("creates todo with minimal input", func(t *testing.T) {
		input := AddTodoInput{
			Description: "Test todo",
		}

		result, output, err := server.handleAddTodo(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleAddTodo failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if output.Description != "Test todo" {
			t.Errorf("description mismatch: got %q, want %q", output.Description, "Test todo")
		}
		if output.Done != false {
			t.Error("new todo should not be done")
		}
	})

	t.Run("creates todo with all fields", func(t *testing.T) {
		project := createTestProject(t, server.storage, "full-test")
		projectID := project.ID.String()
		priority := "high"
		notes := "test notes"
		dueDate := "2025-12-15T00:00:00Z"

		input := AddTodoInput{
			Description: "Full todo",
			ProjectID:   &projectID,
			Priority:    &priority,
			Notes:       &notes,
			Tags:        []string{"tag1", "tag2"},
			DueDate:     &dueDate,
		}

		result, output, err := server.handleAddTodo(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleAddTodo failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if output.ProjectID != projectID {
			t.Errorf("project_id mismatch: got %q, want %q", output.ProjectID, projectID)
		}
		if output.Priority == nil || *output.Priority != "high" {
			t.Error("priority should be 'high'")
		}
		if output.Notes == nil || *output.Notes != "test notes" {
			t.Error("notes should be set")
		}
		if len(output.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(output.Tags))
		}
	})

	t.Run("fails with invalid project ID", func(t *testing.T) {
		invalidID := "not-a-uuid"
		input := AddTodoInput{
			Description: "Test",
			ProjectID:   &invalidID,
		}

		_, _, err := server.handleAddTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid project ID")
		}
	})

	t.Run("fails with invalid priority", func(t *testing.T) {
		priority := "invalid"
		input := AddTodoInput{
			Description: "Test",
			Priority:    &priority,
		}

		_, _, err := server.handleAddTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid priority")
		}
	})
}

func TestHandleListTodos(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "list-test")

	// Create some todos
	todo1 := createTestTodo(t, server.storage, project.ID, "Todo 1")
	_ = createTestTodo(t, server.storage, project.ID, "Todo 2")

	// Mark one as done
	_ = server.storage.MarkTodoDone(todo1.ID, true)

	t.Run("lists all todos", func(t *testing.T) {
		input := ListTodosInput{}
		result, output, err := server.handleListTodos(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleListTodos failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if output.Count < 2 {
			t.Errorf("expected at least 2 todos, got %d", output.Count)
		}
	})

	t.Run("filters by done status", func(t *testing.T) {
		done := false
		input := ListTodosInput{Done: &done}
		_, output, err := server.handleListTodos(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleListTodos failed: %v", err)
		}
		// Check filters are recorded
		if output.Filters["done"] != false {
			t.Error("filters should include done=false")
		}
	})

	t.Run("filters by project", func(t *testing.T) {
		projectID := project.ID.String()
		input := ListTodosInput{ProjectID: &projectID}
		_, output, err := server.handleListTodos(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleListTodos failed: %v", err)
		}
		if output.Count != 2 {
			t.Errorf("expected 2 todos in project, got %d", output.Count)
		}
	})

	t.Run("fails with invalid project ID", func(t *testing.T) {
		invalidID := "not-a-uuid"
		input := ListTodosInput{ProjectID: &invalidID}
		_, _, err := server.handleListTodos(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid project ID")
		}
	})
}

func TestHandleMarkDone(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "done-test")
	todo := createTestTodo(t, server.storage, project.ID, "Mark me done")

	t.Run("marks todo as done", func(t *testing.T) {
		input := MarkDoneInput{TodoID: todo.ID.String()}
		result, output, err := server.handleMarkDone(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleMarkDone failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if !output.Done {
			t.Error("todo should be marked done")
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		input := MarkDoneInput{TodoID: "not-a-uuid"}
		_, _, err := server.handleMarkDone(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})

	t.Run("fails with non-existent UUID", func(t *testing.T) {
		input := MarkDoneInput{TodoID: uuid.New().String()}
		_, _, err := server.handleMarkDone(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with non-existent UUID")
		}
	})
}

func TestHandleMarkUndone(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "undone-test")
	todo := createTestTodo(t, server.storage, project.ID, "Mark me undone")

	// First mark it done
	_ = server.storage.MarkTodoDone(todo.ID, true)

	t.Run("marks todo as undone", func(t *testing.T) {
		input := MarkUndoneInput{TodoID: todo.ID.String()}
		result, output, err := server.handleMarkUndone(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleMarkUndone failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if output.Done {
			t.Error("todo should be marked undone")
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		input := MarkUndoneInput{TodoID: "not-a-uuid"}
		_, _, err := server.handleMarkUndone(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})
}

func TestHandleDeleteTodo(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "delete-test")
	todo := createTestTodo(t, server.storage, project.ID, "Delete me")

	t.Run("deletes todo", func(t *testing.T) {
		input := DeleteTodoInput{TodoID: todo.ID.String()}
		result, output, err := server.handleDeleteTodo(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleDeleteTodo failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if !output.Success {
			t.Error("delete should be successful")
		}
		if output.TodoID != todo.ID.String() {
			t.Errorf("todo_id mismatch: got %q, want %q", output.TodoID, todo.ID.String())
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		input := DeleteTodoInput{TodoID: "not-a-uuid"}
		_, _, err := server.handleDeleteTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})
}

//nolint:funlen
func TestHandleUpdateTodo(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "update-test")
	todo := createTestTodo(t, server.storage, project.ID, "Original description")

	t.Run("updates todo description", func(t *testing.T) {
		newDesc := "Updated description"
		input := UpdateTodoInput{
			TodoID:      todo.ID.String(),
			Description: &newDesc,
		}
		result, output, err := server.handleUpdateTodo(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleUpdateTodo failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if output.Description != "Updated description" {
			t.Errorf("description mismatch: got %q", output.Description)
		}
	})

	t.Run("updates todo priority and notes", func(t *testing.T) {
		priority := "high"
		notes := "new notes"
		input := UpdateTodoInput{
			TodoID:   todo.ID.String(),
			Priority: &priority,
			Notes:    &notes,
		}
		_, output, err := server.handleUpdateTodo(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleUpdateTodo failed: %v", err)
		}
		if output.Priority == nil || *output.Priority != "high" {
			t.Error("priority should be high")
		}
		if output.Notes == nil || *output.Notes != "new notes" {
			t.Error("notes should be updated")
		}
	})

	t.Run("updates due date", func(t *testing.T) {
		dueDate := "2025-12-25T00:00:00Z"
		input := UpdateTodoInput{
			TodoID:  todo.ID.String(),
			DueDate: &dueDate,
		}
		_, output, err := server.handleUpdateTodo(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleUpdateTodo failed: %v", err)
		}
		if output.DueDate == nil {
			t.Error("due date should be set")
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		desc := "test"
		input := UpdateTodoInput{
			TodoID:      "not-a-uuid",
			Description: &desc,
		}
		_, _, err := server.handleUpdateTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})

	t.Run("fails with non-existent UUID", func(t *testing.T) {
		desc := "test"
		input := UpdateTodoInput{
			TodoID:      uuid.New().String(),
			Description: &desc,
		}
		_, _, err := server.handleUpdateTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with non-existent UUID")
		}
	})

	t.Run("fails with invalid priority", func(t *testing.T) {
		priority := "invalid"
		input := UpdateTodoInput{
			TodoID:   todo.ID.String(),
			Priority: &priority,
		}
		_, _, err := server.handleUpdateTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid priority")
		}
	})

	t.Run("fails with invalid due date", func(t *testing.T) {
		dueDate := "invalid-date"
		input := UpdateTodoInput{
			TodoID:  todo.ID.String(),
			DueDate: &dueDate,
		}
		_, _, err := server.handleUpdateTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid due date")
		}
	})
}

func TestHandleAddTagToTodo(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "tag-test")
	todo := createTestTodo(t, server.storage, project.ID, "Todo with tags")

	t.Run("adds tag to todo", func(t *testing.T) {
		input := AddTagToTodoInput{
			TodoID:  todo.ID.String(),
			TagName: "urgent",
		}
		result, output, err := server.handleAddTagToTodo(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleAddTagToTodo failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(output.Tags) == 0 {
			t.Error("expected tags to be present")
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		input := AddTagToTodoInput{
			TodoID:  "not-a-uuid",
			TagName: "test",
		}
		_, _, err := server.handleAddTagToTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})
}

func TestHandleRemoveTagFromTodo(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "remove-tag-test")
	todo := createTestTodo(t, server.storage, project.ID, "Todo with tag")

	// Add a tag first
	_ = server.storage.AddTagToTodo(todo.ID, "removeme")

	t.Run("removes tag from todo", func(t *testing.T) {
		input := RemoveTagFromTodoInput{
			TodoID:  todo.ID.String(),
			TagName: "removeme",
		}
		result, _, err := server.handleRemoveTagFromTodo(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleRemoveTagFromTodo failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		input := RemoveTagFromTodoInput{
			TodoID:  "not-a-uuid",
			TagName: "test",
		}
		_, _, err := server.handleRemoveTagFromTodo(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})
}

func TestHandleAddProject(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("creates new project", func(t *testing.T) {
		input := AddProjectInput{
			Name: "new-project",
		}
		result, output, err := server.handleAddProject(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleAddProject failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if output.Name != "new-project" {
			t.Errorf("name mismatch: got %q", output.Name)
		}
	})

	t.Run("creates project with path", func(t *testing.T) {
		path := "/test/path"
		input := AddProjectInput{
			Name: "project-with-path",
			Path: &path,
		}
		_, output, err := server.handleAddProject(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleAddProject failed: %v", err)
		}
		if output.Path == nil || *output.Path != "/test/path" {
			t.Error("path should be set")
		}
	})

	t.Run("returns existing project", func(t *testing.T) {
		_ = createTestProject(t, server.storage, "existing-project")

		input := AddProjectInput{
			Name: "existing-project",
		}
		result, output, err := server.handleAddProject(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleAddProject failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if output.Name != "existing-project" {
			t.Errorf("name mismatch: got %q", output.Name)
		}
	})
}

func TestHandleListProjects(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create some projects
	_ = createTestProject(t, server.storage, "project-alpha")
	_ = createTestProject(t, server.storage, "project-beta")

	t.Run("lists all projects", func(t *testing.T) {
		input := ListProjectsInput{}
		result, output, err := server.handleListProjects(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleListProjects failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if output.Count < 2 {
			t.Errorf("expected at least 2 projects, got %d", output.Count)
		}
	})
}

func TestHandleDeleteProject(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "to-delete")

	t.Run("deletes project", func(t *testing.T) {
		input := DeleteProjectInput{ProjectID: project.ID.String()}
		result, output, err := server.handleDeleteProject(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleDeleteProject failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if !output.Success {
			t.Error("delete should be successful")
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		input := DeleteProjectInput{ProjectID: "not-a-uuid"}
		_, _, err := server.handleDeleteProject(ctx, &mcp.CallToolRequest{}, input)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})
}

func TestHandleSyncStatus(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns sync status", func(t *testing.T) {
		input := SyncStatusInput{}
		result, output, err := server.handleSyncStatus(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleSyncStatus failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// Cloud sync is removed, so configured should be false
		if output.Configured {
			t.Error("sync should not be configured")
		}
	})
}

func TestHandleSyncNow(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns sync result", func(t *testing.T) {
		input := SyncNowInput{}
		result, output, err := server.handleSyncNow(ctx, &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Fatalf("handleSyncNow failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// Cloud sync is removed, so success should be false
		if output.Success {
			t.Error("sync should not succeed (removed)")
		}
		if output.Error == "" {
			t.Error("should have error message about removed sync")
		}
	})
}

func TestBuildAppliedFilters(t *testing.T) {
	t.Run("empty input returns empty filters", func(t *testing.T) {
		input := ListTodosInput{}
		filters := buildAppliedFilters(input)
		if len(filters) != 0 {
			t.Errorf("expected empty filters, got %d", len(filters))
		}
	})

	t.Run("includes all set filters", func(t *testing.T) {
		projectID := "test-project-id"
		done := true
		priority := "high"
		tag := "urgent"
		overdue := true

		input := ListTodosInput{
			ProjectID: &projectID,
			Done:      &done,
			Priority:  &priority,
			Tag:       &tag,
			Overdue:   &overdue,
		}
		filters := buildAppliedFilters(input)
		if filters["project_id"] != projectID {
			t.Error("project_id filter missing")
		}
		if filters["done"] != true {
			t.Error("done filter missing")
		}
		if filters["priority"] != "high" {
			t.Error("priority filter missing")
		}
		if filters["tag"] != "urgent" {
			t.Error("tag filter missing")
		}
		if filters["overdue"] != true {
			t.Error("overdue filter missing")
		}
	})
}

func TestResolveProjectID(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	project := createTestProject(t, server.storage, "resolve-test")

	t.Run("resolves given project ID", func(t *testing.T) {
		projectIDStr := project.ID.String()
		id, err := server.resolveProjectID(&projectIDStr)
		if err != nil {
			t.Fatalf("resolveProjectID failed: %v", err)
		}
		if id != project.ID {
			t.Error("project ID mismatch")
		}
	})

	t.Run("creates default project when nil", func(t *testing.T) {
		id, err := server.resolveProjectID(nil)
		if err != nil {
			t.Fatalf("resolveProjectID failed: %v", err)
		}
		if id == uuid.Nil {
			t.Error("expected non-nil UUID")
		}
	})

	t.Run("creates default project when empty", func(t *testing.T) {
		empty := ""
		id, err := server.resolveProjectID(&empty)
		if err != nil {
			t.Fatalf("resolveProjectID failed: %v", err)
		}
		if id == uuid.Nil {
			t.Error("expected non-nil UUID")
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		invalid := "not-a-uuid"
		_, err := server.resolveProjectID(&invalid)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})

	t.Run("fails with non-existent project", func(t *testing.T) {
		nonExistent := uuid.New().String()
		_, err := server.resolveProjectID(&nonExistent)
		if err == nil {
			t.Error("expected error with non-existent project")
		}
	})
}

func TestResolveOptionalProjectID(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("returns nil for nil input", func(t *testing.T) {
		id, err := server.resolveOptionalProjectID(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		empty := ""
		id, err := server.resolveOptionalProjectID(&empty)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != nil {
			t.Error("expected nil for empty input")
		}
	})

	t.Run("parses valid UUID", func(t *testing.T) {
		validUUID := uuid.New().String()
		id, err := server.resolveOptionalProjectID(&validUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == nil {
			t.Error("expected non-nil UUID")
		}
	})

	t.Run("fails with invalid UUID", func(t *testing.T) {
		invalid := "not-a-uuid"
		_, err := server.resolveOptionalProjectID(&invalid)
		if err == nil {
			t.Error("expected error with invalid UUID")
		}
	})
}

func strPtr(s string) *string {
	return &s
}
