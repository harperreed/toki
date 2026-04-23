// ABOUTME: Tests for git repository detection
// ABOUTME: Creates temporary git repos for testing path detection

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	// Filter out GIT_DIR and GIT_WORK_TREE from environment to ensure
	// git init works correctly in worktree environments and pre-commit hooks
	env := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_DIR=") && !strings.HasPrefix(e, "GIT_WORK_TREE=") {
			env = append(env, e)
		}
	}
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	return tmpDir
}

func TestFindGitRoot(t *testing.T) {
	repoDir := setupGitRepo(t)

	// Create subdirectory
	subDir := filepath.Join(repoDir, "nested", "deep")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatal(err)
	}

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

func TestNormalizePathRealDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := NormalizePath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to normalize temp dir: %v", err)
	}

	// Should return the resolved absolute path
	if !filepath.IsAbs(result) {
		t.Error("Expected absolute path")
	}
}

func TestNormalizePathSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real directory
	realDir := filepath.Join(tmpDir, "real")
	if err := os.MkdirAll(realDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to it
	linkDir := filepath.Join(tmpDir, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skip("Symlinks not supported on this system")
	}

	result, err := NormalizePath(linkDir)
	if err != nil {
		t.Fatalf("Failed to normalize symlink: %v", err)
	}

	// Should resolve symlink to real directory
	// Also resolve the expected path to handle system symlinks (e.g., /var -> /private/var on macOS)
	expectedResolved, _ := filepath.EvalSymlinks(realDir)
	if result != expectedResolved {
		t.Errorf("Expected resolved path %s, got %s", expectedResolved, result)
	}
}

func TestNormalizePathNonexistent(t *testing.T) {
	// Non-existent path should still return absolute path
	// (symlink resolution will fail gracefully)
	result, err := NormalizePath("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("Failed to normalize nonexistent path: %v", err)
	}

	if !filepath.IsAbs(result) {
		t.Error("Expected absolute path for nonexistent path")
	}
}

func TestFindGitRootFromRepoRoot(t *testing.T) {
	repoDir := setupGitRepo(t)

	// Test from the repo root itself
	root, err := FindGitRoot(repoDir)
	if err != nil {
		t.Fatalf("Failed to find git root: %v", err)
	}

	absRepo, _ := filepath.EvalSymlinks(repoDir)
	absRoot, _ := filepath.EvalSymlinks(root)

	if absRoot != absRepo {
		t.Errorf("Expected git root %s, got %s", absRepo, absRoot)
	}
}

func TestFindGitRootFromWorktree(t *testing.T) {
	repoDir := setupGitRepo(t)

	// Create an initial commit so worktree creation works
	cmd := exec.Command("git", "commit", "--allow-empty", "-m", "init")
	cmd.Dir = repoDir
	env := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_DIR=") && !strings.HasPrefix(e, "GIT_WORK_TREE=") {
			env = append(env, e)
		}
	}
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Create a worktree
	worktreeDir := filepath.Join(t.TempDir(), "my-worktree")
	cmd = exec.Command("git", "worktree", "add", "-b", "test-branch", worktreeDir)
	cmd.Dir = repoDir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// FindGitRoot from within the worktree should return the main repo root
	root, err := FindGitRoot(worktreeDir)
	if err != nil {
		t.Fatalf("Failed to find git root from worktree: %v", err)
	}

	absRepo, _ := filepath.EvalSymlinks(repoDir)
	absRoot, _ := filepath.EvalSymlinks(root)

	if absRoot != absRepo {
		t.Errorf("Expected main repo root %s, got %s", absRepo, absRoot)
	}

	// Also test from a subdirectory within the worktree
	subDir := filepath.Join(worktreeDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatal(err)
	}

	root, err = FindGitRoot(subDir)
	if err != nil {
		t.Fatalf("Failed to find git root from worktree subdir: %v", err)
	}

	absRoot, _ = filepath.EvalSymlinks(root)
	if absRoot != absRepo {
		t.Errorf("Expected main repo root %s from subdir, got %s", absRepo, absRoot)
	}
}

func TestFindGitRootWithSymlink(t *testing.T) {
	repoDir := setupGitRepo(t)
	tmpDir := t.TempDir()

	// Create a symlink to the repo
	linkPath := filepath.Join(tmpDir, "repo-link")
	if err := os.Symlink(repoDir, linkPath); err != nil {
		t.Skip("Symlinks not supported on this system")
	}

	// Find git root from the symlink
	root, err := FindGitRoot(linkPath)
	if err != nil {
		t.Fatalf("Failed to find git root from symlink: %v", err)
	}

	// The result should be the resolved (real) path
	absRepo, _ := filepath.EvalSymlinks(repoDir)
	absRoot, _ := filepath.EvalSymlinks(root)

	if absRoot != absRepo {
		t.Errorf("Expected resolved git root %s, got %s", absRepo, absRoot)
	}
}
