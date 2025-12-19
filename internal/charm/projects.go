// ABOUTME: Project CRUD operations for Charm KV
// ABOUTME: Implements create, read, update, delete, and list for projects

package charm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// CreateProject stores a new project in the KV store.
func (c *Client) CreateProject(project *Project) error {
	data, err := json.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	key := ProjectKey(project.ID)
	if err := c.kv.Set([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store project: %w", err)
	}

	return nil
}

// GetProject retrieves a project by ID.
func (c *Client) GetProject(id uuid.UUID) (*Project, error) {
	key := ProjectKey(id)
	data, err := c.kv.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	var project Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project: %w", err)
	}

	return &project, nil
}

// GetProjectByName retrieves a project by name.
func (c *Client) GetProjectByName(name string) (*Project, error) {
	projects, err := c.ListProjects()
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if p.Name == name {
			return p, nil
		}
	}

	return nil, fmt.Errorf("project not found: %s", name)
}

// GetProjectByPath retrieves a project by directory path.
func (c *Client) GetProjectByPath(path string) (*Project, error) {
	projects, err := c.ListProjects()
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if p.DirectoryPath == path {
			return p, nil
		}
	}

	return nil, fmt.Errorf("project not found for path: %s", path)
}

// ListProjects returns all projects, sorted by name.
func (c *Client) ListProjects() ([]*Project, error) {
	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	projects := make([]*Project, 0, len(keys))
	for _, key := range keys {
		keyStr := string(key)
		if !strings.HasPrefix(keyStr, ProjectKeyPrefix) {
			continue
		}

		data, err := c.kv.Get(key)
		if err != nil {
			continue
		}

		var project Project
		if err := json.Unmarshal(data, &project); err != nil {
			continue
		}

		projects = append(projects, &project)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

// UpdateProject updates an existing project.
func (c *Client) UpdateProject(project *Project) error {
	return c.CreateProject(project)
}

// DeleteProject removes a project by ID.
func (c *Client) DeleteProject(id uuid.UUID) error {
	key := ProjectKey(id)
	if err := c.kv.Delete([]byte(key)); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}
