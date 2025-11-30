// ABOUTME: Tests for tag database operations
// ABOUTME: Covers tag creation, retrieval, and todo-tag associations

package db

import (
	"testing"

	"github.com/harper/toki/internal/models"
)

func TestGetOrCreateTag(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

	project := models.NewProject("test", nil)
	if err := CreateProject(db, project); err != nil {
		t.Fatal(err)
	}

	todo := models.NewTodo(project.ID, "test")
	if err := CreateTodo(db, todo); err != nil {
		t.Fatal(err)
	}

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
	defer func() { _ = db.Close() }()

	project := models.NewProject("test", nil)
	if err := CreateProject(db, project); err != nil {
		t.Fatal(err)
	}

	todo := models.NewTodo(project.ID, "test")
	if err := CreateTodo(db, todo); err != nil {
		t.Fatal(err)
	}

	if err := AddTagToTodo(db, todo.ID, "frontend"); err != nil {
		t.Fatal(err)
	}

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
	defer func() { _ = db.Close() }()

	if _, err := GetOrCreateTag(db, "tag1"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetOrCreateTag(db, "tag2"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetOrCreateTag(db, "tag3"); err != nil {
		t.Fatal(err)
	}

	tags, err := ListAllTags(db)
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}

	if len(tags) < 3 {
		t.Errorf("Expected at least 3 tags, got %d", len(tags))
	}
}
