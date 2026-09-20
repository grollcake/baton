package baton

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	docs "github.com/grollcake/baton"
)

func TestNewRoundCollisionBranchAndRandomErrors(t *testing.T) {
	t.Run("artifact key collision", func(t *testing.T) {
		harness := newHarness(t)
		collision := harness.app.batonPath("runs", "20260711-1001-collision-PLAN.md")
		if err := os.WriteFile(collision, []byte("collision"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := harness.fail("new-round", "collision", "--summary", "Collision"); err == nil {
			t.Fatal("new-round accepted an existing artifact key")
		}
	})

	t.Run("branch lifecycle", func(t *testing.T) {
		harness := newHarness(t)
		for _, args := range [][]string{
			{"init"},
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "Test"},
		} {
			if _, err := harness.app.runGit(args...); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(harness.app.ProjectDir, "README.md"), []byte("test\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := harness.app.runGit("add", "README.md"); err != nil {
			t.Fatal(err)
		}
		if _, err := harness.app.runGit("commit", "-m", "initial"); err != nil {
			t.Fatal(err)
		}
		if _, err := harness.app.runGit("branch", "exists"); err != nil {
			t.Fatal(err)
		}
		if err := harness.fail("new-round", "existing", "--summary", "Existing", "--branch", "exists"); err == nil {
			t.Fatal("new-round accepted an existing branch")
		}
		harness.run(t, "new-round", "feature", "--summary", "Feature", "--branch", "feature")
		branch, err := harness.app.runGit("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil || branch != "feature" {
			t.Fatalf("current branch %q, error %v", branch, err)
		}
	})

	t.Run("random source failure", func(t *testing.T) {
		harness := newHarness(t)
		harness.app.Random = errorReader{}
		if err := harness.fail("new-round", "random", "--summary", "Random"); err == nil {
			t.Fatal("new-round ignored a random source failure")
		}
	})

	t.Run("task id retries exhausted", func(t *testing.T) {
		harness := newHarness(t)
		records := []Record{{TaskID: "aaaa"}}
		harness.app.Random = zeroReader{}
		if _, err := harness.app.generateTaskID(records); err == nil {
			t.Fatal("task-id collision retries did not stop")
		}
	})
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type zeroReader struct{}

func (zeroReader) Read(target []byte) (int, error) {
	for index := range target {
		target[index] = 0
	}
	return len(target), nil
}

func TestAgentBlockLintBranches(t *testing.T) {
	active, err := docs.ActiveBlock()
	if err != nil {
		t.Fatal(err)
	}
	neither := "<baton-rules>\nneither active nor paused\n</baton-rules>"
	cases := []struct {
		name, agents, claude string
		wantErrors           int
	}{
		{"matching active", string(active), string(active), 0},
		{"matching paused", pausedBlock, pausedBlock, 0},
		{"agents missing", "plain", string(active), 1},
		{"claude missing", string(active), "plain", 1},
		{"different", string(active), pausedBlock, 1},
		{"matches neither", neither, neither, 1},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newHarness(t)
			if err := os.WriteFile(filepath.Join(harness.app.ProjectDir, "AGENTS.md"), []byte(testCase.agents), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(harness.app.ProjectDir, "CLAUDE.md"), []byte(testCase.claude), 0o644); err != nil {
				t.Fatal(err)
			}
			state := &lintState{app: harness.app}
			state.checkAgentBlocks()
			if state.errors != testCase.wantErrors {
				t.Fatalf("got %d errors, want %d", state.errors, testCase.wantErrors)
			}
		})
	}
}

func TestUpdateAndMergeErrorBranches(t *testing.T) {
	harness := newHarness(t)
	if err := harness.fail("update"); err == nil {
		t.Fatal("update accepted missing flags")
	}
	if err := harness.fail("update", "extra", "--upstream", t.TempDir()); err == nil {
		t.Fatal("update accepted positional arguments")
	}
	if err := harness.fail("update", "--upstream", t.TempDir()); err == nil {
		t.Fatal("update accepted an invalid upstream")
	}

	upstream := t.TempDir()
	if err := os.MkdirAll(filepath.Join(upstream, "bootstrap", ".baton"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("update", "--upstream", upstream); err == nil {
		t.Fatal("update accepted a missing upstream VERSION")
	}

	if err := os.Remove(harness.app.batonPath("VERSION")); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("update", "--upstream", upstream); err == nil {
		t.Fatal("update accepted a missing installed VERSION")
	}

	if !samePath(harness.app.ProjectDir, filepath.Join(harness.app.ProjectDir, ".")) || samePath(harness.app.ProjectDir, upstream) {
		t.Fatal("samePath returned an unexpected result")
	}

	if err := harness.fail("merge-agent-block"); err == nil {
		t.Fatal("merge accepted missing arguments")
	}
	missing := filepath.Join(harness.app.ProjectDir, "missing.md")
	if err := harness.fail("merge-agent-block", filepath.Join(harness.app.ProjectDir, "target.md"), missing); err == nil {
		t.Fatal("merge accepted a missing source")
	}
	invalid := filepath.Join(harness.app.ProjectDir, "invalid.md")
	if err := os.WriteFile(invalid, []byte("no block\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := harness.fail("merge-agent-block", filepath.Join(harness.app.ProjectDir, "target.md"), invalid); err == nil {
		t.Fatal("merge accepted a source without a Baton block")
	}
}

func TestChecksumParsingErrors(t *testing.T) {
	dir := t.TempDir()
	checksumFile := filepath.Join(dir, "SHA256SUMS")
	if err := os.WriteFile(checksumFile, []byte("invalid line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := checksumFor(checksumFile, "missing"); err == nil {
		t.Fatal("checksumFor accepted a missing entry")
	}
	if _, err := checksumFor(filepath.Join(dir, "missing"), "missing"); err == nil {
		t.Fatal("checksumFor accepted a missing file")
	}
	if err := verifyChecksum(filepath.Join(dir, "missing"), checksumFile, "missing"); err == nil {
		t.Fatal("verifyChecksum accepted missing inputs")
	}
	if err := atomicWrite(filepath.Join(dir, "atomic.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if content := string(readFile(t, filepath.Join(dir, "atomic.txt"))); strings.TrimSpace(content) != "ok" {
		t.Fatalf("unexpected atomic write content: %q", content)
	}
}
