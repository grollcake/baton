package memento

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStatusReportsPendingArtifacts(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	taskID, key := parseRoundOutput(t, harness.run(t, "new-round", "pending-flow", "--summary", "Pending flow"))
	planPath := ".memento/runs/" + key + "-PLAN.md"
	runPath := ".memento/runs/" + key + "-RUN-01.md"
	reviewPath := ".memento/runs/" + key + "-REVIEW-01.md"
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
	writePlan(t, project, ".memento/runs/20260711-0900-other-PLAN.md", "zzzz")
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

	harness.run(t, "feedback", "--task-id", taskID, "--summary", "Defect found")
	runFile := filepath.Join(project, filepath.FromSlash(runPath))
	feedbackAt := harness.app.Now()
	if err := os.Chtimes(runFile, feedbackAt, feedbackAt.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	expect("stale run after feedback", "")
	if err := os.Chtimes(runFile, feedbackAt, feedbackAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	expect("rewritten run after feedback", runPath+" (complete, not appended as EXECUTED)")
}
