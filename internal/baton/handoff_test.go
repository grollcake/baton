package baton

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// filledHandoff returns a handoff record with every placeholder filled in
// except Date: and Log Line:, which are left exactly as the template ships
// them so a test can assert sealHandoff stamps them.
func filledHandoff() string {
	return `# HANDOFF

Date: <YYYY-MM-DD, stamped by ` + "`baton handoff`" + `>
Log Line: <last BATON-LOG.txt line number, stamped by ` + "`baton handoff`" + `>

Do not copy anything already recorded in ` + "`BATON-LOG.txt`" + `, ` + "`.baton/runs/`" + `,
` + "`GUIDANCE.md`" + `, ` + "`lesson-learned/`" + `, or ` + "`baton models get`" + ` into this file; record
here only what dies with this session.

## Branch Strategy

- Working on a feature branch, not yet merged.

## Unrecorded Decisions

- User agreed to skip the changelog for this round.

## Delegates

<filled automatically by ` + "`baton handoff`" + ` from BATON-LOG.txt and the run
artifacts on disk; do not edit this section by hand>

## Next Intent

- Resume by awaiting RUN-02 and appending EXECUTED.
`
}

func writeHandoff(t *testing.T, project, content string) {
	t.Helper()
	path := filepath.Join(project, ".baton", "HANDOFF.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHandoffSealRefusesWhenAbsent(t *testing.T) {
	harness := newHarness(t)
	err := harness.fail("handoff")
	if err == nil {
		t.Fatal("expected seal to refuse when .baton/HANDOFF.md is absent")
	}
	if !strings.Contains(err.Error(), "templates/handoff.md") {
		t.Fatalf("refusal should name templates/handoff.md: %v", err)
	}
}

func TestHandoffSealRefusesIncompleteRecord(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"missing section", strings.Replace(filledHandoff(), "## Next Intent\n\n- Resume by awaiting RUN-02 and appending EXECUTED.\n", "", 1)},
		{"unfilled placeholder", strings.Replace(filledHandoff(), "Working on a feature branch, not yet merged.", "<branch approach>", 1)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newHarness(t)
			project := harness.app.ProjectDir
			writeHandoff(t, project, testCase.content)
			if err := harness.fail("handoff"); err == nil {
				t.Fatal("expected seal to refuse")
			}
			path := filepath.Join(project, ".baton", "HANDOFF.md")
			got := string(readFile(t, path))
			if got != testCase.content {
				t.Fatalf("refused seal must leave the file byte-identical:\nwant %q\ngot  %q", testCase.content, got)
			}
		})
	}
}

func TestHandoffSealStampsDateAndLogLine(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	writeHandoff(t, project, filledHandoff())
	harness.run(t, "handoff")

	path := filepath.Join(project, ".baton", "HANDOFF.md")
	got := string(readFile(t, path))

	wantDate := "Date: " + harness.app.Now().Format("2006-01-02")
	if !strings.Contains(got, wantDate) {
		t.Fatalf("missing stamped date %q in %q", wantDate, got)
	}
	if !strings.Contains(got, "Log Line: 2") {
		t.Fatalf("expected Log Line: 2 (the boot log has two records) in %q", got)
	}

	// Every line outside Date:, Log Line:, and the stamped ## Delegates body
	// must survive sealing unchanged; Branch Strategy, Unrecorded Decisions,
	// and Next Intent are Director's, not the binary's, to rewrite.
	for _, unchanged := range []string{
		"- Working on a feature branch, not yet merged.",
		"- User agreed to skip the changelog for this round.",
		"- Resume by awaiting RUN-02 and appending EXECUTED.",
	} {
		if !strings.Contains(got, unchanged) {
			t.Fatalf("expected %q to survive sealing unchanged, got %q", unchanged, got)
		}
	}
}

func TestStatusReportsSealedHandoffWithNoOpenTask(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	writeHandoff(t, project, filledHandoff())
	harness.run(t, "handoff")
	wantDate := "sealed: " + harness.app.Now().Format("2006-01-02")

	statusNoTask := harness.run(t, "status")
	if !strings.Contains(statusNoTask, "handoff: .baton/HANDOFF.md | "+wantDate) {
		t.Fatalf("status with no open task did not report the sealed handoff: %s", statusNoTask)
	}
}

func TestStatusReportsSealedHandoffWithOpenTask(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	taskID, _ := parseRoundOutput(t, harness.run(t, "new-round", "handoff-flow", "--summary", "Handoff flow"))
	writeHandoff(t, project, filledHandoff())
	harness.run(t, "handoff")
	wantDate := "sealed: " + harness.app.Now().Format("2006-01-02")

	statusTask := harness.run(t, "status", "--task-id", taskID)
	if !strings.Contains(statusTask, "handoff: .baton/HANDOFF.md | "+wantDate) {
		t.Fatalf("status --task-id did not report the sealed handoff: %s", statusTask)
	}

	statusOpen := harness.run(t, "status", "--open")
	if !strings.Contains(statusOpen, "handoff: .baton/HANDOFF.md | "+wantDate) {
		t.Fatalf("status --open did not report the sealed handoff: %s", statusOpen)
	}
}

func TestStatusReportsSupersededHandoff(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	writeHandoff(t, project, filledHandoff())
	harness.run(t, "handoff")

	taskID, _ := parseRoundOutput(t, harness.run(t, "new-round", "handoff-super", "--summary", "Handoff superseded"))
	status := harness.run(t, "status", "--task-id", taskID)
	if !strings.Contains(status, "handoff_superseded: .baton/HANDOFF.md | ") {
		t.Fatalf("expected handoff_superseded after another event was appended: %s", status)
	}
	if !strings.Contains(status, "1 event(s) appended since") {
		t.Fatalf("expected exactly 1 event appended since sealing: %s", status)
	}
	if strings.Contains(status, "handoff: .baton/HANDOFF.md") {
		t.Fatalf("a superseded handoff must not also print as live: %s", status)
	}
}

func TestHandoffStampsNoOpenTasks(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	writeHandoff(t, project, filledHandoff())
	harness.run(t, "handoff")

	got := string(readFile(t, filepath.Join(project, ".baton", "HANDOFF.md")))
	if !strings.Contains(got, handoffDelegatesMarker) {
		t.Fatalf("expected the stamped marker in the Delegates section: %s", got)
	}
	if !strings.Contains(got, "- no open tasks") {
		t.Fatalf("expected 'no open tasks' with nothing open: %s", got)
	}
}

func TestHandoffStampsDelegatesForOpenTask(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	taskID, key := parseRoundOutput(t, harness.run(t, "new-round", "handoff-delegate", "--summary", "Handoff delegate"))
	writeHandoff(t, project, filledHandoff())
	harness.run(t, "handoff")

	got := string(readFile(t, filepath.Join(project, ".baton", "HANDOFF.md")))
	planPath := ".baton/runs/" + key + "-PLAN.md"
	wanted := "- " + taskID + " (key " + key + "): last REQUEST, next PLANNED (delegate Planner), artifact " + planPath + ": missing"
	if !strings.Contains(got, wanted) {
		t.Fatalf("expected %q in stamped Delegates, got %s", wanted, got)
	}

	// Advance to a checkpoint RUN and reseal: the artifact state must track it.
	writePlan(t, project, planPath, taskID)
	harness.run(t, "append", eventPlanned, "--task-id", taskID, "--role", "Planner", "--summary", "Plan complete", "--path", planPath)
	runPath := ".baton/runs/" + key + "-RUN-01.md"
	writeRun(t, project, runPath, taskID, "01")
	checkpointContent := strings.Replace(string(readFile(t, filepath.Join(project, filepath.FromSlash(runPath)))), "Status: complete", "Status: checkpoint", 1)
	writeArtifact(t, project, runPath, checkpointContent)
	harness.run(t, "handoff")
	got = string(readFile(t, filepath.Join(project, ".baton", "HANDOFF.md")))
	wanted = "- " + taskID + " (key " + key + "): last PLANNED, next EXECUTED (delegate Executor), artifact " + runPath + ": checkpoint"
	if !strings.Contains(got, wanted) {
		t.Fatalf("expected %q after a checkpoint RUN, got %s", wanted, got)
	}

	// Complete the RUN on disk but do not append EXECUTED yet.
	writeRun(t, project, runPath, taskID, "01")
	harness.run(t, "handoff")
	got = string(readFile(t, filepath.Join(project, ".baton", "HANDOFF.md")))
	wanted = "- " + taskID + " (key " + key + "): last PLANNED, next EXECUTED (delegate Executor), artifact " + runPath + ": complete, not appended"
	if !strings.Contains(got, wanted) {
		t.Fatalf("expected %q once the RUN is complete but not appended, got %s", wanted, got)
	}
}

func TestHandoffSealRefusesHandWrittenDelegates(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	content := strings.Replace(filledHandoff(),
		"<filled automatically by `baton handoff` from BATON-LOG.txt and the run\nartifacts on disk; do not edit this section by hand>",
		"- Executor agent, mid-RUN-02, about to run go test.", 1)
	writeHandoff(t, project, content)

	if err := harness.fail("handoff"); err == nil {
		t.Fatal("expected seal to refuse a hand-written Delegates section")
	}
	got := string(readFile(t, filepath.Join(project, ".baton", "HANDOFF.md")))
	if got != content {
		t.Fatalf("refused seal must leave the file byte-identical:\nwant %q\ngot  %q", content, got)
	}
}

// TestHandoffResealSupersededBecomesLiveAgain pins the documented behaviour:
// sealing is always "current as of now", so re-sealing a record the log has
// already moved past re-stamps Log Line: and reports live again, with no
// separate "still superseded" state.
func TestHandoffResealSupersededBecomesLiveAgain(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	writeHandoff(t, project, filledHandoff())
	harness.run(t, "handoff")

	harness.run(t, "new-round", "handoff-reseal", "--summary", "Handoff reseal")
	status := harness.run(t, "status", "--open")
	if !strings.Contains(status, "handoff_superseded: .baton/HANDOFF.md") {
		t.Fatalf("expected the record to read superseded before resealing: %s", status)
	}

	harness.run(t, "handoff")
	status = harness.run(t, "status", "--open")
	if !strings.Contains(status, "handoff: .baton/HANDOFF.md") {
		t.Fatalf("expected the record to read live again after resealing: %s", status)
	}
	if strings.Contains(status, "handoff_superseded:") {
		t.Fatalf("a freshly resealed record must not still read superseded: %s", status)
	}
}

func TestHandoffClearRemovesRecordAndFailsTwice(t *testing.T) {
	harness := newHarness(t)
	project := harness.app.ProjectDir
	writeHandoff(t, project, filledHandoff())
	harness.run(t, "handoff")

	harness.run(t, "handoff", "--clear")
	path := filepath.Join(project, ".baton", "HANDOFF.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected .baton/HANDOFF.md to be removed, stat err: %v", err)
	}

	status := harness.run(t, "status")
	if strings.Contains(status, "handoff") {
		t.Fatalf("status should report no handoff line after --clear: %s", status)
	}

	if err := harness.fail("handoff", "--clear"); err == nil {
		t.Fatal("expected a second --clear to fail")
	}
}
