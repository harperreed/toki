// ABOUTME: Tests for MCP resource providers
// ABOUTME: Verifies resource URIs, metadata, data structure, and query parameters

package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// testResourceResponse represents the standard response format for all resources (for testing).
type testResourceResponse struct {
	Metadata ResourceMetadata  `json:"metadata"`
	Data     json.RawMessage   `json:"data"`
	Links    map[string]string `json:"links"`
}

func readResource(t *testing.T, ts *testSession, uri string) testResourceResponse {
	t.Helper()

	readResult, err := ts.session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("Failed to read resource %s: %v", uri, err)
	}
	if len(readResult.Contents) == 0 {
		t.Fatalf("No contents returned for resource %s", uri)
	}

	var resp testResourceResponse
	if err := json.Unmarshal([]byte(readResult.Contents[0].Text), &resp); err != nil {
		t.Fatalf("Failed to parse response from %s: %v", uri, err)
	}

	return resp
}

func TestResourceProjects(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	session := setupTestSession(t, database)
	defer session.cleanup()

	// Create test projects
	proj1 := models.NewProject("project-alpha", nil)
	if err := db.CreateProject(database, proj1); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	path := "/tmp/test"
	proj2 := models.NewProject("project-beta", &path)
	if err := db.CreateProject(database, proj2); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Read the resource
	resp := readResource(t, session, "toki://projects")

	// Verify metadata
	if resp.Metadata.ResourceURI != "toki://projects" {
		t.Errorf("Expected resource_uri 'toki://projects', got %s", resp.Metadata.ResourceURI)
	}
	if resp.Metadata.Count != 2 {
		t.Errorf("Expected count 2, got %d", resp.Metadata.Count)
	}

	// Verify data structure
	var projects []map[string]any
	if err := json.Unmarshal(resp.Data, &projects); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	// Verify links exist
	if len(resp.Links) == 0 {
		t.Error("Expected links to be present")
	}
}

func TestResourceTodosAll(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	session := setupTestSession(t, database)
	defer session.cleanup()

	// Create test project and todos
	proj := models.NewProject("test-project", nil)
	if err := db.CreateProject(database, proj); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	todo1 := models.NewTodo(proj.ID, "task one")
	if err := db.CreateTodo(database, todo1); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	todo2 := models.NewTodo(proj.ID, "task two")
	todo2.MarkDone()
	if err := db.CreateTodo(database, todo2); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// Read the resource
	resp := readResource(t, session, "toki://todos")

	// Verify metadata
	if resp.Metadata.ResourceURI != "toki://todos" {
		t.Errorf("Expected resource_uri 'toki://todos', got %s", resp.Metadata.ResourceURI)
	}
	if resp.Metadata.Count != 2 {
		t.Errorf("Expected count 2, got %d", resp.Metadata.Count)
	}

	// Verify data
	var todos []map[string]any
	if err := json.Unmarshal(resp.Data, &todos); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
}

func TestResourceTodosPending(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	session := setupTestSession(t, database)
	defer session.cleanup()

	// Create test project and todos
	proj := models.NewProject("test-project", nil)
	if err := db.CreateProject(database, proj); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	todo1 := models.NewTodo(proj.ID, "pending task")
	if err := db.CreateTodo(database, todo1); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	todo2 := models.NewTodo(proj.ID, "completed task")
	todo2.MarkDone()
	if err := db.CreateTodo(database, todo2); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// Read the resource
	resp := readResource(t, session, "toki://todos/pending")

	// Verify metadata
	if resp.Metadata.ResourceURI != "toki://todos/pending" {
		t.Errorf("Expected resource_uri 'toki://todos/pending', got %s", resp.Metadata.ResourceURI)
	}
	if resp.Metadata.Count != 1 {
		t.Errorf("Expected count 1, got %d", resp.Metadata.Count)
	}
	if done, ok := resp.Metadata.Filters["done"].(bool); !ok || done {
		t.Errorf("Expected done=false filter, got %v", resp.Metadata.Filters["done"])
	}

	// Verify only pending todos returned
	var todos []map[string]any
	if err := json.Unmarshal(resp.Data, &todos); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0]["description"] != "pending task" {
		t.Errorf("Expected 'pending task', got %s", todos[0]["description"])
	}
}

func TestResourceTodosOverdue(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	session := setupTestSession(t, database)
	defer session.cleanup()

	// Create test project and todos
	proj := models.NewProject("test-project", nil)
	if err := db.CreateProject(database, proj); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Overdue todo (due yesterday)
	yesterday := time.Now().Add(-24 * time.Hour)
	todo1 := models.NewTodo(proj.ID, "overdue task")
	todo1.DueDate = &yesterday
	if err := db.CreateTodo(database, todo1); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// Future todo
	tomorrow := time.Now().Add(24 * time.Hour)
	todo2 := models.NewTodo(proj.ID, "future task")
	todo2.DueDate = &tomorrow
	if err := db.CreateTodo(database, todo2); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// No due date todo
	todo3 := models.NewTodo(proj.ID, "no due date")
	if err := db.CreateTodo(database, todo3); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// Read the resource
	resp := readResource(t, session, "toki://todos/overdue")

	// Verify metadata
	if resp.Metadata.ResourceURI != "toki://todos/overdue" {
		t.Errorf("Expected resource_uri 'toki://todos/overdue', got %s", resp.Metadata.ResourceURI)
	}
	if resp.Metadata.Count != 1 {
		t.Errorf("Expected count 1, got %d", resp.Metadata.Count)
	}

	// Verify only overdue todos returned
	var todos []map[string]any
	if err := json.Unmarshal(resp.Data, &todos); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0]["description"] != "overdue task" {
		t.Errorf("Expected 'overdue task', got %s", todos[0]["description"])
	}
}

func TestResourceTodosHighPriority(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	session := setupTestSession(t, database)
	defer session.cleanup()

	// Create test project and todos
	proj := models.NewProject("test-project", nil)
	if err := db.CreateProject(database, proj); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	high := "high"
	todo1 := models.NewTodo(proj.ID, "high priority task")
	todo1.Priority = &high
	if err := db.CreateTodo(database, todo1); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	medium := "medium"
	todo2 := models.NewTodo(proj.ID, "medium priority task")
	todo2.Priority = &medium
	if err := db.CreateTodo(database, todo2); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	todo3 := models.NewTodo(proj.ID, "no priority task")
	if err := db.CreateTodo(database, todo3); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// Read the resource
	resp := readResource(t, session, "toki://todos/high-priority")

	// Verify metadata
	if resp.Metadata.ResourceURI != "toki://todos/high-priority" {
		t.Errorf("Expected resource_uri 'toki://todos/high-priority', got %s", resp.Metadata.ResourceURI)
	}
	if resp.Metadata.Count != 1 {
		t.Errorf("Expected count 1, got %d", resp.Metadata.Count)
	}
	if priority, ok := resp.Metadata.Filters["priority"].(string); !ok || priority != "high" {
		t.Errorf("Expected priority=high filter, got %v", resp.Metadata.Filters["priority"])
	}

	// Verify only high priority todos returned
	var todos []map[string]any
	if err := json.Unmarshal(resp.Data, &todos); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	if todos[0]["description"] != "high priority task" {
		t.Errorf("Expected 'high priority task', got %s", todos[0]["description"])
	}
}

func TestResourceQuery(t *testing.T) {
	// NOTE: In v1, the query resource doesn't support URL parameters due to MCP SDK limitations.
	// It returns all todos, same as toki://todos.
	// For filtered views, use pre-built resources like toki://todos/pending or toki://todos/high-priority.
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	session := setupTestSession(t, database)
	defer session.cleanup()

	// Create test project and todos
	proj := models.NewProject("test-project", nil)
	if err := db.CreateProject(database, proj); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	high := "high"
	todo1 := models.NewTodo(proj.ID, "high priority task")
	todo1.Priority = &high
	if err := db.CreateTodo(database, todo1); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	low := "low"
	todo2 := models.NewTodo(proj.ID, "low priority task")
	todo2.Priority = &low
	if err := db.CreateTodo(database, todo2); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// Query base resource (returns all todos in v1)
	resp := readResource(t, session, "toki://query")

	// Verify it returns all todos
	if resp.Metadata.Count != 2 {
		t.Errorf("Expected count 2, got %d", resp.Metadata.Count)
	}

	var todos []map[string]any
	if err := json.Unmarshal(resp.Data, &todos); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
}

func TestResourceEmptyResults(t *testing.T) {
	// Test that resources handle empty data gracefully
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	session := setupTestSession(t, database)
	defer session.cleanup()

	// No todos exist yet, so todos resource should return empty array
	resp := readResource(t, session, "toki://todos")

	// Verify empty results handled gracefully
	if resp.Metadata.Count != 0 {
		t.Errorf("Expected count 0, got %d", resp.Metadata.Count)
	}

	var todos []map[string]any
	if err := json.Unmarshal(resp.Data, &todos); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("Expected 0 todos, got %d", len(todos))
	}
}

func TestResourceLinksPresent(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	session := setupTestSession(t, database)
	defer session.cleanup()

	// Create test data
	proj := models.NewProject("test-project", nil)
	if err := db.CreateProject(database, proj); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	high := "high"
	todo1 := models.NewTodo(proj.ID, "high priority task")
	todo1.Priority = &high
	if err := db.CreateTodo(database, todo1); err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// Check links in pre-built resource
	resp := readResource(t, session, "toki://todos/high-priority")

	// Verify links section exists and has content
	if len(resp.Links) == 0 {
		t.Error("Expected links to be present")
	}
	if _, ok := resp.Links["all_todos"]; !ok {
		t.Error("Expected 'all_todos' link")
	}
	if _, ok := resp.Links["query"]; !ok {
		t.Error("Expected 'query' link")
	}
}

// Note: TestResourceInvalidQueryParameter and TestResourceInvalidPriority removed
// because query parameters aren't supported in v1 due to MCP SDK limitations.
// For custom filtering, use the list_todos tool instead of resources.
