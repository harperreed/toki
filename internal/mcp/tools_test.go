// ABOUTME: Tests for MCP tool definitions and handlers
// ABOUTME: Validates tool registration, parameter parsing, and error handling

package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	return database
}

type testSession struct {
	session *mcp.ClientSession
	cleanup func()
}

func setupTestSession(t *testing.T, database *sql.DB) *testSession {
	t.Helper()

	server, err := NewServer(database)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	_, err = server.mcp.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}

	return &testSession{
		session: session,
		cleanup: func() { _ = session.Close() },
	}
}

func parseAddTodoResult(t *testing.T, result *mcp.CallToolResult) map[string]interface{} {
	t.Helper()

	if result.IsError {
		t.Fatalf("add_todo returned error: %v", result.Content)
	}

	if len(result.Content) == 0 {
		t.Fatal("add_todo should return content")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content")
	}

	var todo map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &todo); err != nil {
		t.Fatalf("Failed to parse todo JSON: %v", err)
	}

	return todo
}

func TestRegisterTools(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	server, err := NewServer(database)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Connect to server to inspect tools
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	_, err = server.mcp.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer func() { _ = session.Close() }()

	// List tools to verify add_todo exists
	foundAddTodo := false
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}
		if tool.Name == "add_todo" {
			foundAddTodo = true
			if tool.Description == "" {
				t.Error("add_todo tool should have a description")
			}
			if tool.InputSchema == nil {
				t.Error("add_todo tool should have input schema")
			}
		}
	}

	if !foundAddTodo {
		t.Error("add_todo tool was not registered")
	}
}

func TestAddTodoMinimalParams(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_todo",
		Arguments: map[string]any{"description": "implement user authentication endpoint"},
	})
	if err != nil {
		t.Fatalf("Failed to call add_todo: %v", err)
	}

	todo := parseAddTodoResult(t, result)

	if todo["description"] != "implement user authentication endpoint" {
		t.Errorf("Expected description 'implement user authentication endpoint', got %s", todo["description"])
	}

	if todo["done"].(bool) {
		t.Error("New todo should not be done")
	}

	if _, err := uuid.Parse(todo["id"].(string)); err != nil {
		t.Errorf("Todo ID should be valid UUID: %v", err)
	}
}

func TestAddTodoAllParams(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project := createTestProject(t, database)
	ts := setupTestSession(t, database)
	defer ts.cleanup()

	dueDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_todo",
		Arguments: map[string]any{
			"description": "fix authentication bug",
			"project_id":  project.ID.String(),
			"priority":    "high",
			"tags":        []string{"bug", "urgent"},
			"notes":       "User reported login failure on mobile",
			"due_date":    dueDate,
		},
	})
	if err != nil {
		t.Fatalf("Failed to call add_todo: %v", err)
	}

	todo := parseAddTodoResult(t, result)
	verifyTodoAllParams(t, database, todo, project.ID.String())
}

func createTestProject(t *testing.T, database *sql.DB) *models.Project {
	t.Helper()
	project := &models.Project{
		ID:        uuid.New(),
		Name:      "test-project-" + uuid.New().String()[:8],
		CreatedAt: time.Now(),
	}
	if err := db.CreateProject(database, project); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	return project
}

func verifyTodoAllParams(t *testing.T, database *sql.DB, todo map[string]interface{}, expectedProjectID string) {
	t.Helper()

	if todo["description"] != "fix authentication bug" {
		t.Errorf("Expected description 'fix authentication bug', got %v", todo["description"])
	}

	if todo["priority"] != "high" {
		t.Errorf("Expected priority 'high', got %v", todo["priority"])
	}

	if todo["notes"] != "User reported login failure on mobile" {
		t.Errorf("Expected notes to match, got %v", todo["notes"])
	}

	if todo["due_date"] == nil {
		t.Error("Expected due_date to be set")
	}

	if todo["project_id"] != expectedProjectID {
		t.Errorf("Expected project_id %s, got %v", expectedProjectID, todo["project_id"])
	}

	verifyTodoTags(t, database, todo["id"].(string))
}

func verifyTodoTags(t *testing.T, database *sql.DB, todoIDStr string) {
	t.Helper()

	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		t.Fatalf("Failed to parse todo ID: %v", err)
	}

	tags, err := db.GetTodoTags(database, todoID)
	if err != nil {
		t.Fatalf("Failed to get todo tags: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}

	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}

	if !tagNames["bug"] || !tagNames["urgent"] {
		t.Error("Expected tags 'bug' and 'urgent'")
	}
}

func TestAddTodoMissingDescription(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	server, err := NewServer(database)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	_, err = server.mcp.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer func() { _ = session.Close() }()

	// Call add_todo without description - should fail validation
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_todo",
		Arguments: map[string]any{},
	})

	if err == nil {
		t.Error("Expected error when description is missing")
	}
}

func TestAddTodoInvalidPriority(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	server, err := NewServer(database)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	_, err = server.mcp.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer func() { _ = session.Close() }()

	// Call add_todo with invalid priority - should fail validation
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_todo",
		Arguments: map[string]any{
			"description": "test todo",
			"priority":    "super-urgent", // invalid
		},
	})

	if err == nil {
		t.Error("Expected error when priority is invalid")
	}
}

func TestAddTodoInvalidProjectID(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	server, err := NewServer(database)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	_, err = server.mcp.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer func() { _ = session.Close() }()

	// Call add_todo with non-existent project_id
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_todo",
		Arguments: map[string]any{
			"description": "test todo",
			"project_id":  uuid.New().String(), // non-existent project
		},
	})

	if err != nil {
		t.Fatalf("Tool call failed: %v", err)
	}

	// Should return an error result, not a protocol error
	if !result.IsError {
		t.Error("Expected error result when project_id doesn't exist")
	}
}

// Helper function to parse list_todos result.
func parseListTodosResult(t *testing.T, result *mcp.CallToolResult) map[string]interface{} {
	t.Helper()

	if result.IsError {
		t.Fatalf("list_todos returned error: %v", result.Content)
	}

	if len(result.Content) == 0 {
		t.Fatal("list_todos should return content")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content")
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse list_todos JSON: %v", err)
	}

	return response
}

func TestListTodosRegistered(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	server, err := NewServer(database)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	_, err = server.mcp.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer func() { _ = session.Close() }()

	// List tools to verify list_todos exists
	foundListTodos := false
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}
		if tool.Name == "list_todos" {
			foundListTodos = true
			if tool.Description == "" {
				t.Error("list_todos tool should have a description")
			}
			if tool.InputSchema == nil {
				t.Error("list_todos tool should have input schema")
			}
		}
	}

	if !foundListTodos {
		t.Error("list_todos tool was not registered")
	}
}

func TestListTodosNoFilters(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project := createTestProject(t, database)
	createTestTodoInDB(t, database, project.ID, "todo 1", nil, nil)
	createTestTodoInDB(t, database, project.ID, "todo 2", stringPtr("high"), nil)
	createTestTodoInDB(t, database, project.ID, "todo 3", stringPtr("low"), nil)

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_todos",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response := parseListTodosResult(t, result)

	todos, ok := response["todos"].([]interface{})
	if !ok {
		t.Fatal("Expected todos array")
	}

	if len(todos) != 3 {
		t.Errorf("Expected 3 todos, got %d", len(todos))
	}

	if response["count"].(float64) != 3 {
		t.Errorf("Expected count=3, got %v", response["count"])
	}
}

func TestListTodosFilterByProjectID(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project1 := createTestProject(t, database)
	project2 := createTestProject(t, database)
	createTestTodoInDB(t, database, project1.ID, "todo in project 1", nil, nil)
	createTestTodoInDB(t, database, project2.ID, "todo in project 2", nil, nil)

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_todos",
		Arguments: map[string]any{
			"project_id": project1.ID.String(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response := parseListTodosResult(t, result)

	todos, ok := response["todos"].([]interface{})
	if !ok {
		t.Fatal("Expected todos array")
	}

	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}

	todo := todos[0].(map[string]interface{})
	if todo["description"] != "todo in project 1" {
		t.Errorf("Expected 'todo in project 1', got %v", todo["description"])
	}
}

func TestListTodosFilterByDone(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project := createTestProject(t, database)
	_ = createTestTodoInDB(t, database, project.ID, "not done", nil, nil)
	todo2 := createTestTodoInDB(t, database, project.ID, "done", nil, nil)
	todo2.MarkDone()
	if err := db.UpdateTodo(database, todo2); err != nil {
		t.Fatalf("Failed to update todo: %v", err)
	}

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Test done=false
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_todos",
		Arguments: map[string]any{"done": false},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response := parseListTodosResult(t, result)
	todos := response["todos"].([]interface{})
	if len(todos) != 1 {
		t.Errorf("Expected 1 incomplete todo, got %d", len(todos))
	}

	// Test done=true
	result, err = ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_todos",
		Arguments: map[string]any{"done": true},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response = parseListTodosResult(t, result)
	todos = response["todos"].([]interface{})
	if len(todos) != 1 {
		t.Errorf("Expected 1 complete todo, got %d", len(todos))
	}
}

func TestListTodosFilterByPriority(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project := createTestProject(t, database)
	createTestTodoInDB(t, database, project.ID, "high priority", stringPtr("high"), nil)
	createTestTodoInDB(t, database, project.ID, "low priority", stringPtr("low"), nil)
	createTestTodoInDB(t, database, project.ID, "medium priority", stringPtr("medium"), nil)

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_todos",
		Arguments: map[string]any{
			"priority": "high",
		},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response := parseListTodosResult(t, result)
	todos := response["todos"].([]interface{})
	if len(todos) != 1 {
		t.Errorf("Expected 1 high priority todo, got %d", len(todos))
	}

	todo := todos[0].(map[string]interface{})
	if todo["priority"] != "high" {
		t.Errorf("Expected priority 'high', got %v", todo["priority"])
	}
}

func TestListTodosFilterByTag(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project := createTestProject(t, database)
	todo1 := createTestTodoInDB(t, database, project.ID, "bug fix", nil, nil)
	todo2 := createTestTodoInDB(t, database, project.ID, "feature", nil, nil)

	if err := db.AddTagToTodo(database, todo1.ID, "bug"); err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}
	if err := db.AddTagToTodo(database, todo2.ID, "feature"); err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_todos",
		Arguments: map[string]any{
			"tag": "bug",
		},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response := parseListTodosResult(t, result)
	todos := response["todos"].([]interface{})
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo with 'bug' tag, got %d", len(todos))
	}

	todo := todos[0].(map[string]interface{})
	if todo["description"] != "bug fix" {
		t.Errorf("Expected 'bug fix', got %v", todo["description"])
	}
}

func TestListTodosFilterByOverdue(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project := createTestProject(t, database)

	// Create overdue todo (due date in the past)
	pastDate := time.Now().Add(-24 * time.Hour)
	createTestTodoInDB(t, database, project.ID, "overdue task", nil, &pastDate)

	// Create future todo
	futureDate := time.Now().Add(24 * time.Hour)
	createTestTodoInDB(t, database, project.ID, "future task", nil, &futureDate)

	// Create todo with no due date
	createTestTodoInDB(t, database, project.ID, "no deadline", nil, nil)

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_todos",
		Arguments: map[string]any{
			"overdue": true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response := parseListTodosResult(t, result)
	todos := response["todos"].([]interface{})
	if len(todos) != 1 {
		t.Errorf("Expected 1 overdue todo, got %d", len(todos))
	}

	todo := todos[0].(map[string]interface{})
	if todo["description"] != "overdue task" {
		t.Errorf("Expected 'overdue task', got %v", todo["description"])
	}
}

func TestListTodosCombinedFilters(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project := createTestProject(t, database)

	// Create several todos with different attributes
	todo1 := createTestTodoInDB(t, database, project.ID, "high priority bug", stringPtr("high"), nil)
	if err := db.AddTagToTodo(database, todo1.ID, "bug"); err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}

	todo2 := createTestTodoInDB(t, database, project.ID, "low priority bug", stringPtr("low"), nil)
	if err := db.AddTagToTodo(database, todo2.ID, "bug"); err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}

	todo3 := createTestTodoInDB(t, database, project.ID, "high priority feature", stringPtr("high"), nil)
	if err := db.AddTagToTodo(database, todo3.ID, "feature"); err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_todos",
		Arguments: map[string]any{
			"priority": "high",
			"tag":      "bug",
			"done":     false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response := parseListTodosResult(t, result)
	todos := response["todos"].([]interface{})
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo matching all filters, got %d", len(todos))
	}

	todo := todos[0].(map[string]interface{})
	if todo["description"] != "high priority bug" {
		t.Errorf("Expected 'high priority bug', got %v", todo["description"])
	}
}

func TestListTodosEmptyResults(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	project := createTestProject(t, database)
	createTestTodoInDB(t, database, project.ID, "low priority", stringPtr("low"), nil)

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_todos",
		Arguments: map[string]any{
			"priority": "high", // no high priority todos exist
		},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	response := parseListTodosResult(t, result)
	todos := response["todos"].([]interface{})
	if len(todos) != 0 {
		t.Errorf("Expected 0 todos, got %d", len(todos))
	}

	if response["count"].(float64) != 0 {
		t.Errorf("Expected count=0, got %v", response["count"])
	}
}

func TestListTodosInvalidProjectID(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_todos",
		Arguments: map[string]any{
			"project_id": "not-a-uuid",
		},
	})
	if err != nil {
		t.Fatalf("Failed to call list_todos: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result when project_id is invalid")
	}
}

// Helper function to create a todo directly in the database.
func createTestTodoInDB(t *testing.T, database *sql.DB, projectID uuid.UUID, description string, priority *string, dueDate *time.Time) *models.Todo {
	t.Helper()
	todo := models.NewTodo(projectID, description)
	todo.Priority = priority
	todo.DueDate = dueDate
	if err := db.CreateTodo(database, todo); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}
	return todo
}

// Helper function to create a string pointer.
func stringPtr(s string) *string {
	return &s
}
