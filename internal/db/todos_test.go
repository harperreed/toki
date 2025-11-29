// ABOUTME: Tests for todo database operations
// ABOUTME: Covers CRUD, filtering, and UUID prefix matching

package db

import (
	"testing"

	"github.com/harper/toki/internal/models"
)

func TestCreateTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "test todo")
	priority := "high"
	todo.Priority = &priority

	err := CreateTodo(db, todo)
	if err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	retrieved, err := GetTodoByID(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve todo: %v", err)
	}

	if retrieved.Description != todo.Description {
		t.Errorf("Description mismatch")
	}
	if *retrieved.Priority != *todo.Priority {
		t.Errorf("Priority mismatch")
	}
}

func TestGetTodoByPrefix(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "find me")
	CreateTodo(db, todo)

	// Use first 6 characters as prefix
	prefix := todo.ID.String()[:6]

	found, err := GetTodoByPrefix(db, prefix)
	if err != nil {
		t.Fatalf("Failed to find todo by prefix: %v", err)
	}

	if found.ID != todo.ID {
		t.Error("Wrong todo found")
	}
}

func TestGetTodoByPrefixAmbiguous(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	// Create two todos - we can't guarantee prefix collision
	// but we can test the error path
	todo1 := models.NewTodo(project.ID, "todo1")
	CreateTodo(db, todo1)

	// Test with empty prefix should be ambiguous
	_, err := GetTodoByPrefix(db, "")
	if err == nil {
		t.Error("Empty prefix should return error")
	}
}

func TestListTodos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo1 := models.NewTodo(project.ID, "todo1")
	todo2 := models.NewTodo(project.ID, "todo2")
	CreateTodo(db, todo1)
	CreateTodo(db, todo2)

	todos, err := ListTodos(db, &project.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to list todos: %v", err)
	}

	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
}

func TestListTodosFilterDone(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo1 := models.NewTodo(project.ID, "pending")
	todo2 := models.NewTodo(project.ID, "done")
	todo2.MarkDone()

	CreateTodo(db, todo1)
	CreateTodo(db, todo2)

	doneFilter := false
	todos, err := ListTodos(db, nil, &doneFilter, nil, nil)
	if err != nil {
		t.Fatalf("Failed to list todos: %v", err)
	}

	if len(todos) != 1 {
		t.Errorf("Expected 1 pending todo, got %d", len(todos))
	}
	if todos[0].Done {
		t.Error("Should only return pending todos")
	}
}

func TestListTodosFilterByTag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo1 := models.NewTodo(project.ID, "backend todo")
	todo2 := models.NewTodo(project.ID, "frontend todo")
	CreateTodo(db, todo1)
	CreateTodo(db, todo2)

	// Tag only todo1 with "backend"
	AddTagToTodo(db, todo1.ID, "backend")
	AddTagToTodo(db, todo2.ID, "frontend")

	// Filter by backend tag
	tagFilter := "backend"
	todos, err := ListTodos(db, nil, nil, nil, &tagFilter)
	if err != nil {
		t.Fatalf("Failed to list todos by tag: %v", err)
	}

	if len(todos) != 1 {
		t.Errorf("Expected 1 todo with backend tag, got %d", len(todos))
	}

	if todos[0].Description != "backend todo" {
		t.Error("Wrong todo returned for tag filter")
	}
}

func TestUpdateTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "original")
	CreateTodo(db, todo)

	todo.Description = "updated"
	todo.MarkDone()

	err := UpdateTodo(db, todo)
	if err != nil {
		t.Fatalf("Failed to update todo: %v", err)
	}

	retrieved, err := GetTodoByID(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve todo: %v", err)
	}

	if retrieved.Description != "updated" {
		t.Error("Description not updated")
	}
	if !retrieved.Done {
		t.Error("Done status not updated")
	}
}

func TestDeleteTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "to delete")
	CreateTodo(db, todo)

	err := DeleteTodo(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to delete todo: %v", err)
	}

	_, err = GetTodoByID(db, todo.ID)
	if err == nil {
		t.Error("Todo should not exist after deletion")
	}
}
