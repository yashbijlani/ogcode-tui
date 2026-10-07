package server

import (
	"testing"

	"github.com/prasenjeet-symon/ogcode/internal/task"
)

// planID must be at least 8 chars because assignChainBranches uses planID[:8].
const testPlanID = "pln_0123456789"

// collectBranches returns a map of task ID -> assigned ChainBranch.
func collectBranches(tasks []*task.Task) map[string]string {
	out := make(map[string]string, len(tasks))
	for _, t := range tasks {
		out[t.ID] = t.ChainBranch
	}
	return out
}

// assertOneSharedBranch fails unless every listed task has the same non-empty
// ChainBranch.
func assertOneSharedBranch(t *testing.T, tasks []*task.Task, ids ...string) {
	t.Helper()
	branches := collectBranches(tasks)
	var want string
	for _, id := range ids {
		got := branches[id]
		if got == "" {
			t.Fatalf("task %s expected a chain branch, got empty", id)
		}
		if want == "" {
			want = got
		} else if got != want {
			t.Fatalf("tasks expected to share one branch, but %s=%q != %q", id, got, want)
		}
	}
}

// TestAssignChainBranches_TopologicalOrder is the common case: each task's
// dependency points to an earlier slice index. All three must share one branch.
func TestAssignChainBranches_TopologicalOrder(t *testing.T) {
	a := &task.Task{ID: "a", Title: "Schema layer"}
	b := &task.Task{ID: "b", Title: "Backend logic", Dependencies: []string{"a"}}
	c := &task.Task{ID: "c", Title: "Frontend", Dependencies: []string{"b"}}
	tasks := []*task.Task{a, b, c}

	assignChainBranches(tasks, testPlanID)

	assertOneSharedBranch(t, tasks, "a", "b", "c")
}

// TestAssignChainBranches_ForwardReference is the regression case for the
// order-dependence bug: the slice is ordered so that a child appears *before*
// its parent (a dependency pointing to a later slice index). Previously this
// split the single chain across two different branches; all three must now share
// the root's branch.
func TestAssignChainBranches_ForwardReference(t *testing.T) {
	// Chain is x2 -> x1 -> x0 (x0 depends on x1 depends on x2), but the slice is
	// ordered x0, x1, x2 so children are processed before their parents.
	x0 := &task.Task{ID: "x0", Title: "Frontend", Dependencies: []string{"x1"}}
	x1 := &task.Task{ID: "x1", Title: "Backend logic", Dependencies: []string{"x2"}}
	x2 := &task.Task{ID: "x2", Title: "Schema layer"}
	tasks := []*task.Task{x0, x1, x2}

	assignChainBranches(tasks, testPlanID)

	assertOneSharedBranch(t, tasks, "x0", "x1", "x2")

	// The shared branch must be named after the true chain root (x2, the task
	// with no dependency), not an arbitrary mid-chain task.
	if got := x2.ChainBranch; got != "chain/pln_0123-schema-layer" {
		t.Fatalf("chain branch should be named after the root task; got %q", got)
	}
}

// TestAssignChainBranches_StandaloneTask verifies a task with no dependency and
// no dependent stays standalone (no branch), while a separate chain is unaffected.
func TestAssignChainBranches_StandaloneTask(t *testing.T) {
	solo := &task.Task{ID: "solo", Title: "Independent fix"}
	a := &task.Task{ID: "a", Title: "Root"}
	b := &task.Task{ID: "b", Title: "Child", Dependencies: []string{"a"}}
	tasks := []*task.Task{solo, a, b}

	assignChainBranches(tasks, testPlanID)

	if solo.ChainBranch != "" {
		t.Fatalf("standalone task should have no chain branch, got %q", solo.ChainBranch)
	}
	assertOneSharedBranch(t, tasks, "a", "b")
}
