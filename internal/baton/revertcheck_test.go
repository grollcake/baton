package baton

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// newRevertProject creates a scratch Git repository (never this repository)
// for revert-check tests, skipping when git is unavailable.
func newRevertProject(t *testing.T) (*App, *bytes.Buffer) {
	t.Helper()
	project := t.TempDir()
	app := New(project, filepath.Join(project, ".baton"))
	var out, errBuf bytes.Buffer
	app.Stdout = &out
	app.Stderr = &errBuf
	if _, err := app.runGit("init"); err != nil {
		t.Skipf("git unavailable: %s", err)
	}
	if _, err := app.runGit("config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.runGit("config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	return app, &out
}

func writeProjectFile(t *testing.T, app *App, relative, content string) {
	t.Helper()
	path := app.projectPath(relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func revertCommit(t *testing.T, app *App, message string) {
	t.Helper()
	if _, err := app.runGit("add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.runGit("commit", "-m", message); err != nil {
		t.Fatal(err)
	}
}

func outputValues(output string) map[string][]string {
	values := map[string][]string{}
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[parts[0]] = append(values[parts[0]], parts[1])
	}
	return values
}

// TestRevertCheckReportsNewlyFailingTests covers the round's core promise:
// a source change that fixes a check is reported as pinned by the test that
// only fails once the fix is reverted in the copy.
func TestRevertCheckReportsNewlyFailingTests(t *testing.T) {
	app, out := newRevertProject(t)
	writeProjectFile(t, app, "calc.txt", "return_old\n")
	revertCommit(t, app, "committed buggy state")
	writeProjectFile(t, app, "calc.txt", "return_new\n")

	check := "grep -q return_new calc.txt || { echo '--- FAIL: TestCalcFixed'; exit 1; }"
	err := app.Run([]string{
		"revert-check",
		"--check", check,
		"--name-pattern", `--- FAIL: (\S+)`,
		"calc.txt",
	})
	if err != nil {
		t.Fatalf("revert-check failed: %v\noutput: %s", err, out.String())
	}
	values := outputValues(out.String())
	if got := values["baseline"]; len(got) != 1 || got[0] != "pass" {
		t.Fatalf("expected baseline=pass, got %q", out.String())
	}
	if got := values["after_revert"]; len(got) != 1 || got[0] != "fail" {
		t.Fatalf("expected after_revert=fail, got %q", out.String())
	}
	if got := values["pins"]; len(got) != 1 || got[0] != "TestCalcFixed" {
		t.Fatalf("expected pins=TestCalcFixed, got %q", out.String())
	}
	if copies := values["copy"]; len(copies) != 1 {
		t.Fatalf("expected exactly one copy= line, got %q", out.String())
	} else if _, statErr := os.Stat(copies[0]); !os.IsNotExist(statErr) {
		t.Fatalf("copy directory %s should have been removed", copies[0])
	}
}

// TestRevertCheckStopsOnRedBaseline covers Decision 1's exit rule: a copy
// whose check already fails stops before any revert and is reported rather
// than compared against.
func TestRevertCheckStopsOnRedBaseline(t *testing.T) {
	app, out := newRevertProject(t)
	writeProjectFile(t, app, "calc.txt", "return_old\n")
	revertCommit(t, app, "committed state")

	err := app.Run([]string{
		"revert-check",
		"--check", "exit 1",
		"calc.txt",
	})
	if err == nil {
		t.Fatalf("expected an error for a red baseline, output: %s", out.String())
	}
	values := outputValues(out.String())
	if got := values["baseline"]; len(got) != 1 || got[0] != "fail" {
		t.Fatalf("expected baseline=fail, got %q", out.String())
	}
	if _, ok := values["after_revert"]; ok {
		t.Fatalf("a red baseline must not be followed by a revert, got %q", out.String())
	}
	if _, ok := values["pins"]; ok {
		t.Fatalf("no pins should be reported without a second run, got %q", out.String())
	}
}

// TestRevertCheckRefusesEscapingPath covers Decision 3's confinement: a path
// that could resolve outside the copy is refused before any copy is made or
// any file is touched.
func TestRevertCheckRefusesEscapingPath(t *testing.T) {
	app, out := newRevertProject(t)
	writeProjectFile(t, app, "calc.txt", "return_old\n")
	revertCommit(t, app, "committed state")

	cases := [][]string{
		{"revert-check", "--check", "true", "../outside.txt"},
		{"revert-check", "--check", "true", "/etc/passwd"},
	}
	for _, args := range cases {
		out.Reset()
		if err := app.Run(args); err == nil {
			t.Fatalf("expected %v to be refused", args)
		}
		values := outputValues(out.String())
		if _, ok := values["copy"]; ok {
			t.Fatalf("a refused path must not create a copy, got %q", out.String())
		}
	}
}

// TestRevertCheckRemovesItsCopy covers Decision 2: the copy is removed on
// both a successful and a failing run, and kept only with --keep.
func TestRevertCheckRemovesItsCopy(t *testing.T) {
	app, out := newRevertProject(t)
	writeProjectFile(t, app, "calc.txt", "return_old\n")
	revertCommit(t, app, "committed state")
	writeProjectFile(t, app, "calc.txt", "return_new\n")

	check := "grep -q return_new calc.txt"

	if err := app.Run([]string{"revert-check", "--check", check, "calc.txt"}); err != nil {
		t.Fatalf("revert-check failed: %v", err)
	}
	values := outputValues(out.String())
	copyDir := values["copy"][0]
	if _, err := os.Stat(copyDir); !os.IsNotExist(err) {
		t.Fatalf("copy %s should be removed after a successful run", copyDir)
	}

	out.Reset()
	if err := app.Run([]string{"revert-check", "--check", "exit 1", "calc.txt"}); err == nil {
		t.Fatal("expected a red baseline error")
	}
	values = outputValues(out.String())
	copyDir = values["copy"][0]
	if _, err := os.Stat(copyDir); !os.IsNotExist(err) {
		t.Fatalf("copy %s should be removed after a failing run", copyDir)
	}

	out.Reset()
	if err := app.Run([]string{"revert-check", "--check", check, "--keep", "calc.txt"}); err != nil {
		t.Fatalf("revert-check --keep failed: %v", err)
	}
	values = outputValues(out.String())
	copyDir = values["copy"][0]
	defer os.RemoveAll(copyDir)
	if _, err := os.Stat(copyDir); err != nil {
		t.Fatalf("copy %s should be kept with --keep: %v", copyDir, err)
	}
}

// TestRevertCheckNeverCheckoutsOrRunsFromRecords pins the boundary Director
// called out: revertcheck.go must never call a Git subcommand that could
// touch the working tree, and the checked command must come only from argv.
func TestRevertCheckNeverCheckoutsOrRunsFromRecords(t *testing.T) {
	root := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(root, "internal", "baton", "revertcheck.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)

	forbidden := regexp.MustCompile(`"(checkout|restore|stash|clean|worktree)"`)
	if match := forbidden.FindString(text); match != "" {
		t.Fatalf("revertcheck.go must not use a working-tree-mutating Git subcommand, found %s", match)
	}

	for _, banned := range []string{"BATON-LOG", "timelineFile", "GUIDANCE.md", "legacyTimelineFile"} {
		if strings.Contains(text, banned) {
			t.Fatalf("revertcheck.go must not read the check command from %s", banned)
		}
	}

	if !strings.Contains(text, `requireValue(parsed, "--check")`) {
		t.Fatal("revertcheck.go must read --check from the parsed arguments")
	}
}

// TestRevertCheckDeletesPathAbsentAtRev covers Decision 3's last case: a path
// that did not exist at --rev is deleted from the copy rather than fetched.
func TestRevertCheckDeletesPathAbsentAtRev(t *testing.T) {
	app, out := newRevertProject(t)
	writeProjectFile(t, app, "a.txt", "a\n")
	revertCommit(t, app, "only a.txt")
	oldRev, err := app.runGit("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	writeProjectFile(t, app, "b.txt", "b\n")
	revertCommit(t, app, "add b.txt")

	if err := app.Run([]string{
		"revert-check",
		"--check", "true",
		"--rev", oldRev,
		"--keep",
		"a.txt", "b.txt",
	}); err != nil {
		t.Fatalf("revert-check failed: %v\noutput: %s", err, out.String())
	}
	values := outputValues(out.String())
	copyDir := values["copy"][0]
	defer os.RemoveAll(copyDir)
	if _, err := os.Stat(filepath.Join(copyDir, "b.txt")); !os.IsNotExist(err) {
		t.Fatal("b.txt did not exist at --rev and should have been deleted from the copy")
	}
	if _, err := os.Stat(filepath.Join(copyDir, "a.txt")); err != nil {
		t.Fatalf("a.txt should still be present in the copy: %v", err)
	}
}

// TestRevertCheckPinsNoneWhenNothingNewlyFails covers REVIEW-01's first
// blocker: when --name-pattern is supplied but the revert changes nothing
// the check notices, the command must say so explicitly rather than
// printing no pins= line at all.
func TestRevertCheckPinsNoneWhenNothingNewlyFails(t *testing.T) {
	app, out := newRevertProject(t)
	writeProjectFile(t, app, "calc.txt", "unrelated\n")
	revertCommit(t, app, "committed state")

	err := app.Run([]string{
		"revert-check",
		"--check", "true",
		"--name-pattern", `--- FAIL: (\S+)`,
		"calc.txt",
	})
	if err != nil {
		t.Fatalf("revert-check failed: %v\noutput: %s", err, out.String())
	}
	values := outputValues(out.String())
	if got := values["after_revert"]; len(got) != 1 || got[0] != "pass" {
		t.Fatalf("expected after_revert=pass, got %q", out.String())
	}
	if got := values["pins"]; len(got) != 1 || got[0] != "none" {
		t.Fatalf("expected exactly one pins=none line, got %q", out.String())
	}
}

// TestRevertCheckReportsReasonWhenRevertBreaksTheBuild covers REVIEW-01's
// first blocker in the exact shape the reviewer reached it: the reverted
// path stops the check from compiling, so no line matches --name-pattern.
// The command must say what happened rather than printing no pins= line.
func TestRevertCheckReportsReasonWhenRevertBreaksTheBuild(t *testing.T) {
	app, out := newRevertProject(t)
	writeProjectFile(t, app, "helper.txt", "old\n")
	revertCommit(t, app, "committed state")
	// helper.txt did not exist before this commit is what the check reads;
	// simulate "the reverted path is required by the check" by having the
	// check fail differently once the file is gone -- reverting a path
	// absent at --rev deletes it from the copy (Decision 3's last case),
	// which is exactly the "package no longer compiles" shape when the
	// check depends on the file's presence.
	oldRev, err := app.runGit("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	writeProjectFile(t, app, "new.txt", "present\n")
	revertCommit(t, app, "add new.txt")

	check := "test -f new.txt || { echo 'build failed: missing new.txt'; exit 1; }"
	err = app.Run([]string{
		"revert-check",
		"--check", check,
		"--rev", oldRev,
		"--name-pattern", `--- FAIL: (\S+)`,
		"new.txt",
	})
	if err != nil {
		t.Fatalf("revert-check failed: %v\noutput: %s", err, out.String())
	}
	values := outputValues(out.String())
	if got := values["after_revert"]; len(got) != 1 || got[0] != "fail" {
		t.Fatalf("expected after_revert=fail, got %q", out.String())
	}
	if got := values["pins"]; len(got) != 1 || !strings.Contains(got[0], "after_revert failed") {
		t.Fatalf("expected a pins= line stating the run could not name a test, got %q", out.String())
	}
}
