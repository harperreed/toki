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
	run := setupTestBinary(t)

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

func TestSyncInit(t *testing.T) {
	run, configDir := setupTestBinaryWithDirs(t)

	// Run sync init
	output, err := run("sync", "init")
	if err != nil {
		t.Fatalf("Failed to init sync: %v\n%s", err, output)
	}

	if !strings.Contains(output, "Sync initialized") {
		t.Error("Expected success message")
	}

	if !strings.Contains(output, "Device:") {
		t.Error("Expected device ID in output")
	}

	// Verify config file was created (XDG_CONFIG_HOME/toki/sync.json)
	configPath := filepath.Join(configDir, "toki", "sync.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify init fails if run again
	output, err = run("sync", "init")
	if err == nil {
		t.Error("Expected error when running init twice")
	}

	if !strings.Contains(output, "config already exists") {
		t.Error("Expected 'config already exists' error message")
	}
}

func TestSyncStatus_NotConfigured(t *testing.T) {
	run := setupTestBinary(t)

	// Use temp config directory - set XDG_CONFIG_HOME which takes precedence
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Run sync status without init
	output, err := run("sync", "status")
	if err != nil {
		t.Fatalf("Failed to get sync status: %v\n%s", err, output)
	}

	if !strings.Contains(output, "Config:") {
		t.Error("Expected config path in output")
	}

	if !strings.Contains(output, "(not set)") {
		t.Error("Expected '(not set)' for unconfigured values")
	}
}

func TestSyncStatus_AfterInit(t *testing.T) {
	run := setupTestBinary(t)

	// Use temp config directory - set XDG_CONFIG_HOME which takes precedence
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Init sync first
	_, err := run("sync", "init")
	if err != nil {
		t.Fatalf("Failed to init sync: %v", err)
	}

	// Run sync status
	output, err := run("sync", "status")
	if err != nil {
		t.Fatalf("Failed to get sync status: %v\n%s", err, output)
	}

	if !strings.Contains(output, "Device ID:") {
		t.Error("Expected device ID in status")
	}

	if !strings.Contains(output, "https://api.storeusa.org") {
		t.Error("Expected default server URL in status")
	}

	if !strings.Contains(output, "Not logged in") {
		t.Error("Expected 'Not logged in' status")
	}
}

func TestSyncPending_NotConfigured(t *testing.T) {
	run := setupTestBinary(t)

	// Use temp config directory - set XDG_CONFIG_HOME which takes precedence
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Run sync pending without init
	output, err := run("sync", "pending")
	if err != nil {
		t.Fatalf("Failed to get pending changes: %v\n%s", err, output)
	}

	if !strings.Contains(output, "not configured") {
		t.Error("Expected 'not configured' message")
	}
}

func TestSyncPending_AfterInit(t *testing.T) {
	run := setupTestBinary(t)

	// Use temp config directory - set XDG_CONFIG_HOME which takes precedence
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Init sync first
	_, err := run("sync", "init")
	if err != nil {
		t.Fatalf("Failed to init sync: %v", err)
	}

	// Run sync pending (should show not configured since we haven't logged in)
	output, err := run("sync", "pending")
	if err != nil {
		t.Fatalf("Failed to get pending changes: %v\n%s", err, output)
	}

	if !strings.Contains(output, "not configured") {
		t.Error("Expected 'not configured' message when not logged in")
	}
}

func TestSyncLogout_WhenNotLoggedIn(t *testing.T) {
	run := setupTestBinary(t)

	// Use temp config directory - set XDG_CONFIG_HOME which takes precedence
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Init sync first
	_, err := run("sync", "init")
	if err != nil {
		t.Fatalf("Failed to init sync: %v", err)
	}

	// Run logout when not logged in
	output, err := run("sync", "logout")
	if err != nil {
		t.Fatalf("Failed to logout: %v\n%s", err, output)
	}

	if !strings.Contains(output, "Not logged in") {
		t.Error("Expected 'Not logged in' message")
	}
}

func TestOfflineQueueing(t *testing.T) {
	run := setupTestBinary(t)

	// Use temp config directory - set XDG_CONFIG_HOME which takes precedence
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Create a project
	_, err := run("project", "add", "test-project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Add a todo (this should work offline)
	output, err := run("add", "offline todo", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to add todo: %v\n%s", err, output)
	}

	if !strings.Contains(output, "offline todo") {
		t.Error("Expected todo to be created")
	}

	// Verify todo is in the list
	output, err = run("list", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to list todos: %v\n%s", err, output)
	}

	if !strings.Contains(output, "offline todo") {
		t.Error("Todo should be in the list")
	}

	// Mark it done (this should also work offline)
	todoPrefix := extractTodoPrefix(output)
	_, err = run("done", todoPrefix)
	if err != nil {
		t.Fatalf("Failed to mark done: %v", err)
	}

	// Verify it's marked as done
	output, err = run("list", "--project", "test-project", "--done")
	if err != nil {
		t.Fatalf("Failed to list done todos: %v\n%s", err, output)
	}

	if !strings.Contains(output, "✓") {
		t.Error("Todo should be marked as done")
	}
}

func setupTestBinary(t *testing.T) func(args ...string) (string, error) {
	run, _ := setupTestBinaryWithDirs(t)
	return run
}

func setupTestBinaryWithDirs(t *testing.T) (func(args ...string) (string, error), string) {
	t.Helper()
	projectRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Build binary in temp dir to avoid race conditions between tests
	tmpDir := t.TempDir()
	tokiBinary := filepath.Join(tmpDir, "toki")
	buildCmd := exec.Command("go", "build", "-o", tokiBinary, "./cmd/toki") //nolint:gosec // Safe: building our own binary with fixed args
	buildCmd.Dir = projectRoot
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build: %v\nOutput: %s", err, buildOutput)
	}
	// No need for cleanup - t.TempDir() handles it

	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")

	run := func(args ...string) (string, error) {
		cmd := exec.Command(tokiBinary, args...) //nolint:gosec // Safe: executing our own test binary with controlled args
		cmd.Env = append(os.Environ(),
			"XDG_DATA_HOME="+dataDir,
			"XDG_CONFIG_HOME="+configDir,
		)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	return run, configDir
}

func extractTodoPrefix(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for _, field := range fields {
			// Look for 6-character hex prefix
			if len(field) == 6 {
				return field
			}
		}
	}
	return ""
}

func TestDoneCommand_ShowsCheckmark(t *testing.T) {
	run := setupTestBinary(t)

	_, err := run("project", "add", "test-project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	addOutput, err := run("add", "task to complete", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to add todo: %v\n%s", err, addOutput)
	}

	todoPrefix := extractTodoPrefix(addOutput)
	if todoPrefix == "" {
		t.Fatalf("Could not extract todo prefix")
	}

	_, err = run("done", todoPrefix)
	if err != nil {
		t.Fatalf("Failed to mark done: %v", err)
	}

	listOutput, err := run("list", "--project", "test-project", "--done")
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}

	if !strings.Contains(listOutput, "✓") {
		t.Error("Completed todo should show checkmark")
	}
}

func TestListCommand_DefaultShowsPendingOnly(t *testing.T) {
	run := setupTestBinary(t)

	_, err := run("project", "add", "test-project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Add and complete a todo
	addOutput, err := run("add", "done task", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}

	todoPrefix := extractTodoPrefix(addOutput)
	_, err = run("done", todoPrefix)
	if err != nil {
		t.Fatalf("Failed to mark done: %v", err)
	}

	// Add a pending todo
	_, err = run("add", "pending task", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to add pending todo: %v", err)
	}

	// Default list should only show pending
	listOutput, err := run("list", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}

	if strings.Contains(listOutput, "done task") {
		t.Error("Default list should not show completed todos")
	}

	if !strings.Contains(listOutput, "pending task") {
		t.Error("Default list should show pending todos")
	}

	if !strings.Contains(listOutput, "pending") {
		t.Error("Summary should say 'pending'")
	}
}
