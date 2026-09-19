package baton

import (
	"strings"
	"testing"
	"time"
)

func TestAwaitReturnsWhenArtifactCompletes(t *testing.T) {
	harness := newHarness(t)
	path := ".baton/runs/20260711-1000-await-RUN-01.md"
	writeArtifact(t, harness.app.ProjectDir, path, strings.Replace(validRun("abcd", "01"), "Status: complete", "Status: checkpoint", 1))
	sleeps := 0
	harness.app.Sleep = func(time.Duration) {
		sleeps++
		if sleeps == 2 {
			writeRun(t, harness.app.ProjectDir, path, "abcd", "01")
		}
	}
	output := harness.run(t, "await", eventExecuted, path, "--task-id", "abcd")
	if sleeps != 2 || !strings.Contains(output, path) {
		t.Fatalf("await returned after %d sleeps: %s", sleeps, output)
	}
}

func TestAwaitTimesOutAndRejectsBadArguments(t *testing.T) {
	harness := newHarness(t)
	now := harness.app.Now()
	harness.app.Now = func() time.Time { return now }
	harness.app.Sleep = func(duration time.Duration) { now = now.Add(duration) }
	path := ".baton/runs/20260711-1000-await-PLAN.md"
	err := harness.fail("await", eventPlanned, path, "--timeout", "5s")
	if err == nil || !strings.Contains(err.Error(), "timed out") || !strings.Contains(err.Error(), "path not found") {
		t.Fatalf("unexpected await timeout error: %v", err)
	}

	harness.app.Sleep = func(time.Duration) { t.Fatal("await waited on invalid arguments") }
	for _, args := range [][]string{
		{"await"},
		{"await", eventPlanned},
		{"await", eventClose, ".baton/runs/x-CLOSE.md"},
		{"await", eventRequest, path},
		{"await", eventPlanned, ".baton/runs/x-RUN-01.md"},
		{"await", eventPlanned, path, "--timeout", "soon"},
		{"await", eventPlanned, path, "--timeout", "-1s"},
	} {
		if err := harness.fail(args...); err == nil {
			t.Fatalf("baton %v unexpectedly passed", args)
		}
	}
}

func TestAwaitReturnsOnFirstOfSeveralArtifacts(t *testing.T) {
	harness := newHarness(t)
	slow := ".baton/runs/20260711-1000-slow-RUN-01.md"
	fast := ".baton/runs/20260711-1000-fast-RUN-01.md"
	for _, path := range []string{slow, fast} {
		writeArtifact(t, harness.app.ProjectDir, path, strings.Replace(validRun("abcd", "01"), "Status: complete", "Status: checkpoint", 1))
	}
	sleeps := 0
	harness.app.Sleep = func(time.Duration) {
		sleeps++
		if sleeps == 2 {
			writeRun(t, harness.app.ProjectDir, fast, "abcd", "01")
		}
	}
	output := harness.run(t, "await", eventExecuted, slow, eventExecuted, fast)
	if !strings.Contains(output, fast) || strings.Contains(output, slow) {
		t.Fatalf("await did not return on the first completed artifact: %s", output)
	}
	if err := harness.fail("await", eventExecuted, slow, eventExecuted, fast, "--task-id", "abcd"); err == nil {
		t.Fatal("await accepted --task-id while watching several artifacts")
	}
	if err := harness.fail("await", eventExecuted, slow, eventExecuted); err == nil {
		t.Fatal("await accepted an unpaired event")
	}
}
