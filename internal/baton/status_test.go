package baton

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusReportsPendingArtifacts(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	taskID, key := parseRoundOutput(t, harness.run(t, "new-round", "pending-flow", "--summary", "Pending flow"))
	planPath := ".baton/runs/" + key + "-PLAN.md"
	runPath := ".baton/runs/" + key + "-RUN-01.md"
	reviewPath := ".baton/runs/" + key + "-REVIEW-01.md"
	pending := func() string {
		t.Helper()
		var lines []string
		for _, line := range strings.Split(harness.run(t, "status"), "\n") {
			if strings.HasPrefix(line, "pending_artifact: ") {
				lines = append(lines, line)
			}
		}
		return strings.Join(lines, "\n")
	}
	expect := func(stage, wanted string) {
		t.Helper()
		got := pending()
		if wanted == "" && got != "" || wanted != "" && (strings.Count(got, "\n") != 0 || !strings.Contains(got, wanted)) {
			t.Fatalf("%s: unexpected pending artifacts: %q", stage, got)
		}
	}

	expect("no plan yet", "")
	writePlan(t, project, ".baton/runs/20260711-0900-other-PLAN.md", "zzzz")
	writeArtifact(t, project, planPath, strings.Replace(validPlan(taskID), "Status: complete", "Status: checkpoint", 1))
	expect("checkpoint plan", "")
	writePlan(t, project, planPath, taskID)
	expect("complete plan", planPath+" (complete, not appended as PLANNED)")
	harness.run(t, "append", eventPlanned, "--task-id", taskID, "--role", "Planner", "--summary", "Plan complete", "--path", planPath)

	expect("no run yet", "")
	writeRun(t, project, runPath, taskID, "01")
	expect("complete run", runPath+" (complete, not appended as EXECUTED)")
	harness.run(t, "append", eventExecuted, "--task-id", taskID, "--role", "Executor", "--summary", "Run complete", "--path", runPath)

	expect("no review yet", "")
	writeReview(t, project, reviewPath, taskID, "01", "ready-for-user-decision")
	expect("complete review", reviewPath+" (complete, not appended as REVIEW)")
	harness.run(t, "append", eventReview, "--task-id", taskID, "--role", "Planner", "--summary", "Review complete", "--path", reviewPath)
	expect("logged run after review", "")

	// Feedback opens the next round; the logged RUN-01 stays logged.
	harness.run(t, "feedback", "--task-id", taskID, "--summary", "Defect found")
	expect("logged run after feedback", "")
	nextRunPath := ".baton/runs/" + key + "-RUN-02.md"
	writeRun(t, project, nextRunPath, taskID, "02")
	expect("next run after feedback", nextRunPath+" (complete, not appended as EXECUTED)")
}

// statusThroughReview drives new-round -> PLANNED -> EXECUTED(RUN-01) and logs
// a REVIEW-01 with the given result, returning the task id, key, and the
// `status` output right after that REVIEW is logged. When removeAfterLog is
// true, the REVIEW artifact is deleted from disk after being logged (append
// requires the file to exist), simulating a logged REVIEW record whose file
// later went missing.
func statusThroughReview(t *testing.T, slug, result string, removeAfterLog bool) (taskID, key, status string) {
	t.Helper()
	harness := newHarness(t)
	project := harness.app.ProjectDir
	taskID, key = parseRoundOutput(t, harness.run(t, "new-round", slug, "--summary", "Review flow"))

	planPath := ".baton/runs/" + key + "-PLAN.md"
	writePlan(t, project, planPath, taskID)
	harness.run(t, "append", eventPlanned, "--task-id", taskID, "--role", "Planner", "--summary", "Plan complete", "--path", planPath)

	runPath := ".baton/runs/" + key + "-RUN-01.md"
	writeRun(t, project, runPath, taskID, "01")
	harness.run(t, "append", eventExecuted, "--task-id", taskID, "--role", "Executor", "--summary", "Run complete", "--path", runPath)

	reviewPath := ".baton/runs/" + key + "-REVIEW-01.md"
	writeReview(t, project, reviewPath, taskID, "01", result)
	harness.run(t, "append", eventReview, "--task-id", taskID, "--role", "Planner", "--summary", "Review complete", "--path", reviewPath)
	if removeAfterLog {
		if err := os.Remove(filepath.Join(project, filepath.FromSlash(reviewPath))); err != nil {
			t.Fatal(err)
		}
	}
	status = harness.run(t, "status", "--task-id", taskID)
	return taskID, key, status
}

func TestStatusAfterReviewBlockersOffersNextExecutedRound(t *testing.T) {
	taskID, key, status := statusThroughReview(t, "review-blockers", "blockers", false)
	if !strings.Contains(status, "next_gate: EXECUTED (delegate Executor)") {
		t.Fatalf("blockers REVIEW should point at another EXECUTED round: %s", status)
	}
	wanted := "next_command: baton prompt exec --task-id " + taskID + " --key " + key + " --run-number 02"
	if !strings.Contains(status, wanted) {
		t.Fatalf("missing %q in %s", wanted, status)
	}
}

func TestStatusAfterReviewReadyOffersUserApproval(t *testing.T) {
	_, _, status := statusThroughReview(t, "review-ready", "ready-for-user-decision", false)
	if !strings.Contains(status, "next_gate: user approval -> CLOSE") {
		t.Fatalf("ready REVIEW should point at user approval: %s", status)
	}
	for _, line := range strings.Split(status, "\n") {
		if strings.HasPrefix(line, "next_command:") {
			t.Fatalf("ready REVIEW should not offer a next_command: %s", status)
		}
	}
}

func TestStatusAfterMissingReviewArtifactOffersNextExecutedRound(t *testing.T) {
	// The REVIEW was logged as ready-for-user-decision, but its artifact file
	// is gone from disk by the time status runs (removed, moved, or never
	// synced). a.reviewResult(path) then hits its os.ReadFile error branch and
	// returns "", which must NOT be treated as ready-for-user-decision: status
	// must still fail closed toward another EXECUTED round.
	taskID, key, status := statusThroughReview(t, "review-missing", "ready-for-user-decision", true)
	if !strings.Contains(status, "next_gate: EXECUTED (delegate Executor)") {
		t.Fatalf("missing REVIEW artifact should point at another EXECUTED round: %s", status)
	}
	wanted := "next_command: baton prompt exec --task-id " + taskID + " --key " + key + " --run-number 02"
	if !strings.Contains(status, wanted) {
		t.Fatalf("missing %q in %s", wanted, status)
	}
}
