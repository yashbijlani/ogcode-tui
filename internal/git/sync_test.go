package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@local", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s: %v", args, string(out), err)
	}
}

func writeCommit(t *testing.T, dir, file, content, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", msg)
}

// TestBranchSyncStatus_Behind sets up a local "remote" (bare repo) and two
// clones, advances the remote, and verifies BranchSyncStatus reports the branch
// as behind after its best-effort fetch — entirely offline (file:// remote).
func TestBranchSyncStatus_Behind(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	bare := filepath.Join(root, "origin.git")
	git(t, root, "init", "--quiet", "--bare", bare)

	// Author repo: seed main and push to the bare remote.
	author := filepath.Join(root, "author")
	git(t, root, "clone", "--quiet", bare, author)
	writeCommit(t, author, "README.md", "seed\n", "seed")
	git(t, author, "branch", "-M", "main")
	git(t, author, "push", "--quiet", "-u", "origin", "main")

	// Work repo: clone main (in sync), tracking origin/main.
	work := filepath.Join(root, "work")
	git(t, root, "clone", "--quiet", "-b", "main", bare, work)

	// In sync to start.
	st, err := BranchSyncStatus(context.Background(), work)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !st.IsRepo || !st.HasUpstream || st.Branch != "main" {
		t.Fatalf("unexpected initial status: %+v", st)
	}
	if st.Behind != 0 || st.Ahead != 0 {
		t.Fatalf("expected in-sync, got ahead=%d behind=%d", st.Ahead, st.Behind)
	}

	// Advance the remote by two commits from the author.
	writeCommit(t, author, "a.txt", "a\n", "a")
	writeCommit(t, author, "b.txt", "b\n", "b")
	git(t, author, "push", "--quiet", "origin", "main")

	// The work repo's best-effort fetch should now see it 2 behind.
	st, err = BranchSyncStatus(context.Background(), work)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !st.Fetched {
		t.Fatalf("expected the best-effort fetch to succeed, got fetchError=%q", st.FetchError)
	}
	if st.Behind != 2 || st.Ahead != 0 {
		t.Fatalf("expected behind=2 ahead=0, got behind=%d ahead=%d", st.Behind, st.Ahead)
	}
}

// TestBranchSyncStatus_NoUpstream verifies a branch with no tracking branch is
// reported as a repo without an upstream (rather than erroring).
func TestBranchSyncStatus_NoUpstream(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	git(t, dir, "init", "--quiet")
	writeCommit(t, dir, "README.md", "seed\n", "seed")

	st, err := BranchSyncStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !st.IsRepo {
		t.Fatal("expected isRepo=true")
	}
	if st.HasUpstream {
		t.Fatalf("expected no upstream, got %q", st.Upstream)
	}
}

// TestBranchSyncStatus_NotARepo verifies a plain directory is reported as not a repo.
func TestBranchSyncStatus_NotARepo(t *testing.T) {
	st, err := BranchSyncStatus(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.IsRepo {
		t.Fatal("expected isRepo=false for a non-git directory")
	}
}
