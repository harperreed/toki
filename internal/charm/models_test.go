// ABOUTME: Tests for Charm KV data models
// ABOUTME: Verifies JSON serialization/deserialization for projects, todos, and tags

package charm

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProjectSerialization(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	p := Project{
		ID:            id,
		Name:          "test-project",
		DirectoryPath: "/path/to/project",
		CreatedAt:     now,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal project: %v", err)
	}

	var decoded Project
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal project: %v", err)
	}

	if decoded.ID != p.ID {
		t.Errorf("ID mismatch: got %v, want %v", decoded.ID, p.ID)
	}
	if decoded.Name != p.Name {
		t.Errorf("Name mismatch: got %v, want %v", decoded.Name, p.Name)
	}
	if decoded.DirectoryPath != p.DirectoryPath {
		t.Errorf("DirectoryPath mismatch: got %v, want %v", decoded.DirectoryPath, p.DirectoryPath)
	}
	if !decoded.CreatedAt.Equal(p.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", decoded.CreatedAt, p.CreatedAt)
	}
}

func TestTodoSerialization(t *testing.T) {
	id := uuid.New()
	projectID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	dueDate := now.Add(24 * time.Hour)
	completedAt := now.Add(1 * time.Hour)

	todo := Todo{
		ID:          id,
		ProjectID:   projectID,
		ProjectName: "test-project",
		Description: "Test todo",
		Done:        true,
		Priority:    "high",
		Notes:       "Some notes",
		Tags:        []string{"urgent", "backend"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: &completedAt,
		DueDate:     &dueDate,
	}

	data, err := json.Marshal(todo)
	if err != nil {
		t.Fatalf("failed to marshal todo: %v", err)
	}

	var decoded Todo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal todo: %v", err)
	}

	if decoded.ID != todo.ID {
		t.Errorf("ID mismatch")
	}
	if decoded.ProjectID != todo.ProjectID {
		t.Errorf("ProjectID mismatch")
	}
	if decoded.ProjectName != todo.ProjectName {
		t.Errorf("ProjectName mismatch")
	}
	if len(decoded.Tags) != 2 || decoded.Tags[0] != "urgent" || decoded.Tags[1] != "backend" {
		t.Errorf("Tags mismatch: got %v", decoded.Tags)
	}
	if decoded.CompletedAt == nil || !decoded.CompletedAt.Equal(*todo.CompletedAt) {
		t.Errorf("CompletedAt mismatch")
	}
	if decoded.DueDate == nil || !decoded.DueDate.Equal(*todo.DueDate) {
		t.Errorf("DueDate mismatch")
	}
}

func TestTagSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tag := Tag{
		Name:      "urgent",
		CreatedAt: now,
	}

	data, err := json.Marshal(tag)
	if err != nil {
		t.Fatalf("failed to marshal tag: %v", err)
	}

	var decoded Tag
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal tag: %v", err)
	}

	if decoded.Name != tag.Name {
		t.Errorf("Name mismatch: got %v, want %v", decoded.Name, tag.Name)
	}
	if !decoded.CreatedAt.Equal(tag.CreatedAt) {
		t.Errorf("CreatedAt mismatch")
	}
}

func TestTodoWithNilOptionalFields(t *testing.T) {
	id := uuid.New()
	projectID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	todo := Todo{
		ID:          id,
		ProjectID:   projectID,
		ProjectName: "test-project",
		Description: "Test todo",
		Done:        false,
		Priority:    "medium",
		Tags:        []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: nil,
		DueDate:     nil,
	}

	data, err := json.Marshal(todo)
	if err != nil {
		t.Fatalf("failed to marshal todo: %v", err)
	}

	var decoded Todo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal todo: %v", err)
	}

	if decoded.CompletedAt != nil {
		t.Errorf("CompletedAt should be nil")
	}
	if decoded.DueDate != nil {
		t.Errorf("DueDate should be nil")
	}
}
