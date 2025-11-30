// ABOUTME: Git repository detection and path normalization
// ABOUTME: Walks directory tree to find .git and resolves symlinks

package git

import (
	"fmt"
	"os"
	"path/filepath"
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
		if _, err := os.Stat(gitPath); err == nil {
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
