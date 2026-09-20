# RUN-02: Make the two checks the protocol mandates into commands

Task ID: cqso
Date: 2026-09-20
Executor: Claude Sonnet 5
Plan: .baton/runs/20260920-1900-verify-tooling-PLAN.md
Status: complete

This round answers REVIEW-01's three blockers only. RUN-01 and REVIEW-01 are
unmodified. Both boundaries REVIEW-01 confirmed (no working-tree writes, `--check`
reachable only from argv) are untouched by this round's changes; the fixes
below are all in the answer `revert-check` gives and where it can run.

## Changes

- `internal/baton/revertcheck.go` (`pins=` output, blocker 1): the pins
  section now always prints at least one `pins=` line.
  - A match found: unchanged, one `pins=<name>` line per distinct name, in
    first-seen order.
  - `--name-pattern` given, no line matched, and `after_revert=pass`: `pins=none`
    -- a true negative (nothing newly failed, so nothing to name).
  - `--name-pattern` given, no line matched, and `after_revert=fail`: `pins=none
    (after_revert failed but no line matched --name-pattern; see new_output)`
    -- the case REVIEW-01 reached with the `PLAN`'s own success-criterion
    command: the reverted file didn't exist at `HEAD`, the package stopped
    compiling, and the old code printed no `pins=` line at all. It now says
    plainly that the run could not name a test, rather than going quiet.
  - No `--name-pattern`: unchanged, `pins=unavailable (pass --name-pattern)`.
- `internal/baton/revertcheck.go` (shell, blocker 3): `runCheckCommand` no
  longer hardcodes `sh -c`. It now runs `cmd /C <command>` on
  `runtime.GOOS == "windows"` and `sh -c <command>` everywhere else, so the
  shipped `windows-amd64`/`windows-arm64` binaries do not require `sh` on
  `PATH` -- matching `PROTOCOL.md`'s "does not require ... a specific shell".
  The `--check` string's own syntax is still whichever shell runs it (a
  POSIX-shell `--check` string will not parse under `cmd.exe`, and vice
  versa); that is inherent to what `--check` is, not a gap this round can
  close, and is not what REVIEW-01 flagged -- REVIEW-01's objection was that
  the binary failed outright with `exec: "sh": executable file not found` on
  a platform it ships to, which no longer happens.
- `internal/baton/app.go` (usage, blocker 2): the `revert-check` usage line
  now carries the example the `PLAN` promised:
  ```
  revert-check --check <command> [--rev <ref>] [--name-pattern <re>] [--keep] PATH...
      (e.g. --name-pattern '--- FAIL: (\S+)' for go test)
  ```
  `git diff --stat` on this file is now 4 insertions (was 3 in RUN-01; the one
  added line is the example), still confined to `Run`'s switch and `usage`.
- `.baton/PLANNER.md` and `bootstrap/.baton/PLANNER.md` (blocker 2): the
  load-bearing-check paragraph now names `--name-pattern` with the same Go
  example, and restores "confirm they fail for the reason the change was
  made", pointing at `new_output=` as where that reason now lives (both
  points REVIEW-01's nit and blocker raised). The paragraph grew from 3 lines
  to 5 lines in both copies to carry this; the two files remain
  byte-identical (`diff` confirms, below).
- `internal/baton/revertcheck_test.go`: two new tests pinning this round's
  fix --
  `TestRevertCheckPinsNoneWhenNothingNewlyFails` (a revert that changes
  nothing the check notices reports `pins=none`) and
  `TestRevertCheckReportsReasonWhenRevertBreaksTheBuild` (a revert that makes
  the check fail with no line matching `--name-pattern` reports the stated
  reason, not silence).

Not touched, per Director's instruction: `VERSION`, `.baton/VERSION`,
`.baton/bin/`. Not touched, per Director's "leave the nits" instruction: the
source-grep guard's narrower-than-presented coverage, the raw rename error on
a directory argument, and `copy=` printing after the baseline run -- Director
records these at close.

## Validation

- `go build ./...`: clean, no output.
- `go vet ./...`: clean, no output.
- `gofmt -l .`: clean, no output.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton 13.179s`
  (all packages; `cmd/*` have no test files). All 8
  `revertcheck_test.go` tests pass individually
  (`-run TestRevertCheck -v`): `PASS`. `TestLintDoesNotExecutePaths` still
  passes unmodified (re-run in this round, not just RUN-01): `PASS`.
- `go run ./cmd/baton lint`: `baton-lint passed`, run with `go run` so it
  reflects the working tree.

### A real run showing `pins=none`

```
$ go run ./cmd/baton revert-check --check 'true' \
    --name-pattern '--- FAIL: (\S+)' VERSION
copy=<copy>
rev=2f1b1ad08290e9566ea3bf1d9641eeef5b1d01e5
reverted=VERSION
baseline=pass
after_revert=pass
pins=none
```

### A real run showing the build-failure case (REVIEW-01's exact reproduction)

Reran the `PLAN`'s own success-criterion command, unchanged, in this
repository:

```
$ go run ./cmd/baton revert-check --check 'go test ./internal/baton/...' \
    --name-pattern '--- FAIL: (\S+)' internal/baton/revertcheck.go
copy=<copy>
rev=2f1b1ad08290e9566ea3bf1d9641eeef5b1d01e5
reverted=internal/baton/revertcheck.go
baseline=pass
after_revert=fail
pins=none (after_revert failed but no line matched --name-pattern; see new_output)
new_output=# github.com/grollcake/baton/internal/baton [github.com/grollcake/baton/internal/baton.test]
new_output=internal/baton/app.go:143:12: a.runRevertCheck undefined (type *App has no field or method runRevertCheck)
new_output=FAIL	github.com/grollcake/baton/internal/baton [build failed]
new_output=FAIL
```

Before, this printed no `pins=` line at all. It now says what happened.

### Boundary re-confirmed after both real runs above

- Both copy directories (`baton-revert-3078548975`, `baton-revert-925300584`)
  were absent afterward (`ls`: "No such file or directory").
- Snapshotted every tracked and untracked file under the repo (excluding
  `.git`) with `shasum -a 256` before both runs above (367 files) and after:
  `diff before after` produced no output. The working tree, including
  `.baton/BATON-LOG.txt` and every other file under `.baton/`, is
  byte-identical.
- `git status --short`: unaffected by either run (only this round's intended
  edits and RUN-01/REVIEW-01's pre-existing untracked artifacts appear).

### `PLANNER.md` byte-identity

- `diff .baton/PLANNER.md bootstrap/.baton/PLANNER.md`: no output (byte
  identical).

### `VERSION` / `.baton/bin/` untouched

- `git diff --stat -- VERSION .baton/VERSION .baton/bin/`: no output.
- `git status --short -- .baton/bin/ VERSION .baton/VERSION`: no output.

## Report To Director

- Suggested summary: Fixed REVIEW-01's three blockers -- `revert-check` now
  always prints a `pins=` line (`none`, or a stated reason when the run could
  not name a test, most concretely the case where the reverted path breaks
  the build), the `--name-pattern` Go example is now in both the usage line
  and the `PLANNER.md` paragraph (which also restores "fail for the reason
  the change was made"), and `runCheckCommand` no longer hardcodes `sh -c`,
  running `cmd /C` on Windows instead. Both safety boundaries from RUN-01
  remain intact and were re-demonstrated on the real repository. `VERSION`
  and `.baton/bin/` untouched; the three named nits left for Director's close.
- Artifact path: `.baton/runs/20260920-1900-verify-tooling-RUN-02.md`

## Success Criteria Status

- Blocker 1 (`pins=` silence): met -- always prints, distinguishing a true
  negative from a run that could not name a test.
- Blocker 2 (missing usage example / `PLANNER.md` flag name): met -- both
  copies now name `--name-pattern` with the Go example, and the "fail for the
  reason" language is restored.
- Blocker 3 (`sh -c` hardcoded): met -- platform-selected interpreter,
  `cmd /C` on Windows.
- Both safety boundaries (no working-tree writes, argv-only `--check`): met,
  unchanged from RUN-01, re-demonstrated above.
- `VERSION` and `.baton/bin/` untouched: met.

## Unresolved Risks

- `cmd /C <command>` on Windows means a `--check` string written in POSIX
  shell syntax (`||`, `{ ; }`, etc.) will not run there, and vice versa for a
  `cmd.exe`-syntax string on Unix. This is unavoidable given `--check` is a
  single string handed to *some* interpreter; it is now at least the
  platform's own always-present interpreter rather than an absent one. Not
  exercised on an actual Windows machine in this round (none available);
  the change is a straightforward `runtime.GOOS` branch with no other
  platform-specific logic.
- The three nits REVIEW-01 raised (narrower-than-presented source grep, raw
  rename error for a directory argument, `copy=` timing under `--keep`) are
  left as Director instructed, for close.

## Out Of Scope Returned To Director

- None beyond what Director already withheld (`VERSION`, `.baton/bin/`) and
  the three nits Director explicitly said to leave for close.
