// ABOUTME: Database schema migrations
// ABOUTME: Creates tables for projects, todos, tags, and relationships

package db

import (
	"database/sql"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS projects (
	id TEXT PRIMARY KEY,
	name TEXT UNIQUE NOT NULL,
	directory_path TEXT,
	created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS todos (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL,
	description TEXT NOT NULL,
	done BOOLEAN NOT NULL DEFAULT 0,
	priority TEXT,
	notes TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	completed_at DATETIME,
	due_date DATETIME,
	FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tags (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS todo_tags (
	todo_id TEXT NOT NULL,
	tag_id INTEGER NOT NULL,
	PRIMARY KEY (todo_id, tag_id),
	FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE,
	FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_todos_project_id ON todos(project_id);
CREATE INDEX IF NOT EXISTS idx_todos_done ON todos(done);
CREATE INDEX IF NOT EXISTS idx_projects_directory_path ON projects(directory_path);
`

func runMigrations(db *sql.DB) error {
	// Run the main schema
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Migration: Add updated_at column if it doesn't exist
	// Check if column exists first
	var columnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='updated_at'`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check for updated_at column: %w", err)
	}

	if !columnExists {
		// Add the column with default value of created_at for existing rows
		_, err = db.Exec(`ALTER TABLE todos ADD COLUMN updated_at DATETIME`)
		if err != nil {
			return fmt.Errorf("failed to add updated_at column: %w", err)
		}

		// Set updated_at to created_at for existing rows
		_, err = db.Exec(`UPDATE todos SET updated_at = created_at WHERE updated_at IS NULL`)
		if err != nil {
			return fmt.Errorf("failed to populate updated_at column: %w", err)
		}
	}

	return nil
}
