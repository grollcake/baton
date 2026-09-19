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
		t.Fatal("lint executed a path from baton.log")
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
