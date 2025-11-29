// ABOUTME: Tests for core data models
// ABOUTME: Validates Project, Todo, Tag struct behavior

package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewProject(t *testing.T) {
	name := "test-project"
	path := "/home/user/project"

	project := NewProject(name, &path)

	if project.ID == uuid.Nil {
		t.Error("Expected non-nil UUID")
	}
	if project.Name != name {
		t.Errorf("Expected name %s, got %s", name, project.Name)
	}
	if project.DirectoryPath == nil || *project.DirectoryPath != path {
		t.Errorf("Expected path %s, got %v", path, project.DirectoryPath)
	}
	if project.CreatedAt.IsZero() {
		t.Error("Expected non-zero created timestamp")
	}
}

func TestNewTodo(t *testing.T) {
	projectID := uuid.New()
	desc := "test todo"

	todo := NewTodo(projectID, desc)

	if todo.ID == uuid.Nil {
		t.Error("Expected non-nil UUID")
	}
	if todo.ProjectID != projectID {
		t.Error("Project ID mismatch")
	}
	if todo.Description != desc {
		t.Errorf("Expected description %s, got %s", desc, todo.Description)
	}
	if todo.Done {
		t.Error("New todo should not be done")
	}
	if todo.CreatedAt.IsZero() {
		t.Error("Expected non-zero created timestamp")
	}
}

func TestTodoMarkDone(t *testing.T) {
	todo := NewTodo(uuid.New(), "test")

	todo.MarkDone()

	if !todo.Done {
		t.Error("Todo should be marked done")
	}
	if todo.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestTodoMarkUndone(t *testing.T) {
	todo := NewTodo(uuid.New(), "test")
	todo.MarkDone()

	todo.MarkUndone()

	if todo.Done {
		t.Error("Todo should not be done")
	}
	if todo.CompletedAt != nil {
		t.Error("CompletedAt should be nil")
	}
}
