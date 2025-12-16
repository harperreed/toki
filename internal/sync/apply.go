// ABOUTME: Entity change handlers for vault sync
// ABOUTME: Applies remote changes to local database with idempotent upserts and cross-entity lookups

package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"suitesync/vault"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/db"
)

// ProjectPayload represents the vault payload for a project entity.
type ProjectPayload struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	DirectoryPath *string `json:"directory_path,omitempty"`
	CreatedAt     int64   `json:"created_at"`
}

// TodoPayload represents the vault payload for a todo entity.
type TodoPayload struct {
	ID          string  `json:"id"`
	ProjectName string  `json:"project_name"`
	Description string  `json:"description"`
	Done        bool    `json:"done"`
	Priority    *string `json:"priority,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	DueDate     *int64  `json:"due_date,omitempty"`
	CompletedAt *int64  `json:"completed_at,omitempty"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

// TagPayload represents the vault payload for a tag entity.
type TagPayload struct {
	Name string `json:"name"`
}

// TodoTagPayload represents the vault payload for a todo-tag association.
type TodoTagPayload struct {
	TodoID  string `json:"todo_id"`
	TagName string `json:"tag_name"`
}

// applyProjectChange applies a project change to the local database.
func (s *Syncer) applyProjectChange(ctx context.Context, c vault.Change) error {
	if c.Op == vault.OpDelete || c.Deleted {
		_, err := s.appDB.ExecContext(ctx,
			`DELETE FROM projects WHERE id = ?`, c.EntityID)
		return err
	}

	var payload ProjectPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal project payload: %w", err)
	}

	createdAt := time.Unix(payload.CreatedAt, 0)
	_, err := s.appDB.ExecContext(ctx, `
		INSERT OR REPLACE INTO projects (id, name, directory_path, created_at)
		VALUES (?, ?, ?, ?)
	`, payload.ID, payload.Name, payload.DirectoryPath, createdAt)

	return err
}

// applyTodoChange applies a todo change to the local database.
func (s *Syncer) applyTodoChange(ctx context.Context, c vault.Change) error {
	if c.Op == vault.OpDelete || c.Deleted {
		_, err := s.appDB.ExecContext(ctx,
			`DELETE FROM todos WHERE id = ?`, c.EntityID)
		return err
	}

	var payload TodoPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal todo payload: %w", err)
	}

	// Look up project by name to get ID
	var projectID string
	err := s.appDB.QueryRowContext(ctx,
		`SELECT id FROM projects WHERE name = ?`, payload.ProjectName).Scan(&projectID)
	if errors.Is(err, sql.ErrNoRows) {
		// Create project if it doesn't exist
		projectID = uuid.New().String()
		_, err = s.appDB.ExecContext(ctx,
			`INSERT INTO projects (id, name, created_at) VALUES (?, ?, ?)`,
			projectID, payload.ProjectName, time.Now())
		if err != nil {
			return fmt.Errorf("create project for todo: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("lookup project: %w", err)
	}

	// Convert timestamps
	createdAt := time.Unix(payload.CreatedAt, 0)
	updatedAt := time.Unix(payload.UpdatedAt, 0)

	var completedAt *time.Time
	if payload.CompletedAt != nil {
		t := time.Unix(*payload.CompletedAt, 0)
		completedAt = &t
	}

	var dueDate *time.Time
	if payload.DueDate != nil {
		t := time.Unix(*payload.DueDate, 0)
		dueDate = &t
	}

	// Upsert todo
	_, err = s.appDB.ExecContext(ctx, `
		INSERT INTO todos (id, project_id, description, done, priority, notes, created_at, updated_at, completed_at, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			project_id = excluded.project_id,
			description = excluded.description,
			done = excluded.done,
			priority = excluded.priority,
			notes = excluded.notes,
			updated_at = excluded.updated_at,
			completed_at = excluded.completed_at,
			due_date = excluded.due_date
	`, payload.ID, projectID, payload.Description, payload.Done, payload.Priority, payload.Notes,
		createdAt, updatedAt, completedAt, dueDate)

	return err
}

// applyTagChange applies a tag change to the local database.
func (s *Syncer) applyTagChange(_ context.Context, c vault.Change) error {
	// Tags don't support deletion in current design
	// (would need to track if tag was manually created vs auto-created)
	if c.Op == vault.OpDelete || c.Deleted {
		return nil
	}

	var payload TagPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal tag payload: %w", err)
	}

	// Use GetOrCreateTag pattern for idempotent tag creation
	_, err := db.GetOrCreateTag(s.appDB, payload.Name)
	return err
}

// applyTodoTagChange applies a todo-tag association change to the local database.
func (s *Syncer) applyTodoTagChange(ctx context.Context, c vault.Change) error {
	// Parse entity ID (format: "todoID|tagName")
	parts := strings.SplitN(c.EntityID, "|", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid todo_tag entity ID: %s", c.EntityID)
	}

	todoID := parts[0]
	tagName := parts[1]

	if c.Op == vault.OpDelete || c.Deleted {
		// Look up tag ID by name
		var tagID int64
		err := s.appDB.QueryRowContext(ctx,
			`SELECT id FROM tags WHERE name = ?`, tagName).Scan(&tagID)
		if errors.Is(err, sql.ErrNoRows) {
			// Tag doesn't exist, nothing to delete
			return nil
		} else if err != nil {
			return fmt.Errorf("lookup tag: %w", err)
		}

		_, err = s.appDB.ExecContext(ctx,
			`DELETE FROM todo_tags WHERE todo_id = ? AND tag_id = ?`, todoID, tagID)
		return err
	}

	var payload TodoTagPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal todo_tag payload: %w", err)
	}

	// Get or create tag
	tag, err := db.GetOrCreateTag(s.appDB, payload.TagName)
	if err != nil {
		return fmt.Errorf("get or create tag: %w", err)
	}

	// Create association (INSERT OR IGNORE for idempotency)
	_, err = s.appDB.ExecContext(ctx,
		`INSERT OR IGNORE INTO todo_tags (todo_id, tag_id) VALUES (?, ?)`,
		payload.TodoID, tag.ID)

	return err
}
