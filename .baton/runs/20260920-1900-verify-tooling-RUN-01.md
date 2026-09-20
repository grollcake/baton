# RUN-01: Make the two checks the protocol mandates into commands

Task ID: cqso
Date: 2026-09-20
Executor: Claude Sonnet 5
Plan: .baton/runs/20260920-1900-verify-tooling-PLAN.md
Status: complete

## Changes

- `internal/baton/revertcheck.go` (new): `App.runRevertCheck` implementing
  `baton revert-check --check '<command>' [--rev <ref>] [--name-pattern <re>]
  [--keep] PATH...`. Copies the project (tracked + untracked, `git ls-files -c
  -o --exclude-standard`, so ignored build output is excluded) into an
  `os.MkdirTemp` directory, runs `--check` there for a baseline, restores the
  named paths to their `--rev` content **in the copy only** (via `git show` /
  deletion for paths absent at `--rev`), runs `--check` again, and reports
  `copy=`, `rev=`, `reverted=`, `baseline=`, `after_revert=`, `pins=` (only
  with `--name-pattern`; `pins=unavailable (pass --name-pattern)` otherwise),
  and `new_output=` (capped at 200 lines, with `new_output_truncated=`). Exits
  non-zero only when it cannot answer: not a Git repository, an unresolved
  path, missing `--check`, or `baseline=fail` (stops before any revert). Every
  write or delete goes through one `confinedJoin` helper that refuses to
  resolve outside the copy root; positional paths are refused up front if
  absolute or escaping the repository, before any copy or Git call. The
  `--check` string is read only from `parsed.values["--check"]` (argv); the
  command never reads `BATON-LOG.txt`, `GUIDANCE.md`, a `PLAN`'s `Required
  Checks`, or any other file for it. The only four Git subcommands used are
  `rev-parse`, `ls-files`, `cat-file -e`, and `show`, all read-only, all run
  with `Dir` set to the real project root; `checkout`, `restore`, `stash`,
  `clean`, and `worktree` do not appear in the file (see Validation).
- `internal/baton/revertcheck_test.go` (new): the six tests the `PLAN`
  specifies, all against `t.TempDir()` scratch Git repositories, never this
  repository.
- `internal/baton/app.go`: one `case "revert-check"` in `App.Run` and one
  usage line under `lint`. Not added to `refusesWhilePaused`. `git diff
  --stat` on this file shows exactly 3 insertions, 0 deletions, 0 other
  hunks.
- `cmd/make-fixture/main.go` (new): repository-only `go run ./cmd/make-fixture
  [--binary working|released] [--no-git]`. Creates a throwaway installed
  project under `os.MkdirTemp` (honours `TMPDIR`), copies
  `bootstrap/.baton` (excluding `bin/`, which is installed separately),
  `bootstrap/AGENTS.md`, `bootstrap/CLAUDE.md`, seeds `BATON-LOG.txt` with the
  bootstrap `REQUEST`/`RUN_DONE` pair at a real timestamp, installs
  `.baton/bin/baton` (+ matching `SHA256SUMS`) either by building
  `./cmd/baton` from the working tree (`working`, default) or by copying
  `bootstrap/.baton/bin/<goos>-<goarch>/baton` (`released`), and by default
  commits everything to a fresh Git repository (`--no-git` skips this). No
  destination argument; prints `dir=`, `baton_dir=`, `binary=`,
  `binary_source=`, `version=`, `git=<dir> (commit <sha>)` or `git=none`,
  `drive=`, `cleanup=`. It cannot remove its own directory (the caller drives
  it after the process exits); this is the stated, weaker guarantee from
  Decision 2. Nothing in `internal/baton` imports it (it is a `main` package
  under `cmd/`), so it never ships to an installed project.
- `.baton/PLANNER.md` and `bootstrap/.baton/PLANNER.md`: the Review section's
  load-bearing-check paragraph is edited identically in both copies, same
  line count (3 lines, same as before), replacing "revert the source change
  on that copy, confirm which tests fail, and confirm they fail for the
  reason the change was made" with a reference to `baton revert-check`. The
  two files are byte-identical (`diff` confirms).
- `VERSION` / `.baton/VERSION`: **not touched**, per Director's change to the
  `PLAN` — Director bumps this at close alongside the release build.
- `.baton/GUIDANCE.md`: **not edited**, per Decision 6. Draft line for
  Director to append at close, if the user accepts:

  > Use `go run ./cmd/make-fixture` to get a throwaway installed Baton
  > project for hand-testing `update`, `pause`/`resume`, and `remove`; it is
  > repository-only and does not ship.

## Validation

- Evidence collected before fixing a user-reported defect: n/a (feature work,
  not a reported defect).
- `go build ./...`: clean, no output.
- `go vet ./...`: clean, no output.
- `gofmt -l .`: clean, no output.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton 12.667s`
  (all packages; `cmd/*` have no test files). All 6 new `revertcheck_test.go`
  tests pass individually (`-run TestRevertCheck -v`): `PASS`.
- `go run ./cmd/baton lint`: `baton-lint passed` (22 OK lines, including the
  managed-document comparison against `.baton/PLANNER.md`), run with `go run`
  so it reflects the working tree rather than the stale installed binary.

### Director emphasis 1 — the boundary that must survive intact

- `TestLintDoesNotExecutePaths` still passes unmodified:
  `go test ./internal/baton/... -run TestLintDoesNotExecutePaths -v` ->
  `--- PASS: TestLintDoesNotExecutePaths (0.13s)`.
- Confirmed it still fails when its guard is removed: temporarily deleted the
  `validateRequestPath` call in `lint.go`'s `REQUEST` branch (the actual guard
  a malicious `REQUEST` path is checked by), reran the same test:
  `--- FAIL: TestLintDoesNotExecutePaths` with `lint_test.go:23: lint accepted
  a malicious path`. Restored the file (`diff` against the pre-edit backup
  showed no difference) and reran: `PASS` again.
- The two guards this round adds were exercised the same way:
  - Confinement guard (`confinedJoin`'s escaping-path refusal): with the
    refusal removed, `TestRevertCheckRefusesEscapingPath` failed with `a
    refused path must not create a copy, got "copy=...`, i.e. the tool would
    have written outside its copy. Restored, test passes again.
  - No-mutating-subcommand guard (source grep):
    added `exec.Command("git", "checkout", ".")` into `runCheckCommand`,
    `TestRevertCheckNeverCheckoutsOrRunsFromRecords` failed with `revertcheck.go
    must not use a working-tree-mutating Git subcommand, found "checkout"`.
    Restored, test passes again.
- `baton revert-check`'s `--check` command comes from argv only: enforced by
  `parseArguments`/`requireValue(parsed, "--check")`, never from a record,
  artifact, `GUIDANCE.md`, or a `PLAN`'s `Required Checks`; a `PLAN` is where a
  reviewer *reads* the command, typing it is the step that keeps it from
  becoming an instruction executed automatically. Verified structurally by
  `TestRevertCheckNeverCheckoutsOrRunsFromRecords`, which greps the source for
  any reference to `BATON-LOG`, `timelineFile`, `GUIDANCE.md`, or
  `legacyTimelineFile` and fails if any is present (none are).

### Director emphasis 2 — cannot write outside its own temporary copy

Ran the real case in this repository, which is a Baton installation holding
today's real records:

- Snapshotted every tracked and untracked file under the repo (excluding
  `.git`) with `shasum -a 256` before: 365 files.
- Ran `go run ./cmd/baton revert-check --check 'go test ./internal/baton/...
  -run TestRevertCheck -v' --name-pattern '--- FAIL: (\S+)'
  internal/baton/app.go` (see emphasis 3 below for the real output).
- Re-snapshotted the same 365 files after: `diff before after` produced no
  output. The working tree, including `.baton/BATON-LOG.txt` and every other
  file under `.baton/`, is byte-identical before and after.
- The tool's own `copy=/var/folders/.../baton-revert-<n>` directory (printed
  by the run) did not exist afterward (`ls` on it: "No such file or
  directory"), confirming it cleaned up under `TMPDIR`, never under this
  repository.
- Also ran `go run ./cmd/make-fixture` (default, `--no-git`, and `--binary
  released`) three times in this repository: each printed `dir=` under
  `TMPDIR`, was driven directly with `BATON_DIR=<dir>/.baton <dir>/.baton/bin/baton
  <cmd>`, and each fixture directory was removed by hand afterward
  (`cleanup=rm -rf <dir>`, confirmed gone with `ls`/test `-d`). `git status
  --short` in this repository was unaffected by any of the three runs.

### Director emphasis 3 — real output naming which tests pin

Ran `baton revert-check` against this round's own new source, reverting
`internal/baton/app.go` (the file that wires the command in), with `--check
'go test ./internal/baton/... -run TestRevertCheck -v'` and `--name-pattern
'--- FAIL: (\S+)'`. Real output (copy path elided as `<copy>`, full usage text
elided as `<usage>`):

```
copy=<copy>
rev=2f1b1ad08290e9566ea3bf1d9641eeef5b1d01e5
reverted=internal/baton/app.go
baseline=pass
after_revert=fail
pins=TestRevertCheckReportsNewlyFailingTests
pins=TestRevertCheckStopsOnRedBaseline
pins=TestRevertCheckRemovesItsCopy
pins=TestRevertCheckDeletesPathAbsentAtRev
new_output=    revertcheck_test.go:83: revert-check failed: unknown command: revert-check
new_output=        output: <usage>
new_output=--- FAIL: TestRevertCheckReportsNewlyFailingTests (0.07s)
new_output=    revertcheck_test.go:120: expected baseline=fail, got "<usage>"
new_output=--- FAIL: TestRevertCheckStopsOnRedBaseline (0.05s)
new_output=    revertcheck_test.go:165: revert-check failed: unknown command: revert-check
new_output=--- FAIL: TestRevertCheckRemovesItsCopy (0.04s)
new_output=    revertcheck_test.go:242: revert-check failed: unknown command: revert-check
new_output=        output: <usage>
new_output=--- FAIL: TestRevertCheckDeletesPathAbsentAtRev (0.07s)
new_output=FAIL	github.com/grollcake/baton/internal/baton	0.648s
new_output=FAIL
```

Reading: 4 of the round's 6 new tests are pinned to `app.go`'s dispatch wiring
-- with it reverted they fail for the stated reason (`unknown command:
revert-check`), naming the exact line and message.
`TestRevertCheckRefusesEscapingPath` and
`TestRevertCheckNeverCheckoutsOrRunsFromRecords` are **not** pinned to
`app.go` and are reported as such rather than dropped: the escaping-path test
only asserts that *an* error occurs (an "unknown command" error satisfies
that too, so it is not evidence for `app.go` specifically -- it is pinned to
`confinedJoin` instead, shown separately above), and the never-checkouts test
is a static source grep that never calls `app.Run` at all, so it cannot be
pinned to any runtime wiring.

### Director emphasis 4 — `cmd/make-fixture` does not ship

- `grep -rn "make-fixture" internal/ bootstrap/`: no matches. Nothing in
  `internal/baton` imports it.
- It is a `main` package under `cmd/make-fixture/`, structurally identical in
  shape to `cmd/build-release`: a repository tool, not part of the
  `internal/baton` library the shipped `baton` binary is built from.
  `bootstrap/.baton/bin/` (what an install actually receives, verified by
  `go run ./cmd/baton lint`'s "no legacy scripts directory" and managed-file
  checks) contains no trace of it, and no code path in `internal/baton`
  references it.

### Director emphasis 5 — the `PLANNER.md` paragraph is edited, not extended

- `diff .baton/PLANNER.md bootstrap/.baton/PLANNER.md`: no output (byte
  identical).
- The paragraph is unchanged from 3 lines to 3 lines (line 32-34 before and
  after in both files); no other line in either file changed.

## Report To Director

- Suggested summary: Added `baton revert-check` (argv-only `--check`,
  temp-dir-confined writes, refuses a red baseline, reports which tests newly
  fail with verbatim output) and repository-only `cmd/make-fixture`
  (throwaway installed project for hand-testing `update`/`pause`/`remove`);
  edited the `PLANNER.md` load-bearing paragraph identically in both copies
  at the same line count; left `VERSION` and `GUIDANCE.md` untouched per
  Director's standing instruction and Decision 6.
- Artifact path: `.baton/runs/20260920-1900-verify-tooling-RUN-01.md`

## Success Criteria Status

- `revert-check` run in this repository reports the round's own new tests as
  pinned, writes nothing outside its temporary directory, and leaves nothing
  behind: met (see Director emphasis 2 and 3 above).
- `go run ./cmd/make-fixture` prints a fixture an agent can drive with no
  further setup: met (`lint`, `new-round`, `status`, `remove` dry run and
  `--apply --purge`, and `--binary released` + `update --apply` all exercised
  successfully; see Validation).
- Neither command writes anything outside its own temporary directory: met
  (365-file whole-tree checksum unchanged; `git status --short` unaffected by
  three `make-fixture` runs).
- No existing command changes behaviour: met (`app.go` diff is 3 insertions,
  0 deletions, confined to `Run`'s switch and `usage`; no other file in
  `internal/baton` besides the two new files was touched).
- `.baton/PLANNER.md` and `bootstrap/.baton/PLANNER.md` are byte-identical
  and the Review section's line count is unchanged: met.

## Unresolved Risks

- The `--binary released` fixture demonstrates the exact staleness this round
  exists to catch: because the release build is not run this round (per
  Director's explicit instruction), the previous release's embedded
  `PLANNER.md` differs from the working tree's edited `.baton/PLANNER.md`
  copied into the fixture, so `lint` reports drift both before and after
  `update --apply` (the binary itself is not rebuilt by `update`). This is
  the documented, expected state from the `PLAN`'s own risk list, not a
  defect; Director's rebuild at close resolves it for this repository's own
  installed binary.
- `revert-check` copies every tracked-and-unignored file for each run (142
  files in this repository, matching the `PLAN`'s estimate); no exclusion
  flag exists, by design, per the `PLAN`'s stated reasoning that a wrong
  exclusion would silently change what is measured.
- `make-fixture` cannot delete the directory it creates (Decision 2); the
  `cleanup=` line and the drafted `GUIDANCE.md` convention (above, not yet
  applied) are the whole mitigation pending user acceptance.

## Out Of Scope Returned To Director

- None. `VERSION` was explicitly withheld per Director's change to the `PLAN`;
  the `GUIDANCE.md` line is drafted above for Director to append at close if
  the user accepts, per Decision 6's design (not an ambiguity, the `PLAN`'s
  stated mechanism).
