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
	cmd.Env = gitEnv()
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	return tmpDir
}

// gitEnv returns the process environment with GIT_DIR and GIT_WORK_TREE
// removed, so git commands run against the fixture directory even when the
// tests are executed from within a hook or worktree.
func gitEnv() []string {
	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") {
			continue
		}
		env = append(env, e)
	}
	return env
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

// newGitWorktree creates a repository with one commit and a linked worktree,
// returning the main repository root and the worktree path.
func newGitWorktree(t *testing.T) (repoDir, worktreeDir string) {
	t.Helper()

	repoDir = setupGitRepo(t)

	commit := exec.Command(
		"git",
		"-c", "user.name=toki-test",
		"-c", "user.email=toki-test@example.com",
		"commit", "--allow-empty", "-m", "init",
	)
	commit.Dir = repoDir
	commit.Env = gitEnv()
	if out, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create initial commit: %v: %s", err, out)
	}

	worktreeDir = filepath.Join(t.TempDir(), "my-worktree")
	add := exec.Command("git", "worktree", "add", "-b", "test-branch", worktreeDir)
	add.Dir = repoDir
	add.Env = gitEnv()
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create worktree: %v: %s", err, out)
	}

	return repoDir, worktreeDir
}

func TestFindGitRootFromWorktree(t *testing.T) {
	repoDir, worktreeDir := newGitWorktree(t)

	root, err := FindGitRoot(worktreeDir)
	if err != nil {
		t.Fatalf("Failed to find git root from worktree: %v", err)
	}

	// A linked worktree shares the main repository's project, so detection
	// must resolve back to the main repository root.
	absRepo, _ := filepath.EvalSymlinks(repoDir)
	absRoot, _ := filepath.EvalSymlinks(root)

	if absRoot != absRepo {
		t.Errorf("Expected main repo root %s, got %s", absRepo, absRoot)
	}

	// Also test from a subdirectory within the worktree.
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

// newBareWorktree creates a bare repository with a linked worktree, returning
// the bare repository directory and the worktree path. For a bare repository
// the "commondir" pointer resolves to the bare repo directory itself, which is
// not named ".git".
func newBareWorktree(t *testing.T) (bareDir, worktreeDir string) {
	t.Helper()

	tmpDir := t.TempDir()
	bareDir = filepath.Join(tmpDir, "repo.git")

	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = gitEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v: %s", args, err, out)
		}
	}

	run(tmpDir, "init", "--bare", "-q", bareDir)

	seed := filepath.Join(tmpDir, "seed")
	run(tmpDir, "init", "-q", seed)
	run(seed, "-c", "user.name=toki-test", "-c", "user.email=toki-test@example.com", "commit", "--allow-empty", "-m", "init")
	run(seed, "push", "-q", bareDir, "HEAD")

	worktreeDir = filepath.Join(tmpDir, "wt")
	run(bareDir, "worktree", "add", "-q", worktreeDir)

	return bareDir, worktreeDir
}

func TestFindGitRootFromBareWorktree(t *testing.T) {
	bareDir, worktreeDir := newBareWorktree(t)

	root, err := FindGitRoot(worktreeDir)
	if err != nil {
		t.Fatalf("Failed to find git root from bare worktree: %v", err)
	}

	// The worktree's common git dir is the bare repository directory itself,
	// so detection must resolve to the bare repo, not its parent.
	absBare, _ := filepath.EvalSymlinks(bareDir)
	absRoot, _ := filepath.EvalSymlinks(root)
	if absRoot != absBare {
		t.Errorf("Expected bare repo root %s, got %s", absBare, absRoot)
	}
}

func TestResolveWorktreeRootMalformedGitFile(t *testing.T) {
	dir := t.TempDir()
	gitFile := filepath.Join(dir, ".git")
	if err := os.WriteFile(gitFile, []byte("this is not a gitdir pointer\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := resolveWorktreeRoot(gitFile); err == nil {
		t.Error("Expected error for malformed .git file")
	}
}

func TestResolveWorktreeRootMissingTarget(t *testing.T) {
	dir := t.TempDir()
	gitFile := filepath.Join(dir, ".git")
	// Points at a gitdir that does not exist on disk.
	if err := os.WriteFile(gitFile, []byte("gitdir: "+filepath.Join(dir, "missing", ".git", "worktrees", "x")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := resolveWorktreeRoot(gitFile); err == nil {
		t.Error("Expected error for .git file pointing at a missing gitdir")
	}
}

func TestResolveWorktreeRootRelativeGitDirFile(t *testing.T) {
	worktreeDir, repoDir := newWorktreeViaFile(t)

	root, err := FindGitRoot(worktreeDir)
	if err != nil {
		t.Fatalf("Failed to find git root: %v", err)
	}

	absRepo, _ := filepath.EvalSymlinks(repoDir)
	absRoot, _ := filepath.EvalSymlinks(root)
	if absRoot != absRepo {
		t.Errorf("Expected main repo root %s, got %s", absRepo, absRoot)
	}
}

// newWorktreeViaFile builds a worktree-shaped checkout by hand. The real
// worktree commands are covered elsewhere; this fixture keeps the relative
// gitdir pointer and .git-as-file cases deterministic without shelling out.
func newWorktreeViaFile(t *testing.T) (worktreeDir, repoDir string) {
	t.Helper()

	repoDir = setupGitRepo(t)
	worktreeGitDir := filepath.Join(repoDir, ".git", "worktrees", "manual")
	if err := os.MkdirAll(worktreeGitDir, 0750); err != nil {
		t.Fatal(err)
	}

	parent := t.TempDir()
	worktreeDir = filepath.Join(parent, "linked")
	if err := os.MkdirAll(worktreeDir, 0750); err != nil {
		t.Fatal(err)
	}

	// git writes a relative "gitdir:" pointer for worktrees created with a
	// relative path.
	rel, err := filepath.Rel(worktreeDir, worktreeGitDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+rel+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	return worktreeDir, repoDir
}

// TestFindGitRootDoesNotAttributeSubmoduleToSuperproject guards the fallback:
// a submodule's .git file lives at "<super>/.git/modules/<name>" and has no
// commondir. It is its own repository, so detection must stay inside it.
func TestFindGitRootDoesNotAttributeSubmoduleToSuperproject(t *testing.T) {
	super := setupGitRepo(t)
	subGitDir := filepath.Join(super, ".git", "modules", "sub")
	if err := os.MkdirAll(subGitDir, 0750); err != nil {
		t.Fatal(err)
	}

	sub := filepath.Join(super, "sub")
	if err := os.MkdirAll(sub, 0750); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(sub, subGitDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, ".git"), []byte("gitdir: "+rel+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	root, err := FindGitRoot(sub)
	if err != nil {
		t.Fatalf("Failed to find git root: %v", err)
	}

	absSub, _ := filepath.EvalSymlinks(sub)
	absRoot, _ := filepath.EvalSymlinks(root)
	if absRoot != absSub {
		t.Errorf("Expected submodule root %s, got %s", absSub, absRoot)
	}
}

func TestWorktreeCommonGitDirRejectsNonWorktree(t *testing.T) {
	if _, err := worktreeCommonGitDir("/tmp/example/.git/modules/sub"); err == nil {
		t.Error("Expected error for submodule-shaped gitdir")
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

// runGit runs a git command in dir and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s failed: %v: %s", args, dir, err, out)
	}
}

const testIdentityName = "toki-test"
const testIdentityEmail = "toki-test@example.com"

// newSubmoduleWorktree creates a superproject with a submodule that itself has
// a linked worktree, returning the submodule's main checkout and the worktree
// path. The worktree's commondir resolves to "<super>/.git/modules/<name>", a
// git dir that has a working tree (recorded in core.worktree) but is not named
// ".git".
func newSubmoduleWorktree(t *testing.T) (subDir, worktreeDir string) {
	t.Helper()

	tmpDir := t.TempDir()

	subSrc := filepath.Join(tmpDir, "sub-src")
	runGit(t, tmpDir, "init", "-q", subSrc)
	runGit(t, subSrc, "-c", "user.name="+testIdentityName, "-c", "user.email="+testIdentityEmail,
		"commit", "--allow-empty", "-q", "-m", "init")

	super := filepath.Join(tmpDir, "super")
	runGit(t, tmpDir, "init", "-q", super)
	runGit(t, super, "-c", "user.name="+testIdentityName, "-c", "user.email="+testIdentityEmail,
		"commit", "--allow-empty", "-q", "-m", "init")
	runGit(t, super, "-c", "protocol.file.allow=always", "submodule", "add", "-q", subSrc, "thesub")
	runGit(t, super, "-c", "user.name="+testIdentityName, "-c", "user.email="+testIdentityEmail,
		"commit", "-q", "-m", "add submodule")

	subDir = filepath.Join(super, "thesub")
	worktreeDir = filepath.Join(tmpDir, "subwt")
	runGit(t, subDir, "worktree", "add", "-q", worktreeDir, "-b", "wt")

	return subDir, worktreeDir
}

// TestFindGitRootFromSubmoduleWorktree guards against resolving a submodule's
// worktree to the internal git dir ("<super>/.git/modules/<name>"), which is
// not a working directory. The submodule's own working tree must be reported,
// matching what git itself reports for the common git dir.
func TestFindGitRootFromSubmoduleWorktree(t *testing.T) {
	subDir, worktreeDir := newSubmoduleWorktree(t)

	root, err := FindGitRoot(worktreeDir)
	if err != nil {
		t.Fatalf("Failed to find git root from submodule worktree: %v", err)
	}

	// The submodule's worktree shares the submodule's project, whose checkout
	// is recorded in core.worktree. This matches
	// `git --git-dir=<common git dir> rev-parse --show-toplevel`.
	absSub, _ := filepath.EvalSymlinks(subDir)
	absRoot, _ := filepath.EvalSymlinks(root)
	if absRoot != absSub {
		t.Errorf("Expected submodule root %s, got %s", absSub, absRoot)
	}
}

// newSeparateGitDirWorktree creates a repository whose git dir lives outside
// the working tree and a linked worktree of it, returning both paths. The
// worktree's commondir points at the relocated git dir, which is not named
// ".git" but owns a working tree.
func newSeparateGitDirWorktree(t *testing.T) (workDir, worktreeDir string) {
	t.Helper()

	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, "store", "main.git")
	if err := os.MkdirAll(filepath.Dir(gitDir), 0750); err != nil {
		t.Fatal(err)
	}

	workDir = filepath.Join(tmpDir, "work")
	runGit(t, tmpDir, "init", "-q", "--separate-git-dir="+gitDir, workDir)
	runGit(t, workDir, "-c", "user.name="+testIdentityName, "-c", "user.email="+testIdentityEmail,
		"commit", "--allow-empty", "-q", "-m", "init")

	worktreeDir = filepath.Join(tmpDir, "wt")
	runGit(t, workDir, "worktree", "add", "-q", worktreeDir, "-b", "wt")

	return workDir, worktreeDir
}

func TestFindGitRootFromSeparateGitDirWorktree(t *testing.T) {
	_, worktreeDir := newSeparateGitDirWorktree(t)

	root, err := FindGitRoot(worktreeDir)
	if err != nil {
		t.Fatalf("Failed to find git root from separate-git-dir worktree: %v", err)
	}

	absWorktree, _ := filepath.EvalSymlinks(worktreeDir)
	absRoot, _ := filepath.EvalSymlinks(root)
	if absRoot != absWorktree {
		t.Errorf("Expected worktree root %s, got %s", absWorktree, absRoot)
	}
}

func TestFindGitRootFromSeparateGitDirRepo(t *testing.T) {
	workDir, _ := newSeparateGitDirWorktree(t)

	root, err := FindGitRoot(workDir)
	if err != nil {
		t.Fatalf("Failed to find git root from separate-git-dir repo: %v", err)
	}

	absWork, _ := filepath.EvalSymlinks(workDir)
	absRoot, _ := filepath.EvalSymlinks(root)
	if absRoot != absWork {
		t.Errorf("Expected working tree root %s, got %s", absWork, absRoot)
	}
}

func TestMainRootForGitDirUsesCoreWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, "store", "main.git")
	if err := os.MkdirAll(gitDir, 0750); err != nil {
		t.Fatal(err)
	}
	// Relative core.worktree is resolved against the git dir by git.
	if err := os.WriteFile(filepath.Join(gitDir, "config"),
		[]byte("[core]\n\tbare = false\n\tworktree = ../../work\n"), 0600); err != nil {
		t.Fatal(err)
	}

	root, err := mainRootForGitDir(gitDir)
	if err != nil {
		t.Fatalf("mainRootForGitDir returned error: %v", err)
	}

	expected := filepath.Join(tmpDir, "work")
	if root != expected {
		t.Errorf("Expected %s, got %s", expected, root)
	}
}

func TestMainRootForGitDirBareRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, "repo.git")
	if err := os.MkdirAll(gitDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"),
		[]byte("[core]\n\tbare = true\n"), 0600); err != nil {
		t.Fatal(err)
	}

	root, err := mainRootForGitDir(gitDir)
	if err != nil {
		t.Fatalf("mainRootForGitDir returned error: %v", err)
	}
	if root != gitDir {
		t.Errorf("Expected bare repo dir %s, got %s", gitDir, root)
	}
}

func TestMainRootForGitDirNonDotGitWithoutConfig(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, "repo.git")
	if err := os.MkdirAll(gitDir, 0750); err != nil {
		t.Fatal(err)
	}

	// No readable config and not named ".git": refuse to guess.
	if _, err := mainRootForGitDir(gitDir); err == nil {
		t.Error("Expected error for unreadable config in non-.git git dir")
	}
}

func TestReadCoreConfigQuotedWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, "repo.git")
	if err := os.MkdirAll(gitDir, 0750); err != nil {
		t.Fatal(err)
	}
	config := "[core]\n\tbare = false\n\tworktree = \"" + tmpDir + "/my work\"\n[remote \"origin\"]\n\tbare = true\n"
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	worktree, bare, err := readCoreConfig(gitDir)
	if err != nil {
		t.Fatalf("readCoreConfig returned error: %v", err)
	}
	if bare {
		t.Error("Expected bare=false from [core] section")
	}
	if expected := tmpDir + "/my work"; worktree != expected {
		t.Errorf("Expected worktree %q, got %q", expected, worktree)
	}
}

func TestReadCoreConfigBooleanSpellings(t *testing.T) {
	cases := []struct {
		value string
		bare  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"on", true},
		{"1", true},
		{"", true},
		{"false", false},
		{"FALSE", false},
		{"no", false},
		{"off", false},
		{"0", false},
	}

	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			tmpDir := t.TempDir()
			gitDir := filepath.Join(tmpDir, "repo.git")
			if err := os.MkdirAll(gitDir, 0750); err != nil {
				t.Fatal(err)
			}
			config := "[core]\n\tbare\n"
			if tc.value != "" {
				config = "[core]\n\tbare = " + tc.value + "\n"
			}
			if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0600); err != nil {
				t.Fatal(err)
			}

			_, bare, err := readCoreConfig(gitDir)
			if err != nil {
				t.Fatalf("readCoreConfig returned error: %v", err)
			}
			if bare != tc.bare {
				t.Errorf("bare = %q: expected %v, got %v", tc.value, tc.bare, bare)
			}
		})
	}
}

func TestMainRootForGitDirBareWithYesValue(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, "repo.git")
	if err := os.MkdirAll(gitDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"),
		[]byte("[core]\n\tbare = yes\n"), 0600); err != nil {
		t.Fatal(err)
	}

	root, err := mainRootForGitDir(gitDir)
	if err != nil {
		t.Fatalf("mainRootForGitDir returned error: %v", err)
	}
	if root != gitDir {
		t.Errorf("Expected bare repo dir %s, got %s", gitDir, root)
	}
}

func TestReadCoreConfigSectionInlineComment(t *testing.T) {
	for _, comment := range []string{"; comment", "# comment", " ; comment", "\t# comment"} {
		t.Run(comment, func(t *testing.T) {
			tmpDir := t.TempDir()
			gitDir := filepath.Join(tmpDir, "repo.git")
			if err := os.MkdirAll(gitDir, 0750); err != nil {
				t.Fatal(err)
			}
			config := "[core] " + comment + "\n\tbare = false\n\tworktree = ../../work\n"
			if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0600); err != nil {
				t.Fatal(err)
			}

			worktree, bare, err := readCoreConfig(gitDir)
			if err != nil {
				t.Fatalf("readCoreConfig returned error: %v", err)
			}
			if bare {
				t.Error("Expected bare=false from [core] section")
			}
			if worktree != "../../work" {
				t.Errorf("Expected worktree %q, got %q", "../../work", worktree)
			}
		})
	}
}
