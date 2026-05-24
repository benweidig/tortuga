package discovery_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/benweidig/tortuga/internal/discovery"
)

// initRepo creates a real git repo in dir with one commit.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	must(t, run(dir, "git", "init"))
	must(t, run(dir, "git", "config", "user.email", "test@test.com"))
	must(t, run(dir, "git", "config", "user.name", "Test"))
	must(t, os.WriteFile(filepath.Join(dir, "README"), []byte("hello"), 0644))
	must(t, run(dir, "git", "add", "."))
	must(t, run(dir, "git", "commit", "-m", "init"))
}

func run(dir string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	return cmd.Run()
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// --- Strategy 1: root is a repo ---

func TestFindRepos_RootIsRepo(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].Path != root {
		t.Errorf("expected Path=%q, got %q", root, repos[0].Path)
	}
}

func TestFindRepos_Strategy1TakesPriority(t *testing.T) {
	// Root is a repo AND has child repos — strategy 1 should win.
	root := t.TempDir()
	initRepo(t, root)

	child := filepath.Join(root, "child")
	must(t, os.Mkdir(child, 0755))
	initRepo(t, child)

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Path != root {
		t.Errorf("expected only root repo, got %v", repos)
	}
}

// --- Strategy 2: children are repos ---

func TestFindRepos_ChildRepos(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "aaa")
	b := filepath.Join(root, "bbb")
	must(t, os.Mkdir(a, 0755))
	must(t, os.Mkdir(b, 0755))
	initRepo(t, a)
	initRepo(t, b)

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	// ReadDir returns entries alphabetically
	if repos[0].Name != "aaa" || repos[1].Name != "bbb" {
		t.Errorf("unexpected order: %v", repos)
	}
}

func TestFindRepos_IgnoresNonRepoChildren(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "repo")
	plainDir := filepath.Join(root, "plain")
	must(t, os.Mkdir(repoDir, 0755))
	must(t, os.Mkdir(plainDir, 0755))
	initRepo(t, repoDir)

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "repo" {
		t.Errorf("expected only repo child, got %v", repos)
	}
}

func TestFindRepos_DotGitAsFile(t *testing.T) {
	// .git as a regular file is valid (git worktree / submodule)
	root := t.TempDir()
	child := filepath.Join(root, "worktree")
	must(t, os.Mkdir(child, 0755))
	must(t, os.WriteFile(filepath.Join(child, ".git"), []byte("gitdir: ../real/.git"), 0644))

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "worktree" {
		t.Errorf("expected worktree child, got %v", repos)
	}
}

func TestFindRepos_Strategy2TakesPriorityOverAncestor(t *testing.T) {
	// Ancestor has a repo, but direct children also have repos — strategy 2 wins.
	ancestor := t.TempDir()
	initRepo(t, ancestor)

	root := filepath.Join(ancestor, "subdir")
	must(t, os.Mkdir(root, 0755))

	child := filepath.Join(root, "child")
	must(t, os.Mkdir(child, 0755))
	initRepo(t, child)

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Path != child {
		t.Errorf("expected child repo, got %v", repos)
	}
}

// --- Strategy 3: walk upward ---

func TestFindRepos_AncestorRepo(t *testing.T) {
	ancestor := t.TempDir()
	initRepo(t, ancestor)

	root := filepath.Join(ancestor, "subdir")
	must(t, os.Mkdir(root, 0755))

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Path != ancestor {
		t.Errorf("expected ancestor repo, got %v", repos)
	}
}

func TestFindRepos_AncestorMultipleLevels(t *testing.T) {
	ancestor := t.TempDir()
	initRepo(t, ancestor)

	deep := filepath.Join(ancestor, "a", "b", "c")
	must(t, os.MkdirAll(deep, 0755))

	repos, err := discovery.FindRepos(deep, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Path != ancestor {
		t.Errorf("expected ancestor repo, got %v", repos)
	}
}

// --- Shared ---

func TestFindRepos_NoRepos(t *testing.T) {
	root := t.TempDir()

	_, err := discovery.FindRepos(root, false)
	if err != discovery.ErrNoRepos {
		t.Errorf("expected ErrNoRepos, got %v", err)
	}
}

func TestFindRepos_BranchName(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)
	must(t, run(root, "git", "checkout", "-b", "my-feature"))

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if repos[0].Branch != "my-feature" {
		t.Errorf("expected branch=my-feature, got %q", repos[0].Branch)
	}
}

func TestFindRepos_DetachedHEAD(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)

	// Get the current commit SHA and detach HEAD
	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	out, err := cmd.Output()
	must(t, err)
	sha := string(out[:len(out)-1]) // trim newline
	must(t, run(root, "git", "checkout", sha))

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if repos[0].Branch != "(detached)" {
		t.Errorf("expected (detached), got %q", repos[0].Branch)
	}
}

// --- .tortugaignore ---

func TestFindRepos_IgnoreFile_Strategy1(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)
	must(t, os.WriteFile(filepath.Join(root, ".tortugaignore"), nil, 0644))

	_, err := discovery.FindRepos(root, false)
	if err != discovery.ErrNoRepos {
		t.Errorf("expected ErrNoRepos for ignored root, got %v", err)
	}
}

func TestFindRepos_IgnoreFile_Strategy2_Some(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "aaa")
	b := filepath.Join(root, "bbb")
	must(t, os.Mkdir(a, 0755))
	must(t, os.Mkdir(b, 0755))
	initRepo(t, a)
	initRepo(t, b)
	must(t, os.WriteFile(filepath.Join(b, ".tortugaignore"), nil, 0644))

	repos, err := discovery.FindRepos(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "aaa" {
		t.Errorf("expected only aaa, got %v", repos)
	}
}

func TestFindRepos_IgnoreFile_Strategy2_All(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "aaa")
	must(t, os.Mkdir(a, 0755))
	initRepo(t, a)
	must(t, os.WriteFile(filepath.Join(a, ".tortugaignore"), nil, 0644))

	_, err := discovery.FindRepos(root, false)
	if err != discovery.ErrNoRepos {
		t.Errorf("expected ErrNoRepos when all children ignored, got %v", err)
	}
}

func TestFindRepos_IgnoreFile_Strategy3(t *testing.T) {
	ancestor := t.TempDir()
	initRepo(t, ancestor)
	must(t, os.WriteFile(filepath.Join(ancestor, ".tortugaignore"), nil, 0644))

	root := filepath.Join(ancestor, "subdir")
	must(t, os.MkdirAll(root, 0755))

	_, err := discovery.FindRepos(root, false)
	if err != discovery.ErrNoRepos {
		t.Errorf("expected ErrNoRepos for ignored ancestor, got %v", err)
	}
}

func TestFindRepos_NoIgnores_OverridesIgnoreFile(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)
	must(t, os.WriteFile(filepath.Join(root, ".tortugaignore"), nil, 0644))

	repos, err := discovery.FindRepos(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Path != root {
		t.Errorf("expected root repo with noIgnores=true, got %v", repos)
	}
}
