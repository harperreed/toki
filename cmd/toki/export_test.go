// ABOUTME: Tests for export command functionality
// ABOUTME: Covers data building, YAML/JSON/Markdown export formats

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/storage"
	"gopkg.in/yaml.v3"
)

func setupExportTestStorage(t *testing.T) (*storage.SQLiteStorage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "toki-export-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("failed to create storage: %v", err)
	}

	cleanup := func() {
		_ = store.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func createExportTestProject(t *testing.T, store *storage.SQLiteStorage, name, path string) *storage.Project {
	t.Helper()
	project := &storage.Project{
		ID:            uuid.New(),
		Name:          name,
		DirectoryPath: path,
		CreatedAt:     time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	return project
}

func createExportTestTodo(t *testing.T, store *storage.SQLiteStorage, projectID uuid.UUID, description string) *storage.Todo {
	t.Helper()
	now := time.Now().UTC()
	todo := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   projectID,
		Description: description,
		Done:        false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}
	return todo
}

//nolint:gocognit,funlen
func TestBuildExportData(t *testing.T) {
	store, cleanup := setupExportTestStorage(t)
	defer cleanup()

	t.Run("empty database", func(t *testing.T) {
		data, err := buildExportData(store)
		if err != nil {
			t.Fatalf("buildExportData failed: %v", err)
		}

		if data.Version != "1.0" {
			t.Errorf("expected version '1.0', got '%s'", data.Version)
		}
		if data.Tool != "toki" {
			t.Errorf("expected tool 'toki', got '%s'", data.Tool)
		}
		if data.ExportedAt == "" {
			t.Error("expected ExportedAt to be set")
		}
		if len(data.Projects) != 0 {
			t.Errorf("expected 0 projects, got %d", len(data.Projects))
		}
	})

	t.Run("with project and todos", func(t *testing.T) {
		// Create a project
		project := createExportTestProject(t, store, "test-project", "")

		// Create a todo
		todo := createExportTestTodo(t, store, project.ID, "test todo")

		// Create completed todo with all fields
		completedTodo := createExportTestTodo(t, store, project.ID, "completed todo")
		dueDate := time.Now().Add(24 * time.Hour)
		completedTodo.Priority = "high"
		completedTodo.Notes = "some notes"
		completedTodo.DueDate = &dueDate
		if err := store.UpdateTodo(completedTodo); err != nil {
			t.Fatalf("failed to update todo: %v", err)
		}
		if err := store.MarkTodoDone(completedTodo.ID, true); err != nil {
			t.Fatalf("failed to mark todo done: %v", err)
		}

		data, err := buildExportData(store)
		if err != nil {
			t.Fatalf("buildExportData failed: %v", err)
		}

		if len(data.Projects) != 1 {
			t.Fatalf("expected 1 project, got %d", len(data.Projects))
		}

		exportProj := data.Projects[0]
		if exportProj.Name != "test-project" {
			t.Errorf("expected project name 'test-project', got '%s'", exportProj.Name)
		}
		if exportProj.ID != project.ID.String() {
			t.Errorf("project ID mismatch")
		}
		if len(exportProj.Todos) != 2 {
			t.Fatalf("expected 2 todos, got %d", len(exportProj.Todos))
		}

		// Find the pending todo
		var pendingTodo, doneTodo ExportTodo
		for _, et := range exportProj.Todos {
			if et.ID == todo.ID.String() {
				pendingTodo = et
			} else {
				doneTodo = et
			}
		}

		if pendingTodo.Description != "test todo" {
			t.Errorf("expected description 'test todo', got '%s'", pendingTodo.Description)
		}
		if pendingTodo.Done {
			t.Error("expected pending todo to not be done")
		}

		if !doneTodo.Done {
			t.Error("expected completed todo to be done")
		}
		if doneTodo.Priority != "high" {
			t.Errorf("expected priority 'high', got '%s'", doneTodo.Priority)
		}
		if doneTodo.Notes != "some notes" {
			t.Errorf("expected notes 'some notes', got '%s'", doneTodo.Notes)
		}
		if doneTodo.DueDate == "" {
			t.Error("expected due date to be set")
		}
		if doneTodo.CompletedAt == "" {
			t.Error("expected completed_at to be set")
		}
	})

	t.Run("project with path", func(t *testing.T) {
		projectPath := "/path/to/project"
		project := createExportTestProject(t, store, "path-project", projectPath)

		data, err := buildExportData(store)
		if err != nil {
			t.Fatalf("buildExportData failed: %v", err)
		}

		// Find the path project
		var found bool
		for _, p := range data.Projects {
			if p.ID == project.ID.String() {
				found = true
				if p.Path != projectPath {
					t.Errorf("expected path '%s', got '%s'", projectPath, p.Path)
				}
				break
			}
		}
		if !found {
			t.Error("expected to find path-project in export")
		}
	})
}

func TestWriteYAML(t *testing.T) {
	data := &ExportData{
		Version:    "1.0",
		ExportedAt: "2024-01-15T10:00:00Z",
		Tool:       "toki",
		Projects: []ExportProject{
			{
				ID:   "proj-123",
				Name: "test-project",
				Todos: []ExportTodo{
					{
						ID:          "todo-456",
						Description: "test todo",
						Done:        false,
						CreatedAt:   "2024-01-15T10:00:00Z",
						UpdatedAt:   "2024-01-15T10:00:00Z",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := writeYAML(&buf, data)
	if err != nil {
		t.Fatalf("writeYAML failed: %v", err)
	}

	output := buf.String()

	// Verify it's valid YAML
	var parsed ExportData
	err = yaml.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	if parsed.Version != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", parsed.Version)
	}
	if parsed.Tool != "toki" {
		t.Errorf("expected tool 'toki', got '%s'", parsed.Tool)
	}
	if len(parsed.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(parsed.Projects))
	}
	if parsed.Projects[0].Name != "test-project" {
		t.Errorf("expected project name 'test-project', got '%s'", parsed.Projects[0].Name)
	}
}

//nolint:funlen
func TestWriteJSON(t *testing.T) {
	data := &ExportData{
		Version:    "1.0",
		ExportedAt: "2024-01-15T10:00:00Z",
		Tool:       "toki",
		Projects: []ExportProject{
			{
				ID:   "proj-123",
				Name: "test-project",
				Todos: []ExportTodo{
					{
						ID:          "todo-456",
						Description: "test todo",
						Done:        false,
						CreatedAt:   "2024-01-15T10:00:00Z",
						UpdatedAt:   "2024-01-15T10:00:00Z",
					},
				},
			},
		},
	}

	t.Run("compact JSON", func(t *testing.T) {
		var buf bytes.Buffer
		err := writeJSON(&buf, data, false)
		if err != nil {
			t.Fatalf("writeJSON failed: %v", err)
		}

		output := buf.String()

		// Verify it's valid JSON
		var parsed ExportData
		err = json.Unmarshal([]byte(output), &parsed)
		if err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}

		if parsed.Version != "1.0" {
			t.Errorf("expected version '1.0', got '%s'", parsed.Version)
		}

		// Compact JSON should not have indentation
		if strings.Contains(output, "\n  ") {
			t.Error("compact JSON should not have indentation")
		}
	})

	t.Run("pretty JSON", func(t *testing.T) {
		var buf bytes.Buffer
		err := writeJSON(&buf, data, true)
		if err != nil {
			t.Fatalf("writeJSON failed: %v", err)
		}

		output := buf.String()

		// Verify it's valid JSON
		var parsed ExportData
		err = json.Unmarshal([]byte(output), &parsed)
		if err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}

		// Pretty JSON should have indentation
		if !strings.Contains(output, "\n  ") {
			t.Error("pretty JSON should have indentation")
		}
	})
}

//nolint:funlen
func TestWriteMarkdown(t *testing.T) {
	data := &ExportData{
		Version:    "1.0",
		ExportedAt: "2024-01-15T10:00:00Z",
		Tool:       "toki",
		Projects: []ExportProject{
			{
				ID:   "proj-123",
				Name: "test-project",
				Path: "/path/to/project",
				Todos: []ExportTodo{
					{
						ID:          "todo-pending",
						Description: "pending task",
						Done:        false,
						Priority:    "high",
						Tags:        []string{"backend", "urgent"},
						DueDate:     "2024-01-20T10:00:00Z",
						CreatedAt:   "2024-01-15T10:00:00Z",
						UpdatedAt:   "2024-01-15T10:00:00Z",
					},
					{
						ID:          "todo-done",
						Description: "completed task",
						Done:        true,
						Notes:       "some notes here",
						CompletedAt: "2024-01-16T10:00:00Z",
						CreatedAt:   "2024-01-15T10:00:00Z",
						UpdatedAt:   "2024-01-16T10:00:00Z",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	writeMarkdown(&buf, data)
	output := buf.String()

	// Check header
	if !strings.Contains(output, "# Toki Export") {
		t.Error("expected markdown header")
	}
	if !strings.Contains(output, "Generated:") {
		t.Error("expected generated timestamp")
	}

	// Check project
	if !strings.Contains(output, "## Project: test-project") {
		t.Error("expected project header")
	}
	if !strings.Contains(output, "_Path: /path/to/project_") {
		t.Error("expected project path")
	}

	// Check pending section
	if !strings.Contains(output, "### Pending Todos") {
		t.Error("expected pending todos section")
	}
	if !strings.Contains(output, "[ ] **[HIGH]** pending task") {
		t.Error("expected pending todo with priority")
	}
	if !strings.Contains(output, "Due: 2024-01-20") {
		t.Error("expected due date")
	}
	if !strings.Contains(output, "Tags: backend, urgent") {
		t.Error("expected tags")
	}

	// Check completed section
	if !strings.Contains(output, "### Completed Todos") {
		t.Error("expected completed todos section")
	}
	if !strings.Contains(output, "[x] ~~completed task~~") {
		t.Error("expected completed todo with strikethrough")
	}
	if !strings.Contains(output, "Completed: 2024-01-16") {
		t.Error("expected completed date")
	}
	if !strings.Contains(output, "Notes: some notes here") {
		t.Error("expected notes")
	}
}

func TestWriteMarkdown_EmptyProject(t *testing.T) {
	data := &ExportData{
		Version:    "1.0",
		ExportedAt: "2024-01-15T10:00:00Z",
		Tool:       "toki",
		Projects: []ExportProject{
			{
				ID:    "proj-123",
				Name:  "empty-project",
				Todos: []ExportTodo{},
			},
		},
	}

	var buf bytes.Buffer
	writeMarkdown(&buf, data)
	output := buf.String()

	if !strings.Contains(output, "_No todos in this project_") {
		t.Error("expected 'no todos' message for empty project")
	}
}

//nolint:funlen
func TestWriteTodoMarkdown(t *testing.T) {
	t.Run("pending todo", func(t *testing.T) {
		todo := ExportTodo{
			ID:          "todo-123",
			Description: "test task",
			Done:        false,
			CreatedAt:   "2024-01-15T10:00:00Z",
			UpdatedAt:   "2024-01-15T10:00:00Z",
		}

		var buf bytes.Buffer
		writeTodoMarkdown(&buf, todo, false)
		output := buf.String()

		if !strings.Contains(output, "[ ]") {
			t.Error("pending todo should have unchecked box")
		}
		if !strings.Contains(output, "test task") {
			t.Error("expected task description")
		}
		if strings.Contains(output, "~~") {
			t.Error("pending todo should not have strikethrough")
		}
	})

	t.Run("completed todo", func(t *testing.T) {
		todo := ExportTodo{
			ID:          "todo-123",
			Description: "done task",
			Done:        true,
			CreatedAt:   "2024-01-15T10:00:00Z",
			UpdatedAt:   "2024-01-15T10:00:00Z",
		}

		var buf bytes.Buffer
		writeTodoMarkdown(&buf, todo, true)
		output := buf.String()

		if !strings.Contains(output, "[x]") {
			t.Error("completed todo should have checked box")
		}
		if !strings.Contains(output, "~~done task~~") {
			t.Error("completed todo should have strikethrough")
		}
	})

	t.Run("with priority", func(t *testing.T) {
		todo := ExportTodo{
			ID:          "todo-123",
			Description: "priority task",
			Done:        false,
			Priority:    "medium",
			CreatedAt:   "2024-01-15T10:00:00Z",
			UpdatedAt:   "2024-01-15T10:00:00Z",
		}

		var buf bytes.Buffer
		writeTodoMarkdown(&buf, todo, false)
		output := buf.String()

		if !strings.Contains(output, "**[MEDIUM]**") {
			t.Error("expected priority badge")
		}
	})

	t.Run("with all metadata", func(t *testing.T) {
		todo := ExportTodo{
			ID:          "todo-123",
			Description: "full task",
			Done:        true,
			Priority:    "low",
			Notes:       "detailed notes",
			Tags:        []string{"tag1", "tag2"},
			DueDate:     "2024-01-20T10:00:00Z",
			CompletedAt: "2024-01-18T10:00:00Z",
			CreatedAt:   "2024-01-15T10:00:00Z",
			UpdatedAt:   "2024-01-18T10:00:00Z",
		}

		var buf bytes.Buffer
		writeTodoMarkdown(&buf, todo, true)
		output := buf.String()

		if !strings.Contains(output, "Due: 2024-01-20") {
			t.Error("expected due date")
		}
		if !strings.Contains(output, "Tags: tag1, tag2") {
			t.Error("expected tags")
		}
		if !strings.Contains(output, "Completed: 2024-01-18") {
			t.Error("expected completed date")
		}
		if !strings.Contains(output, "Notes: detailed notes") {
			t.Error("expected notes")
		}
	})
}

func TestExportCommandExists(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"export"})
	if err != nil {
		t.Fatalf("export command not found: %v", err)
	}

	if cmd.Name() != "export" {
		t.Errorf("expected command name 'export', got '%s'", cmd.Name())
	}
}

func TestExportSubcommands(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"export"})
	if err != nil {
		t.Fatalf("export command not found: %v", err)
	}

	expectedSubcommands := []string{
		"yaml",
		"json",
		"markdown",
		"sqlite",
	}

	registeredCommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		registeredCommands[subcmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !registeredCommands[expected] {
			t.Errorf("expected subcommand 'export %s' to be registered", expected)
		}
	}
}

func TestMarkdownAlias(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"export", "markdown"})
	if err != nil {
		t.Fatalf("export markdown command not found: %v", err)
	}

	hasAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "md" {
			hasAlias = true
			break
		}
	}

	if !hasAlias {
		t.Error("expected markdown command to have 'md' alias")
	}
}

func TestJSONCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"export", "json"})
	if err != nil {
		t.Fatalf("export json command not found: %v", err)
	}

	prettyFlag := cmd.Flags().Lookup("pretty")
	if prettyFlag == nil {
		t.Error("expected --pretty flag on export json command")
	}
}

func TestSQLiteCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"export", "sqlite"})
	if err != nil {
		t.Fatalf("export sqlite command not found: %v", err)
	}

	outputFlag := cmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("expected --output flag on export sqlite command")
	}
}

//nolint:funlen
func TestExportSQLite(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME to ensure we have a known database location
	originalXDG := os.Getenv("XDG_DATA_HOME")
	t.Cleanup(func() {
		if originalXDG == "" {
			_ = os.Unsetenv("XDG_DATA_HOME")
		} else {
			_ = os.Setenv("XDG_DATA_HOME", originalXDG)
		}
	})
	_ = os.Setenv("XDG_DATA_HOME", tmpDir)

	// Create a source database
	store, cleanup := setupExportTestStorage(t)
	defer cleanup()

	// Create some test data
	project := createExportTestProject(t, store, "export-test", "")
	createExportTestTodo(t, store, project.ID, "test todo 1")
	createExportTestTodo(t, store, project.ID, "test todo 2")

	// Close the store so we can copy the database
	_ = store.Close()

	// Create source DB in the expected location
	sourceDir := filepath.Join(tmpDir, "toki")
	if err := os.MkdirAll(sourceDir, 0750); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Re-create the database at the default path
	sourceDB := filepath.Join(sourceDir, "toki.db")
	sourceStore, err := storage.NewSQLiteStorage(sourceDB)
	if err != nil {
		t.Fatalf("failed to create source storage: %v", err)
	}

	// Create test data in the source database
	proj := createExportTestProject(t, sourceStore, "sqlite-test", "/path/to/project")
	createExportTestTodo(t, sourceStore, proj.ID, "sqlite test todo")
	_ = sourceStore.Close()

	t.Run("successful export", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "export-test.db")
		err := exportSQLite(outputPath)
		if err != nil {
			t.Fatalf("exportSQLite failed: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("expected output file to exist")
		}

		// Verify it's a valid SQLite database by opening it
		exportedStore, err := storage.NewSQLiteStorage(outputPath)
		if err != nil {
			t.Fatalf("failed to open exported database: %v", err)
		}
		defer func() { _ = exportedStore.Close() }()

		// Verify data was copied
		projects, err := exportedStore.ListProjects()
		if err != nil {
			t.Fatalf("failed to list projects: %v", err)
		}
		if len(projects) == 0 {
			t.Error("expected at least one project in exported database")
		}
	})

	t.Run("non-existent source database", func(t *testing.T) {
		// Point to a non-existent directory
		_ = os.Setenv("XDG_DATA_HOME", "/nonexistent/path/that/does/not/exist")
		defer func() { _ = os.Setenv("XDG_DATA_HOME", tmpDir) }()

		outputPath := filepath.Join(tmpDir, "should-not-exist.db")
		err := exportSQLite(outputPath)
		if err == nil {
			t.Error("expected error for non-existent source database")
		}
	})
}

//nolint:funlen
func TestExportToWriterFormats(t *testing.T) {
	// This test verifies the format dispatch in exportToWriter
	// by testing with a real storage

	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME to ensure we have a known database location
	originalXDG := os.Getenv("XDG_DATA_HOME")
	t.Cleanup(func() {
		if originalXDG == "" {
			_ = os.Unsetenv("XDG_DATA_HOME")
		} else {
			_ = os.Setenv("XDG_DATA_HOME", originalXDG)
		}
	})
	_ = os.Setenv("XDG_DATA_HOME", tmpDir)

	// Create source DB in the expected location
	sourceDir := filepath.Join(tmpDir, "toki")
	if err := os.MkdirAll(sourceDir, 0750); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	sourceDB := filepath.Join(sourceDir, "toki.db")
	sourceStore, err := storage.NewSQLiteStorage(sourceDB)
	if err != nil {
		t.Fatalf("failed to create source storage: %v", err)
	}

	// Create test data
	proj := &storage.Project{
		ID:        uuid.New(),
		Name:      "format-test",
		CreatedAt: time.Now().UTC(),
	}
	if err := sourceStore.CreateProject(proj); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	_ = sourceStore.Close()

	t.Run("yaml format", func(t *testing.T) {
		var buf bytes.Buffer
		err := exportToWriter(&buf, "yaml", false)
		if err != nil {
			t.Fatalf("exportToWriter yaml failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "version:") {
			t.Error("expected YAML output to contain 'version:'")
		}
		if !strings.Contains(output, "format-test") {
			t.Error("expected YAML output to contain project name")
		}
	})

	t.Run("json format", func(t *testing.T) {
		var buf bytes.Buffer
		err := exportToWriter(&buf, "json", false)
		if err != nil {
			t.Fatalf("exportToWriter json failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, `"version"`) {
			t.Error("expected JSON output to contain '\"version\"'")
		}
	})

	t.Run("json pretty format", func(t *testing.T) {
		var buf bytes.Buffer
		err := exportToWriter(&buf, "json", true)
		if err != nil {
			t.Fatalf("exportToWriter json pretty failed: %v", err)
		}

		output := buf.String()
		// Pretty printed JSON should have newlines and indentation
		if !strings.Contains(output, "\n  ") {
			t.Error("expected pretty JSON to have indentation")
		}
	})

	t.Run("markdown format", func(t *testing.T) {
		var buf bytes.Buffer
		err := exportToWriter(&buf, "markdown", false)
		if err != nil {
			t.Fatalf("exportToWriter markdown failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "# Toki Export") {
			t.Error("expected markdown header")
		}
	})

	t.Run("unknown format", func(t *testing.T) {
		var buf bytes.Buffer
		err := exportToWriter(&buf, "unknown-format", false)
		if err == nil {
			t.Error("expected error for unknown format")
		}
		if !strings.Contains(err.Error(), "unknown format") {
			t.Errorf("expected 'unknown format' error, got: %v", err)
		}
	})
}

func TestGetStorageForExport(t *testing.T) {
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	t.Cleanup(func() {
		if originalXDG == "" {
			_ = os.Unsetenv("XDG_DATA_HOME")
		} else {
			_ = os.Setenv("XDG_DATA_HOME", originalXDG)
		}
	})
	_ = os.Setenv("XDG_DATA_HOME", tmpDir)

	store, err := getStorageForExport()
	if err != nil {
		t.Fatalf("getStorageForExport failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Verify it returns a valid storage
	if store == nil {
		t.Error("expected non-nil storage")
	}

	// Verify the database was created in the expected location
	expectedPath := filepath.Join(tmpDir, "toki", "toki.db")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected database at %s", expectedPath)
	}
}
