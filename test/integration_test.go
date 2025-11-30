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
	// Get project root directory
	projectRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Build toki binary
	tokiBinary := filepath.Join(projectRoot, "toki")
	buildCmd := exec.Command("go", "build", "-o", tokiBinary, "./cmd/toki") //nolint:gosec // Safe: building our own binary with fixed args
	buildCmd.Dir = projectRoot
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build: %v\nOutput: %s", err, buildOutput)
	}
	defer func() { _ = os.Remove(tokiBinary) }()

	// Use temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	run := func(args ...string) (string, error) {
		fullArgs := append([]string{"--db", dbPath}, args...)
		cmd := exec.Command(tokiBinary, fullArgs...) //nolint:gosec // Safe: executing our own test binary with controlled args
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

func setupTestBinary(t *testing.T) func(args ...string) (string, error) {
	t.Helper()
	projectRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	tokiBinary := filepath.Join(projectRoot, "toki")
	buildCmd := exec.Command("go", "build", "-o", tokiBinary, "./cmd/toki") //nolint:gosec // Safe: building our own binary with fixed args
	buildCmd.Dir = projectRoot
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build: %v\nOutput: %s", err, buildOutput)
	}
	t.Cleanup(func() { _ = os.Remove(tokiBinary) })

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	run := func(args ...string) (string, error) {
		fullArgs := append([]string{"--db", dbPath}, args...)
		cmd := exec.Command(tokiBinary, fullArgs...) //nolint:gosec // Safe: executing our own test binary with controlled args
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	return run
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
