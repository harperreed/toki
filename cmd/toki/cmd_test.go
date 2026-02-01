// ABOUTME: Tests for CLI command helpers and utilities
// ABOUTME: Covers storage functions, list helpers, and root command

package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/storage"
)

// resetStorageOnce resets the storageOnce sync.Once for testing purposes.
// This allows tests to re-initialize storage without restarting the process.
func resetStorageOnce() {
	storageOnce = sync.Once{}
}

// setupTestStorage initializes a test storage database.
func setupTestStorage(t *testing.T) (*storage.SQLiteStorage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "toki-cmd-test-*")
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

//nolint:funlen
func TestStorageToModelsTodo(t *testing.T) {
	now := time.Now().UTC()
	dueDate := now.Add(24 * time.Hour)
	completedAt := now

	tests := []struct {
		name         string
		storageTodo  *storage.Todo
		wantPriority bool
		wantNotes    bool
	}{
		{
			name: "minimal todo",
			storageTodo: &storage.Todo{
				ID:          uuid.New(),
				ProjectID:   uuid.New(),
				Description: "Test todo",
				Done:        false,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantPriority: false,
			wantNotes:    false,
		},
		{
			name: "todo with priority",
			storageTodo: &storage.Todo{
				ID:          uuid.New(),
				ProjectID:   uuid.New(),
				Description: "High priority task",
				Done:        false,
				Priority:    "high",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantPriority: true,
			wantNotes:    false,
		},
		{
			name: "todo with notes",
			storageTodo: &storage.Todo{
				ID:          uuid.New(),
				ProjectID:   uuid.New(),
				Description: "Task with notes",
				Done:        false,
				Notes:       "Some notes",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantPriority: false,
			wantNotes:    true,
		},
		{
			name: "completed todo with all fields",
			storageTodo: &storage.Todo{
				ID:          uuid.New(),
				ProjectID:   uuid.New(),
				Description: "Full todo",
				Done:        true,
				Priority:    "medium",
				Notes:       "Complete notes",
				DueDate:     &dueDate,
				CompletedAt: &completedAt,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantPriority: true,
			wantNotes:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelsTodo := storageToModelsTodo(tt.storageTodo)

			if modelsTodo.ID != tt.storageTodo.ID {
				t.Errorf("ID mismatch: got %v, want %v", modelsTodo.ID, tt.storageTodo.ID)
			}
			if modelsTodo.ProjectID != tt.storageTodo.ProjectID {
				t.Errorf("ProjectID mismatch")
			}
			if modelsTodo.Description != tt.storageTodo.Description {
				t.Errorf("Description mismatch")
			}
			if modelsTodo.Done != tt.storageTodo.Done {
				t.Errorf("Done mismatch")
			}
			if tt.wantPriority && modelsTodo.Priority == nil {
				t.Error("expected Priority to be set")
			}
			if !tt.wantPriority && modelsTodo.Priority != nil {
				t.Error("expected Priority to be nil")
			}
			if tt.wantNotes && modelsTodo.Notes == nil {
				t.Error("expected Notes to be set")
			}
			if !tt.wantNotes && modelsTodo.Notes != nil {
				t.Error("expected Notes to be nil")
			}
		})
	}
}

func TestStorageToModelsProject(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name           string
		storageProject *storage.Project
		wantPath       bool
	}{
		{
			name: "project without path",
			storageProject: &storage.Project{
				ID:        uuid.New(),
				Name:      "test-project",
				CreatedAt: now,
			},
			wantPath: false,
		},
		{
			name: "project with path",
			storageProject: &storage.Project{
				ID:            uuid.New(),
				Name:          "project-with-path",
				DirectoryPath: "/path/to/project",
				CreatedAt:     now,
			},
			wantPath: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelsProject := storageToModelsProject(tt.storageProject)

			if modelsProject.ID != tt.storageProject.ID {
				t.Errorf("ID mismatch")
			}
			if modelsProject.Name != tt.storageProject.Name {
				t.Errorf("Name mismatch")
			}
			if tt.wantPath && modelsProject.DirectoryPath == nil {
				t.Error("expected DirectoryPath to be set")
			}
			if !tt.wantPath && modelsProject.DirectoryPath != nil {
				t.Error("expected DirectoryPath to be nil")
			}
		})
	}
}

func TestRootCommandExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	if rootCmd.Use != "toki" {
		t.Errorf("expected Use 'toki', got '%s'", rootCmd.Use)
	}
}

func TestSubcommandsRegistered(t *testing.T) {
	expectedCommands := []string{
		"add",
		"list",
		"done",
		"undone",
		"remove",
		"project",
		"mcp",
		"version",
		"export",
		"install-skill",
	}

	registeredCommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		registeredCommands[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !registeredCommands[expected] {
			t.Errorf("expected command '%s' to be registered", expected)
		}
	}
}

func TestCloseStorageWithNilStorage(t *testing.T) {
	// Save and reset the global storage
	originalStorage := globalStorage

	// Reset for test
	globalStorage = nil
	resetStorageOnce()

	defer func() {
		globalStorage = originalStorage
		resetStorageOnce()
	}()

	// CloseStorage with nil storage should not error
	err := CloseStorage()
	if err != nil {
		t.Errorf("CloseStorage with nil storage should not error: %v", err)
	}
}

func TestGetStorageReturnsGlobal(t *testing.T) {
	// Save and reset the global storage
	originalStorage := globalStorage

	defer func() {
		globalStorage = originalStorage
		resetStorageOnce()
	}()

	// Create a test storage
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Set as global storage
	globalStorage = store

	// GetStorage should return the global storage
	got := GetStorage()
	if got != store {
		t.Error("GetStorage should return the global storage")
	}
}

func TestAddCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"add"})
	if err != nil {
		t.Fatalf("add command not found: %v", err)
	}

	expectedFlags := []string{
		"project",
		"priority",
		"tags",
		"notes",
		"due",
	}

	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("expected flag '--%s' on add command", flag)
		}
	}
}

func TestListCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("list command not found: %v", err)
	}

	expectedFlags := []string{
		"project",
		"tag",
		"done",
		"pending",
		"priority",
	}

	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("expected flag '--%s' on list command", flag)
		}
	}
}

func TestProjectSubcommands(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"project"})
	if err != nil {
		t.Fatalf("project command not found: %v", err)
	}

	expectedSubcommands := []string{
		"add",
		"list",
		"set-path",
		"remove",
		"cleanup",
	}

	registeredCommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		registeredCommands[subcmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !registeredCommands[expected] {
			t.Errorf("expected subcommand 'project %s' to be registered", expected)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("version command not found: %v", err)
	}

	if cmd.Name() != "version" {
		t.Errorf("expected command name 'version', got '%s'", cmd.Name())
	}

	if cmd.Short == "" {
		t.Error("version command should have a short description")
	}
}

func TestCommandAliases(t *testing.T) {
	tests := []struct {
		command       string
		expectedAlias string
	}{
		{"add", "a"},
		{"list", "ls"},
		{"done", "d"},
		{"undone", "ud"},
		{"remove", "rm"},
		{"project", "p"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.command})
			if err != nil {
				t.Fatalf("%s command not found: %v", tt.command, err)
			}

			hasAlias := false
			for _, alias := range cmd.Aliases {
				if alias == tt.expectedAlias {
					hasAlias = true
					break
				}
			}

			if !hasAlias {
				t.Errorf("expected command '%s' to have alias '%s'", tt.command, tt.expectedAlias)
			}
		})
	}
}

func TestSyncCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"sync"})
	if err != nil {
		t.Fatalf("sync command not found: %v", err)
	}

	if cmd.Name() != "sync" {
		t.Errorf("expected command name 'sync', got '%s'", cmd.Name())
	}
}

func TestSyncSubcommands(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"sync"})
	if err != nil {
		t.Fatalf("sync command not found: %v", err)
	}

	expectedSubcommands := []string{
		"status",
		"repair",
		"reset",
	}

	registeredCommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		registeredCommands[subcmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !registeredCommands[expected] {
			t.Errorf("expected subcommand 'sync %s' to be registered", expected)
		}
	}
}

func TestTagCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"tag"})
	if err != nil {
		t.Fatalf("tag command not found: %v", err)
	}

	if cmd.Name() != "tag" {
		t.Errorf("expected command name 'tag', got '%s'", cmd.Name())
	}
}

func TestTagSubcommands(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"tag"})
	if err != nil {
		t.Fatalf("tag command not found: %v", err)
	}

	expectedSubcommands := []string{
		"add",
		"remove",
		"list",
	}

	registeredCommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		registeredCommands[subcmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !registeredCommands[expected] {
			t.Errorf("expected subcommand 'tag %s' to be registered", expected)
		}
	}
}

func TestTagSubcommandAliases(t *testing.T) {
	t.Run("tag add", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"tag", "add"})
		if err != nil {
			t.Fatalf("tag add command not found: %v", err)
		}

		hasAlias := false
		for _, alias := range cmd.Aliases {
			if alias == "a" {
				hasAlias = true
				break
			}
		}
		if !hasAlias {
			t.Error("expected tag add to have alias 'a'")
		}
	})

	t.Run("tag remove", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"tag", "remove"})
		if err != nil {
			t.Fatalf("tag remove command not found: %v", err)
		}

		hasRmAlias := false
		for _, alias := range cmd.Aliases {
			if alias == "rm" || alias == "r" {
				hasRmAlias = true
				break
			}
		}
		if !hasRmAlias {
			t.Error("expected tag remove to have alias 'rm' or 'r'")
		}
	})
}

func TestMcpCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"mcp"})
	if err != nil {
		t.Fatalf("mcp command not found: %v", err)
	}

	if cmd.Name() != "mcp" {
		t.Errorf("expected command name 'mcp', got '%s'", cmd.Name())
	}
}

func TestDoneCommandArgs(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"done"})
	if err != nil {
		t.Fatalf("done command not found: %v", err)
	}

	// Done requires at least 1 argument
	if cmd.Args == nil {
		t.Error("expected done command to have Args set")
	}
}

func TestUndoneCommandArgs(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"undone"})
	if err != nil {
		t.Fatalf("undone command not found: %v", err)
	}

	// Undone requires at least 1 argument
	if cmd.Args == nil {
		t.Error("expected undone command to have Args set")
	}
}

func TestRemoveCommandArgs(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"remove"})
	if err != nil {
		t.Fatalf("remove command not found: %v", err)
	}

	// Remove requires exactly 1 argument
	if cmd.Args == nil {
		t.Error("expected remove command to have Args set")
	}
}

func TestProjectAddHasPathFlag(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"project", "add"})
	if err != nil {
		t.Fatalf("project add command not found: %v", err)
	}

	pathFlag := cmd.Flags().Lookup("path")
	if pathFlag == nil {
		t.Error("expected --path flag on project add command")
	}
}

func TestListCommandHasLAlias(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("list command not found: %v", err)
	}

	hasLAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "l" {
			hasLAlias = true
			break
		}
	}

	if !hasLAlias {
		t.Error("expected list command to have 'l' alias")
	}
}

func TestProjectListHasAliases(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"project", "list"})
	if err != nil {
		t.Fatalf("project list command not found: %v", err)
	}

	hasAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "ls" || alias == "l" {
			hasAlias = true
			break
		}
	}

	if !hasAlias {
		t.Error("expected project list to have 'ls' or 'l' alias")
	}
}

func TestProjectRemoveHasAliases(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"project", "remove"})
	if err != nil {
		t.Fatalf("project remove command not found: %v", err)
	}

	hasAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "rm" || alias == "r" {
			hasAlias = true
			break
		}
	}

	if !hasAlias {
		t.Error("expected project remove to have 'rm' or 'r' alias")
	}
}

func TestProjectSetPathHasAlias(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"project", "set-path"})
	if err != nil {
		t.Fatalf("project set-path command not found: %v", err)
	}

	hasAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "sp" {
			hasAlias = true
			break
		}
	}

	if !hasAlias {
		t.Error("expected project set-path to have 'sp' alias")
	}
}

func TestStorageToModelsTodoTimestamps(t *testing.T) {
	now := time.Now().UTC()
	dueDate := now.Add(24 * time.Hour)
	completedAt := now.Add(-1 * time.Hour)

	storageTodo := &storage.Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		Description: "Test timestamps",
		Done:        true,
		DueDate:     &dueDate,
		CompletedAt: &completedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	modelsTodo := storageToModelsTodo(storageTodo)

	if modelsTodo.CreatedAt != storageTodo.CreatedAt {
		t.Error("CreatedAt mismatch")
	}
	if modelsTodo.UpdatedAt != storageTodo.UpdatedAt {
		t.Error("UpdatedAt mismatch")
	}
	if modelsTodo.DueDate == nil || *modelsTodo.DueDate != *storageTodo.DueDate {
		t.Error("DueDate mismatch")
	}
	if modelsTodo.CompletedAt == nil || *modelsTodo.CompletedAt != *storageTodo.CompletedAt {
		t.Error("CompletedAt mismatch")
	}
}

func TestStorageToModelsProjectCreatedAt(t *testing.T) {
	now := time.Now().UTC()

	storageProject := &storage.Project{
		ID:        uuid.New(),
		Name:      "test",
		CreatedAt: now,
	}

	modelsProject := storageToModelsProject(storageProject)

	if modelsProject.CreatedAt != storageProject.CreatedAt {
		t.Error("CreatedAt mismatch")
	}
}

func TestGetOrCreateDefaultProject(t *testing.T) {
	// Save and reset the global storage
	originalStorage := globalStorage

	defer func() {
		globalStorage = originalStorage
		resetStorageOnce()
	}()

	// Create a test storage
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Set as global storage
	globalStorage = store
	resetStorageOnce()

	t.Run("creates default project when none exists", func(t *testing.T) {
		project, err := getOrCreateDefaultProject()
		if err != nil {
			t.Fatalf("getOrCreateDefaultProject failed: %v", err)
		}

		if project.Name != "default" {
			t.Errorf("expected project name 'default', got '%s'", project.Name)
		}
	})

	t.Run("returns existing default project", func(t *testing.T) {
		// Call again - should return the same project
		project, err := getOrCreateDefaultProject()
		if err != nil {
			t.Fatalf("getOrCreateDefaultProject failed: %v", err)
		}

		if project.Name != "default" {
			t.Errorf("expected project name 'default', got '%s'", project.Name)
		}
	})
}

func TestGetProjectIDWithProjectFlag(t *testing.T) {
	// Save and reset the global storage
	originalStorage := globalStorage

	defer func() {
		globalStorage = originalStorage
		resetStorageOnce()
	}()

	// Create a test storage
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	// Set as global storage
	globalStorage = store
	resetStorageOnce()

	// Create a project
	now := time.Now().UTC()
	project := &storage.Project{
		ID:        uuid.New(),
		Name:      "my-project",
		CreatedAt: now,
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	t.Run("returns project ID by name", func(t *testing.T) {
		id, err := getProjectID("my-project")
		if err != nil {
			t.Fatalf("getProjectID failed: %v", err)
		}

		if *id != project.ID {
			t.Errorf("expected project ID %v, got %v", project.ID, *id)
		}
	})

	t.Run("error for non-existent project", func(t *testing.T) {
		_, err := getProjectID("non-existent-project")
		if err == nil {
			t.Error("expected error for non-existent project")
		}
	})
}

func TestInitStorageIdempotent(t *testing.T) {
	// Save and reset the global storage
	originalStorage := globalStorage

	defer func() {
		globalStorage = originalStorage
		resetStorageOnce()
	}()

	// Reset for test
	globalStorage = nil
	resetStorageOnce()

	// Create a test storage first time
	store, cleanup := setupTestStorage(t)
	defer cleanup()
	globalStorage = store

	// GetStorage should return the storage
	got := GetStorage()
	if got != store {
		t.Error("GetStorage should return the set storage")
	}
}

func TestCloseStorageWithInitializedStorage(t *testing.T) {
	// Save and reset the global storage
	originalStorage := globalStorage

	defer func() {
		globalStorage = originalStorage
		resetStorageOnce()
	}()

	// Create a test storage
	store, cleanup := setupTestStorage(t)
	// Don't call cleanup in defer - we'll test close manually

	// Set as global storage
	globalStorage = store
	resetStorageOnce()

	// CloseStorage should work
	err := CloseStorage()
	if err != nil {
		t.Errorf("CloseStorage should not error: %v", err)
	}

	// Clean up temp directory
	cleanup()
}

func TestRootCommandHasExpectedFlags(t *testing.T) {
	// Verify root command structure
	if rootCmd.Use != "toki" {
		t.Errorf("expected Use 'toki', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("root command should have short description")
	}

	if rootCmd.Long == "" {
		t.Error("root command should have long description")
	}
}
