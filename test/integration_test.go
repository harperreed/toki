// ABOUTME: Integration tests for full workflow
// ABOUTME: Tests project creation, todo CRUD, git detection end-to-end

package test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFullWorkflow(t *testing.T) {
	run := setupTestBinary(t)

	// Use unique project name to avoid sync conflicts
	projectName := fmt.Sprintf("test-project-%d", time.Now().UnixNano())

	// Create project
	output, err := run("project", "add", projectName)
	if err != nil {
		t.Fatalf("Failed to create project: %v\n%s", err, output)
	}

	if !strings.Contains(output, "Created project") && !strings.Contains(output, "already exists") {
		t.Error("Expected success message")
	}

	// Add todo
	output, err = run("add", "test todo", "--project", projectName, "--priority", "high")
	if err != nil {
		t.Fatalf("Failed to add todo: %v\n%s", err, output)
	}

	// List todos
	output, err = run("list", "--project", projectName)
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

// forceBackend writes a config selecting the given storage backend.
func forceBackend(t *testing.T, configDir, backend string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(configDir, "toki"), 0o750); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	cfg := fmt.Sprintf(`{"backend":%q}`, backend)
	if err := os.WriteFile(filepath.Join(configDir, "toki", "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
}

func TestAddCommand_RejectsWhitespaceOnlyDescriptions(t *testing.T) {
	for _, backend := range []string{"markdown", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			run, configDir := setupTestBinaryWithDirs(t)
			forceBackend(t, configDir, backend)

			if _, err := run("project", "add", "probe"); err != nil {
				t.Fatalf("Failed to create project: %v", err)
			}

			// The project starts empty.
			if got := exportedTodoCount(t, run); got != 0 {
				t.Fatalf("expected 0 stored todos before rejections, got %d", got)
			}

			// Spaces-only descriptions must be rejected and must not be stored.
			if out, err := run("add", "   ", "--project", "probe"); err == nil {
				t.Fatalf("add of spaces-only description should have failed, output: %s", out)
			}
			if got := exportedTodoCount(t, run); got != 0 {
				t.Errorf("rejected spaces-only description should not be stored, stored todo count = %d", got)
			}

			// Padded two-character input trims to under three characters.
			if out, err := run("add", " ab ", "--project", "probe"); err == nil {
				t.Fatalf("add of padded two-character description should have failed, output: %s", out)
			}
			if got := exportedTodoCount(t, run); got != 0 {
				t.Errorf("rejected padded two-character description should not be stored, stored todo count = %d", got)
			}
		})
	}
}

// exportedTodoCount returns the number of todos stored across all projects by
// parsing `toki export json`, which reads directly from the backend. Unlike
// `toki list`, it counts todos regardless of completion status or how they
// render, so an unexpectedly stored todo cannot hide behind a substring check.
func exportedTodoCount(t *testing.T, run func(args ...string) (string, error)) int {
	t.Helper()
	out, err := run("export", "json")
	if err != nil {
		t.Fatalf("Failed to export: %v\n%s", err, out)
	}
	var data struct {
		Projects []struct {
			Todos []struct {
				ID          string `json:"id"`
				Description string `json:"description"`
			} `json:"todos"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		t.Fatalf("Failed to parse export output: %v\n%s", err, out)
	}
	count := 0
	for _, project := range data.Projects {
		count += len(project.Todos)
	}
	return count
}

func TestAddCommand_AcceptsTrimmedThreeCharacterDescription(t *testing.T) {
	for _, backend := range []string{"markdown", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			run, configDir := setupTestBinaryWithDirs(t)
			forceBackend(t, configDir, backend)

			if _, err := run("project", "add", "probe"); err != nil {
				t.Fatalf("Failed to create project: %v", err)
			}

			if out, err := run("add", "abc", "--project", "probe"); err != nil {
				t.Fatalf("add of three-character description failed: %v\n%s", err, out)
			}

			listOutput, err := run("list", "--project", "probe")
			if err != nil {
				t.Fatalf("Failed to list: %v\n%s", err, listOutput)
			}
			if !strings.Contains(listOutput, "abc") {
				t.Errorf("accepted description should be stored unchanged, list output: %s", listOutput)
			}
		})
	}
}

func TestAddCommand_PreservesSurroundingWhitespace(t *testing.T) {
	for _, backend := range []string{"markdown", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			run, configDir := setupTestBinaryWithDirs(t)
			forceBackend(t, configDir, backend)

			if _, err := run("project", "add", "probe"); err != nil {
				t.Fatalf("Failed to create project: %v", err)
			}

			// Padded three-character input passes validation on its trimmed
			// length and must be stored unchanged, whitespace included.
			if out, err := run("add", "  abc  ", "--project", "probe"); err != nil {
				t.Fatalf("add of padded three-character description failed: %v\n%s", err, out)
			}

			listOutput, err := run("list", "--project", "probe")
			if err != nil {
				t.Fatalf("Failed to list: %v\n%s", err, listOutput)
			}
			if !strings.Contains(listOutput, "  abc  ") {
				t.Errorf("surrounding whitespace should be preserved, list output: %q", listOutput)
			}
		})
	}
}

func TestDoneCommand_LiteralWildcardPrefix(t *testing.T) {
	run, configDir := setupTestBinaryWithDirs(t)

	// Force the SQLite backend so this exercises the LIKE-based prefix lookup.
	if err := os.MkdirAll(filepath.Join(configDir, "toki"), 0o750); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(configDir, "toki", "config.json"),
		[]byte(`{"backend":"sqlite"}`),
		0o600,
	); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	_, err := run("project", "add", "test-project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Exactly one todo in the store.
	_, err = run("add", "only task", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}

	// SQL wildcard prefixes must not match: the ID prefix is literal.
	for _, prefix := range []string{"%", "_"} {
		if out, err := run("done", prefix); err == nil {
			t.Fatalf("done %q should have failed, output: %s", prefix, out)
		}
	}

	// The todo must remain pending.
	listOutput, err := run("list", "--project", "test-project")
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}
	if !strings.Contains(listOutput, "only task") {
		t.Errorf("todo should still be pending, list output: %s", listOutput)
	}
	if strings.Contains(listOutput, "✓") {
		t.Error("todo should not have been marked done")
	}
}
