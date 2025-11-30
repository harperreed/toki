// ABOUTME: Todo database operations (CRUD and filtering)
// ABOUTME: Handles todo creation, retrieval with UUID prefix matching, updates, and deletion

package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

// CreateTodo inserts a new todo into the database.
func CreateTodo(db *sql.DB, todo *models.Todo) error {
	query := `INSERT INTO todos (id, project_id, description, done, priority, notes, created_at, updated_at, completed_at, due_date)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		todo.ID.String(),
		todo.ProjectID.String(),
		todo.Description,
		todo.Done,
		todo.Priority,
		todo.Notes,
		todo.CreatedAt,
		todo.UpdatedAt,
		todo.CompletedAt,
		todo.DueDate,
	)

	if err != nil {
		return fmt.Errorf("failed to create todo: %w", err)
	}
	return nil
}

// GetTodoByID retrieves a todo by its UUID.
func GetTodoByID(db *sql.DB, id uuid.UUID) (*models.Todo, error) {
	query := `SELECT id, project_id, description, done, priority, notes, created_at, updated_at, completed_at, due_date
	          FROM todos WHERE id = ?`

	return scanTodo(db.QueryRow(query, id.String()))
}

// GetTodoByPrefix retrieves a todo by UUID prefix (minimum 6 characters).
func GetTodoByPrefix(db *sql.DB, prefix string) (*models.Todo, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	query := `SELECT id, project_id, description, done, priority, notes, created_at, updated_at, completed_at, due_date
	          FROM todos WHERE id LIKE ?`

	rows, err := db.Query(query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query todos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var matches []*models.Todo
	for rows.Next() {
		todo, err := scanTodoFromRows(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, todo)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no todo found with prefix: %s", prefix)
	}

	if len(matches) > 1 {
		ids := make([]string, len(matches))
		for i, t := range matches {
			ids[i] = t.ID.String()[:8]
		}
		return nil, fmt.Errorf("ambiguous prefix '%s', matches: %s", prefix, strings.Join(ids, ", "))
	}

	return matches[0], nil
}

// ListTodos returns todos filtered by project, done status, priority, and/or tag.
func ListTodos(db *sql.DB, projectID *uuid.UUID, done *bool, priority *string, tag *string) ([]*models.Todo, error) {
	query := `SELECT DISTINCT t.id, t.project_id, t.description, t.done, t.priority, t.notes, t.created_at, t.updated_at, t.completed_at, t.due_date
	          FROM todos t`

	var args []interface{}

	// Add JOIN if filtering by tag
	if tag != nil {
		query += `
	          LEFT JOIN todo_tags tt ON t.id = tt.todo_id
	          LEFT JOIN tags tg ON tt.tag_id = tg.id`
	}

	query += ` WHERE 1=1`

	if projectID != nil {
		query += " AND t.project_id = ?"
		args = append(args, projectID.String())
	}

	if done != nil {
		query += " AND t.done = ?"
		args = append(args, *done)
	}

	if priority != nil {
		query += " AND t.priority = ?"
		args = append(args, *priority)
	}

	if tag != nil {
		query += " AND tg.name = ?"
		args = append(args, *tag)
	}

	query += " ORDER BY t.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var todos []*models.Todo
	for rows.Next() {
		todo, err := scanTodoFromRows(rows)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

// UpdateTodo updates an existing todo.
func UpdateTodo(db *sql.DB, todo *models.Todo) error {
	query := `UPDATE todos
	          SET description = ?, done = ?, priority = ?, notes = ?, updated_at = ?, completed_at = ?, due_date = ?
	          WHERE id = ?`

	_, err := db.Exec(query,
		todo.Description,
		todo.Done,
		todo.Priority,
		todo.Notes,
		todo.UpdatedAt,
		todo.CompletedAt,
		todo.DueDate,
		todo.ID.String(),
	)

	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}
	return nil
}

// DeleteTodo deletes a todo.
func DeleteTodo(db *sql.DB, id uuid.UUID) error {
	query := `DELETE FROM todos WHERE id = ?`
	_, err := db.Exec(query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	return nil
}

// Helper function to scan a todo from a single row.
func scanTodo(row *sql.Row) (*models.Todo, error) {
	var todo models.Todo
	var idStr, projectIDStr string

	err := row.Scan(
		&idStr,
		&projectIDStr,
		&todo.Description,
		&todo.Done,
		&todo.Priority,
		&todo.Notes,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&todo.CompletedAt,
		&todo.DueDate,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("todo not found")
		}
		return nil, fmt.Errorf("failed to scan todo: %w", err)
	}

	todo.ID, _ = uuid.Parse(idStr)
	todo.ProjectID, _ = uuid.Parse(projectIDStr)

	return &todo, nil
}

// Helper function to scan a todo from multiple rows.
func scanTodoFromRows(rows *sql.Rows) (*models.Todo, error) {
	var todo models.Todo
	var idStr, projectIDStr string

	err := rows.Scan(
		&idStr,
		&projectIDStr,
		&todo.Description,
		&todo.Done,
		&todo.Priority,
		&todo.Notes,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&todo.CompletedAt,
		&todo.DueDate,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan todo: %w", err)
	}

	todo.ID, _ = uuid.Parse(idStr)
	todo.ProjectID, _ = uuid.Parse(projectIDStr)

	return &todo, nil
}
