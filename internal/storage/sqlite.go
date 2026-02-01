// ABOUTME: SQLite implementation of the Storage interface
// ABOUTME: Uses modernc.org/sqlite for pure Go SQLite support

package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// SQLiteStorage implements Storage using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage at the given path.
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open with WAL mode and foreign keys
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

// DefaultDBPath returns the default database path for Toki.
func DefaultDBPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "toki", "toki.db")
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// CreateProject creates a new project.
func (s *SQLiteStorage) CreateProject(project *Project) error {
	_, err := s.db.Exec(
		`INSERT INTO projects (id, name, directory_path, created_at) VALUES (?, ?, ?, ?)`,
		project.ID.String(), project.Name, project.DirectoryPath, project.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

// GetProject retrieves a project by ID.
func (s *SQLiteStorage) GetProject(id uuid.UUID) (*Project, error) {
	var project Project
	var idStr, dirPath string
	var createdAt time.Time

	err := s.db.QueryRow(
		`SELECT id, name, directory_path, created_at FROM projects WHERE id = ?`,
		id.String(),
	).Scan(&idStr, &project.Name, &dirPath, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	project.DirectoryPath = dirPath
	project.CreatedAt = createdAt

	return &project, nil
}

// GetProjectByName retrieves a project by name.
func (s *SQLiteStorage) GetProjectByName(name string) (*Project, error) {
	var project Project
	var idStr, dirPath string
	var createdAt time.Time

	err := s.db.QueryRow(
		`SELECT id, name, directory_path, created_at FROM projects WHERE name = ?`,
		name,
	).Scan(&idStr, &project.Name, &dirPath, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	project.DirectoryPath = dirPath
	project.CreatedAt = createdAt

	return &project, nil
}

// GetProjectByPath retrieves a project by directory path.
func (s *SQLiteStorage) GetProjectByPath(path string) (*Project, error) {
	var project Project
	var idStr, dirPath string
	var createdAt time.Time

	err := s.db.QueryRow(
		`SELECT id, name, directory_path, created_at FROM projects WHERE directory_path = ?`,
		path,
	).Scan(&idStr, &project.Name, &dirPath, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found for path: %s", path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	project.DirectoryPath = dirPath
	project.CreatedAt = createdAt

	return &project, nil
}

// ListProjects returns all projects sorted by name.
func (s *SQLiteStorage) ListProjects() ([]*Project, error) {
	rows, err := s.db.Query(
		`SELECT id, name, directory_path, created_at FROM projects ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var projects []*Project
	for rows.Next() {
		var idStr, dirPath string
		var project Project
		var createdAt time.Time

		if err := rows.Scan(&idStr, &project.Name, &dirPath, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		project.ID, _ = uuid.Parse(idStr)
		project.DirectoryPath = dirPath
		project.CreatedAt = createdAt
		projects = append(projects, &project)
	}

	return projects, nil
}

// UpdateProject updates an existing project.
func (s *SQLiteStorage) UpdateProject(project *Project) error {
	result, err := s.db.Exec(
		`UPDATE projects SET name = ?, directory_path = ? WHERE id = ?`,
		project.Name, project.DirectoryPath, project.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("project not found: %s", project.ID)
	}

	return nil
}

// DeleteProject removes a project by ID.
func (s *SQLiteStorage) DeleteProject(id uuid.UUID) error {
	result, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("project not found: %s", id)
	}

	return nil
}

// CreateTodo creates a new todo with optional tags.
func (s *SQLiteStorage) CreateTodo(todo *Todo) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Insert todo
	_, err = tx.Exec(
		`INSERT INTO todos (id, project_id, description, done, priority, notes, created_at, updated_at, completed_at, due_date)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		todo.ID.String(), todo.ProjectID.String(), todo.Description, todo.Done,
		todo.Priority, todo.Notes, todo.CreatedAt, todo.UpdatedAt,
		todo.CompletedAt, todo.DueDate,
	)
	if err != nil {
		return fmt.Errorf("failed to create todo: %w", err)
	}

	// Insert tags if provided
	for _, tagName := range todo.Tags {
		// Get or create tag
		var tagID int64
		err := tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, tagName).Scan(&tagID)
		if errors.Is(err, sql.ErrNoRows) {
			result, err := tx.Exec(`INSERT INTO tags (name) VALUES (?)`, tagName)
			if err != nil {
				return fmt.Errorf("failed to create tag: %w", err)
			}
			tagID, _ = result.LastInsertId()
		} else if err != nil {
			return fmt.Errorf("failed to get tag: %w", err)
		}

		// Create association
		_, err = tx.Exec(`INSERT OR IGNORE INTO todo_tags (todo_id, tag_id) VALUES (?, ?)`,
			todo.ID.String(), tagID)
		if err != nil {
			return fmt.Errorf("failed to associate tag: %w", err)
		}
	}

	return tx.Commit()
}

// GetTodo retrieves a todo by ID.
func (s *SQLiteStorage) GetTodo(id uuid.UUID) (*Todo, error) {
	todo, err := s.scanTodo(s.db.QueryRow(
		`SELECT t.id, t.project_id, p.name, t.description, t.done, t.priority, t.notes,
		        t.created_at, t.updated_at, t.completed_at, t.due_date
		 FROM todos t
		 JOIN projects p ON t.project_id = p.id
		 WHERE t.id = ?`,
		id.String(),
	))
	if err != nil {
		return nil, err
	}

	// Get tags
	tags, err := s.GetTagsForTodo(id)
	if err != nil {
		return nil, err
	}
	todo.Tags = tags

	return todo, nil
}

// GetTodoByPrefix retrieves a todo by ID prefix.
func (s *SQLiteStorage) GetTodoByPrefix(prefix string) (*Todo, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.project_id, p.name, t.description, t.done, t.priority, t.notes,
		        t.created_at, t.updated_at, t.completed_at, t.due_date
		 FROM todos t
		 JOIN projects p ON t.project_id = p.id
		 WHERE t.id LIKE ?`,
		prefix+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search todos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var matches []*Todo
	for rows.Next() {
		todo, err := s.scanTodoFromRows(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, todo)
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

	// Get tags for the matched todo
	tags, err := s.GetTagsForTodo(matches[0].ID)
	if err != nil {
		return nil, err
	}
	matches[0].Tags = tags

	return matches[0], nil
}

// ListTodos returns todos matching the given filter.
//
//nolint:funlen,nestif // Query building logic is clearer when kept together
func (s *SQLiteStorage) ListTodos(filter *TodoFilter) ([]*Todo, error) {
	query := `SELECT t.id, t.project_id, p.name, t.description, t.done, t.priority, t.notes,
	                 t.created_at, t.updated_at, t.completed_at, t.due_date
	          FROM todos t
	          JOIN projects p ON t.project_id = p.id`

	var conditions []string
	var args []interface{}

	if filter != nil {
		if filter.ProjectID != nil {
			conditions = append(conditions, "t.project_id = ?")
			args = append(args, filter.ProjectID.String())
		}
		if filter.Done != nil {
			conditions = append(conditions, "t.done = ?")
			if *filter.Done {
				args = append(args, 1)
			} else {
				args = append(args, 0)
			}
		}
		if filter.Priority != nil {
			conditions = append(conditions, "t.priority = ?")
			args = append(args, *filter.Priority)
		}
		if filter.Tag != nil {
			conditions = append(conditions,
				`t.id IN (SELECT todo_id FROM todo_tags tt JOIN tags tg ON tt.tag_id = tg.id WHERE tg.name = ?)`)
			args = append(args, *filter.Tag)
		}
		if filter.Overdue != nil && *filter.Overdue {
			conditions = append(conditions, "t.done = 0 AND t.due_date IS NOT NULL AND t.due_date < ?")
			args = append(args, time.Now().UTC())
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY t.created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var todos []*Todo
	for rows.Next() {
		todo, err := s.scanTodoFromRows(rows)
		if err != nil {
			return nil, err
		}

		// Get tags for each todo
		tags, err := s.GetTagsForTodo(todo.ID)
		if err != nil {
			return nil, err
		}
		todo.Tags = tags

		todos = append(todos, todo)
	}

	return todos, nil
}

// UpdateTodo updates an existing todo.
func (s *SQLiteStorage) UpdateTodo(todo *Todo) error {
	todo.UpdatedAt = time.Now().UTC()

	result, err := s.db.Exec(
		`UPDATE todos SET description = ?, done = ?, priority = ?, notes = ?,
		                  updated_at = ?, completed_at = ?, due_date = ?
		 WHERE id = ?`,
		todo.Description, todo.Done, todo.Priority, todo.Notes,
		todo.UpdatedAt, todo.CompletedAt, todo.DueDate, todo.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("todo not found: %s", todo.ID)
	}

	return nil
}

// DeleteTodo removes a todo by ID.
func (s *SQLiteStorage) DeleteTodo(id uuid.UUID) error {
	result, err := s.db.Exec(`DELETE FROM todos WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("todo not found: %s", id)
	}

	return nil
}

// MarkTodoDone sets the done status of a todo.
func (s *SQLiteStorage) MarkTodoDone(id uuid.UUID, done bool) error {
	var completedAt *time.Time
	if done {
		now := time.Now().UTC()
		completedAt = &now
	}

	result, err := s.db.Exec(
		`UPDATE todos SET done = ?, completed_at = ?, updated_at = ? WHERE id = ?`,
		done, completedAt, time.Now().UTC(), id.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to mark todo: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("todo not found: %s", id)
	}

	return nil
}

// GetOrCreateTag retrieves a tag by name, creating it if it doesn't exist.
func (s *SQLiteStorage) GetOrCreateTag(name string) (*Tag, error) {
	var tag Tag
	var createdAt time.Time

	err := s.db.QueryRow(`SELECT id, name, created_at FROM tags WHERE name = ?`, name).
		Scan(&tag.ID, &tag.Name, &createdAt)

	if err == sql.ErrNoRows {
		// Create the tag
		now := time.Now().UTC()
		result, err := s.db.Exec(`INSERT INTO tags (name, created_at) VALUES (?, ?)`, name, now)
		if err != nil {
			return nil, fmt.Errorf("failed to create tag: %w", err)
		}
		tag.ID, _ = result.LastInsertId()
		tag.Name = name
		tag.CreatedAt = now
		return &tag, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	tag.CreatedAt = createdAt
	return &tag, nil
}

// ListTags returns all tags sorted by name.
func (s *SQLiteStorage) ListTags() ([]*Tag, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM tags ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []*Tag
	for rows.Next() {
		var tag Tag
		var createdAt time.Time
		if err := rows.Scan(&tag.ID, &tag.Name, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tag.CreatedAt = createdAt
		tags = append(tags, &tag)
	}

	return tags, nil
}

// DeleteTag removes a tag by name.
func (s *SQLiteStorage) DeleteTag(name string) error {
	result, err := s.db.Exec(`DELETE FROM tags WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("tag not found: %s", name)
	}

	return nil
}

// AddTagToTodo adds a tag to a todo.
func (s *SQLiteStorage) AddTagToTodo(todoID uuid.UUID, tagName string) error {
	// Get or create the tag
	tag, err := s.GetOrCreateTag(tagName)
	if err != nil {
		return err
	}

	// Create association (ignore if already exists)
	_, err = s.db.Exec(`INSERT OR IGNORE INTO todo_tags (todo_id, tag_id) VALUES (?, ?)`,
		todoID.String(), tag.ID)
	if err != nil {
		return fmt.Errorf("failed to add tag to todo: %w", err)
	}

	return nil
}

// RemoveTagFromTodo removes a tag from a todo.
func (s *SQLiteStorage) RemoveTagFromTodo(todoID uuid.UUID, tagName string) error {
	_, err := s.db.Exec(
		`DELETE FROM todo_tags WHERE todo_id = ? AND tag_id = (SELECT id FROM tags WHERE name = ?)`,
		todoID.String(), tagName,
	)
	if err != nil {
		return fmt.Errorf("failed to remove tag from todo: %w", err)
	}
	return nil
}

// GetTagsForTodo returns all tag names for a todo.
func (s *SQLiteStorage) GetTagsForTodo(todoID uuid.UUID) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT t.name FROM tags t
		 JOIN todo_tags tt ON t.id = tt.tag_id
		 WHERE tt.todo_id = ?
		 ORDER BY t.name`,
		todoID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, name)
	}

	// Sort for consistent output
	sort.Strings(tags)

	return tags, nil
}

// Vacuum optimizes the database.
func (s *SQLiteStorage) Vacuum() error {
	_, err := s.db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("vacuum failed: %w", err)
	}
	return nil
}

// IntegrityCheck runs SQLite integrity checks.
func (s *SQLiteStorage) IntegrityCheck() (bool, error) {
	var result string
	err := s.db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil {
		return false, fmt.Errorf("integrity check failed: %w", err)
	}
	return result == "ok", nil
}

// scanTodo scans a single row into a Todo.
func (s *SQLiteStorage) scanTodo(row *sql.Row) (*Todo, error) {
	var todo Todo
	var idStr, projIDStr, projName string
	var priority, notes sql.NullString
	var completedAt, dueDate sql.NullTime
	var done int

	err := row.Scan(&idStr, &projIDStr, &projName, &todo.Description, &done,
		&priority, &notes, &todo.CreatedAt, &todo.UpdatedAt, &completedAt, &dueDate)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan todo: %w", err)
	}

	todo.ID, _ = uuid.Parse(idStr)
	todo.ProjectID, _ = uuid.Parse(projIDStr)
	todo.ProjectName = projName
	todo.Done = done == 1
	if priority.Valid {
		todo.Priority = priority.String
	}
	if notes.Valid {
		todo.Notes = notes.String
	}
	if completedAt.Valid {
		todo.CompletedAt = &completedAt.Time
	}
	if dueDate.Valid {
		todo.DueDate = &dueDate.Time
	}

	return &todo, nil
}

// scanTodoFromRows scans a row from Rows into a Todo.
func (s *SQLiteStorage) scanTodoFromRows(rows *sql.Rows) (*Todo, error) {
	var todo Todo
	var idStr, projIDStr, projName string
	var priority, notes sql.NullString
	var completedAt, dueDate sql.NullTime
	var done int

	err := rows.Scan(&idStr, &projIDStr, &projName, &todo.Description, &done,
		&priority, &notes, &todo.CreatedAt, &todo.UpdatedAt, &completedAt, &dueDate)
	if err != nil {
		return nil, fmt.Errorf("failed to scan todo: %w", err)
	}

	todo.ID, _ = uuid.Parse(idStr)
	todo.ProjectID, _ = uuid.Parse(projIDStr)
	todo.ProjectName = projName
	todo.Done = done == 1
	if priority.Valid {
		todo.Priority = priority.String
	}
	if notes.Valid {
		todo.Notes = notes.String
	}
	if completedAt.Valid {
		todo.CompletedAt = &completedAt.Time
	}
	if dueDate.Valid {
		todo.DueDate = &dueDate.Time
	}

	return &todo, nil
}
