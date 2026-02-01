// ABOUTME: Storage interface for Toki data persistence
// ABOUTME: Defines contracts for project, todo, and tag operations

package storage

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a todo project in storage.
type Project struct {
	ID            uuid.UUID
	Name          string
	DirectoryPath string
	CreatedAt     time.Time
}

// Todo represents a todo item in storage.
type Todo struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	ProjectName string
	Description string
	Done        bool
	Priority    string
	Notes       string
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	DueDate     *time.Time
}

// Tag represents a tag for categorizing todos.
type Tag struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// TodoFilter specifies filters for listing todos.
type TodoFilter struct {
	ProjectID *uuid.UUID
	Done      *bool
	Priority  *string
	Tag       *string
	Overdue   *bool
}

// Storage defines the interface for data persistence.
type Storage interface {
	// Close closes the storage connection.
	Close() error

	// Project operations
	CreateProject(project *Project) error
	GetProject(id uuid.UUID) (*Project, error)
	GetProjectByName(name string) (*Project, error)
	GetProjectByPath(path string) (*Project, error)
	ListProjects() ([]*Project, error)
	UpdateProject(project *Project) error
	DeleteProject(id uuid.UUID) error

	// Todo operations
	CreateTodo(todo *Todo) error
	GetTodo(id uuid.UUID) (*Todo, error)
	GetTodoByPrefix(prefix string) (*Todo, error)
	ListTodos(filter *TodoFilter) ([]*Todo, error)
	UpdateTodo(todo *Todo) error
	DeleteTodo(id uuid.UUID) error
	MarkTodoDone(id uuid.UUID, done bool) error

	// Tag operations
	GetOrCreateTag(name string) (*Tag, error)
	ListTags() ([]*Tag, error)
	DeleteTag(name string) error
	AddTagToTodo(todoID uuid.UUID, tagName string) error
	RemoveTagFromTodo(todoID uuid.UUID, tagName string) error
	GetTagsForTodo(todoID uuid.UUID) ([]string, error)

	// Database maintenance
	Vacuum() error
	IntegrityCheck() (bool, error)
}
