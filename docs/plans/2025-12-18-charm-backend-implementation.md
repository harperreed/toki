# Charm Backend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the vault sync backend with Charm KV for data storage and synchronization.

**Architecture:** Charm KV becomes the primary data store (replacing SQLite). Data stored as JSON with type-prefixed keys (`project:{uuid}`, `todo:{uuid}`, `tag:{name}`). Tags denormalized into todos. Client-side filtering for queries.

**Tech Stack:** Go, github.com/2389-research/charm (KV, Crypt, Accounts), BadgerDB (under the hood)

**Default Server:** `charm.2389.dev`

---

## Phase 1: Foundation

### Task 1.1: Add Charm Dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the charm dependency**

```bash
cd /Users/harper/Public/src/personal/suite/toki-charm/.worktrees/charm-backend
go get github.com/2389-research/charm@latest
```

**Step 2: Verify dependency added**

```bash
grep "2389-research/charm" go.mod
```

Expected: Line showing `github.com/2389-research/charm vX.X.X`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add 2389-research/charm dependency"
```

---

### Task 1.2: Create Charm Models

**Files:**
- Create: `internal/charm/models.go`
- Create: `internal/charm/models_test.go`

**Step 1: Write the test for model serialization**

Create `internal/charm/models_test.go`:

```go
// ABOUTME: Tests for Charm KV data models
// ABOUTME: Verifies JSON serialization/deserialization for projects, todos, and tags

package charm

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProjectSerialization(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	p := Project{
		ID:            id,
		Name:          "test-project",
		DirectoryPath: "/path/to/project",
		CreatedAt:     now,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal project: %v", err)
	}

	var decoded Project
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal project: %v", err)
	}

	if decoded.ID != p.ID {
		t.Errorf("ID mismatch: got %v, want %v", decoded.ID, p.ID)
	}
	if decoded.Name != p.Name {
		t.Errorf("Name mismatch: got %v, want %v", decoded.Name, p.Name)
	}
	if decoded.DirectoryPath != p.DirectoryPath {
		t.Errorf("DirectoryPath mismatch: got %v, want %v", decoded.DirectoryPath, p.DirectoryPath)
	}
	if !decoded.CreatedAt.Equal(p.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", decoded.CreatedAt, p.CreatedAt)
	}
}

func TestTodoSerialization(t *testing.T) {
	id := uuid.New()
	projectID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	dueDate := now.Add(24 * time.Hour)
	completedAt := now.Add(1 * time.Hour)

	todo := Todo{
		ID:          id,
		ProjectID:   projectID,
		ProjectName: "test-project",
		Description: "Test todo",
		Done:        true,
		Priority:    "high",
		Notes:       "Some notes",
		Tags:        []string{"urgent", "backend"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: &completedAt,
		DueDate:     &dueDate,
	}

	data, err := json.Marshal(todo)
	if err != nil {
		t.Fatalf("failed to marshal todo: %v", err)
	}

	var decoded Todo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal todo: %v", err)
	}

	if decoded.ID != todo.ID {
		t.Errorf("ID mismatch")
	}
	if decoded.ProjectID != todo.ProjectID {
		t.Errorf("ProjectID mismatch")
	}
	if decoded.ProjectName != todo.ProjectName {
		t.Errorf("ProjectName mismatch")
	}
	if len(decoded.Tags) != 2 || decoded.Tags[0] != "urgent" || decoded.Tags[1] != "backend" {
		t.Errorf("Tags mismatch: got %v", decoded.Tags)
	}
	if decoded.CompletedAt == nil || !decoded.CompletedAt.Equal(*todo.CompletedAt) {
		t.Errorf("CompletedAt mismatch")
	}
	if decoded.DueDate == nil || !decoded.DueDate.Equal(*todo.DueDate) {
		t.Errorf("DueDate mismatch")
	}
}

func TestTagSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tag := Tag{
		Name:      "urgent",
		CreatedAt: now,
	}

	data, err := json.Marshal(tag)
	if err != nil {
		t.Fatalf("failed to marshal tag: %v", err)
	}

	var decoded Tag
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal tag: %v", err)
	}

	if decoded.Name != tag.Name {
		t.Errorf("Name mismatch: got %v, want %v", decoded.Name, tag.Name)
	}
	if !decoded.CreatedAt.Equal(tag.CreatedAt) {
		t.Errorf("CreatedAt mismatch")
	}
}

func TestTodoWithNilOptionalFields(t *testing.T) {
	id := uuid.New()
	projectID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	todo := Todo{
		ID:          id,
		ProjectID:   projectID,
		ProjectName: "test-project",
		Description: "Test todo",
		Done:        false,
		Priority:    "medium",
		Tags:        []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: nil,
		DueDate:     nil,
	}

	data, err := json.Marshal(todo)
	if err != nil {
		t.Fatalf("failed to marshal todo: %v", err)
	}

	var decoded Todo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal todo: %v", err)
	}

	if decoded.CompletedAt != nil {
		t.Errorf("CompletedAt should be nil")
	}
	if decoded.DueDate != nil {
		t.Errorf("DueDate should be nil")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/charm/... -v
```

Expected: FAIL - package doesn't exist

**Step 3: Create the models**

Create `internal/charm/models.go`:

```go
// ABOUTME: Data models for Charm KV storage
// ABOUTME: Defines Project, Todo, and Tag types with JSON serialization

package charm

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a todo project stored in Charm KV.
type Project struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	DirectoryPath string    `json:"directory_path,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Todo represents a todo item stored in Charm KV.
// Tags are denormalized as a string slice for simpler querying.
type Todo struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	ProjectName string     `json:"project_name"`
	Description string     `json:"description"`
	Done        bool       `json:"done"`
	Priority    string     `json:"priority"`
	Notes       string     `json:"notes,omitempty"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// Tag represents a known tag stored in Charm KV.
// Tags exist primarily for autocomplete; the source of truth is in Todo.Tags.
type Tag struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Key prefixes for Charm KV storage.
const (
	ProjectKeyPrefix = "project:"
	TodoKeyPrefix    = "todo:"
	TagKeyPrefix     = "tag:"
)

// ProjectKey returns the KV key for a project.
func ProjectKey(id uuid.UUID) string {
	return ProjectKeyPrefix + id.String()
}

// TodoKey returns the KV key for a todo.
func TodoKey(id uuid.UUID) string {
	return TodoKeyPrefix + id.String()
}

// TagKey returns the KV key for a tag.
func TagKey(name string) string {
	return TagKeyPrefix + name
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/charm/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/charm/
git commit -m "feat(charm): add data models for KV storage"
```

---

### Task 1.3: Create Charm Client Wrapper

**Files:**
- Create: `internal/charm/client.go`
- Create: `internal/charm/client_test.go`

**Step 1: Write the test for client initialization**

Create `internal/charm/client_test.go`:

```go
// ABOUTME: Tests for Charm client wrapper
// ABOUTME: Verifies client creation, configuration, and basic operations

package charm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Use temp directory for test data
	tmpDir, err := os.MkdirTemp("", "toki-charm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set CHARM_DATA_DIR to isolate test
	os.Setenv("CHARM_DATA_DIR", tmpDir)
	defer os.Unsetenv("CHARM_DATA_DIR")

	client, err := NewClient("toki-test")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if client.kv == nil {
		t.Error("kv should not be nil")
	}
}

func TestClientConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server != "charm.2389.dev" {
		t.Errorf("default server should be charm.2389.dev, got %s", cfg.Server)
	}
	if !cfg.AutoSync {
		t.Error("AutoSync should be true by default")
	}
}

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("CHARM_HOST", "custom.server.com")
	defer os.Unsetenv("CHARM_HOST")

	cfg := DefaultConfig()
	cfg.ApplyEnv()

	if cfg.Server != "custom.server.com" {
		t.Errorf("server should be custom.server.com, got %s", cfg.Server)
	}
}

func TestConfigPath(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	tmpDir, _ := os.MkdirTemp("", "toki-config-test-*")
	defer os.RemoveAll(tmpDir)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	path := ConfigPath()
	expected := filepath.Join(tmpDir, "toki", "charm.json")
	if path != expected {
		t.Errorf("ConfigPath() = %s, want %s", path, expected)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/charm/... -v -run TestNewClient
```

Expected: FAIL - NewClient not defined

**Step 3: Write minimal implementation**

Create `internal/charm/client.go`:

```go
// ABOUTME: Charm client wrapper for Toki
// ABOUTME: Handles KV store initialization, configuration, and lifecycle

package charm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/2389-research/charm/kv"
)

// Config holds Charm client configuration.
type Config struct {
	Server   string `json:"server"`
	AutoSync bool   `json:"auto_sync"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Server:   "charm.2389.dev",
		AutoSync: true,
	}
}

// ApplyEnv applies environment variable overrides to the config.
func (c *Config) ApplyEnv() {
	if host := os.Getenv("CHARM_HOST"); host != "" {
		c.Server = host
	}
}

// ConfigPath returns the path to the Charm config file.
func ConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "toki", "charm.json")
}

// LoadConfig loads configuration from disk, falling back to defaults.
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			cfg.ApplyEnv()
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.ApplyEnv()
	return cfg, nil
}

// SaveConfig saves configuration to disk.
func SaveConfig(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Client wraps the Charm KV store for Toki operations.
type Client struct {
	kv     *kv.KV
	config *Config
}

// NewClient creates a new Charm client with the given database name.
func NewClient(dbName string) (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Set CHARM_HOST for the underlying library
	if cfg.Server != "" {
		os.Setenv("CHARM_HOST", cfg.Server)
	}

	db, err := kv.OpenWithDefaults(dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to open kv store: %w", err)
	}

	return &Client{
		kv:     db,
		config: cfg,
	}, nil
}

// Close closes the underlying KV store.
func (c *Client) Close() error {
	if c.kv != nil {
		return c.kv.Close()
	}
	return nil
}

// Sync synchronizes local data with the Charm server.
func (c *Client) Sync() error {
	return c.kv.Sync()
}

// KV returns the underlying KV store for direct access.
func (c *Client) KV() *kv.KV {
	return c.kv
}

// Config returns the current configuration.
func (c *Client) Config() *Config {
	return c.config
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/charm/... -v
```

Expected: PASS (may need network/charm setup - tests should be skipped if charm not available)

**Step 5: Commit**

```bash
git add internal/charm/
git commit -m "feat(charm): add client wrapper with configuration"
```

---

## Phase 2: KV Operations

### Task 2.1: Implement Project CRUD

**Files:**
- Create: `internal/charm/projects.go`
- Create: `internal/charm/projects_test.go`

**Step 1: Write the tests**

Create `internal/charm/projects_test.go`:

```go
// ABOUTME: Tests for project CRUD operations
// ABOUTME: Verifies create, read, update, delete, and list for projects

package charm

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupTestClient(t *testing.T) (*Client, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "toki-charm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	os.Setenv("CHARM_DATA_DIR", tmpDir)

	client, err := NewClient("toki-test-" + uuid.New().String()[:8])
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create client: %v", err)
	}

	cleanup := func() {
		client.Close()
		os.RemoveAll(tmpDir)
		os.Unsetenv("CHARM_DATA_DIR")
	}

	return client, cleanup
}

func TestCreateProject(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:            uuid.New(),
		Name:          "test-project",
		DirectoryPath: "/path/to/project",
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
	}

	err := client.CreateProject(project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Verify it was stored
	retrieved, err := client.GetProject(project.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if retrieved.Name != project.Name {
		t.Errorf("Name mismatch: got %v, want %v", retrieved.Name, project.Name)
	}
}

func TestGetProjectByName(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "unique-name",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	retrieved, err := client.GetProjectByName("unique-name")
	if err != nil {
		t.Fatalf("failed to get project by name: %v", err)
	}

	if retrieved.ID != project.ID {
		t.Errorf("ID mismatch")
	}
}

func TestListProjects(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	// Create multiple projects
	for i := 0; i < 3; i++ {
		project := &Project{
			ID:        uuid.New(),
			Name:      fmt.Sprintf("project-%d", i),
			CreatedAt: time.Now().UTC(),
		}
		if err := client.CreateProject(project); err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
	}

	projects, err := client.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("expected 3 projects, got %d", len(projects))
	}
}

func TestUpdateProject(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "original-name",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.DirectoryPath = "/new/path"
	if err := client.UpdateProject(project); err != nil {
		t.Fatalf("failed to update project: %v", err)
	}

	retrieved, err := client.GetProject(project.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if retrieved.DirectoryPath != "/new/path" {
		t.Errorf("DirectoryPath not updated")
	}
}

func TestDeleteProject(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "to-delete",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if err := client.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	_, err := client.GetProject(project.ID)
	if err == nil {
		t.Error("expected error getting deleted project")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/charm/... -v -run TestCreateProject
```

Expected: FAIL - CreateProject not defined

**Step 3: Implement project operations**

Create `internal/charm/projects.go`:

```go
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

	var projects []*Project
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
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/charm/... -v -run "Test.*Project"
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/charm/projects.go internal/charm/projects_test.go
git commit -m "feat(charm): implement project CRUD operations"
```

---

### Task 2.2: Implement Todo CRUD

**Files:**
- Create: `internal/charm/todos.go`
- Create: `internal/charm/todos_test.go`

**Step 1: Write the tests**

Create `internal/charm/todos_test.go`:

```go
// ABOUTME: Tests for todo CRUD operations
// ABOUTME: Verifies create, read, update, delete, list, and filtering for todos

package charm

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateTodo(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		ProjectName: "test-project",
		Description: "Test todo",
		Done:        false,
		Priority:    "medium",
		Tags:        []string{"test"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	retrieved, err := client.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if retrieved.Description != todo.Description {
		t.Errorf("Description mismatch")
	}
}

func TestListTodosWithFilters(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	projectID := uuid.New()

	// Create todos with different attributes
	todos := []*Todo{
		{ID: uuid.New(), ProjectID: projectID, ProjectName: "proj", Description: "Done high", Done: true, Priority: "high", Tags: []string{"urgent"}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), ProjectID: projectID, ProjectName: "proj", Description: "Not done medium", Done: false, Priority: "medium", Tags: []string{"backend"}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), ProjectID: uuid.New(), ProjectName: "other", Description: "Other project", Done: false, Priority: "low", Tags: []string{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}

	for _, todo := range todos {
		if err := client.CreateTodo(todo); err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}
	}

	// Test filter by project
	filter := &TodoFilter{ProjectID: &projectID}
	result, err := client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 todos for project, got %d", len(result))
	}

	// Test filter by done status
	done := false
	filter = &TodoFilter{Done: &done}
	result, err = client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 incomplete todos, got %d", len(result))
	}

	// Test filter by priority
	priority := "high"
	filter = &TodoFilter{Priority: &priority}
	result, err = client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 high priority todo, got %d", len(result))
	}

	// Test filter by tag
	tag := "urgent"
	filter = &TodoFilter{Tag: &tag}
	result, err = client.ListTodos(filter)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 todo with urgent tag, got %d", len(result))
	}
}

func TestGetTodoByPrefix(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	id := uuid.MustParse("abc12345-1234-1234-1234-123456789abc")
	todo := &Todo{
		ID:          id,
		ProjectID:   uuid.New(),
		ProjectName: "test",
		Description: "Test",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Should find by prefix
	retrieved, err := client.GetTodoByPrefix("abc1")
	if err != nil {
		t.Fatalf("failed to get by prefix: %v", err)
	}
	if retrieved.ID != id {
		t.Errorf("wrong todo returned")
	}
}

func TestMarkTodoDone(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		ProjectName: "test",
		Description: "Test",
		Done:        false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := client.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := client.MarkTodoDone(todo.ID, true); err != nil {
		t.Fatalf("failed to mark done: %v", err)
	}

	retrieved, err := client.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if !retrieved.Done {
		t.Error("todo should be done")
	}
	if retrieved.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/charm/... -v -run TestCreateTodo
```

Expected: FAIL - CreateTodo not defined

**Step 3: Implement todo operations**

Create `internal/charm/todos.go`:

```go
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

	var matches []*Todo
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
		var ids []string
		for _, m := range matches {
			ids = append(ids, m.ID.String()[:8])
		}
		return nil, fmt.Errorf("ambiguous prefix '%s', matches: %s", prefix, strings.Join(ids, ", "))
	}

	return matches[0], nil
}

// ListTodos returns todos matching the given filter.
func (c *Client) ListTodos(filter *TodoFilter) ([]*Todo, error) {
	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var todos []*Todo
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

		// Apply filters
		if filter != nil {
			if filter.ProjectID != nil && todo.ProjectID != *filter.ProjectID {
				continue
			}
			if filter.Done != nil && todo.Done != *filter.Done {
				continue
			}
			if filter.Priority != nil && todo.Priority != *filter.Priority {
				continue
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
					continue
				}
			}
			if filter.Overdue != nil && *filter.Overdue {
				if todo.Done || todo.DueDate == nil || todo.DueDate.After(now) {
					continue
				}
			}
		}

		todos = append(todos, &todo)
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
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/charm/... -v -run "Test.*Todo"
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/charm/todos.go internal/charm/todos_test.go
git commit -m "feat(charm): implement todo CRUD operations with filtering"
```

---

### Task 2.3: Implement Tag Operations

**Files:**
- Create: `internal/charm/tags.go`
- Create: `internal/charm/tags_test.go`

**Step 1: Write the tests**

Create `internal/charm/tags_test.go`:

```go
// ABOUTME: Tests for tag operations
// ABOUTME: Verifies create, list, and get for tags

package charm

import (
	"testing"
	"time"
)

func TestCreateTag(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	tag := &Tag{
		Name:      "urgent",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateTag(tag); err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	retrieved, err := client.GetTag("urgent")
	if err != nil {
		t.Fatalf("failed to get tag: %v", err)
	}

	if retrieved.Name != "urgent" {
		t.Errorf("Name mismatch")
	}
}

func TestListTags(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	tags := []string{"urgent", "backend", "frontend"}
	for _, name := range tags {
		tag := &Tag{Name: name, CreatedAt: time.Now().UTC()}
		if err := client.CreateTag(tag); err != nil {
			t.Fatalf("failed to create tag: %v", err)
		}
	}

	result, err := client.ListTags()
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 tags, got %d", len(result))
	}
}

func TestGetOrCreateTag(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	// First call creates
	tag1, err := client.GetOrCreateTag("new-tag")
	if err != nil {
		t.Fatalf("failed to get/create tag: %v", err)
	}

	// Second call returns existing
	tag2, err := client.GetOrCreateTag("new-tag")
	if err != nil {
		t.Fatalf("failed to get/create tag: %v", err)
	}

	if tag1.Name != tag2.Name {
		t.Errorf("tags should be the same")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/charm/... -v -run TestCreateTag
```

Expected: FAIL - CreateTag not defined

**Step 3: Implement tag operations**

Create `internal/charm/tags.go`:

```go
// ABOUTME: Tag operations for Charm KV
// ABOUTME: Implements create, list, and get for tags (used for autocomplete)

package charm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// CreateTag stores a new tag in the KV store.
func (c *Client) CreateTag(tag *Tag) error {
	data, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("failed to marshal tag: %w", err)
	}

	key := TagKey(tag.Name)
	if err := c.kv.Set([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store tag: %w", err)
	}

	return nil
}

// GetTag retrieves a tag by name.
func (c *Client) GetTag(name string) (*Tag, error) {
	key := TagKey(name)
	data, err := c.kv.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("tag not found: %w", err)
	}

	var tag Tag
	if err := json.Unmarshal(data, &tag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tag: %w", err)
	}

	return &tag, nil
}

// GetOrCreateTag retrieves a tag by name, creating it if it doesn't exist.
func (c *Client) GetOrCreateTag(name string) (*Tag, error) {
	tag, err := c.GetTag(name)
	if err == nil {
		return tag, nil
	}

	tag = &Tag{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	if err := c.CreateTag(tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// ListTags returns all tags, sorted by name.
func (c *Client) ListTags() ([]*Tag, error) {
	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var tags []*Tag
	for _, key := range keys {
		keyStr := string(key)
		if !strings.HasPrefix(keyStr, TagKeyPrefix) {
			continue
		}

		data, err := c.kv.Get(key)
		if err != nil {
			continue
		}

		var tag Tag
		if err := json.Unmarshal(data, &tag); err != nil {
			continue
		}

		tags = append(tags, &tag)
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	return tags, nil
}

// DeleteTag removes a tag by name.
func (c *Client) DeleteTag(name string) error {
	key := TagKey(name)
	if err := c.kv.Delete([]byte(key)); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/charm/... -v -run "Test.*Tag"
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/charm/tags.go internal/charm/tags_test.go
git commit -m "feat(charm): implement tag operations"
```

---

## Phase 3: Command Migration

### Task 3.1: Create Adapter Layer

Before migrating commands, create an adapter that provides the same interface as the old db package.

**Files:**
- Create: `internal/charm/adapter.go`

**Step 1: Create adapter that matches db package interface**

Create `internal/charm/adapter.go`:

```go
// ABOUTME: Adapter layer for migrating from db package to charm
// ABOUTME: Provides functions matching the old db package signatures for gradual migration

package charm

import (
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

// Global client for command usage
var globalClient *Client

// InitClient initializes the global Charm client.
func InitClient() error {
	client, err := NewClient("toki")
	if err != nil {
		return err
	}
	globalClient = client
	return nil
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
	return &models.Todo{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Description: t.Description,
		Done:        t.Done,
		Priority:    t.Priority,
		Notes:       t.Notes,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
		DueDate:     t.DueDate,
	}
}

// FromModelsTodo converts a models.Todo to charm.Todo.
func FromModelsTodo(t *models.Todo, projectName string, tags []string) *Todo {
	return &Todo{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		ProjectName: projectName,
		Description: t.Description,
		Done:        t.Done,
		Priority:    t.Priority,
		Notes:       t.Notes,
		Tags:        tags,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
		DueDate:     t.DueDate,
	}
}

// ToModelsProject converts a charm.Project to models.Project.
func ToModelsProject(p *Project) *models.Project {
	return &models.Project{
		ID:            p.ID,
		Name:          p.Name,
		DirectoryPath: p.DirectoryPath,
		CreatedAt:     p.CreatedAt,
	}
}

// FromModelsProject converts a models.Project to charm.Project.
func FromModelsProject(p *models.Project) *Project {
	return &Project{
		ID:            p.ID,
		Name:          p.Name,
		DirectoryPath: p.DirectoryPath,
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
```

**Step 2: Commit**

```bash
git add internal/charm/adapter.go
git commit -m "feat(charm): add adapter layer for command migration"
```

---

### Task 3.2: Migrate Root Command

**Files:**
- Modify: `cmd/toki/root.go`

This task updates the root command to initialize Charm instead of SQLite.

**Step 1: Read current root.go**

Read and understand the current initialization logic.

**Step 2: Update initialization**

Replace SQLite initialization with Charm client initialization. Remove db.InitDB calls, add charm.InitClient calls.

**Step 3: Run tests**

```bash
go test ./cmd/toki/... -v
```

**Step 4: Commit**

```bash
git add cmd/toki/root.go
git commit -m "refactor(cmd): migrate root command to Charm client"
```

---

### Task 3.3: Migrate Add Command

**Files:**
- Modify: `cmd/toki/add.go`

**Step 1: Read current add.go**

**Step 2: Replace db calls with charm calls**

- Replace `db.CreateTodo` with `charm.CreateTodo`
- Replace `db.GetOrCreateTag` with `charm.GetOrCreateTag`
- Remove sync queue calls (Charm syncs automatically)

**Step 3: Run tests**

```bash
go test ./cmd/toki/... -v -run TestAdd
```

**Step 4: Commit**

```bash
git add cmd/toki/add.go
git commit -m "refactor(cmd): migrate add command to Charm"
```

---

### Task 3.4-3.10: Migrate Remaining Commands

Repeat the pattern for each command file:
- `cmd/toki/list.go`
- `cmd/toki/done.go`
- `cmd/toki/remove.go`
- `cmd/toki/project.go`
- `cmd/toki/tag.go`
- `cmd/toki/update.go`

Each follows the same pattern:
1. Read current file
2. Replace db.* calls with charm.* calls
3. Remove sync queue calls
4. Run tests
5. Commit

---

## Phase 4: Sync Commands

### Task 4.1: Rewrite Sync Command

**Files:**
- Rewrite: `cmd/toki/sync.go`

**Step 1: Write new sync command**

```go
// ABOUTME: Sync subcommands for Charm backend
// ABOUTME: Provides status, now, link, unlink, and wipe operations

package main

// Subcommands:
// - toki sync status: Show Charm ID, server, sync state
// - toki sync now: Force immediate sync
// - toki sync link: Initiate device linking
// - toki sync unlink: Remove device from account
// - toki sync wipe: Clear all remote data
```

**Step 2: Remove old sync subcommands**

Remove: init, login, logout, pending (no longer applicable)

**Step 3: Run tests**

```bash
go test ./cmd/toki/... -v -run TestSync
```

**Step 4: Commit**

```bash
git add cmd/toki/sync.go
git commit -m "refactor(cmd): rewrite sync commands for Charm backend"
```

---

## Phase 5: Cleanup

### Task 5.1: Remove Old Packages

**Files:**
- Delete: `internal/db/` (entire directory)
- Delete: `internal/sync/` (entire directory)

**Step 1: Remove directories**

```bash
rm -rf internal/db internal/sync
```

**Step 2: Update go.mod**

```bash
go mod tidy
```

**Step 3: Verify build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: remove old db and sync packages"
```

---

### Task 5.2: Update MCP Tools

**Files:**
- Modify: `internal/mcp/tools.go`

**Step 1: Replace db calls with charm calls**

Follow the same pattern as command migration.

**Step 2: Run tests**

```bash
go test ./internal/mcp/... -v
```

**Step 3: Commit**

```bash
git add internal/mcp/
git commit -m "refactor(mcp): migrate tools to Charm backend"
```

---

### Task 5.3: Update Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Remove old dependencies**

```bash
go mod tidy
```

**Step 2: Verify no sqlite or vault references**

```bash
grep -E "sqlite|sweet" go.mod go.sum
```

Expected: No matches

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: remove old dependencies"
```

---

## Phase 6: Integration Testing

### Task 6.1: End-to-End Test

**Step 1: Build the binary**

```bash
go build -o toki ./cmd/toki
```

**Step 2: Test basic operations**

```bash
# Set up test environment
export CHARM_DATA_DIR=$(mktemp -d)
export CHARM_HOST=charm.2389.dev

# Test project creation
./toki project add test-project
./toki project list

# Test todo operations
./toki add "Test todo" --project test-project
./toki list
./toki done <id>
./toki list --done

# Test sync
./toki sync status
./toki sync now
```

**Step 3: Verify sync works**

Test on second machine or with different CHARM_DATA_DIR.

**Step 4: Clean up test environment**

```bash
rm -rf $CHARM_DATA_DIR
```

---

## Summary

This plan migrates Toki from SQLite + vault sync to Charm KV in these phases:

1. **Foundation** - Add dependency, create models, create client wrapper
2. **KV Operations** - Implement CRUD for projects, todos, tags
3. **Command Migration** - Update each command to use Charm
4. **Sync Commands** - Rewrite sync subcommands for Charm
5. **Cleanup** - Remove old code and dependencies
6. **Integration Testing** - Verify end-to-end functionality

Total estimated tasks: ~20 discrete commits
