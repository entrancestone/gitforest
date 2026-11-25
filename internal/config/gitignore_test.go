package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	base, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}

	bareDir := filepath.Join(base, "remote.git")
	repoDir := filepath.Join(base, "myrepo")

	run(t, "", "git", "init", "--bare", "--initial-branch=main", bareDir)
	run(t, "", "git", "init", "--initial-branch=main", repoDir)
	run(t, repoDir, "git", "config", "user.email", "test@test.com")
	run(t, repoDir, "git", "config", "user.name", "Test")
	run(t, repoDir, "git", "remote", "add", "origin", bareDir)

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "initial")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	return repoDir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

func excludePath(repoDir string) string {
	return filepath.Join(repoDir, ".git", "info", "exclude")
}

func readExclude(t *testing.T, repoDir string) string {
	t.Helper()
	data, err := os.ReadFile(excludePath(repoDir))
	if err != nil {
		t.Fatalf("failed to read .git/info/exclude: %v", err)
	}
	return string(data)
}

func TestEnsureExcluded_WritesToGitInfoExclude(t *testing.T) {
	repoDir := setupGitRepo(t)

	gitforestDir := filepath.Join(repoDir, ".gitforest")

	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatalf("EnsureExcluded failed: %v", err)
	}

	expected := "/.gitforest"
	if got := readExclude(t, repoDir); !strings.Contains(got, expected) {
		t.Errorf("expected .git/info/exclude to contain %q, got: %s", expected, got)
	}

	// The tracked .gitignore must stay clean so the repo does not read dirty.
	if _, err := os.Stat(filepath.Join(repoDir, ".gitignore")); !os.IsNotExist(err) {
		t.Errorf("expected no tracked .gitignore to be created, stat err: %v", err)
	}
}

func TestEnsureExcluded_LeavesRepoClean(t *testing.T) {
	repoDir := setupGitRepo(t)

	gitforestDir := filepath.Join(repoDir, ".gitforest")
	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatalf("EnsureExcluded failed: %v", err)
	}

	out, err := exec.Command("git", "-C", repoDir, "status", "--porcelain").CombinedOutput()
	if err != nil {
		t.Fatalf("git status failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("expected clean git status after EnsureExcluded, got:\n%s", out)
	}
}

func TestEnsureExcluded_Idempotent(t *testing.T) {
	repoDir := setupGitRepo(t)

	gitforestDir := filepath.Join(repoDir, ".gitforest")

	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatal(err)
	}
	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatal(err)
	}

	entry := "/.gitforest"
	count := strings.Count(readExclude(t, repoDir), entry)
	if count != 1 {
		t.Errorf("expected entry exactly once, found %d times in:\n%s", count, readExclude(t, repoDir))
	}
}

func TestEnsureExcluded_PreservesPreexistingGitignoreEntry(t *testing.T) {
	repoDir := setupGitRepo(t)

	gitforestDir := filepath.Join(repoDir, ".gitforest")
	entry := "/.gitforest"

	// Simulate a pool created by an older version that added the entry to the
	// tracked .gitignore.
	gitignorePath := filepath.Join(repoDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(entry+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatalf("EnsureExcluded failed: %v", err)
	}

	// The pre-existing .gitignore entry must be left untouched.
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != entry {
		t.Errorf("expected .gitignore to keep %q untouched, got: %s", entry, data)
	}

	// And no duplicate should be written to .git/info/exclude.
	if data, err := os.ReadFile(excludePath(repoDir)); err == nil {
		if strings.Contains(string(data), entry) {
			t.Errorf("expected no duplicate entry in .git/info/exclude, got: %s", data)
		}
	}
}

func TestEnsureExcluded_NestedRoot(t *testing.T) {
	repoDir := setupGitRepo(t)

	// A relative root like ".worktrees" resolves to <repo>/.worktrees/.gitforest.
	gitforestDir := filepath.Join(repoDir, ".worktrees", ".gitforest")

	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatalf("EnsureExcluded failed: %v", err)
	}

	expected := "/.worktrees/.gitforest"
	if got := readExclude(t, repoDir); !strings.Contains(got, expected) {
		t.Errorf("expected .git/info/exclude to contain %q, got: %s", expected, got)
	}
}

func TestEnsureExcluded_NotInRepo(t *testing.T) {
	// A temp dir that is not a git repo: only the self-ignoring pool root is
	// written; nothing outside it is touched.
	dir := t.TempDir()
	gitforestDir := filepath.Join(dir, ".gitforest")

	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatalf("EnsureExcluded should succeed outside a repo, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Error("expected no .gitignore to be created outside the pool root")
	}
	assertSelfIgnore(t, gitforestDir)
}

func TestEnsureExcluded_DefaultRoot(t *testing.T) {
	repoDir := setupGitRepo(t)

	// When using the default root ($HOME/.gitforest), the gitforest dir is
	// outside the repo, so EnsureExcluded must not touch the repo's exclude
	// file or .gitignore.
	setUserHome(t, t.TempDir())
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	gitforestDir := filepath.Join(home, ".gitforest")

	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatalf("EnsureExcluded failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".gitignore")); err == nil {
		t.Error("expected no .gitignore to be created in the repo for the default root")
	}
	assertSelfIgnore(t, gitforestDir)
}

func assertSelfIgnore(t *testing.T, gitforestDir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(gitforestDir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected self-ignoring .gitignore in the pool root: %v", err)
	}
	if strings.TrimSpace(string(data)) != "*" {
		t.Errorf("expected pool root .gitignore to contain %q, got: %s", "*", data)
	}
}

func TestEnsureExcluded_PoolRootIsSelfIgnoring(t *testing.T) {
	repoDir := setupGitRepo(t)
	gitforestDir := filepath.Join(repoDir, ".gitforest")

	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatalf("EnsureExcluded failed: %v", err)
	}
	assertSelfIgnore(t, gitforestDir)

	// An existing pool root .gitignore is left untouched.
	custom := "custom\n"
	if err := os.WriteFile(filepath.Join(gitforestDir, ".gitignore"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureExcluded(gitforestDir); err != nil {
		t.Fatalf("EnsureExcluded failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(gitforestDir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != custom {
		t.Errorf("expected existing pool root .gitignore to be preserved, got: %s", data)
	}
}
