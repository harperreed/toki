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
