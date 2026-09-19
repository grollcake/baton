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
