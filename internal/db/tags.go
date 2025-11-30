// ABOUTME: Tag database operations
// ABOUTME: Handles tag creation, todo-tag associations, and retrieval

package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

// GetOrCreateTag retrieves a tag by name or creates it if it doesn't exist.
func GetOrCreateTag(db *sql.DB, name string) (*models.Tag, error) {
	// Try to get existing tag
	var tag models.Tag
	err := db.QueryRow("SELECT id, name FROM tags WHERE name = ?", name).Scan(&tag.ID, &tag.Name)

	if err == nil {
		return &tag, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query tag: %w", err)
	}

	// Create new tag
	result, err := db.Exec("INSERT INTO tags (name) VALUES (?)", name)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag ID: %w", err)
	}

	return &models.Tag{ID: id, Name: name}, nil
}

// AddTagToTodo associates a tag with a todo.
func AddTagToTodo(db *sql.DB, todoID uuid.UUID, tagName string) error {
	tag, err := GetOrCreateTag(db, tagName)
	if err != nil {
		return err
	}

	query := `INSERT OR IGNORE INTO todo_tags (todo_id, tag_id) VALUES (?, ?)`
	_, err = db.Exec(query, todoID.String(), tag.ID)
	if err != nil {
		return fmt.Errorf("failed to add tag to todo: %w", err)
	}

	return nil
}

// RemoveTagFromTodo removes a tag association from a todo.
func RemoveTagFromTodo(db *sql.DB, todoID uuid.UUID, tagName string) error {
	query := `DELETE FROM todo_tags
	          WHERE todo_id = ? AND tag_id = (SELECT id FROM tags WHERE name = ?)`

	_, err := db.Exec(query, todoID.String(), tagName)
	if err != nil {
		return fmt.Errorf("failed to remove tag from todo: %w", err)
	}

	return nil
}

// GetTodoTags retrieves all tags associated with a todo.
func GetTodoTags(db *sql.DB, todoID uuid.UUID) ([]*models.Tag, error) {
	query := `SELECT t.id, t.name
	          FROM tags t
	          INNER JOIN todo_tags tt ON t.id = tt.tag_id
	          WHERE tt.todo_id = ?
	          ORDER BY t.name`

	rows, err := db.Query(query, todoID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get todo tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}

	return tags, nil
}

// ListAllTags retrieves all tags in the database.
func ListAllTags(db *sql.DB) ([]*models.Tag, error) {
	query := `SELECT id, name FROM tags ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}

	return tags, nil
}
