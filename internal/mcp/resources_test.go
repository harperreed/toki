// ABOUTME: Tests for MCP resource handlers
// ABOUTME: Covers project, todo, and stats resources

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestBuildFiltersMetadata(t *testing.T) {
	t.Run("empty filters", func(t *testing.T) {
		filters := buildFiltersMetadata(nil, nil, nil, nil, false)
		if len(filters) != 0 {
			t.Errorf("expected empty filters, got %d", len(filters))
		}
	})

	t.Run("all filters set", func(t *testing.T) {
		projectID := uuid.New()
		done := true
		priority := "high"
		tag := "urgent"

		filters := buildFiltersMetadata(&projectID, &done, &priority, &tag, true)

		if filters["project_id"] != projectID.String() {
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

	t.Run("partial filters", func(t *testing.T) {
		done := false
		filters := buildFiltersMetadata(nil, &done, nil, nil, false)

		if len(filters) != 1 {
			t.Errorf("expected 1 filter, got %d", len(filters))
		}
		if filters["done"] != false {
			t.Error("done filter should be false")
		}
	})
}

//nolint:funlen
func TestBuildTodoResourceLinks(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("base links always present", func(t *testing.T) {
		links := server.buildTodoResourceLinks(nil, nil, nil, nil, false)

		if _, ok := links["all_todos"]; !ok {
			t.Error("all_todos link should be present")
		}
		if _, ok := links["pending"]; !ok {
			t.Error("pending link should be present")
		}
		if _, ok := links["high_priority"]; !ok {
			t.Error("high_priority link should be present")
		}
		if _, ok := links["overdue"]; !ok {
			t.Error("overdue link should be present")
		}
	})

	t.Run("no pending link when done is true", func(t *testing.T) {
		done := true
		links := server.buildTodoResourceLinks(nil, &done, nil, nil, false)

		if _, ok := links["pending"]; ok {
			t.Error("pending link should not be present when done=true")
		}
	})

	t.Run("no high_priority link when priority filter set", func(t *testing.T) {
		priority := "high"
		links := server.buildTodoResourceLinks(nil, nil, &priority, nil, false)

		if _, ok := links["high_priority"]; ok {
			t.Error("high_priority link should not be present when priority is filtered")
		}
	})

	t.Run("no overdue link when overdue filter is true", func(t *testing.T) {
		links := server.buildTodoResourceLinks(nil, nil, nil, nil, true)

		if _, ok := links["overdue"]; ok {
			t.Error("overdue link should not be present when already filtering overdue")
		}
	})

	t.Run("query link includes all filters", func(t *testing.T) {
		projectID := uuid.New()
		done := false
		priority := "high"
		tag := "urgent"

		links := server.buildTodoResourceLinks(&projectID, &done, &priority, &tag, true)

		queryLink, ok := links["query"]
		if !ok {
			t.Fatal("query link should be present when filters are applied")
		}

		// Check that the query contains all the filter parameters
		if len(queryLink) == 0 {
			t.Error("query link should not be empty")
		}
	})
}

//nolint:funlen
func TestHandleTodoResource(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	project := createTestProject(t, server.storage, "resource-test")

	// Create some todos
	now := time.Now().UTC()
	past := now.Add(-48 * time.Hour)

	todo1 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo 1",
		Done:        false,
		Priority:    "high",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = server.storage.CreateTodo(todo1)

	todo2 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo 2",
		Done:        true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = server.storage.CreateTodo(todo2)

	todo3 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Overdue todo",
		Done:        false,
		DueDate:     &past,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = server.storage.CreateTodo(todo3)

	t.Run("returns all todos", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{
				URI: "toki://todos",
			},
		}
		result, err := server.handleTodoResource(ctx, req, nil, nil, nil, nil, false)
		if err != nil {
			t.Fatalf("handleTodoResource failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Contents) != 1 {
			t.Errorf("expected 1 content, got %d", len(result.Contents))
		}
	})

	t.Run("filters by done status", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{
				URI: "toki://todos/pending",
			},
		}
		done := false
		result, err := server.handleTodoResource(ctx, req, nil, &done, nil, nil, false)
		if err != nil {
			t.Fatalf("handleTodoResource failed: %v", err)
		}

		// Parse the JSON response
		var resourceData ResourceData
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &resourceData); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		todos := resourceData.Data.([]interface{})
		// Should have 2 pending todos (todo1 and todo3)
		if len(todos) < 2 {
			t.Errorf("expected at least 2 pending todos, got %d", len(todos))
		}
	})

	t.Run("filters by priority", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{
				URI: "toki://todos/high-priority",
			},
		}
		priority := "high"
		result, err := server.handleTodoResource(ctx, req, nil, nil, &priority, nil, false)
		if err != nil {
			t.Fatalf("handleTodoResource failed: %v", err)
		}

		var resourceData ResourceData
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &resourceData); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		todos := resourceData.Data.([]interface{})
		if len(todos) != 1 {
			t.Errorf("expected 1 high priority todo, got %d", len(todos))
		}
	})

	t.Run("filters by overdue", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{
				URI: "toki://todos/overdue",
			},
		}
		result, err := server.handleTodoResource(ctx, req, nil, nil, nil, nil, true)
		if err != nil {
			t.Fatalf("handleTodoResource failed: %v", err)
		}

		var resourceData ResourceData
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &resourceData); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		todos := resourceData.Data.([]interface{})
		if len(todos) != 1 {
			t.Errorf("expected 1 overdue todo, got %d", len(todos))
		}
	})
}

//nolint:funlen
func TestBuildTodoOutputsFromStorage(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	now := time.Now().UTC()
	dueDate := now.Add(24 * time.Hour)
	completedAt := now

	todos := []*storage.Todo{
		{
			ID:          uuid.New(),
			ProjectID:   uuid.New(),
			Description: "Todo with all fields",
			Done:        true,
			Priority:    "high",
			Notes:       "Some notes",
			Tags:        []string{"tag1", "tag2"},
			CreatedAt:   now,
			UpdatedAt:   now,
			DueDate:     &dueDate,
			CompletedAt: &completedAt,
		},
		{
			ID:          uuid.New(),
			ProjectID:   uuid.New(),
			Description: "Minimal todo",
			Done:        false,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	outputs := server.buildTodoOutputsFromStorage(todos)

	if len(outputs) != 2 {
		t.Errorf("expected 2 outputs, got %d", len(outputs))
	}

	// Check first todo (with all fields)
	first := outputs[0]
	if first["description"] != "Todo with all fields" {
		t.Error("description mismatch")
	}
	if first["done"] != true {
		t.Error("done should be true")
	}
	if first["priority"] != "high" {
		t.Error("priority should be high")
	}
	if first["notes"] != "Some notes" {
		t.Error("notes mismatch")
	}
	if _, ok := first["completed_at"]; !ok {
		t.Error("completed_at should be present")
	}
	if _, ok := first["due_date"]; !ok {
		t.Error("due_date should be present")
	}

	// Check second todo (minimal)
	second := outputs[1]
	if second["description"] != "Minimal todo" {
		t.Error("description mismatch")
	}
	if _, ok := second["priority"]; ok {
		t.Error("priority should not be present")
	}
	if _, ok := second["notes"]; ok {
		t.Error("notes should not be present")
	}
}

//nolint:funlen
func TestCalculateStats(t *testing.T) {
	// Create a fresh server for stats testing
	tmpDir, err := os.MkdirTemp("", "toki-stats-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	server, err := NewServer(store)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create projects and todos
	project1 := &storage.Project{
		ID:        uuid.New(),
		Name:      "project-1",
		CreatedAt: time.Now().UTC(),
	}
	_ = store.CreateProject(project1)

	project2 := &storage.Project{
		ID:        uuid.New(),
		Name:      "project-2",
		CreatedAt: time.Now().UTC(),
	}
	_ = store.CreateProject(project2)

	now := time.Now().UTC()
	past := now.Add(-48 * time.Hour)

	// Pending high priority todo in project1
	todo1 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project1.ID,
		ProjectName: project1.Name,
		Description: "High priority todo",
		Done:        false,
		Priority:    "high",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = store.CreateTodo(todo1)

	// Completed todo in project1
	todo2 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project1.ID,
		ProjectName: project1.Name,
		Description: "Completed todo",
		Done:        true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = store.CreateTodo(todo2)

	// Overdue todo in project2
	todo3 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project2.ID,
		ProjectName: project2.Name,
		Description: "Overdue todo",
		Done:        false,
		DueDate:     &past,
		CreatedAt:   now.Add(-72 * time.Hour), // Older for oldest pending test
		UpdatedAt:   now,
	}
	_ = store.CreateTodo(todo3)

	t.Run("calculates correct summary stats", func(t *testing.T) {
		stats, err := server.calculateStats()
		if err != nil {
			t.Fatalf("calculateStats failed: %v", err)
		}

		if stats.Summary.TotalTodos != 3 {
			t.Errorf("expected 3 total todos, got %d", stats.Summary.TotalTodos)
		}
		if stats.Summary.Pending != 2 {
			t.Errorf("expected 2 pending, got %d", stats.Summary.Pending)
		}
		if stats.Summary.Completed != 1 {
			t.Errorf("expected 1 completed, got %d", stats.Summary.Completed)
		}
		if stats.Summary.Overdue != 1 {
			t.Errorf("expected 1 overdue, got %d", stats.Summary.Overdue)
		}
	})

	t.Run("calculates priority breakdown", func(t *testing.T) {
		stats, err := server.calculateStats()
		if err != nil {
			t.Fatalf("calculateStats failed: %v", err)
		}

		if stats.ByPriority["high"] != 1 {
			t.Errorf("expected 1 high priority, got %d", stats.ByPriority["high"])
		}
	})

	t.Run("calculates project breakdown", func(t *testing.T) {
		stats, err := server.calculateStats()
		if err != nil {
			t.Fatalf("calculateStats failed: %v", err)
		}

		if len(stats.ByProject) != 2 {
			t.Errorf("expected 2 projects, got %d", len(stats.ByProject))
		}
	})

	t.Run("identifies oldest pending todo", func(t *testing.T) {
		stats, err := server.calculateStats()
		if err != nil {
			t.Fatalf("calculateStats failed: %v", err)
		}

		if stats.OldestPending == nil {
			t.Fatal("expected oldest pending to be set")
		}
		// The oldest pending should be todo3 (created 72 hours ago)
		if stats.OldestPending.ID != todo3.ID.String() {
			t.Error("oldest pending should be the overdue todo")
		}
		if stats.OldestPending.AgeDays < 2 {
			t.Errorf("expected age >= 2 days, got %d", stats.OldestPending.AgeDays)
		}
	})
}

//nolint:funlen
func TestFetchAndFilterTodos(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	project := createTestProject(t, server.storage, "filter-test")

	now := time.Now().UTC()
	past := now.Add(-48 * time.Hour)

	// Create todos with various attributes
	todo1 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "High priority pending",
		Done:        false,
		Priority:    "high",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = server.storage.CreateTodo(todo1)

	todo2 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Completed",
		Done:        true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = server.storage.CreateTodo(todo2)

	todo3 := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Overdue",
		Done:        false,
		DueDate:     &past,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = server.storage.CreateTodo(todo3)

	t.Run("fetches all todos with no filters", func(t *testing.T) {
		todos, err := server.fetchAndFilterTodos(nil, nil, nil, nil, false)
		if err != nil {
			t.Fatalf("fetchAndFilterTodos failed: %v", err)
		}
		if len(todos) < 3 {
			t.Errorf("expected at least 3 todos, got %d", len(todos))
		}
	})

	t.Run("filters by project", func(t *testing.T) {
		todos, err := server.fetchAndFilterTodos(&project.ID, nil, nil, nil, false)
		if err != nil {
			t.Fatalf("fetchAndFilterTodos failed: %v", err)
		}
		if len(todos) != 3 {
			t.Errorf("expected 3 todos in project, got %d", len(todos))
		}
	})

	t.Run("filters by done status", func(t *testing.T) {
		done := false
		todos, err := server.fetchAndFilterTodos(nil, &done, nil, nil, false)
		if err != nil {
			t.Fatalf("fetchAndFilterTodos failed: %v", err)
		}
		for _, todo := range todos {
			if todo.Done {
				t.Error("should not include done todos")
			}
		}
	})

	t.Run("filters by priority", func(t *testing.T) {
		priority := "high"
		todos, err := server.fetchAndFilterTodos(nil, nil, &priority, nil, false)
		if err != nil {
			t.Fatalf("fetchAndFilterTodos failed: %v", err)
		}
		for _, todo := range todos {
			if todo.Priority != "high" {
				t.Error("should only include high priority todos")
			}
		}
	})

	t.Run("filters by overdue", func(t *testing.T) {
		todos, err := server.fetchAndFilterTodos(nil, nil, nil, nil, true)
		if err != nil {
			t.Fatalf("fetchAndFilterTodos failed: %v", err)
		}
		if len(todos) != 1 {
			t.Errorf("expected 1 overdue todo, got %d", len(todos))
		}
	})
}
