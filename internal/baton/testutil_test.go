package baton

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type countingReader struct {
	next byte
}

func (reader *countingReader) Read(target []byte) (int, error) {
	for index := range target {
		target[index] = reader.next
		reader.next++
	}
	return len(target), nil
}

type testHarness struct {
	app *App
	out bytes.Buffer
	err bytes.Buffer
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func copyTree(source, target string, skip map[string]bool) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(target, 0o755)
		}
		first := strings.Split(relative, string(filepath.Separator))[0]
		if skip[first] {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return atomicWrite(destination, content, info.Mode().Perm())
	})
}

func newHarness(t *testing.T) *testHarness {
	t.Helper()
	root := repositoryRoot(t)
	project := t.TempDir()
	batonDir := filepath.Join(project, ".baton")
	if err := copyTree(filepath.Join(root, "bootstrap", ".baton"), batonDir, map[string]bool{"scripts": true, "bin": true}); err != nil {
		t.Fatal(err)
	}
	agents, err := os.ReadFile(filepath.Join(root, "bootstrap", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "AGENTS.md"), agents, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(batonDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(batonDir, "bin", binaryName(runtime.GOOS))
	if err := os.WriteFile(binary, []byte("test binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := fileSHA256(binary)
	if err != nil {
		t.Fatal(err)
	}
	checksum := fmt.Sprintf("%s  %s\n", digest, filepath.ToSlash(filepath.Join(runtime.GOOS+"-"+runtime.GOARCH, binaryName(runtime.GOOS))))
	if err := os.WriteFile(filepath.Join(batonDir, "bin", "SHA256SUMS"), []byte(checksum), 0o644); err != nil {
		t.Fatal(err)
	}
	writeLog(t, batonDir,
		"2026-07-11T10:00:00 | boot | REQUEST  | Director | Bootstrap Baton",
		"2026-07-11T10:00:00 | boot | RUN_DONE | Director | Baton initialized",
	)
	harness := &testHarness{}
	harness.app = New(project, batonDir)
	harness.app.ConfigDir = filepath.Join(project, "user-config")
	harness.app.Stdout = &harness.out
	harness.app.Stderr = &harness.err
	harness.app.Now = func() time.Time { return time.Date(2026, 7, 11, 10, 1, 2, 0, time.Local) }
	harness.app.Random = &countingReader{}
	return harness
}

func writeLog(t *testing.T, batonDir string, lines ...string) {
	t.Helper()
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(filepath.Join(batonDir, "baton.log"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (harness *testHarness) run(t *testing.T, args ...string) string {
	t.Helper()
	harness.out.Reset()
	harness.err.Reset()
	if err := harness.app.Run(args); err != nil {
		t.Fatalf("baton %s failed: %v\nstderr: %s", strings.Join(args, " "), err, harness.err.String())
	}
	return harness.out.String()
}

func (harness *testHarness) fail(args ...string) error {
	harness.out.Reset()
	harness.err.Reset()
	return harness.app.Run(args)
}

func parseRoundOutput(t *testing.T, output string) (string, string) {
	t.Helper()
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			values[parts[0]] = strings.Trim(parts[1], "'")
		}
	}
	if values["task_id"] == "" || values["key"] == "" {
		t.Fatalf("invalid new-round output: %q", output)
	}
	return values["task_id"], values["key"]
}

func writeArtifact(t *testing.T, project, path, content string) {
	t.Helper()
	fullPath := filepath.Join(project, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePlan(t *testing.T, project, path, taskID string) {
	t.Helper()
	writeArtifact(t, project, path, fmt.Sprintf(`# PLAN: Test flow
Task ID: %s
Date: 2026-07-11
Planner: test
Status: complete
## Director Brief
- Goal: validate the flow
## Success Criteria
- The test passes.
## Validation
- Run the test.
`, taskID))
}

func writeRun(t *testing.T, project, path, taskID, round string) {
	t.Helper()
	writeArtifact(t, project, path, fmt.Sprintf(`# RUN-%s: Test flow
Task ID: %s
Date: 2026-07-11
Executor: test
Status: complete
## Changes
- Touched the test flow.
## Validation
- Test passed.
## Success Criteria Status
- Flow: met
## Unresolved Risks
- None.
`, round, taskID))
}

func writeReview(t *testing.T, project, path, taskID, round, result string) {
	t.Helper()
	writeArtifact(t, project, path, fmt.Sprintf(`# REVIEW-%s: Test flow
Task ID: %s
Date: 2026-07-11
Planner: test
Status: complete
Result: %s
## Suggested User Checks
- Inspect the output.
- Rerun the flow.
- Confirm the artifact path.
## Evidence Reviewed
- Test output.
`, round, taskID, result))
}

func writeClose(t *testing.T, project, path, taskID string) {
	t.Helper()
	writeArtifact(t, project, path, fmt.Sprintf(`# CLOSE: Test flow
Task ID: %s
Date: 2026-07-11
Director: test
Approved By: User
## Acceptance
- Accepted.
## Validation Summary
- Test passed.
`, taskID))
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func mustCopy(t *testing.T, source, target string, mode os.FileMode) {
	t.Helper()
	if err := copyFile(source, target, mode); err != nil {
		t.Fatal(err)
	}
}

func discardWriter() io.Writer {
	return io.Discard
}
