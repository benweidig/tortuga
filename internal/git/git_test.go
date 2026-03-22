package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/benweidig/tortuga/internal/git"
	"github.com/benweidig/tortuga/internal/model"
)

// initRepo creates a git repo in dir with one initial commit.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	must(t, run(dir, "git", "init"))
	must(t, run(dir, "git", "config", "user.email", "test@test.com"))
	must(t, run(dir, "git", "config", "user.name", "Test"))
	must(t, os.WriteFile(filepath.Join(dir, "README"), []byte("hello"), 0644))
	must(t, run(dir, "git", "add", "."))
	must(t, run(dir, "git", "commit", "-m", "init"))
}

// initBareRepo creates a bare git repo and clones it into cloneDir,
// returning the path to the clone (which has the bare repo as its remote).
func initBareRepo(t *testing.T) (bareDir, cloneDir string) {
	t.Helper()
	bareDir = t.TempDir()
	must(t, run(bareDir, "git", "init", "--bare"))

	cloneDir = t.TempDir()
	must(t, run(cloneDir, "git", "clone", bareDir, "."))
	must(t, run(cloneDir, "git", "config", "user.email", "test@test.com"))
	must(t, run(cloneDir, "git", "config", "user.name", "Test"))

	// Create initial commit and push so the remote is not empty
	must(t, os.WriteFile(filepath.Join(cloneDir, "README"), []byte("hello"), 0644))
	must(t, run(cloneDir, "git", "add", "."))
	must(t, run(cloneDir, "git", "commit", "-m", "init"))
	must(t, run(cloneDir, "git", "push", "origin", "HEAD"))
	return bareDir, cloneDir
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

// --- IsAvailable ---

func TestIsAvailable(t *testing.T) {
	if err := git.IsAvailable(); err != nil {
		t.Errorf("expected git to be available: %v", err)
	}
}

// --- GetStatus ---

func TestGetStatus_Clean(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	info, err := git.GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDirty() {
		t.Error("expected clean repo")
	}
	if info.ChangedFiles != 0 || info.UnversionedFiles != 0 {
		t.Errorf("expected no changes, got changed=%d unversioned=%d", info.ChangedFiles, info.UnversionedFiles)
	}
}

func TestGetStatus_ModifiedTrackedFile(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	must(t, os.WriteFile(filepath.Join(dir, "README"), []byte("changed"), 0644))

	info, err := git.GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDirty() {
		t.Error("expected dirty repo")
	}
	if info.ChangedFiles != 1 {
		t.Errorf("expected 1 changed file, got %d", info.ChangedFiles)
	}
}

func TestGetStatus_UntrackedFile(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	must(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644))

	info, err := git.GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDirty() {
		t.Error("expected dirty repo")
	}
	if info.UnversionedFiles != 1 {
		t.Errorf("expected 1 unversioned file, got %d", info.UnversionedFiles)
	}
}

// --- AheadBehind ---

func TestAheadBehind_NoUpstream(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	ahead, behind, hasUpstream, err := git.AheadBehind(dir)
	if err != nil {
		t.Fatal(err)
	}
	if hasUpstream {
		t.Error("expected hasUpstream=false for repo without upstream")
	}
	if ahead != 0 || behind != 0 {
		t.Errorf("expected (0,0), got (%d,%d)", ahead, behind)
	}
}

func TestAheadBehind_Ahead(t *testing.T) {
	_, cloneDir := initBareRepo(t)

	// Make a local commit not pushed yet
	must(t, os.WriteFile(filepath.Join(cloneDir, "new.txt"), []byte("x"), 0644))
	must(t, run(cloneDir, "git", "add", "."))
	must(t, run(cloneDir, "git", "commit", "-m", "local"))

	ahead, behind, _, err := git.AheadBehind(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 1 || behind != 0 {
		t.Errorf("expected (1,0), got (%d,%d)", ahead, behind)
	}
}

func TestAheadBehind_Behind(t *testing.T) {
	bareDir, cloneDir := initBareRepo(t)

	// Make a commit in a second clone and push it
	clone2 := t.TempDir()
	must(t, run(clone2, "git", "clone", bareDir, "."))
	must(t, run(clone2, "git", "config", "user.email", "test@test.com"))
	must(t, run(clone2, "git", "config", "user.name", "Test"))
	must(t, os.WriteFile(filepath.Join(clone2, "other.txt"), []byte("y"), 0644))
	must(t, run(clone2, "git", "add", "."))
	must(t, run(clone2, "git", "commit", "-m", "remote"))
	must(t, run(clone2, "git", "push", "origin", "HEAD"))

	// Fetch in cloneDir so it knows about the remote commit
	must(t, run(cloneDir, "git", "fetch"))

	ahead, behind, _, err := git.AheadBehind(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 0 || behind != 1 {
		t.Errorf("expected (0,1), got (%d,%d)", ahead, behind)
	}
}

func TestAheadBehind_Diverged(t *testing.T) {
	bareDir, cloneDir := initBareRepo(t)

	// Remote gets a commit
	clone2 := t.TempDir()
	must(t, run(clone2, "git", "clone", bareDir, "."))
	must(t, run(clone2, "git", "config", "user.email", "test@test.com"))
	must(t, run(clone2, "git", "config", "user.name", "Test"))
	must(t, os.WriteFile(filepath.Join(clone2, "remote.txt"), []byte("r"), 0644))
	must(t, run(clone2, "git", "add", "."))
	must(t, run(clone2, "git", "commit", "-m", "remote"))
	must(t, run(clone2, "git", "push", "origin", "HEAD"))

	// cloneDir gets a different local commit
	must(t, os.WriteFile(filepath.Join(cloneDir, "local.txt"), []byte("l"), 0644))
	must(t, run(cloneDir, "git", "add", "."))
	must(t, run(cloneDir, "git", "commit", "-m", "local"))

	// Fetch so cloneDir knows about remote commit
	must(t, run(cloneDir, "git", "fetch"))

	ahead, behind, _, err := git.AheadBehind(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 1 || behind != 1 {
		t.Errorf("expected (1,1), got (%d,%d)", ahead, behind)
	}
}

// --- Stash / StashPop ---

func TestStash_StashPop(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	// Dirty the repo
	must(t, os.WriteFile(filepath.Join(dir, "README"), []byte("dirty"), 0644))

	if err := git.Stash(dir); err != nil {
		t.Fatalf("Stash: %v", err)
	}

	info, err := git.GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDirty() {
		t.Error("expected clean after stash")
	}

	if gitErr := git.StashPop(dir); gitErr != nil {
		t.Fatalf("StashPop: %v", gitErr)
	}

	info, err = git.GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDirty() {
		t.Error("expected dirty after stash pop")
	}
}

func TestStash_IncludesUntracked(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	// Only an untracked file
	must(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644))

	if err := git.Stash(dir); err != nil {
		t.Fatalf("Stash: %v", err)
	}

	info, err := git.GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDirty() {
		t.Error("expected clean after stash --include-untracked")
	}
}

// --- Rebase ---

func TestRebase_FastForward(t *testing.T) {
	bareDir, cloneDir := initBareRepo(t)

	// Push a new commit from clone2
	clone2 := t.TempDir()
	must(t, run(clone2, "git", "clone", bareDir, "."))
	must(t, run(clone2, "git", "config", "user.email", "test@test.com"))
	must(t, run(clone2, "git", "config", "user.name", "Test"))
	must(t, os.WriteFile(filepath.Join(clone2, "extra.txt"), []byte("e"), 0644))
	must(t, run(clone2, "git", "add", "."))
	must(t, run(clone2, "git", "commit", "-m", "extra"))
	must(t, run(clone2, "git", "push", "origin", "HEAD"))

	// Fetch so @{u} is up to date (mirrors real Phase 1 → Phase 2 flow)
	must(t, run(cloneDir, "git", "fetch"))

	gitErr := git.Rebase(cloneDir)
	if gitErr != nil {
		t.Fatalf("Rebase: %v", gitErr)
	}

	_, behind, _, err := git.AheadBehind(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if behind != 0 {
		t.Errorf("expected behind=0 after rebase, got %d", behind)
	}
}

func TestRebase_Conflict(t *testing.T) {
	bareDir, cloneDir := initBareRepo(t)

	// Remote edits the same file as cloneDir — forces a rebase conflict
	clone2 := t.TempDir()
	must(t, run(clone2, "git", "clone", bareDir, "."))
	must(t, run(clone2, "git", "config", "user.email", "test@test.com"))
	must(t, run(clone2, "git", "config", "user.name", "Test"))
	must(t, os.WriteFile(filepath.Join(clone2, "README"), []byte("remote"), 0644))
	must(t, run(clone2, "git", "add", "."))
	must(t, run(clone2, "git", "commit", "-m", "remote"))
	must(t, run(clone2, "git", "push", "origin", "HEAD"))

	// cloneDir edits the same file differently
	must(t, os.WriteFile(filepath.Join(cloneDir, "README"), []byte("local"), 0644))
	must(t, run(cloneDir, "git", "add", "."))
	must(t, run(cloneDir, "git", "commit", "-m", "local"))

	// Fetch so @{u} is up to date (mirrors real Phase 1 → Phase 2 flow)
	must(t, run(cloneDir, "git", "fetch"))

	gitErr := git.Rebase(cloneDir)
	if gitErr == nil {
		t.Fatal("expected Rebase to fail on conflicting changes")
	}
	// Clean up the rebase state so t.TempDir() cleanup succeeds
	_ = run(cloneDir, "git", "rebase", "--abort")
}

// --- Push ---

func TestPush(t *testing.T) {
	_, cloneDir := initBareRepo(t)

	must(t, os.WriteFile(filepath.Join(cloneDir, "new.txt"), []byte("n"), 0644))
	must(t, run(cloneDir, "git", "add", "."))
	must(t, run(cloneDir, "git", "commit", "-m", "new"))

	gitErr := git.Push(cloneDir)
	if gitErr != nil {
		t.Fatalf("Push: %v", gitErr)
	}

	ahead, _, _, err := git.AheadBehind(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 0 {
		t.Errorf("expected ahead=0 after push, got %d", ahead)
	}
}

// --- Fetch ---

func TestFetch(t *testing.T) {
	bareDir, cloneDir := initBareRepo(t)

	// Push a new commit from clone2
	clone2 := t.TempDir()
	must(t, run(clone2, "git", "clone", bareDir, "."))
	must(t, run(clone2, "git", "config", "user.email", "test@test.com"))
	must(t, run(clone2, "git", "config", "user.name", "Test"))
	must(t, os.WriteFile(filepath.Join(clone2, "extra.txt"), []byte("e"), 0644))
	must(t, run(clone2, "git", "add", "."))
	must(t, run(clone2, "git", "commit", "-m", "extra"))
	must(t, run(clone2, "git", "push", "origin", "HEAD"))

	// Before fetch, cloneDir should not see the new commit
	_, behindBefore, _, err := git.AheadBehind(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if behindBefore != 0 {
		t.Errorf("expected behind=0 before fetch, got %d", behindBefore)
	}

	gitErr := git.Fetch(cloneDir)
	if gitErr != nil {
		t.Fatalf("Fetch: %v", gitErr)
	}

	// After fetch, cloneDir should be behind by 1
	_, behindAfter, _, err := git.AheadBehind(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if behindAfter != 1 {
		t.Errorf("expected behind=1 after fetch, got %d", behindAfter)
	}
}

// --- ClassifyError ---

func TestClassifyError_Auth(t *testing.T) {
	e := git.ClassifyError("Authentication failed for 'https://github.com/foo'", "fetch")
	if e.Kind != model.ErrAuthFailed {
		t.Errorf("expected ErrAuthFailed, got %v", e.Kind)
	}
}

func TestClassifyError_NoNetwork(t *testing.T) {
	e := git.ClassifyError("fatal: Could not resolve host: github.com", "fetch")
	if e.Kind != model.ErrNoNetwork {
		t.Errorf("expected ErrNoNetwork, got %v", e.Kind)
	}
}

func TestClassifyError_PushRejected(t *testing.T) {
	e := git.ClassifyError("! [rejected]  main -> main (fetch first)\nUpdates were rejected", "push")
	if e.Kind != model.ErrPushRejected {
		t.Errorf("expected ErrPushRejected, got %v", e.Kind)
	}
}

func TestClassifyError_StashConflict(t *testing.T) {
	e := git.ClassifyError("CONFLICT (content): Merge conflict in README", "stash pop")
	if e.Kind != model.ErrStashConflict {
		t.Errorf("expected ErrStashConflict, got %v", e.Kind)
	}
}

func TestClassifyError_StashConflictWrongStep(t *testing.T) {
	// CONFLICT in a non-stash-pop context should not classify as ErrStashConflict
	e := git.ClassifyError("CONFLICT (content): Merge conflict in README", "pull")
	if e.Kind == model.ErrStashConflict {
		t.Error("should not classify as ErrStashConflict for pull step")
	}
}

func TestClassifyError_Unknown(t *testing.T) {
	e := git.ClassifyError("some completely unknown error message", "fetch")
	if e.Kind != model.ErrUnknown {
		t.Errorf("expected ErrUnknown, got %v", e.Kind)
	}
	if e.Message != "some completely unknown error message" {
		t.Errorf("unexpected message: %q", e.Message)
	}
}
