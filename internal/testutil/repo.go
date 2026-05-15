package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo is a temporary git repository for use in tests.
type Repo struct {
	Dir string
	t   *testing.T
}

// NewRepo creates a fresh git repository in a temp directory.
// The directory is automatically removed when the test ends.
func NewRepo(t *testing.T) *Repo {
	t.Helper()
	dir := t.TempDir()
	r := &Repo{Dir: dir, t: t}
	r.Git("init")
	r.Git("config", "user.email", "test@git-next.test")
	r.Git("config", "user.name", "git-next test")
	r.Git("config", "commit.gpgsign", "false")
	return r
}

// Git runs a git command in the repo directory. Fails the test on error.
func (r *Repo) Git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// GitMayFail runs a git command that is expected to sometimes fail.
func (r *Repo) GitMayFail(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// WriteFile writes content to a file in the repository.
func (r *Repo) WriteFile(name, content string) {
	r.t.Helper()
	path := filepath.Join(r.Dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		r.t.Fatal(err)
	}
}

// AddAndCommit writes a file, stages it, and creates a commit.
func (r *Repo) AddAndCommit(filename, content, msg string) {
	r.t.Helper()
	r.WriteFile(filename, content)
	r.Git("add", filename)
	r.Git("commit", "-m", msg)
}

// Stage writes a file and adds it to the index without committing.
func (r *Repo) Stage(filename, content string) {
	r.t.Helper()
	r.WriteFile(filename, content)
	r.Git("add", filename)
}

// ChDir changes the test process working directory to the repo.
// Go's t.Chdir automatically restores the original directory when the test ends.
func (r *Repo) ChDir() {
	r.t.Helper()
	r.t.Chdir(r.Dir)
}

// InitialCommit creates an initial empty commit so the repo has a HEAD.
func (r *Repo) InitialCommit() {
	r.t.Helper()
	r.AddAndCommit("README.md", "# test\n", "initial commit")
}

// CloneFrom creates a second repo that is a clone of r (used as the "remote").
func CloneFrom(t *testing.T, origin *Repo) *Repo {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "clone", origin.Dir, dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	clone := &Repo{Dir: dir, t: t}
	clone.Git("config", "user.email", "test@git-next.test")
	clone.Git("config", "user.name", "git-next test")
	clone.Git("config", "commit.gpgsign", "false")
	return clone
}
