package baton

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintDoesNotExecutePaths(t *testing.T) {
	harness := newHarness(t)
	marker := filepath.Join(t.TempDir(), "command-ran")
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		fmt.Sprintf("2026-07-11T10:01:00 | evil | REQUEST  | Director | Probe | $(touch %s)", marker),
		"2026-07-11T10:01:01 | evil | RUN_DONE | Director | Probe complete",
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a malicious path")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("lint executed a path from the timeline")
	}
}

func TestLintAcceptsPathlessRequest(t *testing.T) {
	harness := newHarness(t)
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | preq | REQUEST  | Director | Pathless request",
	)
	harness.run(t, "lint")
	status := harness.run(t, "status", "--task-id", "preq")
	if strings.Contains(status, "next_command:") {
		t.Fatalf("pathless REQUEST should not offer a next_command: %s", status)
	}

	planPath := ".baton/runs/20260711-1001-preq-PLAN.md"
	writePlan(t, harness.app.ProjectDir, planPath, "preq")
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | preq | REQUEST  | Director | Pathless request",
		"2026-07-11T10:01:01 | preq | PLANNED  | Planner  | Plan complete | "+planPath,
	)
	harness.run(t, "lint")
}

func TestLintRejectsMissingPlannedArtifact(t *testing.T) {
	harness := newHarness(t)
	planPath := ".baton/runs/20260711-1001-missing-PLAN.md"
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | miss | REQUEST  | Director | Missing plan",
		"2026-07-11T10:01:01 | miss | PLANNED  | Planner  | Plan complete | "+planPath,
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a PLANNED artifact that does not exist on disk")
	}
}

// TestLintRejectsInvalidExecutedArtifact pins lint's EXECUTED arm of
// requiresArtifact: an EXECUTED artifact that exists on disk but fails
// checkArtifact (here, a missing Status: complete) must still be rejected,
// not merely checked for presence.
func TestLintRejectsInvalidExecutedArtifact(t *testing.T) {
	harness := newHarness(t)
	planPath := ".baton/runs/20260919-2100-badrun-PLAN.md"
	runPath := ".baton/runs/20260919-2100-badrun-RUN-01.md"
	writePlan(t, harness.app.ProjectDir, planPath, "brun")
	invalid := strings.Replace(validRun("brun", "01"), "Status: complete", "Status: checkpoint", 1)
	writeArtifact(t, harness.app.ProjectDir, runPath, invalid)
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-09-19T21:00:00 | brun | REQUEST  | Director | Bad run",
		"2026-09-19T21:00:01 | brun | PLANNED  | Planner  | Plan complete | "+planPath,
		"2026-09-19T21:00:02 | brun | EXECUTED | Executor | Run complete | "+runPath,
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted an EXECUTED artifact that fails checkArtifact")
	}
}

// TestLintRejectsInvalidReviewArtifact pins lint's REVIEW arm of
// requiresArtifact: a REVIEW artifact that exists on disk but fails
// checkArtifact (here, an invalid Result value) must still be rejected.
func TestLintRejectsInvalidReviewArtifact(t *testing.T) {
	harness := newHarness(t)
	planPath := ".baton/runs/20260919-2100-badreview-PLAN.md"
	runPath := ".baton/runs/20260919-2100-badreview-RUN-01.md"
	reviewPath := ".baton/runs/20260919-2100-badreview-REVIEW-01.md"
	writePlan(t, harness.app.ProjectDir, planPath, "brev")
	writeRun(t, harness.app.ProjectDir, runPath, "brev", "01")
	invalid := strings.Replace(validReview("brev", "01"), "ready-for-user-decision", "unknown", 1)
	writeArtifact(t, harness.app.ProjectDir, reviewPath, invalid)
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-09-19T21:00:00 | brev | REQUEST  | Director | Bad review",
		"2026-09-19T21:00:01 | brev | PLANNED  | Planner  | Plan complete | "+planPath,
		"2026-09-19T21:00:02 | brev | EXECUTED | Executor | Run complete | "+runPath,
		"2026-09-19T21:00:03 | brev | REVIEW   | Planner  | Review complete | "+reviewPath,
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a REVIEW artifact that fails checkArtifact")
	}
}

// TestLintRejectsInvalidCloseArtifact pins lint's CLOSE arm of
// requiresArtifact: a CLOSE artifact that exists on disk but fails
// checkArtifact (here, a missing user approval line) must still be rejected.
func TestLintRejectsInvalidCloseArtifact(t *testing.T) {
	harness := newHarness(t)
	planPath := ".baton/runs/20260919-2100-badclose-PLAN.md"
	runPath := ".baton/runs/20260919-2100-badclose-RUN-01.md"
	reviewPath := ".baton/runs/20260919-2100-badclose-REVIEW-01.md"
	closePath := ".baton/runs/20260919-2100-badclose-CLOSE.md"
	writePlan(t, harness.app.ProjectDir, planPath, "bcls")
	writeRun(t, harness.app.ProjectDir, runPath, "bcls", "01")
	writeReview(t, harness.app.ProjectDir, reviewPath, "bcls", "01", "ready-for-user-decision")
	invalid := strings.Replace(validClose("bcls"), "Approved By: User\n", "", 1)
	writeArtifact(t, harness.app.ProjectDir, closePath, invalid)
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-09-19T21:00:00 | bcls | REQUEST  | Director | Bad close",
		"2026-09-19T21:00:01 | bcls | PLANNED  | Planner  | Plan complete | "+planPath,
		"2026-09-19T21:00:02 | bcls | EXECUTED | Executor | Run complete | "+runPath,
		"2026-09-19T21:00:03 | bcls | REVIEW   | Planner  | Review complete | "+reviewPath,
		"2026-09-19T21:00:04 | bcls | CLOSE    | Director | Closed | "+closePath,
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a CLOSE artifact that fails checkArtifact")
	}
}

func TestLintRejectsMissingFeedbackOrRunDonePath(t *testing.T) {
	harness := newHarness(t)
	missingRun := ".baton/runs/20260711-1001-missing-RUN-01.md"
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | fdbk | REQUEST  | Director | Feedback missing path",
		"2026-07-11T10:01:01 | fdbk | RUN_DONE | Director | Direct work done | "+missingRun,
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a RUN_DONE path that does not exist on disk")
	}

	planPath := ".baton/runs/20260711-1001-fdbk2-PLAN.md"
	runPath := ".baton/runs/20260711-1001-fdbk2-RUN-01.md"
	reviewPath := ".baton/runs/20260711-1001-fdbk2-REVIEW-01.md"
	missingReview := ".baton/runs/20260711-1001-fdbk2-missing.md"
	writePlan(t, harness.app.ProjectDir, planPath, "fdb2")
	writeRun(t, harness.app.ProjectDir, runPath, "fdb2", "01")
	writeReview(t, harness.app.ProjectDir, reviewPath, "fdb2", "01", "blockers")
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | fdb2 | REQUEST  | Director | Feedback missing path",
		"2026-07-11T10:01:01 | fdb2 | PLANNED  | Planner  | Plan complete | "+planPath,
		"2026-07-11T10:01:02 | fdb2 | EXECUTED | Executor | Run complete | "+runPath,
		"2026-07-11T10:01:03 | fdb2 | REVIEW   | Planner  | Review complete | "+reviewPath,
		"2026-07-11T10:01:04 | fdb2 | FEEDBACK | Director | Feedback given | "+missingReview,
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a FEEDBACK path that does not exist on disk")
	}
}

func TestLintRejectsLegacyScriptsDirectory(t *testing.T) {
	harness := newHarness(t)
	if err := os.MkdirAll(harness.app.batonPath("scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a legacy scripts directory")
	}
}

func TestLintRejectsTamperedBinary(t *testing.T) {
	harness := newHarness(t)
	if err := os.WriteFile(harness.app.installedBinaryPath(), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a binary checksum mismatch")
	}
}

func TestLintAcceptsLegacyArtifactsWithoutStatus(t *testing.T) {
	harness := newHarness(t)
	path := ".baton/runs/20260711-1001-legacy-PLAN.md"
	legacy := strings.Replace(validPlan("lgcy"), "Status: complete\n", "", 1)
	writeArtifact(t, harness.app.ProjectDir, path, legacy)
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | lgcy | REQUEST  | Director | Legacy task",
		"2026-07-11T10:01:01 | lgcy | PLANNED  | Planner  | Legacy plan | "+path,
	)
	harness.run(t, "lint")
	if err := harness.fail("check-artifact", eventPlanned, path, "lgcy"); err == nil {
		t.Fatal("check-artifact accepted a PLAN without Status")
	}

	writeArtifact(t, harness.app.ProjectDir, path, strings.Replace(validPlan("lgcy"), "Status: complete", "Status: checkpoint", 1))
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a PLAN with Status: checkpoint")
	}
}

func TestLintRejectsCloseAfterBlockers(t *testing.T) {
	harness := newHarness(t)
	key := ".baton/runs/20260711-1001-blocked"
	planPath, runPath := key+"-PLAN.md", key+"-RUN-01.md"
	reviewPath, closePath := key+"-REVIEW-01.md", key+"-CLOSE.md"
	writePlan(t, harness.app.ProjectDir, planPath, "blkd")
	writeRun(t, harness.app.ProjectDir, runPath, "blkd", "01")
	writeReview(t, harness.app.ProjectDir, reviewPath, "blkd", "01", "blockers")
	writeClose(t, harness.app.ProjectDir, closePath, "blkd")
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | blkd | REQUEST  | Director | Blocked task",
		"2026-07-11T10:01:01 | blkd | PLANNED  | Planner  | Plan complete | "+planPath,
		"2026-07-11T10:01:02 | blkd | EXECUTED | Executor | Run complete | "+runPath,
		"2026-07-11T10:01:03 | blkd | REVIEW   | Planner  | Review complete | "+reviewPath,
		"2026-07-11T10:01:04 | blkd | CLOSE    | Director | Closed | "+closePath,
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a CLOSE that followed a review with blockers")
	}

	writeReview(t, harness.app.ProjectDir, reviewPath, "blkd", "01", "ready-for-user-decision")
	harness.run(t, "lint")
}

// TestLintRejectsCloseAfterUnreadableReview proves that CLOSE is rejected when
// the preceding REVIEW's artifact cannot be read from disk, even though the
// REVIEW was logged with Result: ready-for-user-decision. An unreadable
// REVIEW must fail closed the same way a REVIEW reporting blockers does; only
// the missing artifact, not a blockers line, can be producing the rejection
// here.
func TestLintRejectsCloseAfterUnreadableReview(t *testing.T) {
	harness := newHarness(t)
	key := ".baton/runs/20260711-1002-unreadable"
	planPath, runPath := key+"-PLAN.md", key+"-RUN-01.md"
	reviewPath, closePath := key+"-REVIEW-01.md", key+"-CLOSE.md"
	writePlan(t, harness.app.ProjectDir, planPath, "unrd")
	writeRun(t, harness.app.ProjectDir, runPath, "unrd", "01")
	writeReview(t, harness.app.ProjectDir, reviewPath, "unrd", "01", "ready-for-user-decision")
	writeClose(t, harness.app.ProjectDir, closePath, "unrd")
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:02:00 | unrd | REQUEST  | Director | Unreadable task",
		"2026-07-11T10:02:01 | unrd | PLANNED  | Planner  | Plan complete | "+planPath,
		"2026-07-11T10:02:02 | unrd | EXECUTED | Executor | Run complete | "+runPath,
		"2026-07-11T10:02:03 | unrd | REVIEW   | Planner  | Review complete | "+reviewPath,
		"2026-07-11T10:02:04 | unrd | CLOSE    | Director | Closed | "+closePath,
	)
	if err := os.Remove(harness.app.projectPath(reviewPath)); err != nil {
		t.Fatal(err)
	}
	err := harness.fail("lint")
	if err == nil {
		t.Fatal("lint accepted a CLOSE that followed a REVIEW whose artifact could not be read")
	}
	if !strings.Contains(harness.err.String(), "CLOSE on line") || !strings.Contains(harness.err.String(), "follows a REVIEW reporting blockers") {
		t.Fatalf("expected the CLOSE-after-blockers message, got: %s", harness.err.String())
	}
}

func TestLintDetectsManagedDocumentDrift(t *testing.T) {
	harness := newHarness(t)
	harness.run(t, "lint")

	path := harness.app.batonPath("PROTOCOL.md")
	content := append(readFile(t, path), []byte("\nLocally edited rule.\n")...)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a managed document that differs from the shipped copy")
	}
}

func TestGuidePrintsShippedDocuments(t *testing.T) {
	harness := newHarness(t)
	if output := harness.run(t, "guide", "protocol"); !strings.Contains(output, "# Baton Protocol") {
		t.Fatalf("unexpected protocol guide: %s", output)
	}
	if output := harness.run(t, "guide", "director"); !strings.Contains(output, "# Director Protocol") {
		t.Fatalf("unexpected director guide: %s", output)
	}
	if err := harness.fail("guide", "nope"); err == nil {
		t.Fatal("guide accepted an unknown document")
	}
	if err := harness.fail("guide"); err == nil {
		t.Fatal("guide accepted missing arguments")
	}
}

func TestLintRejectsIgnoredBatonDirectory(t *testing.T) {
	harness := newHarness(t)
	if _, err := harness.app.runGit("init"); err != nil {
		t.Skipf("git unavailable: %s", err)
	}
	harness.run(t, "lint")

	path := filepath.Join(harness.app.ProjectDir, ".gitignore")
	if err := os.WriteFile(path, []byte(".baton/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a .baton directory ignored by Git")
	}
}

// TestLintRejectsIgnoredRequiredDocument covers Decision 5: a .gitignore rule
// that matches a single required document (not the whole .baton directory)
// must still fail lint, naming the ignored path.
func TestLintRejectsIgnoredRequiredDocument(t *testing.T) {
	harness := newHarness(t)
	if _, err := harness.app.runGit("init"); err != nil {
		t.Skipf("git unavailable: %s", err)
	}
	harness.run(t, "lint")

	path := filepath.Join(harness.app.ProjectDir, ".gitignore")
	if err := os.WriteFile(path, []byte("GUIDANCE.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted an ignored GUIDANCE.md")
	} else if !strings.Contains(harness.err.String(), "GUIDANCE.md") {
		t.Fatalf("lint error does not name the ignored document: %s", harness.err.String())
	}
}

// TestLintRejectsIgnoredRunsDirectory covers Decision 5: a .gitignore rule
// that matches .baton/runs/ alone must still fail lint.
func TestLintRejectsIgnoredRunsDirectory(t *testing.T) {
	harness := newHarness(t)
	if _, err := harness.app.runGit("init"); err != nil {
		t.Skipf("git unavailable: %s", err)
	}
	harness.run(t, "lint")

	path := filepath.Join(harness.app.ProjectDir, ".gitignore")
	if err := os.WriteFile(path, []byte("runs/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted an ignored .baton/runs")
	} else if !strings.Contains(harness.err.String(), filepath.Join(".baton", "runs")) {
		t.Fatalf("lint error does not name the ignored directory: %s", harness.err.String())
	}
}

// TestLintToleratesIgnoredBin covers Decision 5's mandatory carve-out: this
// repository (and any project following its convention) ignores
// .baton/bin/, the installed binary, on purpose, and lint must keep passing
// when only that path is ignored.
func TestLintToleratesIgnoredBin(t *testing.T) {
	harness := newHarness(t)
	if _, err := harness.app.runGit("init"); err != nil {
		t.Skipf("git unavailable: %s", err)
	}
	harness.run(t, "lint")

	path := filepath.Join(harness.app.ProjectDir, ".gitignore")
	if err := os.WriteFile(path, []byte("/.baton/bin/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	harness.run(t, "lint")
}

func TestLintToleratesLegacyRunSections(t *testing.T) {
	harness := newHarness(t)
	key := ".baton/runs/20260711-1001-legacy-run"
	planPath, runPath := key+"-PLAN.md", key+"-RUN-01.md"
	writePlan(t, harness.app.ProjectDir, planPath, "lgrn")
	legacy := strings.Replace(validRun("lgrn", "01"), "## Changes\n", "", 1)
	writeArtifact(t, harness.app.ProjectDir, runPath, strings.Replace(legacy, "## Unresolved Risks\n", "", 1))
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | lgrn | REQUEST  | Director | Legacy run",
		"2026-07-11T10:01:01 | lgrn | PLANNED  | Planner  | Plan complete | "+planPath,
		"2026-07-11T10:01:02 | lgrn | EXECUTED | Executor | Run complete | "+runPath,
	)
	harness.run(t, "lint")
	if err := harness.fail("check-artifact", eventExecuted, runPath, "lgrn"); err == nil {
		t.Fatal("check-artifact accepted a RUN without the required sections")
	}
}

func TestLintToleratesLegacyCloseSections(t *testing.T) {
	harness := newHarness(t)
	key := ".baton/runs/20260711-1001-legacy-close"
	planPath, runPath := key+"-PLAN.md", key+"-RUN-01.md"
	reviewPath, closePath := key+"-REVIEW-01.md", key+"-CLOSE.md"
	writePlan(t, harness.app.ProjectDir, planPath, "lgcl")
	writeRun(t, harness.app.ProjectDir, runPath, "lgcl", "01")
	writeReview(t, harness.app.ProjectDir, reviewPath, "lgcl", "01", "ready-for-user-decision")
	legacy := strings.Replace(validClose("lgcl"), "## Plan Deviations\n- none.\n", "", 1)
	writeArtifact(t, harness.app.ProjectDir, closePath, legacy)
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:01:00 | lgcl | REQUEST  | Director | Legacy close",
		"2026-07-11T10:01:01 | lgcl | PLANNED  | Planner  | Plan complete | "+planPath,
		"2026-07-11T10:01:02 | lgcl | EXECUTED | Executor | Run complete | "+runPath,
		"2026-07-11T10:01:03 | lgcl | REVIEW   | Planner  | Review complete | "+reviewPath,
		"2026-07-11T10:01:04 | lgcl | CLOSE    | Director | Closed | "+closePath,
	)
	harness.run(t, "lint")
	if err := harness.fail("check-artifact", eventClose, closePath, "lgcl"); err == nil {
		t.Fatal("check-artifact accepted a CLOSE without Plan Deviations")
	}
}
