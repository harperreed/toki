# Toki Todo Manager Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a git-aware CLI todo manager with projects, rich metadata, and smart context detection.

**Architecture:** SQLite for data storage, Cobra for CLI framework, git detection for auto-context, TDD throughout with unit/integration tests.

**Tech Stack:** Go 1.21+, Cobra CLI, SQLite (modernc.org/sqlite for CGO-free), UUID library, fatih/color for output.

---

## Task 1: Project Initialization

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `.gitignore`
- Create: `README.md`

**Step 1: Initialize Go module**

```bash
go mod init github.com/harper/toki
```

Expected: Creates `go.mod` with module declaration

**Step 2: Create .gitignore**

Create `.gitignore`:
```
# Binaries
toki
*.exe
*.dll
*.so
*.dylib

# Test binaries
*.test
*.out

# Go workspace
go.work
go.work.sum

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# Test coverage
coverage.txt
*.cover

# SQLite
*.db
*.db-shm
*.db-wal
```

**Step 3: Create basic README**

Create `README.md`:
```markdown
# Toki

A super simple git-aware CLI todo manager.

## Installation

```bash
go install github.com/harper/toki/cmd/toki@latest
```

## Quick Start

```bash
# Add a todo
toki add "implement feature"

# List todos
toki list

# Mark done
toki done a3f2b9
```

See `docs/plans/2025-11-29-toki-todo-manager-design.md` for full design.
```

**Step 4: Commit initialization**

```bash
git add go.mod .gitignore README.md
git commit -m "chore: initialize Go module and project structure"
```

---

## Task 2: Data Models

**Files:**
- Create: `internal/models/models.go`
- Create: `internal/models/models_test.go`

**Step 1: Write the failing test for model creation**

Create `internal/models/models_test.go`:
```go
// ABOUTME: Tests for core data models
// ABOUTME: Validates Project, Todo, Tag struct behavior

package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewProject(t *testing.T) {
	name := "test-project"
	path := "/home/user/project"

	project := NewProject(name, &path)

	if project.ID == uuid.Nil {
		t.Error("Expected non-nil UUID")
	}
	if project.Name != name {
		t.Errorf("Expected name %s, got %s", name, project.Name)
	}
	if project.DirectoryPath == nil || *project.DirectoryPath != path {
		t.Errorf("Expected path %s, got %v", path, project.DirectoryPath)
	}
	if project.CreatedAt.IsZero() {
		t.Error("Expected non-zero created timestamp")
	}
}

func TestNewTodo(t *testing.T) {
	projectID := uuid.New()
	desc := "test todo"

	todo := NewTodo(projectID, desc)

	if todo.ID == uuid.Nil {
		t.Error("Expected non-nil UUID")
	}
	if todo.ProjectID != projectID {
		t.Error("Project ID mismatch")
	}
	if todo.Description != desc {
		t.Errorf("Expected description %s, got %s", desc, todo.Description)
	}
	if todo.Done {
		t.Error("New todo should not be done")
	}
	if todo.CreatedAt.IsZero() {
		t.Error("Expected non-zero created timestamp")
	}
}

func TestTodoMarkDone(t *testing.T) {
	todo := NewTodo(uuid.New(), "test")

	todo.MarkDone()

	if !todo.Done {
		t.Error("Todo should be marked done")
	}
	if todo.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestTodoMarkUndone(t *testing.T) {
	todo := NewTodo(uuid.New(), "test")
	todo.MarkDone()

	todo.MarkUndone()

	if todo.Done {
		t.Error("Todo should not be done")
	}
	if todo.CompletedAt != nil {
		t.Error("CompletedAt should be nil")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go get github.com/google/uuid
go test ./internal/models/... -v
```

Expected: FAIL with "undefined: NewProject", "undefined: NewTodo"

**Step 3: Implement models**

Create `internal/models/models.go`:
```go
// ABOUTME: Core data models for projects, todos, and tags
// ABOUTME: Provides constructor functions and business logic methods

package models

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a collection of todos
type Project struct {
	ID            uuid.UUID
	Name          string
	DirectoryPath *string
	CreatedAt     time.Time
}

// Todo represents a single task
type Todo struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Description string
	Done        bool
	Priority    *string
	Notes       *string
	CreatedAt   time.Time
	CompletedAt *time.Time
	DueDate     *time.Time
}

// Tag represents a label that can be applied to todos
type Tag struct {
	ID   int64
	Name string
}

// TodoTag represents the many-to-many relationship
type TodoTag struct {
	TodoID uuid.UUID
	TagID  int64
}

// NewProject creates a new project with generated UUID and timestamp
func NewProject(name string, directoryPath *string) *Project {
	return &Project{
		ID:            uuid.New(),
		Name:          name,
		DirectoryPath: directoryPath,
		CreatedAt:     time.Now(),
	}
}

// NewTodo creates a new todo with generated UUID and timestamp
func NewTodo(projectID uuid.UUID, description string) *Todo {
	return &Todo{
		ID:          uuid.New(),
		ProjectID:   projectID,
		Description: description,
		Done:        false,
		CreatedAt:   time.Now(),
	}
}

// MarkDone marks a todo as complete
func (t *Todo) MarkDone() {
	t.Done = true
	now := time.Now()
	t.CompletedAt = &now
}

// MarkUndone marks a todo as incomplete
func (t *Todo) MarkUndone() {
	t.Done = false
	t.CompletedAt = nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/models/... -v
```

Expected: PASS (all 4 tests)

**Step 5: Commit models**

```bash
git add internal/models/
git commit -m "feat: add core data models with constructors"
```

---

## Task 3: Database Layer - Connection & Migrations

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/db_test.go`
- Create: `internal/db/migrations.go`

**Step 1: Write the failing test for database initialization**

Create `internal/db/db_test.go`:
```go
// ABOUTME: Tests for database connection and migrations
// ABOUTME: Uses temporary test databases for isolation

package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitDB(t *testing.T) {
	// Create temp dir for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify tables exist
	tables := []string{"projects", "todos", "tags", "todo_tags"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("Error checking table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

func TestGetDefaultDBPath(t *testing.T) {
	path := GetDefaultDBPath()

	if path == "" {
		t.Error("Default path should not be empty")
	}

	// Should contain .local/share/toki
	if !filepath.IsAbs(path) {
		t.Error("Path should be absolute")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v
```

Expected: FAIL with "undefined: InitDB", "undefined: GetDefaultDBPath"

**Step 3: Implement database initialization**

Create `internal/db/db.go`:
```go
// ABOUTME: Database connection management and initialization
// ABOUTME: Handles SQLite connection and migration execution

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitDB initializes the database connection and runs migrations
func InitDB(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// GetDefaultDBPath returns the default database path following XDG standards
func GetDefaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(dataDir, "toki", "toki.db")
}
```

Create `internal/db/migrations.go`:
```go
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
```

**Step 4: Install dependencies and run tests**

```bash
go get modernc.org/sqlite
go test ./internal/db/... -v
```

Expected: PASS (all 2 tests)

**Step 5: Commit database layer**

```bash
git add internal/db/
git commit -m "feat: add database initialization and migrations"
```

---

## Task 4: Database Layer - Project Operations

**Files:**
- Create: `internal/db/projects.go`
- Create: `internal/db/projects_test.go`

**Step 1: Write failing tests for project CRUD**

Create `internal/db/projects_test.go`:
```go
// ABOUTME: Tests for project database operations
// ABOUTME: Covers CRUD operations and path-based lookups

package db

import (
	"path/filepath"
	"testing"

	"github.com/harper/toki/internal/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	return db
}

func TestCreateProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	path := "/home/user/project"
	project := models.NewProject("test-project", &path)

	err := CreateProject(db, project)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Verify it was created
	retrieved, err := GetProjectByID(db, project.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve project: %v", err)
	}

	if retrieved.Name != project.Name {
		t.Errorf("Expected name %s, got %s", project.Name, retrieved.Name)
	}
}

func TestGetProjectByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("findme", nil)
	CreateProject(db, project)

	found, err := GetProjectByName(db, "findme")
	if err != nil {
		t.Fatalf("Failed to get project: %v", err)
	}

	if found.ID != project.ID {
		t.Error("Retrieved wrong project")
	}
}

func TestGetProjectByPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	path := "/home/user/myproject"
	project := models.NewProject("myproject", &path)
	CreateProject(db, project)

	found, err := GetProjectByPath(db, path)
	if err != nil {
		t.Fatalf("Failed to get project by path: %v", err)
	}

	if found.ID != project.ID {
		t.Error("Retrieved wrong project")
	}
}

func TestListProjects(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	CreateProject(db, models.NewProject("project1", nil))
	CreateProject(db, models.NewProject("project2", nil))

	projects, err := ListProjects(db)
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
}

func TestDeleteProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("todelete", nil)
	CreateProject(db, project)

	err := DeleteProject(db, project.ID)
	if err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	_, err = GetProjectByID(db, project.ID)
	if err == nil {
		t.Error("Project should not exist after deletion")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v
```

Expected: FAIL with "undefined: CreateProject", etc.

**Step 3: Implement project operations**

Create `internal/db/projects.go`:
```go
// ABOUTME: Project database operations (CRUD)
// ABOUTME: Handles project creation, retrieval, updates, and deletion

package db

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

// CreateProject inserts a new project into the database
func CreateProject(db *sql.DB, project *models.Project) error {
	query := `INSERT INTO projects (id, name, directory_path, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, project.ID.String(), project.Name, project.DirectoryPath, project.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

// GetProjectByID retrieves a project by its UUID
func GetProjectByID(db *sql.DB, id uuid.UUID) (*models.Project, error) {
	query := `SELECT id, name, directory_path, created_at FROM projects WHERE id = ?`

	var project models.Project
	var idStr string

	err := db.QueryRow(query, id.String()).Scan(&idStr, &project.Name, &project.DirectoryPath, &project.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	return &project, nil
}

// GetProjectByName retrieves a project by its name
func GetProjectByName(db *sql.DB, name string) (*models.Project, error) {
	query := `SELECT id, name, directory_path, created_at FROM projects WHERE name = ?`

	var project models.Project
	var idStr string

	err := db.QueryRow(query, name).Scan(&idStr, &project.Name, &project.DirectoryPath, &project.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	return &project, nil
}

// GetProjectByPath retrieves a project by its directory path
func GetProjectByPath(db *sql.DB, path string) (*models.Project, error) {
	query := `SELECT id, name, directory_path, created_at FROM projects WHERE directory_path = ?`

	var project models.Project
	var idStr string

	err := db.QueryRow(query, path).Scan(&idStr, &project.Name, &project.DirectoryPath, &project.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	return &project, nil
}

// ListProjects returns all projects
func ListProjects(db *sql.DB) ([]*models.Project, error) {
	query := `SELECT id, name, directory_path, created_at FROM projects ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		var project models.Project
		var idStr string

		if err := rows.Scan(&idStr, &project.Name, &project.DirectoryPath, &project.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		project.ID, _ = uuid.Parse(idStr)
		projects = append(projects, &project)
	}

	return projects, nil
}

// UpdateProjectPath updates the directory path for a project
func UpdateProjectPath(db *sql.DB, id uuid.UUID, path *string) error {
	query := `UPDATE projects SET directory_path = ? WHERE id = ?`
	_, err := db.Exec(query, path, id.String())
	if err != nil {
		return fmt.Errorf("failed to update project path: %w", err)
	}
	return nil
}

// DeleteProject deletes a project (cascades to todos)
func DeleteProject(db *sql.DB, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = ?`
	_, err := db.Exec(query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}
```

**Step 4: Fix import in test file and run tests**

Add to top of `internal/db/projects_test.go`:
```go
import (
	"database/sql"
	// ... existing imports
)
```

Run:
```bash
go test ./internal/db/... -v
```

Expected: PASS (all tests)

**Step 5: Commit project operations**

```bash
git add internal/db/projects.go internal/db/projects_test.go
git commit -m "feat: add project CRUD operations"
```

---

## Task 5: Git Detection Logic

**Files:**
- Create: `internal/git/detect.go`
- Create: `internal/git/detect_test.go`

**Step 1: Write failing tests for git detection**

Create `internal/git/detect_test.go`:
```go
// ABOUTME: Tests for git repository detection
// ABOUTME: Creates temporary git repos for testing path detection

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	return tmpDir
}

func TestFindGitRoot(t *testing.T) {
	repoDir := setupGitRepo(t)

	// Create subdirectory
	subDir := filepath.Join(repoDir, "nested", "deep")
	os.MkdirAll(subDir, 0755)

	// Test from subdirectory
	root, err := FindGitRoot(subDir)
	if err != nil {
		t.Fatalf("Failed to find git root: %v", err)
	}

	// Should resolve to repo root (handling symlinks)
	absRepo, _ := filepath.EvalSymlinks(repoDir)
	absRoot, _ := filepath.EvalSymlinks(root)

	if absRoot != absRepo {
		t.Errorf("Expected git root %s, got %s", absRepo, absRoot)
	}
}

func TestFindGitRootNotInRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindGitRoot(tmpDir)
	if err == nil {
		t.Error("Expected error when not in git repo")
	}
}

func TestNormalizePath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"relative/path", "relative/path"},
		{"/absolute/path", "/absolute/path"},
	}

	for _, tc := range testCases {
		result, err := NormalizePath(tc.input)
		if err != nil {
			t.Errorf("Failed to normalize %s: %v", tc.input, err)
		}

		if !filepath.IsAbs(result) {
			t.Errorf("Expected absolute path for %s", tc.input)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/git/... -v
```

Expected: FAIL with "undefined: FindGitRoot", etc.

**Step 3: Implement git detection**

Create `internal/git/detect.go`:
```go
// ABOUTME: Git repository detection and path normalization
// ABOUTME: Walks directory tree to find .git and resolves symlinks

package git

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindGitRoot walks up the directory tree looking for .git
// Returns the absolute path to the repository root
func FindGitRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	currentPath := absPath
	for {
		gitPath := filepath.Join(currentPath, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			// Resolve symlinks
			resolved, err := filepath.EvalSymlinks(currentPath)
			if err != nil {
				return currentPath, nil
			}
			return resolved, nil
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			// Reached root without finding .git
			return "", fmt.Errorf("not in a git repository")
		}
		currentPath = parent
	}
}

// NormalizePath converts a path to absolute and resolves symlinks
func NormalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If symlink resolution fails, return absolute path
		return absPath, nil
	}

	return resolved, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/git/... -v
```

Expected: PASS (all tests)

**Step 5: Commit git detection**

```bash
git add internal/git/
git commit -m "feat: add git repository detection"
```

---

## Task 6: Database Layer - Todo Operations

**Files:**
- Create: `internal/db/todos.go`
- Create: `internal/db/todos_test.go`

**Step 1: Write failing tests for todo CRUD**

Create `internal/db/todos_test.go`:
```go
// ABOUTME: Tests for todo database operations
// ABOUTME: Covers CRUD, filtering, and UUID prefix matching

package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

func TestCreateTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "test todo")
	priority := "high"
	todo.Priority = &priority

	err := CreateTodo(db, todo)
	if err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	retrieved, err := GetTodoByID(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve todo: %v", err)
	}

	if retrieved.Description != todo.Description {
		t.Errorf("Description mismatch")
	}
	if *retrieved.Priority != *todo.Priority {
		t.Errorf("Priority mismatch")
	}
}

func TestGetTodoByPrefix(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "find me")
	CreateTodo(db, todo)

	// Use first 6 characters as prefix
	prefix := todo.ID.String()[:6]

	found, err := GetTodoByPrefix(db, prefix)
	if err != nil {
		t.Fatalf("Failed to find todo by prefix: %v", err)
	}

	if found.ID != todo.ID {
		t.Error("Wrong todo found")
	}
}

func TestGetTodoByPrefixAmbiguous(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	// Create two todos - we can't guarantee prefix collision
	// but we can test the error path
	todo1 := models.NewTodo(project.ID, "todo1")
	CreateTodo(db, todo1)

	// Test with empty prefix should be ambiguous
	_, err := GetTodoByPrefix(db, "")
	if err == nil {
		t.Error("Empty prefix should return error")
	}
}

func TestListTodos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo1 := models.NewTodo(project.ID, "todo1")
	todo2 := models.NewTodo(project.ID, "todo2")
	CreateTodo(db, todo1)
	CreateTodo(db, todo2)

	todos, err := ListTodos(db, &project.ID, nil, nil)
	if err != nil {
		t.Fatalf("Failed to list todos: %v", err)
	}

	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
}

func TestListTodosFilterDone(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo1 := models.NewTodo(project.ID, "pending")
	todo2 := models.NewTodo(project.ID, "done")
	todo2.MarkDone()

	CreateTodo(db, todo1)
	CreateTodo(db, todo2)

	doneFilter := false
	todos, err := ListTodos(db, nil, &doneFilter, nil)
	if err != nil {
		t.Fatalf("Failed to list todos: %v", err)
	}

	if len(todos) != 1 {
		t.Errorf("Expected 1 pending todo, got %d", len(todos))
	}
	if todos[0].Done {
		t.Error("Should only return pending todos")
	}
}

func TestUpdateTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "original")
	CreateTodo(db, todo)

	todo.Description = "updated"
	todo.MarkDone()

	err := UpdateTodo(db, todo)
	if err != nil {
		t.Fatalf("Failed to update todo: %v", err)
	}

	retrieved, err := GetTodoByID(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve todo: %v", err)
	}

	if retrieved.Description != "updated" {
		t.Error("Description not updated")
	}
	if !retrieved.Done {
		t.Error("Done status not updated")
	}
}

func TestDeleteTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "to delete")
	CreateTodo(db, todo)

	err := DeleteTodo(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to delete todo: %v", err)
	}

	_, err = GetTodoByID(db, todo.ID)
	if err == nil {
		t.Error("Todo should not exist after deletion")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v -run TestCreateTodo
```

Expected: FAIL with "undefined: CreateTodo", etc.

**Step 3: Implement todo operations (part 1 - basic CRUD)**

Create `internal/db/todos.go`:
```go
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

// CreateTodo inserts a new todo into the database
func CreateTodo(db *sql.DB, todo *models.Todo) error {
	query := `INSERT INTO todos (id, project_id, description, done, priority, notes, created_at, completed_at, due_date)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		todo.ID.String(),
		todo.ProjectID.String(),
		todo.Description,
		todo.Done,
		todo.Priority,
		todo.Notes,
		todo.CreatedAt,
		todo.CompletedAt,
		todo.DueDate,
	)

	if err != nil {
		return fmt.Errorf("failed to create todo: %w", err)
	}
	return nil
}

// GetTodoByID retrieves a todo by its UUID
func GetTodoByID(db *sql.DB, id uuid.UUID) (*models.Todo, error) {
	query := `SELECT id, project_id, description, done, priority, notes, created_at, completed_at, due_date
	          FROM todos WHERE id = ?`

	return scanTodo(db.QueryRow(query, id.String()))
}

// GetTodoByPrefix retrieves a todo by UUID prefix (minimum 6 characters)
func GetTodoByPrefix(db *sql.DB, prefix string) (*models.Todo, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	query := `SELECT id, project_id, description, done, priority, notes, created_at, completed_at, due_date
	          FROM todos WHERE id LIKE ?`

	rows, err := db.Query(query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query todos: %w", err)
	}
	defer rows.Close()

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

// ListTodos returns todos filtered by project, done status, and/or priority
func ListTodos(db *sql.DB, projectID *uuid.UUID, done *bool, priority *string) ([]*models.Todo, error) {
	query := `SELECT id, project_id, description, done, priority, notes, created_at, completed_at, due_date
	          FROM todos WHERE 1=1`

	var args []interface{}

	if projectID != nil {
		query += " AND project_id = ?"
		args = append(args, projectID.String())
	}

	if done != nil {
		query += " AND done = ?"
		args = append(args, *done)
	}

	if priority != nil {
		query += " AND priority = ?"
		args = append(args, *priority)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos: %w", err)
	}
	defer rows.Close()

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

// UpdateTodo updates an existing todo
func UpdateTodo(db *sql.DB, todo *models.Todo) error {
	query := `UPDATE todos
	          SET description = ?, done = ?, priority = ?, notes = ?, completed_at = ?, due_date = ?
	          WHERE id = ?`

	_, err := db.Exec(query,
		todo.Description,
		todo.Done,
		todo.Priority,
		todo.Notes,
		todo.CompletedAt,
		todo.DueDate,
		todo.ID.String(),
	)

	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}
	return nil
}

// DeleteTodo deletes a todo
func DeleteTodo(db *sql.DB, id uuid.UUID) error {
	query := `DELETE FROM todos WHERE id = ?`
	_, err := db.Exec(query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	return nil
}

// Helper function to scan a todo from a single row
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

// Helper function to scan a todo from multiple rows
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
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/db/... -v -run TestTodo
```

Expected: PASS (all todo tests)

**Step 5: Commit todo operations**

```bash
git add internal/db/todos.go internal/db/todos_test.go
git commit -m "feat: add todo CRUD operations with prefix matching"
```

---

## Task 7: Database Layer - Tag Operations

**Files:**
- Create: `internal/db/tags.go`
- Create: `internal/db/tags_test.go`

**Step 1: Write failing tests for tag operations**

Create `internal/db/tags_test.go`:
```go
// ABOUTME: Tests for tag database operations
// ABOUTME: Covers tag creation, retrieval, and todo-tag associations

package db

import (
	"testing"

	"github.com/harper/toki/internal/models"
)

func TestGetOrCreateTag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tag, err := GetOrCreateTag(db, "urgent")
	if err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	if tag.Name != "urgent" {
		t.Errorf("Expected name 'urgent', got %s", tag.Name)
	}

	// Getting same tag should return same ID
	tag2, err := GetOrCreateTag(db, "urgent")
	if err != nil {
		t.Fatalf("Failed to get existing tag: %v", err)
	}

	if tag2.ID != tag.ID {
		t.Error("Should return same tag ID")
	}
}

func TestAddTagToTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "test")
	CreateTodo(db, todo)

	err := AddTagToTodo(db, todo.ID, "backend")
	if err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}

	tags, err := GetTodoTags(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to get tags: %v", err)
	}

	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}

	if tags[0].Name != "backend" {
		t.Errorf("Expected tag 'backend', got %s", tags[0].Name)
	}
}

func TestRemoveTagFromTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := models.NewProject("test", nil)
	CreateProject(db, project)

	todo := models.NewTodo(project.ID, "test")
	CreateTodo(db, todo)

	AddTagToTodo(db, todo.ID, "frontend")

	err := RemoveTagFromTodo(db, todo.ID, "frontend")
	if err != nil {
		t.Fatalf("Failed to remove tag: %v", err)
	}

	tags, err := GetTodoTags(db, todo.ID)
	if err != nil {
		t.Fatalf("Failed to get tags: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(tags))
	}
}

func TestListAllTags(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	GetOrCreateTag(db, "tag1")
	GetOrCreateTag(db, "tag2")
	GetOrCreateTag(db, "tag3")

	tags, err := ListAllTags(db)
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}

	if len(tags) < 3 {
		t.Errorf("Expected at least 3 tags, got %d", len(tags))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v -run TestTag
```

Expected: FAIL with "undefined: GetOrCreateTag", etc.

**Step 3: Implement tag operations**

Create `internal/db/tags.go`:
```go
// ABOUTME: Tag database operations
// ABOUTME: Handles tag creation, todo-tag associations, and retrieval

package db

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

// GetOrCreateTag retrieves a tag by name or creates it if it doesn't exist
func GetOrCreateTag(db *sql.DB, name string) (*models.Tag, error) {
	// Try to get existing tag
	var tag models.Tag
	err := db.QueryRow("SELECT id, name FROM tags WHERE name = ?", name).Scan(&tag.ID, &tag.Name)

	if err == nil {
		return &tag, nil
	}

	if err != sql.ErrNoRows {
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

// AddTagToTodo associates a tag with a todo
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

// RemoveTagFromTodo removes a tag association from a todo
func RemoveTagFromTodo(db *sql.DB, todoID uuid.UUID, tagName string) error {
	query := `DELETE FROM todo_tags
	          WHERE todo_id = ? AND tag_id = (SELECT id FROM tags WHERE name = ?)`

	_, err := db.Exec(query, todoID.String(), tagName)
	if err != nil {
		return fmt.Errorf("failed to remove tag from todo: %w", err)
	}

	return nil
}

// GetTodoTags retrieves all tags associated with a todo
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
	defer rows.Close()

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

// ListAllTags retrieves all tags in the database
func ListAllTags(db *sql.DB) ([]*models.Tag, error) {
	query := `SELECT id, name FROM tags ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

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
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/db/... -v -run TestTag
```

Expected: PASS (all tag tests)

**Step 5: Commit tag operations**

```bash
git add internal/db/tags.go internal/db/tags_test.go
git commit -m "feat: add tag operations and todo-tag associations"
```

---

## Task 8: CLI Foundation with Cobra

**Files:**
- Create: `cmd/toki/main.go`
- Create: `cmd/toki/root.go`

**Step 1: Install Cobra and create main entry point**

```bash
go get github.com/spf13/cobra
go get github.com/fatih/color
```

**Step 2: Create main.go**

Create `cmd/toki/main.go`:
```go
// ABOUTME: CLI entry point for toki
// ABOUTME: Initializes and executes root command

package main

import (
	"fmt"
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 3: Create root command**

Create `cmd/toki/root.go`:
```go
// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and database connection

package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/harper/toki/internal/db"
	"github.com/spf13/cobra"
)

var (
	dbPath string
	dbConn *sql.DB
)

var rootCmd = &cobra.Command{
	Use:   "toki",
	Short: "A super simple git-aware todo manager",
	Long: `Toki is a CLI todo manager that organizes tasks by project,
supports rich metadata (priority, tags, notes, due dates),
and automatically detects project context from git repositories.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize database connection
		var err error
		dbConn, err = db.InitDB(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close database connection
		if dbConn != nil {
			return dbConn.Close()
		}
		return nil
	},
}

func init() {
	defaultPath := db.GetDefaultDBPath()
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultPath, "database file path")
}
```

**Step 4: Test basic CLI**

```bash
go run cmd/toki/main.go --help
```

Expected: Shows help output with "toki" description

**Step 5: Commit CLI foundation**

```bash
git add cmd/toki/
git commit -m "feat: add CLI foundation with Cobra"
```

---

## Task 9: Project Commands (part 1 of 2 - add, list)

**Files:**
- Create: `cmd/toki/project.go`

**Step 1: Create project command structure**

Create `cmd/toki/project.go`:
```go
// ABOUTME: Project management commands
// ABOUTME: Handles add, list, set-path, and remove operations

package main

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/git"
	"github.com/harper/toki/internal/models"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"p"},
	Short:   "Manage projects",
}

var projectAddCmd = &cobra.Command{
	Use:     "add <name>",
	Aliases: []string{"a"},
	Short:   "Add a new project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		pathFlag, _ := cmd.Flags().GetString("path")
		var dirPath *string

		if pathFlag != "" {
			normalized, err := git.NormalizePath(pathFlag)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}
			dirPath = &normalized
		}

		project := models.NewProject(name, dirPath)

		if err := db.CreateProject(dbConn, project); err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		color.Green("✓ Created project '%s'", name)
		if dirPath != nil {
			fmt.Printf("  Path: %s\n", *dirPath)
		}

		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := db.ListProjects(dbConn)
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("No projects yet. Create one with 'toki project add <name>'")
			return nil
		}

		color.New(color.Bold).Println("PROJECTS")
		fmt.Println(color.New(color.Faint).Sprint("────────────────────────────────────────"))

		for _, p := range projects {
			fmt.Printf("%s\n", color.New(color.Bold, color.FgCyan).Sprint(p.Name))
			if p.DirectoryPath != nil {
				fmt.Printf("  %s\n", color.New(color.Faint).Sprint(*p.DirectoryPath))
			}
		}

		return nil
	},
}

func init() {
	projectAddCmd.Flags().String("path", "", "directory path to associate with project")

	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
	rootCmd.AddCommand(projectCmd)
}
```

**Step 2: Test project commands**

```bash
go run cmd/toki/main.go project add test-project
go run cmd/toki/main.go project list
```

Expected: Creates project and displays in list

**Step 3: Commit project commands (part 1)**

```bash
git add cmd/toki/project.go
git commit -m "feat: add project add and list commands"
```

---

## Task 10: Project Commands (part 2 of 2 - set-path, remove)

**Files:**
- Modify: `cmd/toki/project.go`

**Step 1: Add set-path command**

Add to `cmd/toki/project.go` before `init()`:
```go
var projectSetPathCmd = &cobra.Command{
	Use:     "set-path <name> <path>",
	Aliases: []string{"sp"},
	Short:   "Set directory path for a project",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		pathArg := args[1]

		project, err := db.GetProjectByName(dbConn, name)
		if err != nil {
			return fmt.Errorf("project not found: %w", err)
		}

		normalized, err := git.NormalizePath(pathArg)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		if err := db.UpdateProjectPath(dbConn, project.ID, &normalized); err != nil {
			return fmt.Errorf("failed to update path: %w", err)
		}

		color.Green("✓ Updated path for '%s'", name)
		fmt.Printf("  Path: %s\n", normalized)

		return nil
	},
}

var projectRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "r"},
	Short:   "Remove a project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		project, err := db.GetProjectByName(dbConn, name)
		if err != nil {
			return fmt.Errorf("project not found: %w", err)
		}

		if err := db.DeleteProject(dbConn, project.ID); err != nil {
			return fmt.Errorf("failed to delete project: %w", err)
		}

		color.Yellow("✓ Removed project '%s' and all its todos", name)

		return nil
	},
}
```

**Step 2: Register new commands in init()**

Modify the `init()` function:
```go
func init() {
	projectAddCmd.Flags().String("path", "", "directory path to associate with project")

	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectSetPathCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	rootCmd.AddCommand(projectCmd)
}
```

**Step 3: Test new commands**

```bash
go run cmd/toki/main.go project set-path test-project .
go run cmd/toki/main.go project list
```

Expected: Path is updated and shown in list

**Step 4: Commit project commands (part 2)**

```bash
git add cmd/toki/project.go
git commit -m "feat: add project set-path and remove commands"
```

---

## Task 11: Todo Add Command with Git Detection

**Files:**
- Create: `cmd/toki/add.go`
- Create: `cmd/toki/context.go`

**Step 1: Create context detection helper**

Create `cmd/toki/context.go`:
```go
// ABOUTME: Context detection for git-aware project lookup
// ABOUTME: Determines current project from git repo or prompts user

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/git"
	"github.com/harper/toki/internal/models"
)

// detectProjectContext attempts to find project from current directory
func detectProjectContext() (*uuid.UUID, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Try to find git root
	gitRoot, err := git.FindGitRoot(cwd)
	if err != nil {
		// Not in a git repo
		return nil, nil
	}

	// Look up project by path
	project, err := db.GetProjectByPath(dbConn, gitRoot)
	if err == nil {
		return &project.ID, nil
	}

	// Project not found - offer to create
	projectName := filepath.Base(gitRoot)
	fmt.Printf("Git repository detected: %s\n", gitRoot)
	fmt.Printf("Create project '%s'? [Y/n]: ", projectName)

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" || response == "y" || response == "yes" {
		project := models.NewProject(projectName, &gitRoot)
		if err := db.CreateProject(dbConn, project); err != nil {
			return nil, fmt.Errorf("failed to create project: %w", err)
		}
		fmt.Printf("✓ Created project '%s'\n", projectName)
		return &project.ID, nil
	}

	return nil, nil
}

// getProjectID gets project ID from flag or context
func getProjectID(projectFlag string) (*uuid.UUID, error) {
	if projectFlag != "" {
		project, err := db.GetProjectByName(dbConn, projectFlag)
		if err != nil {
			return nil, fmt.Errorf("project '%s' not found", projectFlag)
		}
		return &project.ID, nil
	}

	// Try context detection
	projectID, err := detectProjectContext()
	if err != nil {
		return nil, err
	}

	if projectID != nil {
		return projectID, nil
	}

	return nil, fmt.Errorf("no project context detected. Use --project or run in a git repository")
}
```

**Step 2: Create add command**

Create `cmd/toki/add.go`:
```go
// ABOUTME: Todo add command with git-aware context detection
// ABOUTME: Creates todos with metadata (priority, tags, notes, due date)

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:     "add <description>",
	Aliases: []string{"a"},
	Short:   "Add a new todo",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := strings.Join(args, " ")

		if len(description) < 3 {
			return fmt.Errorf("description must be at least 3 characters")
		}

		projectFlag, _ := cmd.Flags().GetString("project")
		projectID, err := getProjectID(projectFlag)
		if err != nil {
			return err
		}

		todo := models.NewTodo(*projectID, description)

		// Handle optional flags
		if priority, _ := cmd.Flags().GetString("priority"); priority != "" {
			priority = strings.ToLower(priority)
			if priority != "low" && priority != "medium" && priority != "high" {
				return fmt.Errorf("priority must be low, medium, or high")
			}
			todo.Priority = &priority
		}

		if notes, _ := cmd.Flags().GetString("notes"); notes != "" {
			todo.Notes = &notes
		}

		if dueStr, _ := cmd.Flags().GetString("due"); dueStr != "" {
			dueDate, err := time.Parse("2006-01-02", dueStr)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
			todo.DueDate = &dueDate
		}

		// Create todo
		if err := db.CreateTodo(dbConn, todo); err != nil {
			return fmt.Errorf("failed to create todo: %w", err)
		}

		// Handle tags
		if tagsStr, _ := cmd.Flags().GetString("tags"); tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					if err := db.AddTagToTodo(dbConn, todo.ID, tag); err != nil {
						return fmt.Errorf("failed to add tag: %w", err)
					}
				}
			}
		}

		color.Green("✓ Added todo")
		fmt.Printf("  %s %s\n", color.New(color.Faint).Sprint(todo.ID.String()[:6]), description)

		return nil
	},
}

func init() {
	addCmd.Flags().StringP("project", "p", "", "project name")
	addCmd.Flags().String("priority", "", "priority (low, medium, high)")
	addCmd.Flags().String("tags", "", "comma-separated tags")
	addCmd.Flags().String("notes", "", "additional notes")
	addCmd.Flags().String("due", "", "due date (YYYY-MM-DD)")

	rootCmd.AddCommand(addCmd)
}
```

**Step 3: Test add command**

```bash
# In a git repo directory
go run cmd/toki/main.go add "test todo" --priority high --tags "backend,urgent"
```

Expected: Creates todo with context detection

**Step 4: Commit add command**

```bash
git add cmd/toki/add.go cmd/toki/context.go
git commit -m "feat: add todo add command with git-aware context"
```

---

## Task 12: Todo List Command with Formatting

**Files:**
- Create: `cmd/toki/list.go`
- Create: `internal/ui/format.go`

**Step 1: Create formatting utilities**

Create `internal/ui/format.go`:
```go
// ABOUTME: Output formatting and color utilities
// ABOUTME: Formats todos and projects for display with colors

package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/models"
)

var (
	boldCyan   = color.New(color.Bold, color.FgCyan)
	faint      = color.New(color.Faint)
	red        = color.New(color.FgRed)
	yellow     = color.New(color.FgYellow)
	priorityHigh   = color.New(color.FgRed, color.Bold)
	priorityMedium = color.New(color.FgYellow)
	priorityLow    = color.New(color.Faint)
)

// FormatTodo formats a single todo for display
func FormatTodo(todo *models.Todo, tags []*models.Tag) string {
	var builder strings.Builder

	// First line: ID + Priority + Description
	builder.WriteString("  ")
	builder.WriteString(faint.Sprint(todo.ID.String()[:6]))
	builder.WriteString("  ")

	if todo.Priority != nil {
		priority := strings.ToUpper(*todo.Priority)
		switch *todo.Priority {
		case "high":
			builder.WriteString(priorityHigh.Sprintf("[%s] ", priority))
		case "medium":
			builder.WriteString(priorityMedium.Sprintf("[%s] ", priority))
		case "low":
			builder.WriteString(priorityLow.Sprintf("[%s] ", priority))
		}
	}

	builder.WriteString(todo.Description)
	builder.WriteString("\n")

	// Second line: Metadata
	var metadata []string

	if todo.DueDate != nil {
		dueStr := todo.DueDate.Format("2006-01-02")
		if todo.DueDate.Before(time.Now()) {
			dueStr = red.Sprint(dueStr + " (overdue)")
		}
		metadata = append(metadata, "Due: "+dueStr)
	}

	if len(tags) > 0 {
		tagNames := make([]string, len(tags))
		for i, tag := range tags {
			tagNames[i] = tag.Name
		}
		metadata = append(metadata, "Tags: "+strings.Join(tagNames, ", "))
	}

	if len(metadata) > 0 {
		builder.WriteString("          ")
		builder.WriteString(faint.Sprint(strings.Join(metadata, " | ")))
		builder.WriteString("\n")
	}

	return builder.String()
}

// FormatProjectHeader formats a project header
func FormatProjectHeader(project *models.Project) string {
	header := fmt.Sprintf("PROJECT: %s", boldCyan.Sprint(project.Name))
	if project.DirectoryPath != nil {
		header += faint.Sprintf(" (%s)", *project.DirectoryPath)
	}
	return header
}

// FormatSeparator creates a separator line
func FormatSeparator() string {
	return faint.Sprint("─────────────────────────────────────────────")
}
```

**Step 2: Create list command**

Create `cmd/toki/list.go`:
```go
// ABOUTME: Todo list command with filtering and formatting
// ABOUTME: Supports project, tag, status, and priority filters

package main

import (
	"database/sql"
	"fmt"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List todos",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse filters
		var projectID *uuid.UUID
		var done *bool
		var priority *string

		projectFlag, _ := cmd.Flags().GetString("project")
		if projectFlag != "" {
			id, err := getProjectID(projectFlag)
			if err != nil {
				return err
			}
			projectID = id
		} else {
			// Try context detection
			id, _ := detectProjectContext()
			if id != nil {
				projectID = id
			}
		}

		if cmd.Flags().Changed("done") {
			doneVal := true
			done = &doneVal
		} else if cmd.Flags().Changed("pending") {
			pendingVal := false
			done = &pendingVal
		}

		if priorityFlag, _ := cmd.Flags().GetString("priority"); priorityFlag != "" {
			priority = &priorityFlag
		}

		// Get todos
		todos, err := db.ListTodos(dbConn, projectID, done, priority)
		if err != nil {
			return fmt.Errorf("failed to list todos: %w", err)
		}

		if len(todos) == 0 {
			fmt.Println("No todos found. Add one with 'toki add <description>'")
			return nil
		}

		// Group by project
		projectTodos := make(map[uuid.UUID][]*struct {
			todo *db.Todo
			tags []*db.Tag
		})

		for _, todo := range todos {
			tags, _ := db.GetTodoTags(dbConn, todo.ID)
			if projectTodos[todo.ProjectID] == nil {
				projectTodos[todo.ProjectID] = []*struct {
					todo *db.Todo
					tags []*db.Tag
				}{}
			}
			projectTodos[todo.ProjectID] = append(projectTodos[todo.ProjectID], &struct {
				todo *db.Todo
				tags []*db.Tag
			}{todo, tags})
		}

		// Display grouped by project
		totalCount := 0
		for projID, items := range projectTodos {
			project, err := db.GetProjectByID(dbConn, projID)
			if err != nil {
				continue
			}

			fmt.Println(ui.FormatProjectHeader(project))
			fmt.Println(ui.FormatSeparator())

			for _, item := range items {
				fmt.Print(ui.FormatTodo(item.todo, item.tags))
				totalCount++
			}

			fmt.Println()
		}

		fmt.Println(ui.FormatSeparator())
		statusText := "pending"
		if done != nil && *done {
			statusText = "completed"
		}
		fmt.Printf("%d %s todo(s) across %d project(s)\n", totalCount, statusText, len(projectTodos))

		return nil
	},
}

func init() {
	listCmd.Flags().StringP("project", "p", "", "filter by project")
	listCmd.Flags().StringP("tag", "t", "", "filter by tag")
	listCmd.Flags().Bool("done", false, "show completed todos")
	listCmd.Flags().Bool("pending", false, "show pending todos only")
	listCmd.Flags().String("priority", "", "filter by priority")

	rootCmd.AddCommand(listCmd)
}
```

**Step 3: Fix type issues in list.go**

The list command references `db.Todo` and `db.Tag` but should use `models.Todo` and `models.Tag`. Update the imports and type references:

```go
// In cmd/toki/list.go, change:
projectTodos := make(map[uuid.UUID][]*struct {
	todo *models.Todo
	tags []*models.Tag
})

// And update the append:
projectTodos[todo.ProjectID] = append(projectTodos[todo.ProjectID], &struct {
	todo *models.Todo
	tags []*models.Tag
}{todo, tags})
```

**Step 4: Test list command**

```bash
go run cmd/toki/main.go list
```

Expected: Displays todos grouped by project with formatting

**Step 5: Commit list command**

```bash
git add cmd/toki/list.go internal/ui/format.go
git commit -m "feat: add todo list command with formatting"
```

---

## Task 13: Todo Done/Undone Commands

**Files:**
- Create: `cmd/toki/done.go`

**Step 1: Create done command**

Create `cmd/toki/done.go`:
```go
// ABOUTME: Todo done and undone commands
// ABOUTME: Marks todos as complete or incomplete using UUID prefixes

package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:     "done <uuid-prefix>",
	Aliases: []string{"d"},
	Short:   "Mark a todo as done",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		todo, err := db.GetTodoByPrefix(dbConn, prefix)
		if err != nil {
			return err
		}

		todo.MarkDone()

		if err := db.UpdateTodo(dbConn, todo); err != nil {
			return fmt.Errorf("failed to update todo: %w", err)
		}

		color.Green("✓ Marked todo as done")
		fmt.Printf("  %s\n", todo.Description)

		return nil
	},
}

var undoneCmd = &cobra.Command{
	Use:     "undone <uuid-prefix>",
	Aliases: []string{"ud"},
	Short:   "Mark a todo as not done",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		todo, err := db.GetTodoByPrefix(dbConn, prefix)
		if err != nil {
			return err
		}

		todo.MarkUndone()

		if err := db.UpdateTodo(dbConn, todo); err != nil {
			return fmt.Errorf("failed to update todo: %w", err)
		}

		color.Yellow("✓ Marked todo as not done")
		fmt.Printf("  %s\n", todo.Description)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(undoneCmd)
}
```

**Step 2: Test done/undone commands**

```bash
# Get a todo ID from list
go run cmd/toki/main.go list
# Use first 6 chars of a todo ID
go run cmd/toki/main.go done a3f2b9
go run cmd/toki/main.go list --done
go run cmd/toki/main.go undone a3f2b9
```

Expected: Todo status changes correctly

**Step 3: Commit done commands**

```bash
git add cmd/toki/done.go
git commit -m "feat: add done and undone commands"
```

---

## Task 14: Todo Remove and Tag Commands

**Files:**
- Create: `cmd/toki/remove.go`
- Create: `cmd/toki/tag.go`

**Step 1: Create remove command**

Create `cmd/toki/remove.go`:
```go
// ABOUTME: Todo remove command
// ABOUTME: Deletes todos by UUID prefix

package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <uuid-prefix>",
	Aliases: []string{"rm"},
	Short:   "Remove a todo",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		todo, err := db.GetTodoByPrefix(dbConn, prefix)
		if err != nil {
			return err
		}

		desc := todo.Description

		if err := db.DeleteTodo(dbConn, todo.ID); err != nil {
			return fmt.Errorf("failed to delete todo: %w", err)
		}

		color.Yellow("✓ Removed todo")
		fmt.Printf("  %s\n", desc)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
```

**Step 2: Create tag commands**

Create `cmd/toki/tag.go`:
```go
// ABOUTME: Tag management commands
// ABOUTME: Add/remove tags from todos and list all tags

package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
}

var tagAddCmd = &cobra.Command{
	Use:     "add <uuid-prefix> <tag>",
	Aliases: []string{"a"},
	Short:   "Add a tag to a todo",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		tagName := strings.ToLower(args[1])

		todo, err := db.GetTodoByPrefix(dbConn, prefix)
		if err != nil {
			return err
		}

		if err := db.AddTagToTodo(dbConn, todo.ID, tagName); err != nil {
			return fmt.Errorf("failed to add tag: %w", err)
		}

		color.Green("✓ Added tag '%s'", tagName)
		fmt.Printf("  %s\n", todo.Description)

		return nil
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:     "remove <uuid-prefix> <tag>",
	Aliases: []string{"rm", "r"},
	Short:   "Remove a tag from a todo",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		tagName := strings.ToLower(args[1])

		todo, err := db.GetTodoByPrefix(dbConn, prefix)
		if err != nil {
			return err
		}

		if err := db.RemoveTagFromTodo(dbConn, todo.ID, tagName); err != nil {
			return fmt.Errorf("failed to remove tag: %w", err)
		}

		color.Yellow("✓ Removed tag '%s'", tagName)
		fmt.Printf("  %s\n", todo.Description)

		return nil
	},
}

var tagsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		tags, err := db.ListAllTags(dbConn)
		if err != nil {
			return fmt.Errorf("failed to list tags: %w", err)
		}

		if len(tags) == 0 {
			fmt.Println("No tags yet.")
			return nil
		}

		color.New(color.Bold).Println("TAGS")
		for _, tag := range tags {
			fmt.Printf("  • %s\n", tag.Name)
		}

		return nil
	},
}

func init() {
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRemoveCmd)
	tagCmd.AddCommand(tagsListCmd)
	rootCmd.AddCommand(tagCmd)
}
```

**Step 3: Test remove and tag commands**

```bash
go run cmd/toki/main.go tag add a3f2b9 urgent
go run cmd/toki/main.go tag list
go run cmd/toki/main.go tag remove a3f2b9 urgent
go run cmd/toki/main.go remove a3f2b9
```

Expected: Tags are added/removed, todo is deleted

**Step 4: Commit remove and tag commands**

```bash
git add cmd/toki/remove.go cmd/toki/tag.go
git commit -m "feat: add remove and tag commands"
```

---

## Task 15: Integration Tests and Final Build

**Files:**
- Create: `test/integration_test.go`
- Create: `Makefile`

**Step 1: Create integration test**

Create `test/integration_test.go`:
```go
// ABOUTME: Integration tests for full workflow
// ABOUTME: Tests project creation, todo CRUD, git detection end-to-end

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullWorkflow(t *testing.T) {
	// Build toki binary
	buildCmd := exec.Command("go", "build", "-o", "toki", "./cmd/toki")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}
	defer os.Remove("toki")

	// Use temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	run := func(args ...string) (string, error) {
		fullArgs := append([]string{"--db", dbPath}, args...)
		cmd := exec.Command("./toki", fullArgs...)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Create project
	output, err := run("project", "add", "test-project")
	if err != nil {
		t.Fatalf("Failed to create project: %v\n%s", err, output)
	}

	if !strings.Contains(output, "Created project") {
		t.Error("Expected success message")
	}

	// Add todo
	output, err = run("add", "test todo", "--project", "test-project", "--priority", "high")
	if err != nil {
		t.Fatalf("Failed to add todo: %v\n%s", err, output)
	}

	// List todos
	output, err = run("list", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to list todos: %v\n%s", err, output)
	}

	if !strings.Contains(output, "test todo") {
		t.Error("Todo not found in list")
	}

	if !strings.Contains(output, "HIGH") {
		t.Error("Priority not shown")
	}

	t.Logf("Integration test passed!\n%s", output)
}
```

**Step 2: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: build test install clean

build:
	go build -o toki cmd/toki/main.go

test:
	go test ./internal/... -v
	go test ./test/... -v

install:
	go install ./cmd/toki

clean:
	rm -f toki
	go clean

.DEFAULT_GOAL := build
```

**Step 3: Run full test suite**

```bash
make test
```

Expected: All tests pass (unit + integration)

**Step 4: Build and test binary**

```bash
make build
./toki --help
./toki project add demo
./toki add "test the binary" --project demo
./toki list
```

Expected: Binary works correctly

**Step 5: Commit tests and build setup**

```bash
git add test/ Makefile
git commit -m "test: add integration tests and build configuration"
```

---

## Task 16: Documentation and Final Polish

**Files:**
- Modify: `README.md`
- Create: `.github/workflows/test.yml` (optional)

**Step 1: Update README with usage examples**

Update `README.md`:
```markdown
# Toki

A super simple git-aware CLI todo manager.

## Features

- **Git-aware context detection** - Automatically associates todos with projects based on your current directory
- **Rich metadata** - Priority, tags, notes, and due dates
- **UUID-based identifiers** - Stable IDs with short prefix matching
- **Clean CLI** - Intuitive commands with short aliases
- **SQLite storage** - Fast, reliable, single-file database

## Installation

```bash
go install github.com/harper/toki/cmd/toki@latest
```

Or build from source:

```bash
git clone https://github.com/harper/toki
cd toki
make install
```

## Quick Start

```bash
# Create a project
toki project add myproject --path ~/code/myproject

# Add a todo (from within the project directory)
cd ~/code/myproject
toki add "implement feature" --priority high --tags backend,api

# List todos
toki list

# Mark done (use first 6+ chars of UUID)
toki done a3f2b9

# Add tags
toki tag add a3f2b9 urgent

# Remove todo
toki remove a3f2b9
```

## Commands

### Projects

```bash
toki project add <name> [--path <dir>]    # Create project
toki project list                          # List projects
toki project set-path <name> <path>        # Link directory
toki project remove <name>                 # Delete project
```

### Todos

```bash
toki add <description> [flags]             # Create todo
  --project, -p <name>                     # Specify project
  --priority <low|medium|high>             # Set priority
  --tags <tag1,tag2>                       # Add tags
  --notes <text>                           # Add notes
  --due <YYYY-MM-DD>                       # Set due date

toki list [flags]                          # List todos
  --project, -p <name>                     # Filter by project
  --tag, -t <tag>                          # Filter by tag
  --done / --pending                       # Filter by status
  --priority <level>                       # Filter by priority

toki done <uuid-prefix>                    # Mark complete
toki undone <uuid-prefix>                  # Mark incomplete
toki remove <uuid-prefix>                  # Delete todo
```

### Tags

```bash
toki tag add <uuid-prefix> <tag>           # Add tag to todo
toki tag remove <uuid-prefix> <tag>        # Remove tag
toki tag list                              # Show all tags
```

## Git-Aware Context

When you run `toki add` or `toki list` from within a git repository:

1. Toki finds the repository root
2. Looks up the associated project
3. If no project exists, offers to create one
4. Uses that project as the default context

This means you can just run `toki add "task"` without specifying `--project` when you're in the right directory.

## Development

```bash
# Run tests
make test

# Build binary
make build

# Install locally
make install
```

## Data Storage

Toki stores all data in `~/.local/share/toki/toki.db` (XDG standard).

## Design

See `docs/plans/2025-11-29-toki-todo-manager-design.md` for the full design document.

## License

MIT
```

**Step 2: Commit documentation**

```bash
git add README.md
git commit -m "docs: update README with comprehensive usage guide"
```

**Step 3: Final verification**

```bash
make clean
make test
make build
./toki --version || ./toki --help
```

Expected: Everything builds and runs cleanly

**Step 4: Create final commit**

```bash
git log --oneline
git status
```

Verify clean working tree and good commit history.

---

## Summary

This plan implements Toki in 16 bite-sized tasks following TDD:

1. ✅ Project initialization
2. ✅ Data models
3. ✅ Database layer (connection & migrations)
4. ✅ Database layer (project operations)
5. ✅ Git detection logic
6. ✅ Database layer (todo operations)
7. ✅ Database layer (tag operations)
8. ✅ CLI foundation with Cobra
9. ✅ Project commands (add, list)
10. ✅ Project commands (set-path, remove)
11. ✅ Todo add command with git detection
12. ✅ Todo list command with formatting
13. ✅ Todo done/undone commands
14. ✅ Todo remove and tag commands
15. ✅ Integration tests and build
16. ✅ Documentation and polish

Each task follows the TDD cycle: write test → run to see fail → implement → run to see pass → commit.

**Related skills:**
- @superpowers:test-driven-development
- @superpowers:verification-before-completion
- @superpowers:systematic-debugging (if issues arise)
