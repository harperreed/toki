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
//
// A linked worktree has a .git *file* (a "gitdir:" pointer) instead of a .git
// directory. Worktrees share the main repository, so detection resolves back to
// the main repository root rather than treating each worktree as its own repo.
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
				// .git is a file (linked worktree or submodule) containing
				// "gitdir: <path>". For a worktree, resolve to the main
				// repository root so the worktree shares the same project.
				// If resolution fails, fall back to the containing directory
				// so an unusual setup still yields something usable.
				if mainRoot, err := resolveWorktreeRoot(gitPath); err == nil {
					return mainRoot, nil
				}
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

// resolveWorktreeRoot reads a .git file (as found in worktrees and submodules)
// and resolves back to the main repository root.
//
// The file contains "gitdir: <path>", where path points at the main
// repository's .git/worktrees/<name> directory. That directory contains a
// "commondir" file pointing back at the main .git directory. Both the pointer
// and its target are validated so a malformed or missing gitdir makes the
// caller fall back to the old behavior instead of returning a bogus root.
func resolveWorktreeRoot(gitFilePath string) (string, error) {
	data, err := os.ReadFile(gitFilePath) // #nosec G304 - path derived from the caller's .git pointer
	if err != nil {
		return "", fmt.Errorf("failed to read git file: %w", err)
	}

	gitdir, err := parseGitDirPointer(string(data), filepath.Dir(gitFilePath))
	if err != nil {
		return "", err
	}

	info, err := os.Stat(gitdir)
	if err != nil {
		return "", fmt.Errorf("invalid gitdir target %q: %w", gitdir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("invalid gitdir target %q: not a directory", gitdir)
	}

	commonGitDir, err := resolveCommonGitDir(gitdir)
	if err != nil {
		return "", err
	}

	mainRoot, err := mainRootForGitDir(commonGitDir)
	if err != nil {
		return "", err
	}

	resolved, err := filepath.EvalSymlinks(mainRoot)
	if err != nil {
		return mainRoot, nil //nolint:nilerr // Intentional: symlink resolution failure is not critical, return unresolved path
	}
	return resolved, nil
}

// mainRootForGitDir maps a common git dir back to the working tree it belongs
// to.
//
// The common git dir is the working tree's ".git" directory for a normal
// repository, so the root is its parent. Some layouts have a git dir that is
// not named ".git" yet still owns a working tree: a submodule keeps its git dir
// at "<super>/.git/modules/<name>" and records the checkout in core.worktree,
// while a bare repository has no working tree at all. Guessing from the
// directory name alone would return the internal git dir as the project root,
// so consult the git dir's configuration first and only fall back to the
// name-based heuristic.
func mainRootForGitDir(commonGitDir string) (string, error) {
	worktree, bare, err := readCoreConfig(commonGitDir)
	if err != nil {
		// Without readable configuration the only safe guess is the
		// conventional "<root>/.git" layout; anything else is reported as
		// unresolved so the caller keeps the historical behavior.
		if filepath.Base(commonGitDir) == ".git" {
			return filepath.Dir(commonGitDir), nil
		}
		return "", err
	}

	// A bare repository is its own root: there is no working tree to escape to.
	if bare {
		return commonGitDir, nil
	}

	// An explicit core.worktree (submodules, --separate-git-dir checkouts with
	// a recorded work tree) is authoritative. Git resolves a relative value
	// against the git dir.
	if worktree != "" {
		if !filepath.IsAbs(worktree) {
			worktree = filepath.Join(commonGitDir, worktree)
		}
		return filepath.Clean(worktree), nil
	}

	if filepath.Base(commonGitDir) == ".git" {
		return filepath.Dir(commonGitDir), nil
	}

	return "", fmt.Errorf("cannot determine working tree for git dir %q", commonGitDir)
}

// readCoreConfig extracts the core.bare and core.worktree settings from a git
// dir's config file. It is a deliberately small parser: only the two keys that
// affect working tree resolution are read, and unknown keys or sections are
// ignored.
func readCoreConfig(gitDir string) (worktree string, bare bool, err error) {
	data, err := os.ReadFile(filepath.Join(gitDir, "config")) // #nosec G304 - path is the git dir being inspected
	if err != nil {
		return "", false, err
	}

	inCore := false
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(rawLine, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			// A section header ends at the first "]" (or the end of the
			// line); anything after it, such as an inline comment, is not
			// part of the section name.
			section := strings.TrimPrefix(line, "[")
			if idx := strings.IndexByte(section, ']'); idx >= 0 {
				section = section[:idx]
			}
			inCore = strings.EqualFold(strings.TrimSpace(section), "core")
			continue
		}

		if !inCore {
			continue
		}

		// Git treats a key with no "=" as set to the empty value, which is
		// true for boolean variables.
		key, value, _ := strings.Cut(line, "=")

		switch strings.ToLower(strings.TrimSpace(key)) {
		case "bare":
			bare = parseGitBool(strings.TrimSpace(value))
		case "worktree":
			worktree = unquoteConfigValue(strings.TrimSpace(value))
		}
	}

	return worktree, bare, nil
}

// parseGitBool interprets a git config boolean value using git's own syntax:
// true/yes/on/1 (case-insensitive, and the empty value) are true, while
// false/no/off/0 are false. Anything else is treated as false, matching the
// conservative default of the previous literal comparison.
func parseGitBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true", "yes", "on", "1":
		return true
	default:
		return false
	}
}

// unquoteConfigValue removes the surrounding double quotes git adds when a
// config value contains characters that need escaping, leaving plain values
// untouched.
func unquoteConfigValue(value string) string {
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		unquoted := value[1 : len(value)-1]
		unquoted = strings.ReplaceAll(unquoted, `\\`, `\`)
		unquoted = strings.ReplaceAll(unquoted, `\"`, `"`)
		return unquoted
	}
	return value
}

// parseGitDirPointer extracts the path from a "gitdir: <path>" pointer,
// resolving relative paths against the directory containing the .git file.
func parseGitDirPointer(contents, baseDir string) (string, error) {
	line := strings.TrimSpace(contents)
	if line == "" {
		return "", fmt.Errorf("empty .git file")
	}

	// Git writes exactly one pointer line; ignore any trailing content.
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}

	const prefix = "gitdir:"
	if !strings.HasPrefix(line, prefix) {
		return "", fmt.Errorf("unexpected .git file format: %q", line)
	}

	gitdir := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if gitdir == "" {
		return "", fmt.Errorf("empty gitdir pointer in .git file")
	}

	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(baseDir, gitdir)
	}

	return filepath.Clean(gitdir), nil
}

// resolveCommonGitDir locates the main .git directory that owns the given
// worktree gitdir. It prefers the explicit "commondir" pointer and falls back
// to the conventional ".git/worktrees/<name>" layout.
//
// The fallback is deliberately restricted to worktree-shaped gitdirs. A
// submodule also has a .git file, but its gitdir lives at
// "<superproject>/.git/modules/<name>"; treating it as a worktree would
// wrongly attribute the submodule to its superproject. For anything else we
// return an error so FindGitRoot keeps the historical behavior.
func resolveCommonGitDir(worktreeGitDir string) (string, error) {
	commonDir, err := readCommonDir(worktreeGitDir)
	if err != nil {
		commonDir, err = worktreeCommonGitDir(worktreeGitDir)
		if err != nil {
			return "", err
		}
	}

	info, err := os.Stat(commonDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("could not resolve main .git directory from gitdir %q", worktreeGitDir)
	}

	return commonDir, nil
}

// readCommonDir reads the "commondir" file inside a worktree gitdir. Its
// contents are relative to the worktree gitdir (typically "../..").
func readCommonDir(worktreeGitDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(worktreeGitDir, "commondir")) // #nosec G304 - path is the worktree git dir being inspected
	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("empty commondir file in %q", worktreeGitDir)
	}

	if !filepath.IsAbs(value) {
		value = filepath.Join(worktreeGitDir, value)
	}

	return filepath.Clean(value), nil
}

// worktreeCommonGitDir derives the main .git directory from the conventional
// "<main>/.git/worktrees/<name>" layout. It returns an error when the path is
// not worktree-shaped.
func worktreeCommonGitDir(worktreeGitDir string) (string, error) {
	dir := filepath.Clean(worktreeGitDir)

	// Expect: <main>/.git/worktrees/<name>
	if filepath.Base(filepath.Dir(dir)) != "worktrees" || filepath.Base(filepath.Dir(filepath.Dir(dir))) != ".git" {
		return "", fmt.Errorf("gitdir %q is not a worktree gitdir", worktreeGitDir)
	}

	commonDir := filepath.Dir(filepath.Dir(dir)) // <main>/.git
	if info, err := os.Stat(commonDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("invalid worktree gitdir %q: main .git directory missing", worktreeGitDir)
	}

	return commonDir, nil
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
