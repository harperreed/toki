// ABOUTME: Tests for todo CRUD operations
// ABOUTME: Verifies create, read, update, delete, list, and filtering for todos

package charm

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateTodo(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		ProjectName: "test-project",
		Description: "Test todo",
		Done:        false,
		Priority:    "medium",
		Tags:        []string{"test"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	retrieved, err := client.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if retrieved.Description != todo.Description {
		t.Errorf("Description mismatch")
	}
}

func TestListTodosWithFilters(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	projectID := uuid.New()

	// Create todos with different attributes
	todos := []*Todo{
		{ID: uuid.New(), ProjectID: projectID, ProjectName: "proj", Description: "Done high", Done: true, Priority: "high", Tags: []string{"urgent"}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), ProjectID: projectID, ProjectName: "proj", Description: "Not done medium", Done: false, Priority: "medium", Tags: []string{"backend"}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "other", Description: "Other project", Done: false, Priority: "low", Tags: []string{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}

	for _, todo := range todos {
		if err := client.CreateTodo(todo); err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}
	}

	// Test filter by project
	filter := &TodoFilter{ProjectID: &projectID}
	result, err := client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 todos for project, got %d", len(result))
	}

	// Test filter by done status
	done := false
	filter = &TodoFilter{Done: &done}
	result, err = client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 incomplete todos, got %d", len(result))
	}

	// Test filter by priority
	priority := "high"
	filter = &TodoFilter{Priority: &priority}
	result, err = client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 high priority todo, got %d", len(result))
	}

	// Test filter by tag
	tag := "urgent"
	filter = &TodoFilter{Tag: &tag}
	result, err = client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 todo with urgent tag, got %d", len(result))
	}
}

func TestGetTodoByPrefix(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	id := uuid.MustParse("abc12345-1234-1234-1234-123456789abc")
	todo := &Todo{
		ID:          id,
		ProjectID:   uuid.New(),
		ProjectName: "test",
		Description: "Test",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Should find by prefix
	retrieved, err := client.GetTodoByPrefix("abc1")
	if err != nil {
		t.Fatalf("failed to get by prefix: %v", err)
	}
	if retrieved.ID != id {
		t.Errorf("wrong todo returned")
	}
}

func TestMarkTodoDone(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		ProjectName: "test",
		Description: "Test",
		Done:        false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := client.MarkTodoDone(todo.ID, true); err != nil {
		t.Fatalf("failed to mark done: %v", err)
	}

	retrieved, err := client.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if !retrieved.Done {
		t.Error("todo should be done")
	}
	if retrieved.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestUpdateTodo(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		ProjectName: "test",
		Description: "Original description",
		Priority:    "low",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Update the todo
	todo.Description = "Updated description"
	todo.Priority = "high"
	if err := client.UpdateTodo(todo); err != nil {
		t.Fatalf("failed to update todo: %v", err)
	}

	retrieved, err := client.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if retrieved.Description != "Updated description" {
		t.Errorf("Description not updated")
	}
	if retrieved.Priority != "high" {
		t.Errorf("Priority not updated")
	}
}

func TestDeleteTodo(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		ProjectName: "test",
		Description: "To delete",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := client.DeleteTodo(todo.ID); err != nil {
		t.Fatalf("failed to delete todo: %v", err)
	}

	_, err := client.GetTodo(todo.ID)
	if err == nil {
		t.Error("expected error getting deleted todo")
	}
}

func TestAddTagToTodo(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		ProjectName: "test",
		Description: "Test",
		Tags:        []string{"existing"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := client.AddTagToTodo(todo.ID, "new-tag"); err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}

	retrieved, err := client.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if len(retrieved.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(retrieved.Tags))
	}

	hasTag := false
	for _, tag := range retrieved.Tags {
		if tag == "new-tag" {
			hasTag = true
			break
		}
	}
	if !hasTag {
		t.Error("new-tag not found in todo tags")
	}
}

func TestRemoveTagFromTodo(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		ProjectName: "test",
		Description: "Test",
		Tags:        []string{"tag1", "tag2", "tag3"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := client.RemoveTagFromTodo(todo.ID, "tag2"); err != nil {
		t.Fatalf("failed to remove tag: %v", err)
	}

	retrieved, err := client.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if len(retrieved.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(retrieved.Tags))
	}

	for _, tag := range retrieved.Tags {
		if tag == "tag2" {
			t.Error("tag2 should have been removed")
		}
	}
}

func TestListTodosOverdueFilter(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	now := time.Now().UTC()
	pastDue := now.Add(-24 * time.Hour)
	futureDue := now.Add(24 * time.Hour)

	todos := []*Todo{
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "test", Description: "Overdue", Done: false, DueDate: &pastDue, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "test", Description: "Future", Done: false, DueDate: &futureDue, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "test", Description: "No due date", Done: false, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "test", Description: "Done overdue", Done: true, DueDate: &pastDue, CreatedAt: now, UpdatedAt: now},
	}

	for _, todo := range todos {
		if err := client.CreateTodo(todo); err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}
	}

	overdue := true
	filter := &TodoFilter{Overdue: &overdue}
	result, err := client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 overdue todo, got %d", len(result))
	}
	if len(result) > 0 && result[0].Description != "Overdue" {
		t.Errorf("wrong overdue todo returned")
	}
}

func TestListTodosSortedByCreatedAt(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	now := time.Now().UTC()

	todos := []*Todo{
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "test", Description: "First", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now},
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "test", Description: "Third", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "test", Description: "Second", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now},
	}

	for _, todo := range todos {
		if err := client.CreateTodo(todo); err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}
	}

	result, err := client.ListTodos(nil)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 todos, got %d", len(result))
	}

	// Should be sorted by CreatedAt descending (most recent first)
	if result[0].Description != "Third" {
		t.Errorf("first todo should be 'Third', got %s", result[0].Description)
	}
	if result[1].Description != "Second" {
		t.Errorf("second todo should be 'Second', got %s", result[1].Description)
	}
	if result[2].Description != "First" {
		t.Errorf("third todo should be 'First', got %s", result[2].Description)
	}
}
