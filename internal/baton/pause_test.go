package baton

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// addClaudeMD gives a harness project a CLAUDE.md that carries the same
// active block AGENTS.md does, so round-trip and refusal tests exercise both
// instruction files, the case Decision 7's rollback exists for.
func addClaudeMD(t *testing.T, harness *testHarness) {
	t.Helper()
	agents := readFile(t, filepath.Join(harness.app.ProjectDir, "AGENTS.md"))
	if err := os.WriteFile(filepath.Join(harness.app.ProjectDir, "CLAUDE.md"), agents, 0o644); err != nil {
		t.Fatal(err)
	}
}

// snapshotTree reads every regular file under root into a map keyed by its
// path relative to root, so a test can compare a whole project tree before
// and after an operation.
func snapshotTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = content
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestPauseResumeRoundTrip is V1: pause then resume must leave every file
// byte-identical to what it held before, except the timeline, which gains
// exactly the four REQUEST/RUN_DONE lines the two commands append and does
// not change any line that was already there.
func TestPauseResumeRoundTrip(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)

	before := snapshotTree(t, harness.app.ProjectDir)

	harness.run(t, "pause")
	harness.run(t, "resume")

	after := snapshotTree(t, harness.app.ProjectDir)

	logRel, err := filepath.Rel(harness.app.ProjectDir, filepath.Join(harness.app.BatonDir, timelineFile))
	if err != nil {
		t.Fatal(err)
	}

	for path, beforeContent := range before {
		if path == logRel {
			continue
		}
		afterContent, ok := after[path]
		if !ok {
			t.Fatalf("%s missing after round trip", path)
		}
		if !bytes.Equal(beforeContent, afterContent) {
			t.Fatalf("%s changed by the pause/resume round trip", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Fatalf("round trip created an unexpected file: %s", path)
		}
	}

	beforeLines := strings.Split(strings.TrimRight(string(before[logRel]), "\n"), "\n")
	afterLines := strings.Split(strings.TrimRight(string(after[logRel]), "\n"), "\n")
	if len(afterLines) != len(beforeLines)+4 {
		t.Fatalf("expected exactly 4 new timeline lines, got %d -> %d", len(beforeLines), len(afterLines))
	}
	for i, line := range beforeLines {
		if afterLines[i] != line {
			t.Fatalf("existing timeline line %d changed:\nbefore: %s\nafter:  %s", i, line, afterLines[i])
		}
	}
}

// TestPauseWritesTheExactPausedBlock checks the written blocks are
// byte-identical to each other and to Decision 2's text, and that resume
// restores the exact active block this binary ships.
func TestPauseWritesTheExactPausedBlock(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)
	activeBefore := readFile(t, filepath.Join(harness.app.ProjectDir, "AGENTS.md"))

	harness.run(t, "pause")
	agents := readFile(t, filepath.Join(harness.app.ProjectDir, "AGENTS.md"))
	claude := readFile(t, filepath.Join(harness.app.ProjectDir, "CLAUDE.md"))
	agentsBlock := batonBlockPattern.Find(agents)
	claudeBlock := batonBlockPattern.Find(claude)
	if string(agentsBlock) != pausedBlock {
		t.Fatalf("AGENTS.md block does not match Decision 2 text:\n%s", agentsBlock)
	}
	if string(claudeBlock) != pausedBlock {
		t.Fatalf("CLAUDE.md block does not match Decision 2 text:\n%s", claudeBlock)
	}

	harness.run(t, "resume")
	agentsAfter := readFile(t, filepath.Join(harness.app.ProjectDir, "AGENTS.md"))
	if !bytes.Equal(agentsAfter, activeBefore) {
		t.Fatalf("resume did not restore the exact active AGENTS.md content")
	}
}

// TestPauseRefusesDriftedBlock covers the "block matches neither shipped
// text" refusal: pausing over drift would make a later resume lossy.
func TestPauseRefusesDriftedBlock(t *testing.T) {
	harness := newHarness(t)
	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("<baton-rules>\nhand-edited\n</baton-rules>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, agentsPath)
	if err := harness.fail("pause"); err == nil {
		t.Fatal("pause accepted a drifted block")
	}
	after := readFile(t, agentsPath)
	if !bytes.Equal(before, after) {
		t.Fatal("a refused pause modified AGENTS.md")
	}
}

// TestResumeRefusesEditedPausedBlock covers "edited while paused": splicing
// the active block back in would silently destroy the edit.
func TestResumeRefusesEditedPausedBlock(t *testing.T) {
	harness := newHarness(t)
	harness.run(t, "pause")
	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("<baton-rules>\nedited while paused\n</baton-rules>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, agentsPath)
	if err := harness.fail("resume"); err == nil {
		t.Fatal("resume accepted a block edited while paused")
	}
	after := readFile(t, agentsPath)
	if !bytes.Equal(before, after) {
		t.Fatal("a refused resume modified AGENTS.md")
	}
}

// TestPauseAndResumeAreIdempotentNoOps: pausing an already-paused project, or
// resuming an already-active one, reports the state and writes nothing.
func TestPauseAndResumeAreIdempotentNoOps(t *testing.T) {
	harness := newHarness(t)
	logBefore := readFile(t, harness.app.batonPath(timelineFile))
	harness.run(t, "resume") // already active
	if !bytes.Equal(logBefore, readFile(t, harness.app.batonPath(timelineFile))) {
		t.Fatal("resume on an active project wrote to the timeline")
	}

	harness.run(t, "pause")
	logAfterPause := readFile(t, harness.app.batonPath(timelineFile))
	harness.run(t, "pause") // already paused
	if !bytes.Equal(logAfterPause, readFile(t, harness.app.batonPath(timelineFile))) {
		t.Fatal("pause on an already-paused project wrote to the timeline")
	}
}

// TestPauseOpenTaskRefusesThenForce covers Decision 5: an open task blocks
// pause by default, is named in the refusal, and --force proceeds and names
// it in the recorded summary.
func TestPauseOpenTaskRefusesThenForce(t *testing.T) {
	harness := newHarness(t)
	taskID, _ := parseRoundOutput(t, harness.run(t, "new-round", "open-task", "--summary", "Open work"))

	logBefore := readFile(t, harness.app.batonPath(timelineFile))
	if err := harness.fail("pause"); err == nil {
		t.Fatal("pause accepted an open task without --force")
	}
	if !bytes.Equal(logBefore, readFile(t, harness.app.batonPath(timelineFile))) {
		t.Fatal("a refused pause wrote to the timeline")
	}

	output := harness.run(t, "pause", "--force")
	if !strings.Contains(output, taskID) {
		t.Fatalf("forced pause summary does not name the open task: %s", output)
	}
	log := string(readFile(t, harness.app.batonPath(timelineFile)))
	if !strings.Contains(log, taskID) || !strings.Contains(log, "RUN_DONE") {
		t.Fatalf("forced pause did not record the open task in RUN_DONE: %s", log)
	}
}

// refusingCommands lists the plan's refuse-list commands other than
// merge-agent-block and update --apply, which have their own tests below
// because they carry the Decision 4a message instead of the generic one.
var refusingCommandsForTest = [][]string{
	{"append", "REQUEST", "--task-id", "boot", "--role", "Director", "--summary", "x"},
	{"new-round", "another-task", "--summary", "x"},
	{"feedback", "--task-id", "boot", "--summary", "x"},
	{"gate", "before-execute", "--task-id", "boot"},
	{"prompt", "plan", "--task-id", "boot", "--key", "20260711-1000-test"},
	{"prompt", "plan", "--task-id", "boot", "--key", "20260711-1000-test"},
	{"await", "RUN_DONE", ".baton/runs/x.md"},
}

// TestRefusalSweepWhilePaused is V3: every command in the refuse list exits
// non-zero and writes nothing to the timeline while the project is paused.
func TestRefusalSweepWhilePaused(t *testing.T) {
	harness := newHarness(t)
	harness.run(t, "pause")
	logBefore := readFile(t, harness.app.batonPath(timelineFile))

	for _, args := range refusingCommandsForTest {
		if err := harness.fail(args...); err == nil {
			t.Fatalf("%v was accepted while paused", args)
		} else if !strings.Contains(err.Error(), "paused") {
			t.Fatalf("%v refusal did not mention the pause: %v", args, err)
		}
	}

	target := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	source := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	if err := harness.fail("merge-agent-block", target, source); err == nil {
		t.Fatal("merge-agent-block was accepted while paused")
	}

	if err := harness.fail("update", "--upstream", t.TempDir(), "--apply"); err == nil {
		t.Fatal("update --apply was accepted while paused")
	}

	logAfter := readFile(t, harness.app.batonPath(timelineFile))
	if !bytes.Equal(logBefore, logAfter) {
		t.Fatal("the refusal sweep wrote to the timeline")
	}
}

// TestUpdateDryRunAllowedWhilePausedAndMentionsResume covers Decision 4a:
// update without --apply still runs and its output points at "baton resume"
// so a paused project learns the detour before it tries to update.
func TestUpdateDryRunAllowedWhilePausedAndMentionsResume(t *testing.T) {
	harness := newHarness(t)
	harness.run(t, "pause")
	upstream := t.TempDir()
	if err := os.MkdirAll(filepath.Join(upstream, "bootstrap", ".baton"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(upstream, "VERSION"), []byte("9.9.9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := harness.run(t, "update", "--upstream", upstream)
	if !strings.Contains(output, "baton resume") {
		t.Fatalf("paused dry run does not mention baton resume: %s", output)
	}
}

// TestUpdateApplyRefusesWhilePaused is the regression test REVIEW-01's B1
// asked for: update --apply is the one command that rewrites the rules
// block, so its pause refusal must be under test. It must exit non-zero and
// leave AGENTS.md, CLAUDE.md, and .baton/VERSION exactly as they were.
func TestUpdateApplyRefusesWhilePaused(t *testing.T) {
	harness := newHarness(t)
	addClaudeMD(t, harness)
	harness.run(t, "pause")

	upstream := t.TempDir()
	if err := os.MkdirAll(filepath.Join(upstream, "bootstrap", ".baton"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(upstream, "VERSION"), []byte("9.9.9\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	agentsBefore := readFile(t, filepath.Join(harness.app.ProjectDir, "AGENTS.md"))
	claudeBefore := readFile(t, filepath.Join(harness.app.ProjectDir, "CLAUDE.md"))
	versionBefore := readFile(t, harness.app.batonPath("VERSION"))

	err := harness.fail("update", "--upstream", upstream, "--apply")
	if err == nil {
		t.Fatal("update --apply was accepted while paused")
	}
	if !strings.Contains(err.Error(), "paused") {
		t.Fatalf("refusal did not mention the pause: %v", err)
	}

	if !bytes.Equal(agentsBefore, readFile(t, filepath.Join(harness.app.ProjectDir, "AGENTS.md"))) {
		t.Fatal("update --apply changed AGENTS.md despite refusing")
	}
	if !bytes.Equal(claudeBefore, readFile(t, filepath.Join(harness.app.ProjectDir, "CLAUDE.md"))) {
		t.Fatal("update --apply changed CLAUDE.md despite refusing")
	}
	if !bytes.Equal(versionBefore, readFile(t, harness.app.batonPath("VERSION"))) {
		t.Fatal("update --apply changed .baton/VERSION despite refusing")
	}
}

// TestLintReportsPause is V4: lint passes and reports the pause on a paused
// project, and fails with the "matches neither" error once a block is
// hand-edited away from the shipped paused text.
func TestLintReportsPause(t *testing.T) {
	harness := newHarness(t)
	harness.run(t, "pause")
	output := harness.run(t, "lint")
	if !strings.Contains(output, "Baton is paused") {
		t.Fatalf("lint on a paused project did not report the pause: %s", output)
	}

	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("<baton-rules>\nhand-edited\n</baton-rules>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("lint"); err == nil {
		t.Fatal("lint passed over a block that matches neither shipped text")
	}
	if !strings.Contains(harness.err.String(), "matches neither") {
		t.Fatalf("lint error does not name the drifted block: %s", harness.err.String())
	}
}

// TestWriteInstructionBlocksRestoresOnPartialFailure constructs the failure
// Director asked for directly, in the shape instructionFiles() actually
// produces: both files in the same directory (the project root), as
// AGENTS.md and CLAUDE.md always are. A separate-directories shape cannot
// happen here and does not fail atomicWrite the same way (a temp file still
// renames cleanly within a writable directory), so this marks the second
// file immutable with chflags uchg instead -- the one thing that makes
// os.Rename fail onto an existing target while its directory stays
// writable. The first file, already written, must come back exactly as it
// was, so the two instruction files are never left disagreeing because of a
// partial pause or resume. Skipped where chflags is unavailable (chflags is
// BSD/Darwin-only; there is no portable equivalent -- Linux's chattr +i
// needs root, which changes what the test proves) or where root would
// ignore the flag anyway.
func TestWriteInstructionBlocksRestoresOnPartialFailure(t *testing.T) {
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
	original := []byte("before\n<baton-rules>\nold\n</baton-rules>\nafter\n")
	if err := os.WriteFile(file1, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command(chflags, "uchg", file2).Run(); err != nil {
		t.Fatalf("could not mark %s immutable: %v", file2, err)
	}
	defer exec.Command(chflags, "nouchg", file2).Run()

	content, err := spliceBlockIntoFiles([]string{file1, file2}, []byte(pausedBlock))
	if err != nil {
		t.Fatal(err)
	}
	writeErr := writeInstructionBlocks([]string{file1, file2}, content)
	if writeErr == nil {
		t.Fatal("expected the second file's write to fail")
	}

	got1 := readFile(t, file1)
	if !bytes.Equal(got1, original) {
		t.Fatalf("the first file was not restored after the second write failed:\n%s", got1)
	}
	if err := exec.Command(chflags, "nouchg", file2).Run(); err != nil {
		t.Fatal(err)
	}
	got2 := readFile(t, file2)
	if !bytes.Equal(got2, original) {
		t.Fatalf("the second file changed even though its write failed:\n%s", got2)
	}
}

// TestLintClaudeOnlyProject closes gap A: a project with only CLAUDE.md still
// has its block checked, rather than lint returning silently.
func TestLintClaudeOnlyProject(t *testing.T) {
	harness := newHarness(t)
	agentsPath := filepath.Join(harness.app.ProjectDir, "AGENTS.md")
	claudePath := filepath.Join(harness.app.ProjectDir, "CLAUDE.md")
	if err := os.Rename(agentsPath, claudePath); err != nil {
		t.Fatal(err)
	}
	state := &lintState{app: harness.app}
	state.checkAgentBlocks()
	if state.errors != 0 {
		t.Fatalf("Claude-only project with an active block should pass, got %d errors", state.errors)
	}

	if err := os.WriteFile(claudePath, []byte("<baton-rules>\nhand-edited\n</baton-rules>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	state = &lintState{app: harness.app}
	state.checkAgentBlocks()
	if state.errors == 0 {
		t.Fatal("Claude-only project with a drifted block should fail")
	}
}
