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
		Name:      "test-project",
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
