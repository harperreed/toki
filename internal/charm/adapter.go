// ABOUTME: Adapter layer for migrating from db package to charm
// ABOUTME: Provides functions matching the old db package signatures for gradual migration

package charm

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

// Global client for command usage (thread-safe initialization).
var (
	globalClient *Client
	clientOnce   sync.Once
	clientErr    error
)

// InitClient initializes the global Charm client (thread-safe).
func InitClient() error {
	clientOnce.Do(func() {
		globalClient, clientErr = NewClient("toki")
	})
	return clientErr
}

// CloseClient closes the global client.
func CloseClient() error {
	if globalClient != nil {
		return globalClient.Close()
	}
	return nil
}

// GetClient returns the global client.
func GetClient() *Client {
	return globalClient
}

// ToModelsTodo converts a charm.Todo to models.Todo.
func ToModelsTodo(t *Todo) *models.Todo {
	var priority *string
	if t.Priority != "" {
		priority = &t.Priority
	}

	var notes *string
	if t.Notes != "" {
		notes = &t.Notes
	}

	return &models.Todo{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Description: t.Description,
		Done:        t.Done,
		Priority:    priority,
		Notes:       notes,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
		DueDate:     t.DueDate,
	}
}

// FromModelsTodo converts a models.Todo to charm.Todo.
func FromModelsTodo(t *models.Todo, projectName string, tags []string) *Todo {
	priority := ""
	if t.Priority != nil {
		priority = *t.Priority
	}

	notes := ""
	if t.Notes != nil {
		notes = *t.Notes
	}

	if tags == nil {
		tags = []string{}
	}

	return &Todo{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		ProjectName: projectName,
		Description: t.Description,
		Done:        t.Done,
		Priority:    priority,
		Notes:       notes,
		Tags:        tags,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
		DueDate:     t.DueDate,
	}
}

// ToModelsProject converts a charm.Project to models.Project.
func ToModelsProject(p *Project) *models.Project {
	var dirPath *string
	if p.DirectoryPath != "" {
		dirPath = &p.DirectoryPath
	}

	return &models.Project{
		ID:            p.ID,
		Name:          p.Name,
		DirectoryPath: dirPath,
		CreatedAt:     p.CreatedAt,
	}
}

// FromModelsProject converts a models.Project to charm.Project.
func FromModelsProject(p *models.Project) *Project {
	dirPath := ""
	if p.DirectoryPath != nil {
		dirPath = *p.DirectoryPath
	}

	return &Project{
		ID:            p.ID,
		Name:          p.Name,
		DirectoryPath: dirPath,
		CreatedAt:     p.CreatedAt,
	}
}

// ToModelsTag converts a charm.Tag to models.Tag.
func ToModelsTag(t *Tag) *models.Tag {
	return &models.Tag{
		Name: t.Name,
	}
}

// CreateDefaultProject creates or returns the default project.
func (c *Client) CreateDefaultProject() (*Project, error) {
	project, err := c.GetProjectByName("default")
	if err == nil {
		return project, nil
	}

	project = &Project{
		ID:        uuid.New(),
		Name:      "default",
		CreatedAt: time.Now().UTC(),
	}

	if err := c.CreateProject(project); err != nil {
		return nil, err
	}

	return project, nil
}

// GetTagsForTodo returns tags for a todo (already inline).
func (c *Client) GetTagsForTodo(todoID uuid.UUID) ([]string, error) {
	todo, err := c.GetTodo(todoID)
	if err != nil {
		return nil, err
	}
	return todo.Tags, nil
}
