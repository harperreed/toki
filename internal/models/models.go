// ABOUTME: Core data models for projects, todos, and tags
// ABOUTME: Provides constructor functions and business logic methods

package models

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a collection of todos
type Project struct {
	ID            uuid.UUID
	Name          string
	DirectoryPath *string
	CreatedAt     time.Time
}

// Todo represents a single task
type Todo struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Description string
	Done        bool
	Priority    *string
	Notes       *string
	CreatedAt   time.Time
	CompletedAt *time.Time
	DueDate     *time.Time
}

// Tag represents a label that can be applied to todos
type Tag struct {
	ID   int64
	Name string
}

// TodoTag represents the many-to-many relationship
type TodoTag struct {
	TodoID uuid.UUID
	TagID  int64
}

// NewProject creates a new project with generated UUID and timestamp
func NewProject(name string, directoryPath *string) *Project {
	return &Project{
		ID:            uuid.New(),
		Name:          name,
		DirectoryPath: directoryPath,
		CreatedAt:     time.Now(),
	}
}

// NewTodo creates a new todo with generated UUID and timestamp
func NewTodo(projectID uuid.UUID, description string) *Todo {
	return &Todo{
		ID:          uuid.New(),
		ProjectID:   projectID,
		Description: description,
		Done:        false,
		CreatedAt:   time.Now(),
	}
}

// MarkDone marks a todo as complete
func (t *Todo) MarkDone() {
	t.Done = true
	now := time.Now()
	t.CompletedAt = &now
}

// MarkUndone marks a todo as incomplete
func (t *Todo) MarkUndone() {
	t.Done = false
	t.CompletedAt = nil
}
