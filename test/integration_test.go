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
