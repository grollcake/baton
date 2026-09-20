package baton

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	docs "github.com/grollcake/baton"
)

// gitInit initialises a Git repository in harness's project directory with a
// local identity, or skips the test when git is unavailable -- the same
// pattern lint_test.go uses for its own Git-dependent tests.
func gitInit(t *testing.T, harness *testHarness) {
	t.Helper()
	if _, err := harness.app.runGit("init"); err != nil {
		t.Skipf("git unavailable: %s", err)
	}
	if _, err := harness.app.runGit("config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.app.runGit("config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
}

func gitCommitAll(t *testing.T, harness *testHarness, message string) {
	t.Helper()
	if _, err := harness.app.runGit("add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.app.runGit("commit", "-m", message); err != nil {
		t.Fatal(err)
	}
}

// TestRemoveNotInstalledReportsAndExitsZero covers Decision 4: remove in a
// project with no Baton block and nothing left to delete reports so and
// changes nothing.
func TestRemoveNotInstalledReportsAndExitsZero(t *testing.T) {
	project := t.TempDir()
	app := New(project, filepath.Join(project, ".baton"))
	var out, errBuf bytes.Buffer
	app.Stdout, app.Stderr = &out, &errBuf
	if err := app.Run([]string{"remove", "--apply"}); err != nil {
		t.Fatalf("remove failed in a project with nothing installed: %v", err)
	}
	if !strings.Contains(out.String(), "not installed") {
		t.Fatalf("remove did not report 'not installed': %s", out.String())
	}
}

// TestRemoveDeletesWholeFileBatonWroteInFull covers Decision 3's whole-file
// comparison: newHarness's AGENTS.md is exactly what bootstrap ships, so
// remove deletes the file rather than leaving an empty stub.
func TestRemoveDeletesWholeFileBatonWroteInFull(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)
	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	claudePath := filepath.Join(harness.app.ProjectDir, "CLAUDE.md")

	harness.run(t, "remove", "--apply")

	if _, err := os.Stat(agentsPath); !os.IsNotExist(err) {
		t.Fatalf("AGENTS.md was not deleted: err=%v", err)
	}
	if _, err := os.Stat(claudePath); !os.IsNotExist(err) {
		t.Fatalf("CLAUDE.md was not deleted: err=%v", err)
	}
}

// TestRemoveInstallRoundTripWithSurroundingText is V1: a project's own text
// above the block, ending in a newline, comes back byte-identical after
// merge-agent-block then remove --apply.
func TestRemoveInstallRoundTripWithSurroundingText(t *testing.T) {
	harness := newHarness(t)
	original := []byte("Project instructions.\n\nDo the thing.\n")
	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	claudePath := filepath.Join(harness.app.ProjectDir, "CLAUDE.md")
	if err := os.WriteFile(agentsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claudePath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	active, err := docs.ActiveBlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceBlock(agentsPath, active); err != nil {
		t.Fatal(err)
	}
	if err := replaceBlock(claudePath, active); err != nil {
		t.Fatal(err)
	}

	before := readFile(t, agentsPath)
	if bytes.Equal(before, original) {
		t.Fatal("merge did not change AGENTS.md")
	}

	harness.run(t, "remove", "--apply")

	after := readFile(t, agentsPath)
	if !bytes.Equal(after, original) {
		t.Fatalf("AGENTS.md not byte-identical after remove:\nbefore: %q\nafter:  %q", original, after)
	}
	afterClaude := readFile(t, claudePath)
	if !bytes.Equal(afterClaude, original) {
		t.Fatalf("CLAUDE.md not byte-identical after remove:\nbefore: %q\nafter:  %q", original, afterClaude)
	}
}

// TestRemoveInstallRoundTripNoFinalNewline is R3: a project file whose block
// sat last and which had no final newline before the merge gains exactly one
// newline on removal -- shown, not just claimed.
func TestRemoveInstallRoundTripNoFinalNewline(t *testing.T) {
	harness := newHarness(t)
	original := []byte("Project instructions with no trailing newline")
	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	active, err := docs.ActiveBlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceBlock(agentsPath, active); err != nil {
		t.Fatal(err)
	}

	harness.run(t, "remove", "--apply")

	after := readFile(t, agentsPath)
	want := append(append([]byte{}, original...), '\n')
	if !bytes.Equal(after, want) {
		t.Fatalf("expected original content plus one newline (R3), got:\nbefore: %q\nafter:  %q", original, after)
	}
}

// TestRemoveDeletesFileThatIsNothingButTheBlock covers Decision 3's other
// seam case: a file whose whole content, after the block is cut, is empty is
// deleted rather than left as an empty file. Merging into an empty CLAUDE.md
// produces just the block (no bootstrap heading), so it does not match
// bootstrap/CLAUDE.md and takes the "nothing remains" path rather than the
// whole-file-shipped path.
func TestRemoveDeletesFileThatIsNothingButTheBlock(t *testing.T) {
	harness := newHarness(t)
	claudePath := filepath.Join(harness.app.ProjectDir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	active, err := docs.ActiveBlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceBlock(claudePath, active); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, claudePath)
	if len(bytes.TrimSpace(bytes.TrimPrefix(before, active))) != 0 {
		t.Fatalf("fixture did not produce a block-only file: %q", before)
	}

	harness.run(t, "remove", "--apply")

	if _, err := os.Stat(claudePath); !os.IsNotExist(err) {
		t.Fatalf("a block-only CLAUDE.md was not deleted: err=%v", err)
	}
}

// TestRemoveKeepsRecordsByteIdentical is V2: every kept path under .baton/ is
// byte-identical before and after, and the timeline gains exactly two
// appended lines with no existing line changed.
func TestRemoveKeepsRecordsByteIdentical(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)
	writeArtifact(t, harness.app.ProjectDir, ".baton/runs/kept.md", "custom record\n")
	if err := os.WriteFile(harness.app.batonPath("GUIDANCE.md"), []byte("custom guidance\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logBefore := readFile(t, harness.app.batonPath(timelineFile))
	guidanceBefore := readFile(t, harness.app.batonPath("GUIDANCE.md"))
	runBefore := readFile(t, harness.app.batonPath("runs", "kept.md"))

	harness.run(t, "remove", "--apply")

	if _, err := os.Stat(harness.app.batonPath("PROTOCOL.md")); !os.IsNotExist(err) {
		t.Fatal("PROTOCOL.md survived remove --apply")
	}
	if _, err := os.Stat(harness.app.batonPath("VERSION")); !os.IsNotExist(err) {
		t.Fatal("VERSION survived remove --apply")
	}
	if _, err := os.Stat(harness.app.batonPath("bin")); !os.IsNotExist(err) {
		t.Fatal("bin/ survived remove --apply")
	}
	if _, err := os.Stat(harness.app.batonPath("templates")); !os.IsNotExist(err) {
		t.Fatal("templates/ survived remove --apply")
	}

	guidanceAfter := readFile(t, harness.app.batonPath("GUIDANCE.md"))
	if !bytes.Equal(guidanceBefore, guidanceAfter) {
		t.Fatal("GUIDANCE.md changed by remove")
	}
	runAfter := readFile(t, harness.app.batonPath("runs", "kept.md"))
	if !bytes.Equal(runBefore, runAfter) {
		t.Fatal("runs/kept.md changed by remove")
	}

	logAfter := readFile(t, harness.app.batonPath(timelineFile))
	beforeLines := strings.Split(strings.TrimRight(string(logBefore), "\n"), "\n")
	afterLines := strings.Split(strings.TrimRight(string(logAfter), "\n"), "\n")
	if len(afterLines) != len(beforeLines)+2 {
		t.Fatalf("expected exactly 2 new timeline lines, got %d -> %d", len(beforeLines), len(afterLines))
	}
	for i, line := range beforeLines {
		if afterLines[i] != line {
			t.Fatalf("existing timeline line %d changed:\nbefore: %s\nafter:  %s", i, line, afterLines[i])
		}
	}
	if !strings.Contains(afterLines[len(afterLines)-1], "RUN_DONE") {
		t.Fatalf("last timeline line is not RUN_DONE: %s", afterLines[len(afterLines)-1])
	}
}

// TestRemoveRefusesHandEditedBlock is Director emphasis #4 and part of V3: a
// hand-written sentence inside the markers stops removal entirely, names the
// file, and leaves the whole tree byte-identical. --force does not override
// it.
func TestRemoveRefusesHandEditedBlock(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)
	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("<baton-rules>\nhand-edited\n</baton-rules>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before := snapshotTree(t, harness.app.ProjectDir)

	if err := harness.fail("remove", "--apply"); err == nil {
		t.Fatal("remove accepted a hand-edited block")
	} else if !strings.Contains(err.Error(), "AGENTS.md") {
		t.Fatalf("refusal does not name AGENTS.md: %v", err)
	}
	if err := harness.fail("remove", "--apply", "--force"); err == nil {
		t.Fatal("--force overrode the hand-edited-block refusal")
	}

	after := snapshotTree(t, harness.app.ProjectDir)
	for path, content := range before {
		if !bytes.Equal(content, after[path]) {
			t.Fatalf("%s changed despite the refusal", path)
		}
	}
	if len(after) != len(before) {
		t.Fatal("the refused remove created or deleted a file")
	}
}

// TestRemoveKeepsUnrecognizedBatonContent is Director emphasis #4: a locally
// modified managed document and a project's own template under .baton/ are
// not Baton's to delete; remove --apply keeps both and names them as kept.
func TestRemoveKeepsUnrecognizedBatonContent(t *testing.T) {
	harness := newHarness(t)
	protocolPath := harness.app.batonPath("PROTOCOL.md")
	original := readFile(t, protocolPath)
	locallyModified := append(append([]byte{}, original...), []byte("\nlocal note\n")...)
	if err := os.WriteFile(protocolPath, locallyModified, 0o644); err != nil {
		t.Fatal(err)
	}
	customTemplate := harness.app.batonPath("templates", "custom.md")
	if err := os.WriteFile(customTemplate, []byte("<custom-token>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output := harness.run(t, "remove", "--apply")

	if !bytes.Equal(readFile(t, protocolPath), locallyModified) {
		t.Fatal("remove deleted or changed a locally modified PROTOCOL.md")
	}
	if !bytes.Equal(readFile(t, customTemplate), []byte("<custom-token>\n")) {
		t.Fatal("remove deleted or changed a project's own template")
	}
	if !strings.Contains(output, "PROTOCOL.md") {
		t.Fatalf("remove did not report the kept PROTOCOL.md: %s", output)
	}
}

// TestRemoveOpenTaskRefusesThenForce mirrors pause's Decision 5 behaviour for
// remove: an open task blocks by default, is named in the refusal, and
// --force proceeds and names it in the recorded summary.
func TestRemoveOpenTaskRefusesThenForce(t *testing.T) {
	harness := newHarness(t)
	taskID, _ := parseRoundOutput(t, harness.run(t, "new-round", "open-task", "--summary", "Open work"))

	logBefore := readFile(t, harness.app.batonPath(timelineFile))
	if err := harness.fail("remove", "--apply"); err == nil {
		t.Fatal("remove accepted an open task without --force")
	}
	if !bytes.Equal(logBefore, readFile(t, harness.app.batonPath(timelineFile))) {
		t.Fatal("a refused remove wrote to the timeline")
	}

	output := harness.run(t, "remove", "--apply", "--force")
	if !strings.Contains(output, taskID) {
		t.Fatalf("forced remove summary does not name the open task: %s", output)
	}
	log := string(readFile(t, harness.app.batonPath(timelineFile)))
	if !strings.Contains(log, taskID) || !strings.Contains(log, "RUN_DONE") {
		t.Fatalf("forced remove did not record the open task in RUN_DONE: %s", log)
	}
}

// TestRemoveDryRunChangesNothing is V3's "dry run is a no-op": remove without
// --apply exits 0 and the whole tree is byte-identical afterwards.
func TestRemoveDryRunChangesNothing(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)
	before := snapshotTree(t, harness.app.ProjectDir)

	output := harness.run(t, "remove")
	if !strings.Contains(output, "Dry run") {
		t.Fatalf("remove dry run output missing 'Dry run': %s", output)
	}

	after := snapshotTree(t, harness.app.ProjectDir)
	for path, content := range before {
		if !bytes.Equal(content, after[path]) {
			t.Fatalf("%s changed by a dry run", path)
		}
	}
	if len(after) != len(before) {
		t.Fatal("a dry run created or deleted a file")
	}
}

// TestRemovePausedProjectThenResumeRefuses is V6: a paused project removes
// without resuming first, and afterwards resume refuses, naming the removal.
func TestRemovePausedProjectThenResumeRefuses(t *testing.T) {
	harness := newHarness(t)
	harness.run(t, "pause")
	harness.run(t, "remove", "--apply")

	if err := harness.fail("resume"); err == nil {
		t.Fatal("resume was accepted after Baton was removed")
	} else if !strings.Contains(err.Error(), "removed") {
		t.Fatalf("resume's refusal does not mention removal: %v", err)
	}
}

// TestUpdateApplyRefusesInRemovedProject covers Decision 6: update --apply
// after a removal refuses and leaves the instruction files byte-identical,
// rather than silently reinstalling Baton through MergeAgentBlock's
// append-when-absent behaviour.
func TestUpdateApplyRefusesInRemovedProject(t *testing.T) {
	harness := newHarness(t)
	harness.run(t, "remove", "--apply")

	// update needs an upstream with its own VERSION to get past the earlier
	// argument checks; the removed project's own .baton/VERSION is gone, so
	// runUpdate fails before reaching the removed-project check -- which is
	// the accident Decision 6 names, and it still refuses either way.
	upstream := t.TempDir()
	if err := os.MkdirAll(filepath.Join(upstream, "bootstrap", ".baton"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(upstream, "VERSION"), []byte("9.9.9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("update", "--upstream", upstream, "--apply"); err == nil {
		t.Fatal("update --apply was accepted in a removed project")
	}
	if _, err := os.Stat(filepath.Join(harness.app.ProjectDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("update --apply recreated AGENTS.md in a removed project")
	}
}

// TestLintOnRemovedProject is V4: lint exits 0 on a removed project and
// reports the removed state; deleting a kept runs/ artifact then makes it
// fail, naming that artifact.
func TestLintOnRemovedProject(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)
	planPath := ".baton/runs/20260711-1005-removed-check-PLAN.md"
	writePlan(t, harness.app.ProjectDir, planPath, "rmvd")
	writeLog(t, harness.app.BatonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
		"2026-07-11T10:05:00 | rmvd | REQUEST  | Director | Removed check",
		"2026-07-11T10:05:01 | rmvd | PLANNED  | Planner  | Plan complete | "+planPath,
	)

	harness.run(t, "remove", "--apply", "--force")

	output := harness.run(t, "lint")
	if !strings.Contains(output, "not in effect") {
		t.Fatalf("lint on a removed project did not report the removed state: %s", output)
	}

	if err := os.Remove(filepath.Join(harness.app.ProjectDir, planPath)); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint passed after a runs/ artifact was deleted from a removed project")
	} else if !strings.Contains(harness.err.String(), planPath) {
		t.Fatalf("lint error does not name the missing artifact: %s", harness.err.String())
	}
}

// TestLintDisagreeingInstructionFiles covers Decision 6's disagreement error:
// one instruction file carrying a block while the other carries none is an
// error, not a silent pass.
func TestLintDisagreeingInstructionFiles(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)
	if err := os.Remove(filepath.Join(harness.app.ProjectDir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("no block here\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint passed when AGENTS.md carries no block but CLAUDE.md does")
	} else if !strings.Contains(harness.err.String(), "AGENTS.md") || !strings.Contains(harness.err.String(), "CLAUDE.md") {
		t.Fatalf("lint error does not name both files: %s", harness.err.String())
	}
}

// TestRemovePurgeRefusesOnUncommittedBaton is Director emphasis #2: --purge
// must refuse and delete nothing when .baton/ carries uncommitted or
// untracked content, constructing the exact case the guard exists for.
func TestRemovePurgeRefusesOnUncommittedBaton(t *testing.T) {
	harness := newHarness(t)
	gitInit(t, harness)
	gitCommitAll(t, harness, "initial commit")

	// Untrack one file under .baton/ by modifying it after the commit --
	// exactly the case the guard exists to catch.
	guidancePath := harness.app.batonPath("GUIDANCE.md")
	if err := os.WriteFile(guidancePath, []byte("uncommitted change\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before := snapshotTree(t, harness.app.ProjectDir)

	if err := harness.fail("remove", "--apply", "--purge"); err == nil {
		t.Fatal("--purge was accepted with an uncommitted change under .baton/")
	} else if !strings.Contains(err.Error(), "committed") {
		t.Fatalf("refusal does not mention the commit precondition: %v", err)
	}

	after := snapshotTree(t, harness.app.ProjectDir)
	for path, content := range before {
		if !bytes.Equal(content, after[path]) {
			t.Fatalf("%s changed despite --purge refusing", path)
		}
	}
	if len(after) != len(before) {
		t.Fatal("a refused --purge created or deleted a file")
	}
	if _, err := os.Stat(harness.app.BatonDir); err != nil {
		t.Fatal(".baton/ was deleted despite --purge refusing")
	}
}

// TestRemovePurgeRefusesOutsideGit covers Director emphasis #2: a project
// that is not a Git repository can never --purge.
func TestRemovePurgeRefusesOutsideGit(t *testing.T) {
	harness := newHarness(t)
	if err := harness.fail("remove", "--apply", "--purge"); err == nil {
		t.Fatal("--purge was accepted outside a Git repository")
	}
	if _, err := os.Stat(harness.app.BatonDir); err != nil {
		t.Fatal(".baton/ was deleted despite --purge refusing outside Git")
	}
}

// TestRemovePurgeSucceedsOnCleanCommittedBaton is Director emphasis #2's
// converse: a fully committed, clean .baton/ lets --purge through, and Git
// still holds the deleted timeline at the commit named in the report.
func TestRemovePurgeSucceedsOnCleanCommittedBaton(t *testing.T) {
	harness := newHarness(t)
	gitInit(t, harness)
	gitCommitAll(t, harness, "initial commit")

	output := harness.run(t, "remove", "--apply", "--purge")
	if !strings.Contains(output, "purged") {
		t.Fatalf("purge did not report success: %s", output)
	}

	if _, err := os.Stat(harness.app.BatonDir); !os.IsNotExist(err) {
		t.Fatalf(".baton/ was not deleted by --purge: err=%v", err)
	}

	committed, err := harness.app.runGit("show", "HEAD:.baton/"+timelineFile)
	if err != nil {
		t.Fatalf("git show could not read the committed timeline: %v", err)
	}
	if !strings.Contains(committed, "Bootstrap Baton") {
		t.Fatalf("the committed timeline does not hold the pre-purge record: %s", committed)
	}
}

// TestPurgePreconditionReportsFullPathForFirstLine is REVIEW-01's N1: runGit
// trims the whole `git status --porcelain` output as one string, which
// strips the leading separator space from the first status line only (e.g.
// " M .baton/GUIDANCE.md" -> "M .baton/GUIDANCE.md"), shifting it one byte
// left. A naive line[3:] then cuts one byte too many from that line's path,
// reporting "baton/GUIDANCE.md" instead of ".baton/GUIDANCE.md" -- a path
// that does not exist, which sends a user looking in the wrong place. This
// constructs exactly that shape (a modified-in-worktree file sorting before
// an untracked one, so the affected line is genuinely first) and checks two
// offending paths, so a fix that only special-cases a single reported line
// cannot pass.
func TestPurgePreconditionReportsFullPathForFirstLine(t *testing.T) {
	harness := newHarness(t)
	gitInit(t, harness)
	gitCommitAll(t, harness, "initial commit")

	guidancePath := harness.app.batonPath("GUIDANCE.md")
	if err := os.WriteFile(guidancePath, []byte("changed, not staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	extraPath := harness.app.batonPath("ZZZ-extra.md")
	if err := os.WriteFile(extraPath, []byte("untracked\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := harness.app.runGit("status", "--porcelain", "--untracked-files=all", "--", harness.app.BatonDir)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(status, "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "M ") {
		t.Fatalf("fixture did not produce the expected shifted first line: %q", lines)
	}

	commit, blocked, err := harness.app.purgePrecondition()
	if err != nil {
		t.Fatalf("purgePrecondition failed: %v", err)
	}
	if commit != "" {
		t.Fatalf("purgePrecondition reported a commit despite a dirty .baton/: %s", commit)
	}
	if len(blocked) != 2 {
		t.Fatalf("expected 2 blocked paths, got %d: %v", len(blocked), blocked)
	}

	joined := strings.Join(blocked, "\n")
	if strings.Contains(joined, "baton/GUIDANCE.md") && !strings.Contains(joined, ".baton/GUIDANCE.md") {
		t.Fatalf("first blocked path is missing its leading byte: %v", blocked)
	}
	if !strings.Contains(joined, ".baton/GUIDANCE.md") {
		t.Fatalf("blocked paths do not contain the full first path: %v", blocked)
	}
	if !strings.Contains(joined, ".baton/ZZZ-extra.md") {
		t.Fatalf("blocked paths do not contain the second path: %v", blocked)
	}
}

// TestWriteInstructionBlocksRestoresDeletedFileOnPartialFailure is Director
// emphasis #5: the rollback path writeInstructionBlocks provides must still
// hold when the operation is a delete rather than a rewrite. It constructs
// the same immutable-second-file failure
// TestWriteInstructionBlocksRestoresOnPartialFailure uses, but plans to
// delete the first file, and checks that a failed second write restores the
// first file exactly rather than leaving it deleted.
func TestWriteInstructionBlocksRestoresDeletedFileOnPartialFailure(t *testing.T) {
	chflags, err := exec.LookPath("chflags")
	if err != nil {
		t.Skip("chflags not available on this platform; cannot construct the real write failure portably")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores the immutable flag this test relies on")
	}

	dir := t.TempDir()
	file1 := filepath.Join(dir, "AGENTS.md")
	file2 := filepath.Join(dir, "CLAUDE.md")
	original1 := []byte("<baton-rules>\nold\n</baton-rules>\n")
	original2 := []byte("before\n<baton-rules>\nold\n</baton-rules>\nafter\n")
	if err := os.WriteFile(file1, original1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, original2, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command(chflags, "uchg", file2).Run(); err != nil {
		t.Fatalf("could not mark %s immutable: %v", file2, err)
	}
	defer exec.Command(chflags, "nouchg", file2).Run()

	// file1's planned content is nil (delete); file2's planned content is a
	// rewrite that will fail because it is immutable.
	writeErr := writeInstructionBlocks([]string{file1, file2}, [][]byte{nil, []byte("won't be written")})
	if writeErr == nil {
		t.Fatal("expected the second file's write to fail")
	}

	got1 := readFile(t, file1)
	if !bytes.Equal(got1, original1) {
		t.Fatalf("the first file was not restored after a delete-then-failure:\n%s", got1)
	}
	if err := exec.Command(chflags, "nouchg", file2).Run(); err != nil {
		t.Fatal(err)
	}
	got2 := readFile(t, file2)
	if !bytes.Equal(got2, original2) {
		t.Fatalf("the second file changed even though its write failed:\n%s", got2)
	}
}
