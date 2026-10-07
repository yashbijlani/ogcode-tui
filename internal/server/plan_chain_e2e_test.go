package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prasenjeet-symon/ogcode/internal/bus"
	"github.com/prasenjeet-symon/ogcode/internal/db"
	"github.com/prasenjeet-symon/ogcode/internal/git"
	"github.com/prasenjeet-symon/ogcode/internal/plan"
	"github.com/prasenjeet-symon/ogcode/internal/session"
	"github.com/prasenjeet-symon/ogcode/internal/task"
)

// --- git-repo helpers ---------------------------------------------------------

func runGitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=test", "-c", "user.email=test@local"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s: %v", args, string(out), err)
	}
	return string(out)
}

// initRepoWithCommit creates a throwaway git repo with one commit so worktrees
// can branch from HEAD, mirroring how ogcode operates on a real project.
func initRepoWithCommit(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	runGitTest(t, dir, "init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, dir, "add", "-A")
	runGitTest(t, dir, "commit", "-q", "-m", "seed")
	return dir
}

// commitFileInWorktree writes a file inside a task worktree and commits it on the
// task's own branch — the equivalent of a build agent doing and committing work.
func commitFileInWorktree(t *testing.T, worktreePath, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(worktreePath, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := git.CommitAllChanges(worktreePath, "task work: "+name); err != nil {
		t.Fatalf("commit in worktree: %v", err)
	}
}

// --- Fix #3: chain merge ordering --------------------------------------------

// TestChain_DependentSeesPriorWork replays the corrected sequence: a chain task
// is merged into the shared chain branch BEFORE the dependent task's worktree is
// created from that branch. The dependent must therefore see the predecessor's
// work. This is the invariant the autoCompleteTask reordering restores.
func TestChain_DependentSeesPriorWork(t *testing.T) {
	repo := initRepoWithCommit(t)
	chainBranch := "chain/pln_test-feature"

	if err := git.CreateChainBranch(repo, chainBranch); err != nil {
		t.Fatalf("create chain branch: %v", err)
	}

	// Task A: branch from chain, do work, then MERGE into chain (correct order).
	wtA, err := git.CreateTaskWorktree(repo, "tsk_A", "schema", chainBranch)
	if err != nil {
		t.Fatalf("worktree A: %v", err)
	}
	commitFileInWorktree(t, wtA.Path, "schema.go", "package db\n")
	if err := git.MergeTaskBranch(repo, chainBranch, wtA.BranchName, "Schema layer"); err != nil {
		t.Fatalf("merge A into chain: %v", err)
	}

	// Task B: branch from chain AFTER A is merged — it must contain A's file.
	wtB, err := git.CreateTaskWorktree(repo, "tsk_B", "backend", chainBranch)
	if err != nil {
		t.Fatalf("worktree B: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wtB.Path, "schema.go")); err != nil {
		t.Fatalf("dependent task B does NOT see predecessor A's work (schema.go missing): %v", err)
	}

	// B builds on top of A, then merges. The chain branch ends with both files.
	commitFileInWorktree(t, wtB.Path, "backend.go", "package backend\n")
	if err := git.MergeTaskBranch(repo, chainBranch, wtB.BranchName, "Backend logic"); err != nil {
		t.Fatalf("merge B into chain: %v", err)
	}

	files := runGitTest(t, repo, "ls-tree", "-r", "--name-only", chainBranch)
	for _, want := range []string{"schema.go", "backend.go", "README.md"} {
		if !strings.Contains(files, want) {
			t.Fatalf("chain branch missing %q after full chain; got:\n%s", want, files)
		}
	}
}

// TestChain_BuggyOrderingMissesPriorWork documents the bug the reordering fixes:
// if the dependent's worktree is created from the chain branch BEFORE the
// predecessor is merged (the old autoCompleteTask order), the dependent does not
// see the predecessor's work. This asserts the old order was genuinely broken.
func TestChain_BuggyOrderingMissesPriorWork(t *testing.T) {
	repo := initRepoWithCommit(t)
	chainBranch := "chain/pln_test-buggy"
	if err := git.CreateChainBranch(repo, chainBranch); err != nil {
		t.Fatalf("create chain branch: %v", err)
	}

	wtA, err := git.CreateTaskWorktree(repo, "tsk_A", "schema", chainBranch)
	if err != nil {
		t.Fatalf("worktree A: %v", err)
	}
	commitFileInWorktree(t, wtA.Path, "schema.go", "package db\n")

	// BUGGY ORDER: create B's worktree before merging A.
	wtB, err := git.CreateTaskWorktree(repo, "tsk_B", "backend", chainBranch)
	if err != nil {
		t.Fatalf("worktree B: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wtB.Path, "schema.go")); err == nil {
		t.Fatal("expected B to miss A's work under the buggy ordering, but schema.go was present")
	}
}

// --- Fixes #1/#2/#4: PR-outcome and chain-blocked recording -------------------

func newChainTestServer(t *testing.T) *Server {
	t.Helper()
	tmp := t.TempDir()
	pdb, err := db.Open(filepath.Join(tmp, "ogcode.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { pdb.Close() })
	return &Server{
		db:        pdb,
		dir:       tmp,
		bus:       bus.New(256),
		store:     session.NewStore(pdb),
		planStore: plan.NewStore(pdb),
		taskStore: task.NewStore(pdb),
	}
}

// makeChainPlan creates the session + (locked) plan a task must reference to
// satisfy the task→plan→session foreign keys.
func makeChainPlan(t *testing.T, s *Server, planID string) {
	t.Helper()
	sess := &session.Session{
		ID: session.NewSessionID(), ProjectID: s.dir, Directory: s.dir,
		Title: "Plan", SessionType: "plan",
		CreatedAt: session.Now(), UpdatedAt: session.Now(),
	}
	if err := s.store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}
	p := &plan.Plan{
		ID: planID, SessionID: string(sess.ID), ProjectID: s.dir, Directory: s.dir,
		Title: "Plan", Status: plan.StatusLocked,
		CreatedAt: plan.Now(), UpdatedAt: plan.Now(),
	}
	if err := s.planStore.Create(p); err != nil {
		t.Fatalf("create plan: %v", err)
	}
}

func makeChainTask(t *testing.T, s *Server, planID, id, title, chainBranch, status string) {
	t.Helper()
	now := task.Now()
	tk := &task.Task{
		ID: id, PlanID: planID, Title: title,
		Effort: task.EffortM, Complexity: task.ComplexityMedium,
		Status: status, ChainBranch: chainBranch,
		BranchName: "task/" + id, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.taskStore.Create(tk); err != nil {
		t.Fatalf("create task %s: %v", id, err)
	}
}

func drainTaskUpdated(ch <-chan bus.Event) int {
	n := 0
	for {
		select {
		case e := <-ch:
			if e.Type == "task.updated" {
				n++
			}
		default:
			return n
		}
	}
}

// TestRecordChainPROutcome_Success verifies a successful chain PR is stamped onto
// every task in the chain (URL + number) and that a task.updated event fires for
// each — so the UI surfaces the shared PR (fix #1).
func TestRecordChainPROutcome_Success(t *testing.T) {
	s := newChainTestServer(t)
	const planID, chain = "pln_1", "chain/pln_1-feat"
	makeChainPlan(t, s, planID)
	makeChainTask(t, s, planID, "tsk_a", "A", chain, task.StatusCompleted)
	makeChainTask(t, s, planID, "tsk_b", "B", chain, task.StatusCompleted)

	chainTasks, _ := s.taskStore.ListByPlan(planID)
	events := s.bus.SubscribeAll()

	s.recordChainPROutcome(chainTasks, &git.PullRequest{URL: "https://github.com/x/y/pull/7", Number: 7}, "")

	if got := drainTaskUpdated(events); got != 2 {
		t.Fatalf("expected 2 task.updated events, got %d", got)
	}
	for _, id := range []string{"tsk_a", "tsk_b"} {
		fresh, _ := s.taskStore.Get(id)
		if fresh.PRURL != "https://github.com/x/y/pull/7" {
			t.Errorf("task %s PRURL not persisted, got %q", id, fresh.PRURL)
		}
		if fresh.PRNumber == nil || *fresh.PRNumber != 7 {
			t.Errorf("task %s PRNumber not persisted, got %v", id, fresh.PRNumber)
		}
		if fresh.PRError != "" {
			t.Errorf("task %s PRError should be cleared, got %q", id, fresh.PRError)
		}
	}
}

// TestRecordChainPROutcome_Failure verifies a failed/skipped chain PR records a
// visible reason on every task in the chain instead of failing silently (fix #2).
func TestRecordChainPROutcome_Failure(t *testing.T) {
	s := newChainTestServer(t)
	const planID, chain = "pln_2", "chain/pln_2-feat"
	makeChainPlan(t, s, planID)
	makeChainTask(t, s, planID, "tsk_c", "C", chain, task.StatusCompleted)
	makeChainTask(t, s, planID, "tsk_d", "D", chain, task.StatusCompleted)

	chainTasks, _ := s.taskStore.ListByPlan(planID)
	s.recordChainPROutcome(chainTasks, nil, "no remote configured — add a GitHub remote")

	for _, id := range []string{"tsk_c", "tsk_d"} {
		fresh, _ := s.taskStore.Get(id)
		if fresh.PRError == "" {
			t.Errorf("task %s should have a PRError recorded on failure", id)
		}
		if fresh.PRURL != "" {
			t.Errorf("task %s should have no PRURL on failure, got %q", id, fresh.PRURL)
		}
	}
}

// TestPlanStore_BaseBranchRoundTrip verifies the base_branch column is persisted
// and read back through the plan store (the schema/store change for targeting the
// active branch as the PR base).
func TestPlanStore_BaseBranchRoundTrip(t *testing.T) {
	s := newChainTestServer(t)
	makeChainPlan(t, s, "pln_bb")

	p, _ := s.planStore.Get("pln_bb")
	if p.BaseBranch != "" {
		t.Fatalf("new plan should have empty base branch, got %q", p.BaseBranch)
	}
	p.BaseBranch = "feature/my-active-branch"
	if err := s.planStore.Update(p); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := s.planStore.Get("pln_bb")
	if got.BaseBranch != "feature/my-active-branch" {
		t.Fatalf("base branch not persisted, got %q", got.BaseBranch)
	}
}

// TestTaskStore_ModelRoundTrip verifies the per-task model override persists and
// can be cleared through the task store (the schema/store change for choosing a
// model per task).
func TestTaskStore_ModelRoundTrip(t *testing.T) {
	s := newChainTestServer(t)
	makeChainPlan(t, s, "pln_m")

	now := task.Now()
	tk := &task.Task{
		ID: "tsk_m", PlanID: "pln_m", Title: "T",
		Effort: task.EffortM, Complexity: task.ComplexityMedium, Status: task.StatusPending,
		Model: "anthropic/claude-sonnet-4-6", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.taskStore.Create(tk); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, _ := s.taskStore.Get("tsk_m")
	if got.Model != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("model not persisted, got %q", got.Model)
	}

	// Clearing the override should round-trip as empty (inherit plan default).
	got.Model = ""
	got.UpdatedAt = task.Now()
	if err := s.taskStore.Update(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	got2, _ := s.taskStore.Get("tsk_m")
	if got2.Model != "" {
		t.Fatalf("model not cleared, got %q", got2.Model)
	}
}

// TestMarkChainBlocked verifies that when a task in a chain fails, the chain's
// already-completed (stranded) tasks get a visible "chain blocked" reason so the
// merged work isn't silently lost without a PR (fix #4).
func TestMarkChainBlocked(t *testing.T) {
	s := newChainTestServer(t)
	const planID, chain = "pln_3", "chain/pln_3-feat"
	makeChainPlan(t, s, planID)
	makeChainTask(t, s, planID, "tsk_done", "Already done", chain, task.StatusCompleted)
	makeChainTask(t, s, planID, "tsk_fail", "Failed one", chain, task.StatusFailed)
	// A standalone completed task in the same plan must be left untouched.
	makeChainTask(t, s, planID, "tsk_solo", "Solo", "", task.StatusCompleted)

	failed, _ := s.taskStore.Get("tsk_fail")
	s.markChainBlocked(failed)

	done, _ := s.taskStore.Get("tsk_done")
	if done.PRError == "" {
		t.Error("completed task in the blocked chain should have a 'chain blocked' reason")
	}
	solo, _ := s.taskStore.Get("tsk_solo")
	if solo.PRError != "" {
		t.Errorf("standalone task must not be marked blocked, got %q", solo.PRError)
	}
}
