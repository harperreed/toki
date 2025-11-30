// ABOUTME: End-to-end integration tests for MCP server
// ABOUTME: Validates server initialization, capability registration, and real-world usage scenarios

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

// TestServerInitialization verifies that the MCP server initializes correctly.
func TestServerInitialization(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	server, err := NewServer(database)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.mcp == nil {
		t.Fatal("Expected MCP server to be initialized")
	}

	if server.db == nil {
		t.Fatal("Expected database to be set")
	}
}

// TestServerInitializationWithNilDB verifies that server rejects nil database.
func TestServerInitializationWithNilDB(t *testing.T) {
	_, err := NewServer(nil)
	if err == nil {
		t.Error("Expected error when creating server with nil database")
	}
}

// TestAllToolsRegistered verifies that all 11 expected tools are registered.
func TestAllToolsRegistered(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Expected tools
	expectedTools := map[string]bool{
		"add_todo":             false,
		"list_todos":           false,
		"mark_done":            false,
		"mark_undone":          false,
		"delete_todo":          false,
		"update_todo":          false,
		"add_tag_to_todo":      false,
		"remove_tag_from_todo": false,
		"add_project":          false,
		"list_projects":        false,
		"delete_project":       false,
	}

	// List all tools
	for tool, err := range ts.session.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}
		if _, expected := expectedTools[tool.Name]; expected {
			expectedTools[tool.Name] = true
			// Verify each tool has required metadata
			if tool.Description == "" {
				t.Errorf("Tool %s should have a description", tool.Name)
			}
			if tool.InputSchema == nil {
				t.Errorf("Tool %s should have input schema", tool.Name)
			}
		}
	}

	// Check that all tools were found
	for name, found := range expectedTools {
		if !found {
			t.Errorf("Expected tool '%s' was not registered", name)
		}
	}
}

// TestAllResourcesRegistered verifies that all 6 expected resources are registered.
func TestAllResourcesRegistered(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Expected resources (based on actual registration in resources.go)
	expectedResources := map[string]bool{
		"toki://projects":            false,
		"toki://todos":               false,
		"toki://todos/pending":       false,
		"toki://todos/overdue":       false,
		"toki://todos/high-priority": false,
		"toki://query":               false,
		"toki://stats":               false,
	}

	// List all resources
	resourcesResp, err := ts.session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}

	for _, resource := range resourcesResp.Resources {
		if _, expected := expectedResources[resource.URI]; expected {
			expectedResources[resource.URI] = true
			// Verify each resource has required metadata
			if resource.Name == "" {
				t.Errorf("Resource %s should have a name", resource.URI)
			}
			if resource.Description == "" {
				t.Errorf("Resource %s should have a description", resource.URI)
			}
		}
	}

	// Check that all resources were found
	for uri, found := range expectedResources {
		if !found {
			t.Errorf("Expected resource '%s' was not registered", uri)
		}
	}
}

// TestAllPromptsRegistered verifies that all 6 expected prompts are registered.
func TestAllPromptsRegistered(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Expected prompts
	expectedPrompts := map[string]bool{
		"plan-project":     false,
		"daily-review":     false,
		"sprint-planning":  false,
		"track-agent-work": false,
		"coordinate-tasks": false,
		"report-status":    false,
	}

	// List all prompts
	promptsResp, err := ts.session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		t.Fatalf("Failed to list prompts: %v", err)
	}

	for _, prompt := range promptsResp.Prompts {
		if _, expected := expectedPrompts[prompt.Name]; expected {
			expectedPrompts[prompt.Name] = true
			// Verify each prompt has required metadata
			if prompt.Description == "" {
				t.Errorf("Prompt %s should have a description", prompt.Name)
			}
		}
	}

	// Check that all prompts were found
	for name, found := range expectedPrompts {
		if !found {
			t.Errorf("Expected prompt '%s' was not registered", name)
		}
	}
}

//nolint:funlen // Integration test requires multiple steps for end-to-end workflow validation
func TestEndToEndWorkflowCreateTodoViaToolReadViaResource(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Step 1: Create a project using add_project tool
	projectResult, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_project",
		Arguments: map[string]any{
			"name": "integration-test-project",
			"path": "/tmp/integration-test",
		},
	})
	if err != nil {
		t.Fatalf("Failed to call add_project: %v", err)
	}

	project := parseToolResult(t, projectResult)
	projectID := project["id"].(string)

	// Step 2: Create a todo using add_todo tool
	todoResult, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_todo",
		Arguments: map[string]any{
			"description": "implement integration tests",
			"project_id":  projectID,
			"priority":    "high",
			"tags":        []string{"testing", "integration"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to call add_todo: %v", err)
	}

	todo := parseToolResult(t, todoResult)
	todoID := todo["id"].(string)

	// Step 3: Read todos via resource (should include our new todo)
	resourceResp, err := ts.session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "toki://todos",
	})
	if err != nil {
		t.Fatalf("Failed to read todos resource: %v", err)
	}

	if len(resourceResp.Contents) == 0 {
		t.Fatal("Expected resource contents")
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(resourceResp.Contents[0].Text), &response); err != nil {
		t.Fatalf("Failed to parse resource response: %v", err)
	}

	data := response["data"].([]any)
	found := false
	for _, item := range data {
		todoItem := item.(map[string]any)
		if todoItem["id"].(string) == todoID {
			found = true
			if todoItem["description"] != "implement integration tests" {
				t.Errorf("Expected description 'implement integration tests', got %v", todoItem["description"])
			}
			if todoItem["priority"] != "high" {
				t.Errorf("Expected priority 'high', got %v", todoItem["priority"])
			}
		}
	}

	if !found {
		t.Error("Todo created via tool should be visible in resource")
	}

	// Step 4: Mark todo as done using mark_done tool
	_, err = ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "mark_done",
		Arguments: map[string]any{"todo_id": todoID},
	})
	if err != nil {
		t.Fatalf("Failed to call mark_done: %v", err)
	}

	// Step 5: Verify todo is marked as done in all todos resource
	allResp, err := ts.session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "toki://todos",
	})
	if err != nil {
		t.Fatalf("Failed to read todos resource: %v", err)
	}

	var allResponse map[string]any
	if err := json.Unmarshal([]byte(allResp.Contents[0].Text), &allResponse); err != nil {
		t.Fatalf("Failed to parse todos resource response: %v", err)
	}

	allData := allResponse["data"].([]any)
	foundDone := false
	for _, item := range allData {
		todoItem := item.(map[string]any)
		if todoItem["id"].(string) == todoID {
			foundDone = true
			if !todoItem["done"].(bool) {
				t.Error("Todo should be marked as done")
			}
		}
	}

	if !foundDone {
		t.Error("Done todo should appear in todos resource")
	}
}

//nolint:funlen // Integration test requires multiple steps for complete lifecycle validation
func TestEndToEndProjectWorkflow(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Step 1: List projects (should be empty initially)
	initialResult, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_projects",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Failed to call list_projects: %v", err)
	}

	initialList := parseToolResult(t, initialResult)
	initialCount := len(initialList["projects"].([]any))

	// Step 2: Create a project
	createResult, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_project",
		Arguments: map[string]any{
			"name": "workflow-test-project",
		},
	})
	if err != nil {
		t.Fatalf("Failed to call add_project: %v", err)
	}

	project := parseToolResult(t, createResult)
	projectID := project["id"].(string)

	// Step 3: Read projects resource
	resourceResp, err := ts.session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "toki://projects",
	})
	if err != nil {
		t.Fatalf("Failed to read projects resource: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(resourceResp.Contents[0].Text), &response); err != nil {
		t.Fatalf("Failed to parse resource response: %v", err)
	}

	projects := response["data"].([]any)
	if len(projects) != initialCount+1 {
		t.Errorf("Expected %d projects, got %d", initialCount+1, len(projects))
	}

	// Step 4: Delete the project
	_, err = ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "delete_project",
		Arguments: map[string]any{
			"project_id": projectID,
		},
	})
	if err != nil {
		t.Fatalf("Failed to call delete_project: %v", err)
	}

	// Step 5: Verify project is gone
	finalResult, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_projects",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Failed to call list_projects: %v", err)
	}

	finalList := parseToolResult(t, finalResult)
	finalCount := len(finalList["projects"].([]any))

	if finalCount != initialCount {
		t.Errorf("Expected %d projects after deletion, got %d", initialCount, finalCount)
	}
}

// TestPromptRetrieval tests that prompts can be retrieved and contain expected content.
func TestPromptRetrieval(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Get plan-project prompt
	result, err := ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "plan-project",
	})
	if err != nil {
		t.Fatalf("Failed to get plan-project prompt: %v", err)
	}

	if result == nil {
		t.Fatal("Expected prompt result, got nil")
	}

	if len(result.Messages) == 0 {
		t.Fatal("Expected prompt messages, got empty array")
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	content := textContent.Text
	if content == "" {
		t.Fatal("Expected prompt content, got empty string")
	}

	// Prompt should contain markdown and workflow information
	if len(content) < 100 {
		t.Error("Prompt content seems too short")
	}
}

// TestDatabaseIntegrationToolsPersistData verifies tools actually write to database.
func TestDatabaseIntegrationToolsPersistData(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Create project via tool
	projectResult, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_project",
		Arguments: map[string]any{"name": "db-test-project"},
	})
	if err != nil {
		t.Fatalf("Failed to call add_project: %v", err)
	}

	project := parseToolResult(t, projectResult)
	projectIDStr := project["id"].(string)
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		t.Fatalf("Failed to parse project ID: %v", err)
	}

	// Verify project exists in database directly
	dbProject, err := db.GetProjectByID(database, projectID)
	if err != nil {
		t.Fatalf("Project should exist in database: %v", err)
	}

	if dbProject.Name != "db-test-project" {
		t.Errorf("Expected project name 'db-test-project', got %s", dbProject.Name)
	}

	// Create todo via tool
	todoResult, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_todo",
		Arguments: map[string]any{
			"description": "test database persistence",
			"project_id":  projectIDStr,
		},
	})
	if err != nil {
		t.Fatalf("Failed to call add_todo: %v", err)
	}

	todo := parseToolResult(t, todoResult)
	todoIDStr := todo["id"].(string)
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		t.Fatalf("Failed to parse todo ID: %v", err)
	}

	// Verify todo exists in database directly
	dbTodo, err := db.GetTodoByID(database, todoID)
	if err != nil {
		t.Fatalf("Todo should exist in database: %v", err)
	}

	if dbTodo.Description != "test database persistence" {
		t.Errorf("Expected todo description 'test database persistence', got %s", dbTodo.Description)
	}
}

//nolint:funlen // Integration test requires setup and verification of concurrent database access
func TestConcurrentAccessCLIAndMCP(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "shared.db")

	// Initialize database as CLI would
	cliDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init CLI database: %v", err)
	}

	// Create project via CLI database connection
	cliProject := models.NewProject("cli-created-project", nil)
	if err := db.CreateProject(cliDB, cliProject); err != nil {
		t.Fatalf("Failed to create project via CLI: %v", err)
	}

	// Close CLI connection
	if err := cliDB.Close(); err != nil {
		t.Fatalf("Failed to close CLI database: %v", err)
	}

	// Open database as MCP server would
	mcpDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open MCP database: %v", err)
	}
	defer func() { _ = mcpDB.Close() }()

	// Create MCP server with same database
	ts := setupTestSession(t, mcpDB)
	defer ts.cleanup()

	ctx := context.Background()

	// Read projects via MCP resource - should see CLI-created project
	resourceResp, err := ts.session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "toki://projects",
	})
	if err != nil {
		t.Fatalf("Failed to read projects resource: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(resourceResp.Contents[0].Text), &response); err != nil {
		t.Fatalf("Failed to parse resource response: %v", err)
	}

	projects := response["data"].([]any)
	found := false
	for _, item := range projects {
		projectItem := item.(map[string]any)
		if projectItem["name"].(string) == "cli-created-project" {
			found = true
		}
	}

	if !found {
		t.Error("MCP server should see project created by CLI")
	}

	// Create todo via MCP
	todoResult, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_todo",
		Arguments: map[string]any{
			"description": "mcp-created-todo",
			"project_id":  cliProject.ID.String(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to call add_todo: %v", err)
	}

	todo := parseToolResult(t, todoResult)
	todoIDStr := todo["id"].(string)
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		t.Fatalf("Failed to parse todo ID: %v", err)
	}

	// Reopen database as CLI and verify todo exists
	cliDB2, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen CLI database: %v", err)
	}
	defer func() { _ = cliDB2.Close() }()

	dbTodo, err := db.GetTodoByID(cliDB2, todoID)
	if err != nil {
		t.Fatalf("CLI should see todo created by MCP: %v", err)
	}

	if dbTodo.Description != "mcp-created-todo" {
		t.Errorf("Expected description 'mcp-created-todo', got %s", dbTodo.Description)
	}
}

//nolint:funlen // Integration test requires creating varied test data and verifying all stats fields
func TestStatsResource(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Create project and various todos
	project := createTestProject(t, database)

	// Create 3 pending todos
	createTestTodoInDB(t, database, project.ID, "pending 1", nil, nil)
	createTestTodoInDB(t, database, project.ID, "pending 2", nil, nil)
	createTestTodoInDB(t, database, project.ID, "pending 3", nil, nil)

	// Create 2 done todos
	doneTodo1 := createTestTodoInDB(t, database, project.ID, "done 1", nil, nil)
	doneTodo1.MarkDone()
	if err := db.UpdateTodo(database, doneTodo1); err != nil {
		t.Fatalf("Failed to mark todo done: %v", err)
	}

	doneTodo2 := createTestTodoInDB(t, database, project.ID, "done 2", nil, nil)
	doneTodo2.MarkDone()
	if err := db.UpdateTodo(database, doneTodo2); err != nil {
		t.Fatalf("Failed to mark todo done: %v", err)
	}

	// Create 1 overdue todo
	pastDate := time.Now().Add(-24 * time.Hour)
	createTestTodoInDB(t, database, project.ID, "overdue", nil, &pastDate)

	// Read stats resource
	resourceResp, err := ts.session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "toki://stats",
	})
	if err != nil {
		t.Fatalf("Failed to read stats resource: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(resourceResp.Contents[0].Text), &response); err != nil {
		t.Fatalf("Failed to parse stats response: %v", err)
	}

	statsData := response["data"].(map[string]any)
	summary := statsData["summary"].(map[string]any)

	// Verify counts in summary
	if summary["total_todos"].(float64) != 6 {
		t.Errorf("Expected 6 total todos, got %v", summary["total_todos"])
	}
	if summary["pending"].(float64) != 4 {
		t.Errorf("Expected 4 pending todos, got %v", summary["pending"])
	}
	if summary["completed"].(float64) != 2 {
		t.Errorf("Expected 2 completed todos, got %v", summary["completed"])
	}
	if summary["overdue"].(float64) != 1 {
		t.Errorf("Expected 1 overdue todo, got %v", summary["overdue"])
	}

	// Verify by_project exists and has data
	byProject := statsData["by_project"].([]any)
	if len(byProject) != 1 {
		t.Errorf("Expected 1 project in stats, got %d", len(byProject))
	}
}

// TestInvalidToolCall verifies proper error handling.
func TestInvalidToolCall(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Call tool with invalid UUID
	result, err := ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "mark_done",
		Arguments: map[string]any{"todo_id": "not-a-valid-uuid"},
	})
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for invalid UUID")
	}

	// Call tool with non-existent resource
	result, err = ts.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "mark_done",
		Arguments: map[string]any{"todo_id": uuid.New().String()},
	})
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for non-existent todo")
	}
}

// TestOverdueResource tests the overdue todos resource.
func TestOverdueResource(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	project := createTestProject(t, database)

	// Create overdue todo
	pastDate := time.Now().Add(-48 * time.Hour)
	overdueTodo := createTestTodoInDB(t, database, project.ID, "overdue task", nil, &pastDate)

	// Create future todo
	futureDate := time.Now().Add(48 * time.Hour)
	createTestTodoInDB(t, database, project.ID, "future task", nil, &futureDate)

	// Read overdue resource
	resourceResp, err := ts.session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "toki://todos/overdue",
	})
	if err != nil {
		t.Fatalf("Failed to read overdue resource: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(resourceResp.Contents[0].Text), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	todos := response["data"].([]any)
	if len(todos) != 1 {
		t.Errorf("Expected 1 overdue todo, got %d", len(todos))
	}

	if len(todos) > 0 {
		todo := todos[0].(map[string]any)
		if todo["id"].(string) != overdueTodo.ID.String() {
			t.Error("Expected overdue todo to be the one we created")
		}
	}
}
