package memento

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFromEnvironmentAndWorkingDirectory(t *testing.T) {
	harness := newHarness(t)
	t.Setenv("MEMENTO_DIR", harness.app.MementoDir)
	discovered, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(discovered.MementoDir, harness.app.MementoDir) {
		t.Fatalf("discovered %s, want %s", discovered.MementoDir, harness.app.MementoDir)
	}

	t.Setenv("MEMENTO_DIR", "")
	nested := filepath.Join(harness.app.ProjectDir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	discovered, err = Discover()
	if err != nil {
		t.Fatal(err)
	}
	discoveredProject, err := filepath.EvalSymlinks(discovered.ProjectDir)
	if err != nil {
		t.Fatal(err)
	}
	wantedProject, err := filepath.EvalSymlinks(harness.app.ProjectDir)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(discoveredProject, wantedProject) {
		t.Fatalf("discovered project %s, want %s", discovered.ProjectDir, harness.app.ProjectDir)
	}
}

func TestRunHelpUnknownAndArgumentErrors(t *testing.T) {
	harness := newHarness(t)
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		output := harness.run(t, args...)
		if !strings.Contains(output, "memento append") {
			t.Fatalf("help output missing commands: %s", output)
		}
	}
	for _, args := range [][]string{
		{},
		{"unknown"},
		{"append"},
		{"append", "INVALID", "--task-id", "abcd", "--role", "Director", "--summary", "x"},
		{"append", eventRequest, "--role", "Director", "--summary", "x"},
		{"append", eventRequest, "--task-id", "abcd", "--summary", "x"},
		{"append", eventRequest, "--task-id", "abcd", "--role", "Planner", "--summary", "x"},
		{"status", "extra"},
		{"feedback", "extra"},
		{"version", "extra"},
	} {
		if err := harness.fail(args...); err == nil {
			t.Fatalf("memento %v unexpectedly passed", args)
		}
	}
	if binaryName("windows") != "memento.exe" || binaryName("linux") != "memento" {
		t.Fatal("platform binary names are incorrect")
	}
}

func TestCheckArtifactCommandAndValidationErrors(t *testing.T) {
	harness := newHarness(t)
	path := ".memento/runs/20260711-1000-check-PLAN.md"
	writePlan(t, harness.app.ProjectDir, path, "abcd")
	harness.run(t, "check-artifact", eventPlanned, path, "abcd")
	if err := harness.fail("check-artifact", eventPlanned); err == nil {
		t.Fatal("check-artifact accepted missing arguments")
	}

	cases := []struct {
		name, event, path, content, task string
	}{
		{"unsupported event", eventRequest, path, "", ""},
		{"outside runs", eventPlanned, "PLAN.md", "", ""},
		{"unsafe path", eventPlanned, ".memento/runs/../PLAN.md", "", ""},
		{"wrong name", eventPlanned, ".memento/runs/check-RUN-01.md", "", ""},
		{"missing file", eventPlanned, ".memento/runs/missing-PLAN.md", "", ""},
		{"placeholder", eventPlanned, ".memento/runs/placeholder-PLAN.md", "<todo>", ""},
		{"run todo", eventExecuted, ".memento/runs/todo-RUN-01.md", validRun("abcd", "01") + "TODO\n", "abcd"},
		{"task mismatch", eventPlanned, ".memento/runs/task-PLAN.md", validPlan("wrong"), "abcd"},
		{"invalid date", eventPlanned, ".memento/runs/date-PLAN.md", strings.Replace(validPlan("abcd"), "2026-07-11", "11/07/2026", 1), "abcd"},
		{"run status", eventExecuted, ".memento/runs/status-RUN-01.md", strings.Replace(validRun("abcd", "01"), "Status: complete", "Status: checkpoint", 1), "abcd"},
		{"plan status missing", eventPlanned, ".memento/runs/nostatus-PLAN.md", strings.Replace(validPlan("abcd"), "Status: complete\n", "", 1), "abcd"},
		{"plan status checkpoint", eventPlanned, ".memento/runs/checkpoint-PLAN.md", strings.Replace(validPlan("abcd"), "Status: complete", "Status: checkpoint", 1), "abcd"},
		{"review status missing", eventReview, ".memento/runs/nostatus-REVIEW-01.md", strings.Replace(validReview("abcd", "01"), "Status: complete\n", "", 1), "abcd"},
		{"review status checkpoint", eventReview, ".memento/runs/checkpoint-REVIEW-01.md", strings.Replace(validReview("abcd", "01"), "Status: complete", "Status: checkpoint", 1), "abcd"},
		{"review result", eventReview, ".memento/runs/result-REVIEW-01.md", strings.Replace(validReview("abcd", "01"), "ready-for-user-decision", "unknown", 1), "abcd"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.content != "" {
				writeArtifact(t, harness.app.ProjectDir, testCase.path, testCase.content)
			}
			if err := harness.app.CheckArtifact(testCase.event, testCase.path, testCase.task); err == nil {
				t.Fatalf("%s unexpectedly passed", testCase.name)
			}
		})
	}
}

func validPlan(taskID string) string {
	return `# PLAN: Test
Task ID: ` + taskID + `
Date: 2026-07-11
Planner: test
Status: complete
## Director Brief
## Success Criteria
## Validation
`
}

func validRun(taskID, round string) string {
	return `# RUN-` + round + `: Test
Task ID: ` + taskID + `
Date: 2026-07-11
Executor: test
Status: complete
## Validation
## Success Criteria Status
`
}

func validReview(taskID, round string) string {
	return `# REVIEW-` + round + `: Test
Task ID: ` + taskID + `
Date: 2026-07-11
Planner: test
Status: complete
Result: ready-for-user-decision
## Suggested User Checks
## Evidence Reviewed
`
}

func TestStatusPromptAndGateBranches(t *testing.T) {
	harness := newHarness(t)
	status := harness.run(t, "status")
	if !strings.Contains(status, "no open task") || !strings.Contains(status, "not-a-git-repo") {
		t.Fatalf("unexpected closed status: %s", status)
	}
	harness.run(t, "append", eventRequest, "--task-id", "open", "--role", "Director", "--summary", "Open task")
	status = harness.run(t, "status")
	if !strings.Contains(status, "task_id: open") || !strings.Contains(status, "next_gate: PLANNED") {
		t.Fatalf("unexpected open status: %s", status)
	}
	if err := harness.fail("gate", "unknown", "--task-id", "open"); err == nil {
		t.Fatal("unknown gate passed")
	}
	if err := harness.fail("gate", "before-review", "--task-id", "open"); err == nil {
		t.Fatal("gate passed after the wrong event")
	}
	if err := harness.fail("gate"); err == nil {
		t.Fatal("gate accepted missing arguments")
	}

	for _, testCase := range []struct {
		role, wanted string
	}{
		{"plan", "Planner for round"},
		{"review", "Planner REVIEW phase"},
		{"exec", "Executor for assigned round"},
	} {
		output := harness.run(t, "prompt", testCase.role, "--task-id", "abcd", "--key", "20260711-1000-test", "--run-number", "02")
		if !strings.Contains(output, testCase.wanted) {
			t.Fatalf("unexpected %s prompt: %s", testCase.role, output)
		}
	}
	if err := harness.fail("prompt", "unknown", "--task-id", "abcd", "--key", "key"); err == nil {
		t.Fatal("unknown prompt role passed")
	}
	if err := harness.fail("prompt", "plan", "--task-id", "abcd"); err == nil {
		t.Fatal("prompt accepted a missing key")
	}
}
