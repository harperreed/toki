// ABOUTME: Git repository detection and path normalization
// ABOUTME: Walks directory tree to find .git and resolves symlinks

package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindGitRoot walks up the directory tree looking for .git
// Returns the absolute path to the repository root.
func FindGitRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	currentPath := absPath
	for {
		gitPath := filepath.Join(currentPath, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			if !info.IsDir() {
				// Worktree: .git is a file containing "gitdir: <path>"
				if mainRoot, err := resolveWorktreeRoot(gitPath); err == nil {
					return mainRoot, nil
				}
				// Fall through to return worktree path if resolution fails
			}
			// Resolve symlinks
			resolved, err := filepath.EvalSymlinks(currentPath)
			if err != nil {
				return currentPath, nil //nolint:nilerr // Intentional: symlink resolution failure is not critical, return unresolved path
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

// resolveWorktreeRoot reads a .git file (as found in worktrees) and resolves
// back to the main repository root. The file contains "gitdir: <path>" where
// path points to .git/worktrees/<name> in the main repo.
func resolveWorktreeRoot(gitFilePath string) (string, error) {
	data, err := os.ReadFile(gitFilePath)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", fmt.Errorf("unexpected .git file format: %s", line)
	}
	gitdir := strings.TrimPrefix(line, "gitdir: ")

	// Resolve relative paths against the worktree directory
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(gitFilePath), gitdir)
	}

	// gitdir is typically <main-repo>/.git/worktrees/<name>
	// Walk up to find the directory containing .git as a directory
	dir := filepath.Clean(gitdir)
	for {
		parent := filepath.Dir(dir)
		if filepath.Base(dir) == ".git" {
			resolved, err := filepath.EvalSymlinks(parent)
			if err != nil {
				return parent, nil
			}
			return resolved, nil
		}
		if parent == dir {
			return "", fmt.Errorf("could not resolve main repo root from gitdir: %s", gitdir)
		}
		dir = parent
	}
}

// NormalizePath converts a path to absolute and resolves symlinks.
func NormalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If symlink resolution fails, return absolute path
		return absPath, nil //nolint:nilerr // Intentional: symlink resolution failure is not critical, return unresolved path
	}

	return resolved, nil
}
