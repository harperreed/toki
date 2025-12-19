// ABOUTME: Data models for Charm KV storage
// ABOUTME: Defines Project, Todo, and Tag types with JSON serialization

package charm

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a todo project stored in Charm KV.
type Project struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	DirectoryPath string    `json:"directory_path,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Todo represents a todo item stored in Charm KV.
// Tags are denormalized as a string slice for simpler querying.
type Todo struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	ProjectName string     `json:"project_name"`
	Description string     `json:"description"`
	Done        bool       `json:"done"`
	Priority    string     `json:"priority"`
	Notes       string     `json:"notes,omitempty"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// Tag represents a known tag stored in Charm KV.
// Tags exist primarily for autocomplete; the source of truth is in Todo.Tags.
type Tag struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Key prefixes for Charm KV storage.
const (
	ProjectKeyPrefix = "project:"
	TodoKeyPrefix    = "todo:"
	TagKeyPrefix     = "tag:"
)

// ProjectKey returns the KV key for a project.
func ProjectKey(id uuid.UUID) string {
	return ProjectKeyPrefix + id.String()
}

// TodoKey returns the KV key for a todo.
func TodoKey(id uuid.UUID) string {
	return TodoKeyPrefix + id.String()
}

// TagKey returns the KV key for a tag.
func TagKey(name string) string {
	return TagKeyPrefix + name
}
