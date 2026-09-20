package baton

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestAppendMigratesLegacyTimeline covers Decision 1's write-path migration:
// a project holding only the legacy baton.log gets renamed to BATON-LOG.txt
// by the first append, and the prior lines survive unchanged ahead of the
// newly appended one.
func TestAppendMigratesLegacyTimeline(t *testing.T) {
	harness := newHarness(t)
	before := readFile(t, harness.app.batonPath(timelineFile))
	if err := os.Rename(harness.app.batonPath(timelineFile), harness.app.batonPath(legacyTimelineFile)); err != nil {
		t.Fatal(err)
	}

	taskID, key := parseRoundOutput(t, harness.run(t, "new-round", "legacy-append", "--summary", "Legacy append flow"))
	_ = key

	if _, err := os.Stat(harness.app.batonPath(legacyTimelineFile)); !os.IsNotExist(err) {
		t.Fatal("legacy baton.log still present after append")
	}
	after := readFile(t, harness.app.batonPath(timelineFile))
	if !bytes.HasPrefix(after, before) {
		t.Fatal("migrated timeline lost its prior lines")
	}
	if !strings.Contains(string(after), taskID) {
		t.Fatalf("migrated timeline missing the new append: %s", after)
	}
}

// TestBothTimelineNamesRefuseReadAndWrite covers Decision 1's refusal case:
// when BATON-LOG.txt and baton.log are both present, every command that
// reads or writes the timeline fails naming both paths, and neither file is
// modified.
func TestBothTimelineNamesRefuseReadAndWrite(t *testing.T) {
	harness := newHarness(t)
	currentBefore := readFile(t, harness.app.batonPath(timelineFile))
	if err := os.WriteFile(harness.app.batonPath(legacyTimelineFile), currentBefore, 0o644); err != nil {
		t.Fatal(err)
	}
	legacyBefore := readFile(t, harness.app.batonPath(legacyTimelineFile))

	statusErr := harness.fail("status")
	if statusErr == nil {
		t.Fatal("status accepted both timeline names present")
	}
	if !strings.Contains(statusErr.Error(), timelineFile) || !strings.Contains(statusErr.Error(), legacyTimelineFile) {
		t.Fatalf("status error does not name both paths: %v", statusErr)
	}

	appendErr := harness.fail("append", eventRequest, "--task-id", "botn", "--role", "Director", "--summary", "Should not append")
	if appendErr == nil {
		t.Fatal("append accepted both timeline names present")
	}
	if !strings.Contains(appendErr.Error(), timelineFile) || !strings.Contains(appendErr.Error(), legacyTimelineFile) {
		t.Fatalf("append error does not name both paths: %v", appendErr)
	}

	gateErr := harness.fail("gate", "before-execute", "--task-id", "botn")
	if gateErr == nil {
		t.Fatal("gate accepted both timeline names present")
	}
	if !strings.Contains(gateErr.Error(), timelineFile) || !strings.Contains(gateErr.Error(), legacyTimelineFile) {
		t.Fatalf("gate error does not name both paths: %v", gateErr)
	}

	if after := readFile(t, harness.app.batonPath(timelineFile)); !bytes.Equal(after, currentBefore) {
		t.Fatal("BATON-LOG.txt was modified while both names were present")
	}
	if after := readFile(t, harness.app.batonPath(legacyTimelineFile)); !bytes.Equal(after, legacyBefore) {
		t.Fatal("baton.log was modified while both names were present")
	}
}

// TestLintReportsBothTimelineNames covers the same refusal surfacing through
// lint, which must report the conflict rather than silently pick a file.
func TestLintReportsBothTimelineNames(t *testing.T) {
	harness := newHarness(t)
	content := readFile(t, harness.app.batonPath(timelineFile))
	if err := os.WriteFile(harness.app.batonPath(legacyTimelineFile), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint accepted both timeline names present")
	}
	if !strings.Contains(harness.err.String(), timelineFile) || !strings.Contains(harness.err.String(), legacyTimelineFile) {
		t.Fatalf("lint error does not name both paths: %s", harness.err.String())
	}
}
