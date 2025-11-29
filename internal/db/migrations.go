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
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}
	return nil
}
