// ABOUTME: Todo CRUD operations for Charm KV
// ABOUTME: Implements create, read, update, delete, list with filtering, and status changes

package charm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TodoFilter specifies filters for listing todos.
type TodoFilter struct {
	ProjectID *uuid.UUID
	Done      *bool
	Priority  *string
	Tag       *string
	Overdue   *bool
}

// CreateTodo stores a new todo in the KV store.
func (c *Client) CreateTodo(todo *Todo) error {
	if todo.Tags == nil {
		todo.Tags = []string{}
	}

	data, err := json.Marshal(todo)
	if err != nil {
		return fmt.Errorf("failed to marshal todo: %w", err)
	}

	key := TodoKey(todo.ID)
	if err := c.kv.Set([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store todo: %w", err)
	}

	return nil
}

// GetTodo retrieves a todo by ID.
func (c *Client) GetTodo(id uuid.UUID) (*Todo, error) {
	key := TodoKey(id)
	data, err := c.kv.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("todo not found: %w", err)
	}

	var todo Todo
	if err := json.Unmarshal(data, &todo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal todo: %w", err)
	}

	return &todo, nil
}

// GetTodoByPrefix retrieves a todo by ID prefix.
func (c *Client) GetTodoByPrefix(prefix string) (*Todo, error) {
	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	matches := make([]*Todo, 0, 1)
	for _, key := range keys {
		keyStr := string(key)
		if !strings.HasPrefix(keyStr, TodoKeyPrefix) {
			continue
		}

		idStr := strings.TrimPrefix(keyStr, TodoKeyPrefix)
		if !strings.HasPrefix(idStr, prefix) {
			continue
		}

		data, err := c.kv.Get(key)
		if err != nil {
			continue
		}

		var todo Todo
		if err := json.Unmarshal(data, &todo); err != nil {
			continue
		}

		matches = append(matches, &todo)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no todo found with prefix: %s", prefix)
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.ID.String()[:8])
		}
		return nil, fmt.Errorf("ambiguous prefix '%s', matches: %s", prefix, strings.Join(ids, ", "))
	}

	return matches[0], nil
}

// matchesFilter checks if a todo matches the given filter criteria.
func matchesFilter(todo *Todo, filter *TodoFilter, now time.Time) bool {
	if filter == nil {
		return true
	}

	if filter.ProjectID != nil && todo.ProjectID != *filter.ProjectID {
		return false
	}

	if filter.Done != nil && todo.Done != *filter.Done {
		return false
	}

	if filter.Priority != nil && todo.Priority != *filter.Priority {
		return false
	}

	if filter.Tag != nil {
		hasTag := false
		for _, t := range todo.Tags {
			if t == *filter.Tag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	if filter.Overdue != nil && *filter.Overdue {
		if todo.Done || todo.DueDate == nil || todo.DueDate.After(now) {
			return false
		}
	}

	return true
}

// ListTodos returns todos matching the given filter.
func (c *Client) ListTodos(filter *TodoFilter) ([]*Todo, error) {
	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	todos := make([]*Todo, 0, len(keys))
	now := time.Now()

	for _, key := range keys {
		keyStr := string(key)
		if !strings.HasPrefix(keyStr, TodoKeyPrefix) {
			continue
		}

		data, err := c.kv.Get(key)
		if err != nil {
			continue
		}

		var todo Todo
		if err := json.Unmarshal(data, &todo); err != nil {
			continue
		}

		if matchesFilter(&todo, filter, now) {
			todos = append(todos, &todo)
		}
	}

	// Sort by created_at descending
	sort.Slice(todos, func(i, j int) bool {
		return todos[i].CreatedAt.After(todos[j].CreatedAt)
	})

	return todos, nil
}

// UpdateTodo updates an existing todo.
func (c *Client) UpdateTodo(todo *Todo) error {
	todo.UpdatedAt = time.Now().UTC()
	return c.CreateTodo(todo)
}

// DeleteTodo removes a todo by ID.
func (c *Client) DeleteTodo(id uuid.UUID) error {
	key := TodoKey(id)
	if err := c.kv.Delete([]byte(key)); err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	return nil
}

// MarkTodoDone sets the done status of a todo.
func (c *Client) MarkTodoDone(id uuid.UUID, done bool) error {
	todo, err := c.GetTodo(id)
	if err != nil {
		return err
	}

	todo.Done = done
	todo.UpdatedAt = time.Now().UTC()

	if done {
		now := time.Now().UTC()
		todo.CompletedAt = &now
	} else {
		todo.CompletedAt = nil
	}

	return c.CreateTodo(todo)
}

// AddTagToTodo adds a tag to a todo.
func (c *Client) AddTagToTodo(todoID uuid.UUID, tagName string) error {
	todo, err := c.GetTodo(todoID)
	if err != nil {
		return err
	}

	// Check if tag already exists
	for _, t := range todo.Tags {
		if t == tagName {
			return nil
		}
	}

	todo.Tags = append(todo.Tags, tagName)
	todo.UpdatedAt = time.Now().UTC()

	return c.CreateTodo(todo)
}

// RemoveTagFromTodo removes a tag from a todo.
func (c *Client) RemoveTagFromTodo(todoID uuid.UUID, tagName string) error {
	todo, err := c.GetTodo(todoID)
	if err != nil {
		return err
	}

	var newTags []string
	for _, t := range todo.Tags {
		if t != tagName {
			newTags = append(newTags, t)
		}
	}

	todo.Tags = newTags
	todo.UpdatedAt = time.Now().UTC()

	return c.CreateTodo(todo)
}
