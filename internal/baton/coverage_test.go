package baton

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFromEnvironmentAndWorkingDirectory(t *testing.T) {
	harness := newHarness(t)
	t.Setenv("BATON_DIR", harness.app.BatonDir)
	discovered, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(discovered.BatonDir, harness.app.BatonDir) {
		t.Fatalf("discovered %s, want %s", discovered.BatonDir, harness.app.BatonDir)
	}

	t.Setenv("BATON_DIR", "")
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
		for _, command := range []string{"new-round", "prompt", "append", "gate", "status", "guide", "lint", "update"} {
			if !strings.Contains(output, command) {
				t.Fatalf("help output missing %s: %s", command, output)
			}
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
			t.Fatalf("baton %v unexpectedly passed", args)
		}
	}
	if binaryName("windows") != "baton.exe" || binaryName("linux") != "baton" {
		t.Fatal("platform binary names are incorrect")
	}
}

func TestCheckArtifactCommandAndValidationErrors(t *testing.T) {
	harness := newHarness(t)
	path := ".baton/runs/20260711-1000-check-PLAN.md"
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
		{"unsafe path", eventPlanned, ".baton/runs/../PLAN.md", "", ""},
		{"wrong name", eventPlanned, ".baton/runs/check-RUN-01.md", "", ""},
		{"missing file", eventPlanned, ".baton/runs/missing-PLAN.md", "", ""},
		{"placeholder", eventPlanned, ".baton/runs/placeholder-PLAN.md", "<todo>", ""},
		{"run todo", eventExecuted, ".baton/runs/todo-RUN-01.md", validRun("abcd", "01") + "TODO\n", "abcd"},
		{"task mismatch", eventPlanned, ".baton/runs/task-PLAN.md", validPlan("wrong"), "abcd"},
		{"invalid date", eventPlanned, ".baton/runs/date-PLAN.md", strings.Replace(validPlan("abcd"), "2026-07-11", "11/07/2026", 1), "abcd"},
		{"run status", eventExecuted, ".baton/runs/status-RUN-01.md", strings.Replace(validRun("abcd", "01"), "Status: complete", "Status: checkpoint", 1), "abcd"},
		{"plan status missing", eventPlanned, ".baton/runs/nostatus-PLAN.md", strings.Replace(validPlan("abcd"), "Status: complete\n", "", 1), "abcd"},
		{"plan status checkpoint", eventPlanned, ".baton/runs/checkpoint-PLAN.md", strings.Replace(validPlan("abcd"), "Status: complete", "Status: checkpoint", 1), "abcd"},
		{"review status missing", eventReview, ".baton/runs/nostatus-REVIEW-01.md", strings.Replace(validReview("abcd", "01"), "Status: complete\n", "", 1), "abcd"},
		{"review status checkpoint", eventReview, ".baton/runs/checkpoint-REVIEW-01.md", strings.Replace(validReview("abcd", "01"), "Status: complete", "Status: checkpoint", 1), "abcd"},
		{"review result", eventReview, ".baton/runs/result-REVIEW-01.md", strings.Replace(validReview("abcd", "01"), "ready-for-user-decision", "unknown", 1), "abcd"},
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
## Changes
## Validation
## Success Criteria Status
## Unresolved Risks
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
- Inspect the output.
- Rerun the flow.
- Confirm the artifact path.
## Evidence Reviewed
`
}

func validClose(taskID string) string {
	return `# CLOSE: Test
Task ID: ` + taskID + `
Date: 2026-07-11
Director: test
Approved By: User
## Acceptance
## Validation Summary
## Lesson Candidates
- none.
## Plan Deviations
- none.
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
		{"plan", "You are Planner for task `abcd`, PLAN phase"},
		{"review", "You are Planner for task `abcd`, REVIEW phase of round `02`"},
		{"exec", "You are Executor for task `abcd`, round `02`"},
	} {
		output := harness.run(t, "prompt", testCase.role, "--task-id", "abcd", "--key", "20260711-1000-test", "--run-number", "02")
		if !strings.Contains(output, testCase.wanted) {
			t.Fatalf("unexpected %s prompt: %s", testCase.role, output)
		}
		if !strings.Contains(output, "Director-owned commands (new-round, append, gate, feedback, update) are not yours to run") {
			t.Fatalf("%s prompt does not name the Director-owned commands: %s", testCase.role, output)
		}
	}
	if err := harness.fail("prompt", "unknown", "--task-id", "abcd", "--key", "key"); err == nil {
		t.Fatal("unknown prompt role passed")
	}
	if err := harness.fail("prompt", "plan", "--task-id", "abcd"); err == nil {
		t.Fatal("prompt accepted a missing key")
	}
}

func TestStatusNextCommand(t *testing.T) {
	harness := newHarness(t)
	taskID, key := parseRoundOutput(t, harness.run(t, "new-round", "next-command", "--summary", "Next command flow"))
	wantedPlan := "next_command: baton prompt plan --task-id " + taskID + " --key " + key
	if status := harness.run(t, "status", "--task-id", taskID); !strings.Contains(status, wantedPlan) {
		t.Fatalf("missing %q in %s", wantedPlan, status)
	}

	planPath := ".baton/runs/" + key + "-PLAN.md"
	writePlan(t, harness.app.ProjectDir, planPath, taskID)
	harness.run(t, "append", eventPlanned, "--task-id", taskID, "--role", "Planner", "--summary", "Plan complete", "--path", planPath)
	wanted := "next_command: baton prompt exec --task-id " + taskID + " --key " + key + " --run-number 01"
	if status := harness.run(t, "status", "--task-id", taskID); !strings.Contains(status, wanted) {
		t.Fatalf("missing %q in %s", wanted, status)
	}

	runPath := ".baton/runs/" + key + "-RUN-01.md"
	writeRun(t, harness.app.ProjectDir, runPath, taskID, "01")
	harness.run(t, "append", eventExecuted, "--task-id", taskID, "--role", "Executor", "--summary", "Run complete", "--path", runPath)
	wanted = "next_command: baton prompt review --task-id " + taskID + " --key " + key + " --run-number 01"
	if status := harness.run(t, "status", "--task-id", taskID); !strings.Contains(status, wanted) {
		t.Fatalf("missing %q in %s", wanted, status)
	}

	reviewPath := ".baton/runs/" + key + "-REVIEW-01.md"
	writeReview(t, harness.app.ProjectDir, reviewPath, taskID, "01", "blockers")
	harness.run(t, "append", eventReview, "--task-id", taskID, "--role", "Planner", "--summary", "Review complete", "--path", reviewPath)
	harness.run(t, "feedback", "--task-id", taskID, "--summary", "User feedback")
	wanted = "next_command: baton prompt exec --task-id " + taskID + " --key " + key + " --run-number 02"
	if status := harness.run(t, "status", "--task-id", taskID); !strings.Contains(status, wanted) {
		t.Fatalf("missing %q in %s", wanted, status)
	}
}

func TestArtifactRequiresProtocolSections(t *testing.T) {
	harness := newHarness(t)
	runPath := ".baton/runs/20260711-1001-sections-RUN-01.md"
	for _, heading := range []string{"## Changes", "## Unresolved Risks"} {
		writeArtifact(t, harness.app.ProjectDir, runPath, strings.Replace(validRun("sect", "01"), heading+"\n", "", 1))
		if err := harness.fail("check-artifact", eventExecuted, runPath, "sect"); err == nil {
			t.Fatalf("check-artifact accepted a RUN without %s", heading)
		}
	}
	writeArtifact(t, harness.app.ProjectDir, runPath, validRun("sect", "01"))
	harness.run(t, "check-artifact", eventExecuted, runPath, "sect")

	reviewPath := ".baton/runs/20260711-1001-sections-REVIEW-01.md"
	for _, checks := range []string{"- One.\n- Two.\n", "- One.\n- Two.\n- Three.\n- Four.\n- Five.\n- Six.\n"} {
		body := strings.Replace(validReview("sect", "01"),
			"- Inspect the output.\n- Rerun the flow.\n- Confirm the artifact path.\n", checks, 1)
		writeArtifact(t, harness.app.ProjectDir, reviewPath, body)
		if err := harness.fail("check-artifact", eventReview, reviewPath, "sect"); err == nil {
			t.Fatalf("check-artifact accepted a REVIEW with these checks: %s", checks)
		}
	}
	writeArtifact(t, harness.app.ProjectDir, reviewPath, validReview("sect", "01"))
	harness.run(t, "check-artifact", eventReview, reviewPath, "sect")

	closePath := ".baton/runs/20260711-1001-sections-CLOSE.md"
	writeArtifact(t, harness.app.ProjectDir, closePath, strings.Replace(validClose("sect"), "## Plan Deviations\n", "", 1))
	if err := harness.fail("check-artifact", eventClose, closePath, "sect"); err == nil {
		t.Fatal("check-artifact accepted a CLOSE without ## Plan Deviations")
	}
	writeArtifact(t, harness.app.ProjectDir, closePath, strings.Replace(validClose("sect"), "- none.\n", "", 1))
	if err := harness.fail("check-artifact", eventClose, closePath, "sect"); err == nil {
		t.Fatal("check-artifact accepted a CLOSE with an empty Plan Deviations section")
	}
	writeArtifact(t, harness.app.ProjectDir, closePath, validClose("sect"))
	harness.run(t, "check-artifact", eventClose, closePath, "sect")
}

func TestStatusListsOpenTasks(t *testing.T) {
	harness := newHarness(t)
	first, _ := parseRoundOutput(t, harness.run(t, "new-round", "open-one", "--summary", "Open one"))
	second, _ := parseRoundOutput(t, harness.run(t, "new-round", "open-two", "--summary", "Open two"))

	plain := harness.run(t, "status")
	if strings.Contains(plain, first) {
		t.Fatalf("status without --open should show one task: %s", plain)
	}
	output := harness.run(t, "status", "--open")
	for _, taskID := range []string{first, second} {
		if !strings.Contains(output, "open_task: "+taskID) {
			t.Fatalf("status --open missing %s: %s", taskID, output)
		}
	}
	if !strings.Contains(output, "open_tasks: 2") {
		t.Fatalf("unexpected open count: %s", output)
	}
	if err := harness.fail("status", "--open", "--task-id", first); err == nil {
		t.Fatal("status accepted --open with --task-id")
	}
}
