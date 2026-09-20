# REVIEW-02: Make the two checks the protocol mandates into commands

Task ID: cqso
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1900-verify-tooling-RUN-02.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

All three REVIEW-01 blockers are answered, each re-driven here rather than read
off the `RUN`. The command now answers in every case I could reach; the flag is
discoverable from `PLANNER.md` alone; and the binary no longer requires a shell
that is absent on a platform it ships to. Round 02 stayed in its lane: the three
nits Director held back reproduce unchanged, `VERSION` and `.baton/bin/` are
untouched, and the working tree is byte-identical after every run below.

The one thing worth saying plainly rather than filing as a nit is under
blocker 3: `cmd /C` fixes *the binary*, not *the check string*. See below.

## Findings

### Blockers

- none

### Nits

- **The Windows branch is unverified, and the suite that would verify it is
  POSIX-only.** `runCheckCommand` is the only `runtime.GOOS` branch in
  `revertcheck.go` and no test covers it; removing the branch entirely (back to
  `sh -c`) on a copy still gives `go build ./...` clean and `go test ./...` `ok`
  in 13.9s. It could not be otherwise: every `--check` string in
  `revertcheck_test.go` is POSIX (`true`, `exit 1`,
  `test -f new.txt || { echo ...; exit 1; }`), so on Windows those tests would
  fail on the check string before ever exercising the fix. The change is a
  two-line `GOOS` branch and is very likely correct; it is simply asserted, not
  demonstrated, and the round says so honestly.
- **The usage example is likewise untested.** Deleting the
  `(e.g. --name-pattern '--- FAIL: (\S+)' for go test)` line from `app.go` on a
  copy leaves `go build`, `go test ./...` and `go run ./cmd/baton lint` all
  green. Nothing pins the one line blocker 2 turned on. Acceptable for usage
  text; recorded so it is not mistaken for covered.
- The three nits REVIEW-01 left are unchanged, as instructed, and each still
  reproduces: the source grep still covers only `"(checkout|restore|stash|clean|
  worktree)"` plus four record names (`revertcheck_test.go:206-216`);
  `revert-check --check true internal/baton` still prints `baseline=pass` then
  the raw `rename .../internal/.baton-787586581 .../internal/baton: file exists`,
  exit 1; and `copy=` is still printed at `revertcheck.go:149`, after the
  baseline run at line 144, while the directory is made at line 110.
- REVIEW-01's fourth nit (the dropped "fail for the reason the change was made")
  was *not* left: round 02 folded it into blocker 2 and restored it. Noting the
  discrepancy so Director's close does not carry it as an open nit.

## Blockers Re-Driven

### 1. The command always answers, and the two cases are distinguishable

Both cases were driven in this repository, not read from the `RUN`.

REVIEW-01's exact case -- the `PLAN`'s own success-criterion command, reverting
a file that did not exist at `HEAD`, so it is deleted from the copy and the
package stops compiling:

```
$ go run ./cmd/baton revert-check --check 'go test ./internal/baton/...' \
    --name-pattern '--- FAIL: (\S+)' internal/baton/revertcheck.go
copy=/var/.../baton-revert-2437043768
rev=2f1b1ad08290e9566ea3bf1d9641eeef5b1d01e5
reverted=internal/baton/revertcheck.go
baseline=pass
after_revert=fail
pins=none (after_revert failed but no line matched --name-pattern; see new_output)
new_output=# github.com/grollcake/baton/internal/baton [...test]
new_output=internal/baton/app.go:143:12: a.runRevertCheck undefined (...)
new_output=FAIL	github.com/grollcake/baton/internal/baton [build failed]
new_output=FAIL
```

A run that completed, with the pattern matching nothing (`.baton/GUIDANCE.md`
reverted, tests still green):

```
$ go run ./cmd/baton revert-check --check 'go test ./internal/baton/... -run TestRevertCheck' \
    --name-pattern '--- FAIL: (\S+)' .baton/GUIDANCE.md
baseline=pass
after_revert=pass
pins=none
new_output=ok  	github.com/grollcake/baton/internal/baton	(cached)
```

Both are actionable and the two are distinguishable twice over: by the `pins=`
line itself, and independently by `after_revert=`. `pins=none` means "nothing
newly failed"; `pins=none (after_revert failed ...)` means "something newly
failed and I could not name it -- read `new_output=`", and `new_output=` then
carries the compile error. An agent splitting on the first `=` gets `none`
versus `none (after_revert failed ...)`, so even a naive parser sees a
difference, though one comparing the value to the literal `none` would need the
`after_revert` line to tell them apart.

No silent path remains. The `pins` switch (`revertcheck.go:203-236`) has three
terminal arms plus `nameRegexp == nil`, all of which print; the only earlier
return that skips the section is `baseline=fail`, which returns the explicit
`baseline check failed before any revert; refusing to compare against a red
baseline`.

Both new tests are load-bearing, verified by reverting the fix on a copy (never
the working tree). With the two new arms removed, both fail, and for the right
reason -- each failure message shows output ending at `after_revert=` with no
`pins=` line at all, which is precisely the defect:

```
--- FAIL: TestRevertCheckPinsNoneWhenNothingNewlyFails
    expected exactly one pins=none line, got "...baseline=pass\nafter_revert=pass\n"
--- FAIL: TestRevertCheckReportsReasonWhenRevertBreaksTheBuild
    expected a pins= line stating the run could not name a test, got
    "...after_revert=fail\nnew_output=build failed: missing new.txt\n"
```

`TestRevertCheckReportsReasonWhenRevertBreaksTheBuild` simulates the broken
build with a missing-file check rather than a real compile failure. The test
says so in its own comment, and the real compile case is driven above, so this
is disclosed, not hidden.

### 2. The flag is discoverable

- Usage line now reads, at `app.go:182-183`:
  `revert-check --check <command> [--rev <ref>] [--name-pattern <re>] [--keep] PATH...`
  followed by `(e.g. --name-pattern '--- FAIL: (\S+)' for go test)`.
- `diff .baton/PLANNER.md bootstrap/.baton/PLANNER.md` is empty: byte-identical.
  Both moved 3 lines out, 5 lines in (`9 ++++++---` each), confined to the
  Review section's load-bearing-check paragraph. No other line in either file is
  touched.
- The restored phrase is present: "confirms they fail for the reason the change
  was made -- see `new_output=`", which also does what the old text could not,
  namely point at where the reason lives.

To REVIEW-01's question, answered no last round: **yes.** Reading only
`PLANNER.md`, an agent is given the flag by name, a concrete regexp to copy
(`--- FAIL: (\S+)`), the tool to pair it with (`go test`), and the output key to
read the reason from. Nothing else needs to be guessed, and nothing sends the
reviewer to the usage text to recover it.

### 3. The shell assumption, and what it did and did not fix

What is actually executed, established from the source (`revertcheck.go:277-290`)
and confirmed by every run above: the `--check` string is handed to exactly one
interpreter, chosen by `runtime.GOOS` -- `cmd /C <string>` on Windows,
`sh -c <string>` everywhere else. It is still the only non-`git` `exec.Command`
in the binary, its working directory is still the copy, and the string still
comes only from `requireValue(parsed, "--check")` (one assignment, line 69; used
only at 144 and 185), so round 02 did not widen where it can come from.

REVIEW-01's blocker was that a shipped `PLANNER.md` instruction died outright on
two shipped platforms with `exec: "sh": executable file not found`. That is
fixed: `cmd.exe` is always present on Windows, so the binary runs on every
platform Baton ships.

Plainly, though, `cmd /C` behaves the *same* only for check strings that are
shell-neutral. A plain `go test ./internal/baton/...`, which is what `PLANNER.md`
actually prescribes, is shell-neutral and works under both. A compound string
using POSIX syntax -- `||`, `{ ; }`, `&&` chains with POSIX quoting, the very
shape `revertcheck_test.go:304` itself uses -- will not parse under `cmd.exe`,
and the reverse holds on Unix. So the cross-platform promise now reads: the same
*command* works everywhere; the same *check string* does not necessarily.

I do not think `PROTOCOL.md`'s "does not require ... a specific shell" has to
give way. Round 02 makes it true of the binary, which is what the sentence is
about, and the instruction `PLANNER.md` ships is one the promise covers. What
is not covered is the reviewer's own freedom to write a compound check string
portably, and neither `PLANNER.md` nor the usage text says so. That is a
documentation gap of one sentence, not a broken claim -- and it is the honest
residue of a fix that could not have gone further without inventing a
cross-platform check language.

### 4. Scope and safety boundaries

- **The three held-back nits are untouched**, each reproduced above.
- **`VERSION` and `.baton/bin/` unchanged.** `git diff --stat -- VERSION
  .baton/VERSION .baton/bin/` and `git status --short` on the same paths are
  both empty, and `.baton/bin/baton` has the same SHA-256 before and after every
  run in this review.
- **Working tree byte-identity, re-run after round 02 changed the copying code.**
  Snapshotted all 368 files under the repository (`.git` pruned, `.baton/bin/`
  included) with `shasum -a 256` before any run and again after all five
  `revert-check` runs below. `diff` of the two listings is empty, the file set is
  identical, `git status --short` is unchanged, and no `baton-revert-*` directory
  survives under `TMPDIR`. This covers the failing directory-argument run, which
  errors out mid-revert and still cleans up.
- **Argv-only `--check` still holds.** `revertcheck.go` has exactly one
  `os.ReadFile` (line 134, the `git ls-files` copy loop) and one `gitShowRaw`;
  no record path can reach `checkCommand`.
- Round 02's `git diff --stat` touches only `.baton/PLANNER.md`,
  `bootstrap/.baton/PLANNER.md`, `internal/baton/app.go` (4 insertions, all
  inside `Run`'s switch and `usage`), and `.baton/BATON-LOG.txt` (Director's).
  `revertcheck.go` and `revertcheck_test.go` are untracked so no per-round diff
  exists for them; their round-02 content matches what `RUN-02` describes and
  nothing else in either file departs from what REVIEW-01 read.

## Suggested User Checks

- Run `go run ./cmd/baton revert-check --check 'go test ./internal/baton/...'
  --name-pattern '--- FAIL: (\S+)' internal/baton/revertcheck.go` and decide
  whether `pins=none (after_revert failed but no line matched --name-pattern;
  see new_output)` plus the compile error under `new_output=` tells you enough
  to act, now that the old version of this command printed nothing.
- Run the same command against `.baton/GUIDANCE.md` instead and confirm you can
  tell the two `pins=none` answers apart at a glance.
- Read only the five-line paragraph in `.baton/PLANNER.md`'s Review section and
  judge whether you could run `revert-check` correctly and report *why* a test
  failed, without looking at the usage text or asking anyone.
- Run `go run ./cmd/baton` with no arguments, read the `revert-check` usage line
  and its example, and decide whether the example belongs there permanently.
- Decide the Windows question: `cmd /C` means the binary runs everywhere, but a
  POSIX-syntax `--check` string still will not parse on Windows and no test
  covers the branch. Judge whether that needs a sentence in `PLANNER.md` now or
  can wait.

## Evidence Reviewed

- Working-tree byte-identity: 368-file `shasum -a 256` listing before and after
  all runs, `diff` empty; `git status --short` unchanged; no leftover
  `baton-revert-*` under `TMPDIR`. Also verified the file set itself is
  identical, so nothing was added or removed.
- Real runs driven here: the two blocker-1 cases above; `--check 'true'
  --name-pattern ... VERSION` -> `pins=none`; `--check true internal/baton`
  (directory nit) -> raw rename error, exit 1; `.baton/bin/baton revert-check
  ...` -> prints usage, confirming the installed binary predates this work and
  every check must use `go run ./cmd/baton`.
- Load-bearing check, on a copy in the scratchpad, never the working tree:
  removing the two new `pins` arms fails both new tests with the no-`pins=`-line
  output quoted above; restored.
- Coverage probe, on a second copy: removing the `runtime.GOOS` branch *and* the
  usage example leaves `go build ./...` clean, `go test ./...` `ok ... 13.907s`,
  `go run ./cmd/baton lint` -> `baton-lint passed`.
- `RUN-02` validation reproduced on the working tree: `gofmt -l .` empty,
  `go vet ./...` clean, `go test ./...` `ok github.com/grollcake/baton/
  internal/baton`, `go run ./cmd/baton lint` -> `baton-lint passed`.
- `diff .baton/PLANNER.md bootstrap/.baton/PLANNER.md`: empty. `git diff` on
  both: `9 ++++++---`, one paragraph each.
- Read: `internal/baton/revertcheck.go` (lines 60-300), the eight tests in
  `revertcheck_test.go`, the `app.go` usage block, and the full round-02
  `git diff`.

## Report To Director

- Suggested summary: REVIEW-02 finds all three REVIEW-01 blockers answered and
  re-driven -- `revert-check` now prints a `pins=` line in every reachable case
  and the "nothing matched" and "could not name a test" answers are
  distinguishable twice over; `--name-pattern` with the `--- FAIL: (\S+)` example
  now reaches an agent from `PLANNER.md` alone, with both copies byte-identical
  and the "fail for the right reason" phrase restored; and the binary picks
  `cmd /C` or `sh -c` by platform so it no longer dies where it ships. Round 02
  stayed in its lane: the three held-back nits reproduce unchanged, `VERSION` and
  `.baton/bin/` are untouched, and the tree is byte-identical after every run.
  Residual, stated plainly rather than filed away: `cmd /C` fixes the binary, not
  the check string -- a POSIX-syntax `--check` still will not parse on Windows,
  and neither the usage text nor `PLANNER.md` says so.
- Artifact path: .baton/runs/20260920-1900-verify-tooling-REVIEW-02.md

## Required Next Step

- Ask Director to request user approval. No blocker remains. The open judgement
  for the user is whether the Windows residue above needs a sentence in
  `PLANNER.md` in this task or is acceptable as-is; if the user wants it, it is
  a one-line `RUN-03` that can carry the three held-back nits with it.
