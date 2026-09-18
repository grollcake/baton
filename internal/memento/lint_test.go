package memento

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
	writeLog(t, harness.app.MementoDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Memento AI",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Memento AI initialized",
		fmt.Sprintf("2026-07-11T10:01:00 | evil | REQUEST  | Director | Probe | $(touch %s)", marker),
		"2026-07-11T10:01:01 | evil | RUN_DONE | Director | Probe complete",
	)
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted a malicious path")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("lint executed a path from memento.log")
	}
}

func TestLintRejectsLegacyScriptsDirectory(t *testing.T) {
	harness := newHarness(t)
	if err := os.MkdirAll(harness.app.mementoPath("scripts"), 0o755); err != nil {
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
	path := ".memento/runs/20260711-1001-legacy-PLAN.md"
	legacy := strings.Replace(validPlan("lgcy"), "Status: complete\n", "", 1)
	writeArtifact(t, harness.app.ProjectDir, path, legacy)
	writeLog(t, harness.app.MementoDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Memento AI",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Memento AI initialized",
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
