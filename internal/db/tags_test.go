// ABOUTME: Tests for tag database operations
// ABOUTME: Covers tag creation, retrieval, and todo-tag associations

package db

import (
	"testing"

	"github.com/harper/toki/internal/models"
)

func TestGetOrCreateTag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tag, err := GetOrCreateTag(db, "urgent")
	if err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	if tag.Name != "urgent" {
		t.Errorf("Expected name 'urgent', got %s", tag.Name)
	}

	// Getting same tag should return same ID
	tag2, err := GetOrCreateTag(db, "urgent")
	if err != nil {
		t.Fatalf("Failed to get existing tag: %v", err)
	}

	if tag2.ID != tag.ID {
		t.Error("Should return same tag ID")
	}
}

func TestAddTagToTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "test")
	CreateTodo(db, todo)

	err := AddTagToTodo(db, todo.ID, "backend")
	if err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}

	tags, err := GetTodoTags(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to get tags: %v", err)
	}

	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}

	if tags[0].Name != "backend" {
		t.Errorf("Expected tag 'backend', got %s", tags[0].Name)
	}
}

func TestRemoveTagFromTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "test")
	CreateTodo(db, todo)

	AddTagToTodo(db, todo.ID, "frontend")

	err := RemoveTagFromTodo(db, todo.ID, "frontend")
	if err != nil {
		t.Fatalf("Failed to remove tag: %v", err)
	}

	tags, err := GetTodoTags(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to get tags: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(tags))
	}
}

func TestListAllTags(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	GetOrCreateTag(db, "tag1")
	GetOrCreateTag(db, "tag2")
	GetOrCreateTag(db, "tag3")

	tags, err := ListAllTags(db)
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}

	if len(tags) < 3 {
		t.Errorf("Expected at least 3 tags, got %d", len(tags))
	}
}
