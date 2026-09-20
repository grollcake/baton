# PLAN: Rename the event timeline to `.baton/BATON-LOG.txt`

Task ID: nnah
Date: 2026-09-20
Planner: Claude Opus 5
Status: complete

## Director Brief

- Goal: Rename `.baton/baton.log` to `.baton/BATON-LOG.txt` so a project's `.gitignore` cannot swallow the append-only handoff record, migrate installed projects without ever losing a timeline, and make `lint` check that each protocol-required path is individually un-ignored.
- Scope: `internal/baton/events.go`, `internal/baton/lint.go`, `internal/baton/update.go` and their tests; the 70 live document references across 25 files; the paired `.baton/` and `bootstrap/.baton/` copies; this repository's own timeline via `git mv`.
- Success Criteria: With `BATON-LOG.txt` present every command behaves exactly as it does today on `main`; with only `baton.log` present every read still works and the first append renames the file; with both present every timeline command refuses and names both paths; `lint` fails when any required document, `templates/`, `runs/` or `lesson-learned/` is individually ignored, and still passes in this repository whose `.baton/bin/` is deliberately ignored.
- Risks: The rename lands mid-round in a repository whose Director is appending to the very file being renamed with an installed binary that predates the change. See Risk R1 for the required Director action. Second risk: a per-file ignore check that includes `.baton/bin/` makes `lint` fail in this repository on the first run; the carve-out in Decision 5 is not optional.
- Required Checks: `go test ./...`, `go vet ./...`, `gofmt -l .`, `go run ./cmd/baton lint`, plus the four migration scenarios under Validation.
- Executor Prompt: Implement `.baton/runs/20260920-1531-timeline-rename-PLAN.md` steps 1-9 in order. Rename `.baton/baton.log` to `.baton/BATON-LOG.txt` and sweep its live references, following the Decisions section exactly. Do not change the log line format, the event names, the transition table, or any artifact contract. Do not rewrite anything under `.baton/runs/` or any line inside the timeline file itself; both are append-only history. Use `git mv` for this repository's own file and `go run ./cmd/baton` for every check that must reflect your working tree, because the installed binary lags it. Record in the RUN, per step, the evidence that the intended thing changed and the evidence that the named must-not-change did not.

## Goal

- The protocol's append-only handoff record is named `.baton/BATON-LOG.txt`. The `*.log` glob that most projects' `.gitignore` carries no longer matches it.
- No installed project can reach a state where a Baton command starts from an empty timeline. Every mismatch between binary, documents and file name either works correctly or fails loudly.
- `lint` no longer reports that Baton state is tracked when an individual required file is ignored.

## Decision 0: the name

Director set the target to `.baton/BATON-LOG.txt`. The deciding factor is
recognition at migration time: every existing installation has a `baton.log`
holding real history, and a user or agent meeting `BATON-LOG.txt` after an
update knows without being told that it is the same artifact renamed.

The accepted costs, recorded so they are not rediscovered as defects:

- The name keeps the word *log*, so it keeps part of the misclassification the
  rename was meant to correct.
- The `BATON-` prefix restates the directory the file sits in, which no other
  file under `.baton/` does.

Neither cost changes any decision below. One thing does change: `*.log` is the
glob that actually occurs in the wild, and `BATON-LOG.txt` does not match it,
so the rename alone closes the specific reported defect. A rule like `*log*`
or `*.txt` still matches, which is why Decision 5 belongs in this round rather
than a later one.

## Decision 1: migration of installed projects

**The binary performs the rename with `os.Rename` inside `.baton/`, never a
copy-then-delete, and it does so from three places with different triggers.**

### Why not `update` alone

`update` is the obvious place, and it is not sufficient. `HOW-TO-UPDATE.md`
step 4 says to use the *installed* binary for the update on every platform
except Windows. The installed binary during an old-to-new upgrade is the old
one, which has no migration code. A rename implemented only in `update` would
therefore never run for the migration it was written for. This is the single
most important finding in this plan: an `update`-only migration is dead code.

### What runs where

1. **Read path (`readRecords`, `lint`'s checks).** Resolve the timeline: if
   `BATON-LOG.txt` exists use it; else if `baton.log` exists use it; else
   report the missing file naming both. Reads never rename. A read-only
   command must stay read-only.
2. **Write path (`appendRecord`).** Before opening the file for append, if
   `BATON-LOG.txt` is absent and `baton.log` is present, rename it, then
   append. `appendRecord` already mutates `.baton/`, so moving the file it is
   about to write is in character, and it guarantees the appended line and the
   prior history stay in one file. This is the mechanism that actually
   migrates an active project, because the first Director event after the new
   binary is installed triggers it.
3. **`update --apply`.** Perform the same rename as the *first* mutation of
   the apply phase, before any managed document, template, binary or `VERSION`
   is copied, and print that it happened. This covers the cases where the
   binary running `update` is already new: Windows, an upstream-run or
   temporary-path binary, `go run`, and any second `update` after the first.

### What `update` must never do

- Never delete, truncate, overwrite or merge either file.
- Never create `BATON-LOG.txt` from a template or as an empty file when
  `baton.log` exists.
- Never copy-then-delete. `os.Rename` within one directory is atomic on every
  supported platform, so there is no intermediate state in which the content
  exists in neither file or is partially written to both.
- Never continue past a failed rename. If `os.Rename` returns an error,
  `update` aborts before copying anything, so the project is left exactly as
  it was found.

### Refusal case: both files present

If `BATON-LOG.txt` and `baton.log` both exist, the binary cannot know which is
authoritative, and merging two append-only records is not a rename. Every
command that reads or writes the timeline returns an error naming both paths
and asking the user to reconcile them. `update` checks this in
`preflightUpdate`, before any file is copied, so a project in this state is
refused rather than half-updated. This is the "losing a timeline is worse than
refusing to update" case, and it is the only one that stops an update.

### Half-completed rename

Within `.baton/` there is no half-completed rename; `os.Rename` is atomic. The
partial states that do exist are process states, and each is loud and
recoverable:

- Renamed, then `update` fails before copying the new binary: the project has
  `BATON-LOG.txt`, old documents and the old binary, which fails on every
  command with `open .baton/baton.log: no such file or directory` (verified
  below). Recovery is to re-run `update`, whose rename step is a no-op the
  second time.
- Renamed, then `update` fails partway through copying documents: same
  recovery, and `lint`'s existing `checkManagedDocuments` already reports the
  documents that do not match the binary.

The rename is idempotent in all three trigger sites, so re-running is always
the recovery.

### A note for projects where `baton.log` was ignored

In exactly the projects this task is about, `baton.log` is untracked, so it has
no Git history to preserve there. The rename is a plain filesystem `os.Rename`,
not `git mv`, which is correct for that case: afterwards Git sees a new,
no-longer-ignored file and the project can commit its timeline for the first
time. `git mv` is used only for this repository (Decision 4), where the file is
tracked.

## Decision 2: backward compatibility

**Yes, the binary reads `baton.log` when `BATON-LOG.txt` is absent. The
fallback is permanent, and this plan deliberately schedules no removal.**

The reason for "permanent": the fallback costs one `os.Stat` on paths that
already open a file, and the only consequence of removing it is that some
repository we cannot observe loses access to its history. There is no date or
version at which removal becomes verifiably safe, because we cannot enumerate
installed projects. A removal scheduled on an unverifiable condition is a
guess, and the guideline this project follows is to state the assumption
rather than hide it. Director can overrule this and pin removal to a version;
the plan records that the only honest basis for such a date would be an
inventory of installations that does not exist.

Decision 1's write-path migration makes the fallback self-clearing in any
active project: the first recorded event renames the file. The fallback then
only serves dormant projects and read-only inspection.

### Combination matrix

| Binary | Files present | Result |
|---|---|---|
| new | `BATON-LOG.txt` | Normal operation. |
| new | `baton.log` only | All reads work via fallback. The first append renames the file. `lint` prints a notice naming the legacy path. |
| new | both | Every timeline command refuses and names both paths. `update` refuses in preflight. |
| new | neither | Error naming `BATON-LOG.txt` and the accepted legacy name. Same failure class as today. |
| old | `baton.log` | Normal operation. The old binary is unaffected by this round. |
| old | `BATON-LOG.txt` only | Every command fails loudly. Verified: `status` printed `open .../.baton/baton.log: no such file or directory` and exited 1; `lint` printed `ERROR: missing .../.baton/baton.log` and failed. Never a silent empty timeline. |

### Documents updated but binary not, and the reverse

- **New documents, old binary.** Already detected today, not by this round:
  `lint`'s `checkManagedDocuments` (`internal/baton/lint.go:83`) compares each
  installed managed document against the copy embedded in the running binary
  and errors when they differ. A project in this state fails `lint` before any
  timeline question arises. The Executor must not weaken this check.
- **Old documents, new binary.** The same check reports it. The new binary's
  fallback means the timeline still works, so this state is inconvenient, not
  destructive.
- Neither combination can produce an empty timeline, because no code path
  creates the timeline file. `appendRecord` opens with
  `os.O_APPEND|os.O_WRONLY` and no `O_CREATE` (`internal/baton/events.go:154`),
  so a missing file is an error, not a fresh empty log. The Executor must keep
  it that way; adding `O_CREATE` anywhere in this round would convert every
  failure in the matrix above into silent data loss.

## Decision 3: reference sweep

**Yes, the Go source names the file in one place.** Add to `events.go`:

- `const timelineFile = "BATON-LOG.txt"`
- `const legacyTimelineFile = "baton.log"`
- one resolver, `func (a *App) timelinePath() (string, error)`, implementing
  the read-path rule and the both-present refusal from Decision 1.

This is not abstraction for its own sake. The two-name resolution and the
both-present refusal must be byte-identical at every call site; duplicating
them across four sites is precisely the bug class that produced this task.
Each of `readRecords`, `appendRecord`, `lint.checkLog` and `lint`'s
required-file list calls the resolver instead of spelling a name.

### Inventory

70 live references across 25 files. Confirmed by
`grep -rn "baton\.log" . | grep -v /.git/ | grep -v '\.baton/runs/' | grep -v '\.baton/baton\.log:'`.

- Go, behavioural: `events.go` (2), `lint.go` (3, one of which is the
  `cannot read baton.log` message), `update.go` (1, the dry-run "Preserved"
  list).
- Go, tests: `update_test.go` (8), `commands_test.go` (3), `testutil_test.go`
  (1), `status_test.go` (1), `lint_test.go` (1).
- Paired protocol documents, which must stay byte-identical between `.baton/`
  and `bootstrap/.baton/`: `DIRECTOR.md` (4 each), `HOW-TO-UPDATE.md` (2 each),
  `PROTOCOL.md`, `PLANNER.md`, `EXECUTOR.md`, `GUIDANCE.md` and
  `templates/guidance.md` (1 each).
- Top-level documents: `PROTOCOL-GUIDE.md` (17), `BOOTSTRAP.md` (7),
  `README.md` (4).
- The seed file `bootstrap/.baton/baton.log` is renamed to
  `bootstrap/.baton/BATON-LOG.txt`; its two placeholder lines are unchanged.

### Explicitly out of the sweep

33 references live under `.baton/runs/` and inside `.baton/baton.log` itself.
Those are closed-round artifacts and recorded history. `PROTOCOL.md` requires
older rounds to stay append-only and forbids overwriting an artifact, so the
Executor must not touch them. A historical record that says `baton.log` is
correct: that is what the file was called when the record was written. This
includes the `REQUEST` line already recorded for this task.

`DIRECTOR.md` already calls the record the Event Timeline. Rename the file
path only; leave that prose, the event names, the line format, the transition
table and every artifact contract untouched.

## Decision 4: this repository's own timeline

**`git mv .baton/baton.log .baton/BATON-LOG.txt`, in this round.**

Why `git mv` rather than a plain `mv`: Git stores no rename, it detects one at
diff time from identical content, and both commands leave content identical, so
`git log --follow` works either way. `git mv` is still the right command
because it stages the removal and the addition in one step, leaving no window
in which the tracked file is deleted and the new one untracked, and no chance
of the new name being caught by a future ignore rule before it is added.

Why the same round rather than a separate one: this repository is itself a
Baton installation. If its documents and binary say `BATON-LOG.txt` while its
file is still `baton.log`, then every check run during this round exercises the
legacy fallback rather than the intended path, and the round cannot be
validated end to end. Splitting would also mean shipping a version whose own
repository contradicts its documents.

## Decision 5: the `lint` per-file ignore check

**This round.**

The reason: the rename's entire justification is that a file under `.baton/`
gets silently ignored while `lint` reports that Baton state is tracked.
Shipping the rename alone ships the claim that the problem is fixed while
leaving the mechanism that hid it intact, and Director's own note on the name
change points at `*log*` as a rule that still bites. The change is also small
and local: it edits `checkGitTracking`, the same function and the same required
path list the rename already touches.

The argument for splitting, recorded and rejected: the check changes `lint`'s
failure surface, so a project that ignores `runs/` starts failing a lint that
passed before. That failure is the defect being reported, not a regression
introduced by this round.

### Evidence that the hole is real, and that it survives the rename

In a scratch repository seeded from `bootstrap/.baton/`:

- With `.gitignore` containing `*.log`: `git check-ignore -q .baton` exits 1
  while `git check-ignore -v .baton/baton.log` reports
  `.gitignore:1:*.log	.baton/baton.log`, and `lint` still printed
  `OK: .baton is not ignored by Git`.
- With `*.txt`: `.baton/BATON-LOG.txt` is ignored. The rename does not close
  this.
- With `runs/`: `.baton/runs` is ignored, and no rename can close that one.

### Scope of the check, and a mandatory carve-out

Check exactly the paths the protocol already requires, by calling
`git check-ignore --quiet` with one path argument, once per path, in the existing style of
`internal/baton/lint.go:74`, and report each ignored path by name:

- `PROTOCOL.md`, `DIRECTOR.md`, `PLANNER.md`, `EXECUTOR.md`,
  `HOW-TO-UPDATE.md`, `VERSION`, `GUIDANCE.md`, `LESSON-LEARNED.md`, and the
  resolved timeline file
- the directories `templates/`, `runs/`, `lesson-learned/`

**Exclude `.baton/bin/`, the installed binary and `bin/SHA256SUMS`.** This is
not a preference. This repository's own `.gitignore` carries `/.baton/bin/`
with a comment explaining that the binary would otherwise be a seventh
committed copy. Verified: `git check-ignore -q .baton/bin` exits 0 here. A
check that included `bin/` would make `lint` fail in this repository on its
first run, and the binary is a build artifact a project may legitimately
ignore, unlike the handoff data.

Keep the existing whole-directory check as it is and add the per-path check
after it. A project that ignores all of `.baton` should still get the clearer
existing message rather than twelve individual ones.

Do not add a broader scan of `.baton/`. The requirement is that protocol-required
paths are tracked, not that nothing under `.baton/` may ever be ignored.

## Scope

In scope:

- `internal/baton/events.go`, `internal/baton/lint.go`, `internal/baton/update.go`
- `internal/baton/update_test.go`, `commands_test.go`, `testutil_test.go`, `status_test.go`, `lint_test.go`
- `.baton/` and `bootstrap/.baton/` protocol documents and `templates/guidance.md`, kept byte-identical in pairs
- `README.md`, `BOOTSTRAP.md`, `PROTOCOL-GUIDE.md`
- `.baton/baton.log` and `bootstrap/.baton/baton.log`, renamed

Out of scope:

- The log line format, event names, the transition table, and every artifact contract
- Everything under `.baton/runs/` and every line inside the timeline file
- `VERSION` and `.baton/bin/`, which Director owns at close
- Any change to `HOW-TO-UPDATE.md` step 4's choice of which binary runs an update; Decision 1 makes that choice irrelevant to correctness

## Plan

1. Add `timelineFile`, `legacyTimelineFile` and `timelinePath()` to
   `internal/baton/events.go`; point `readRecords` at the resolver.
   Verify: `go test ./...` still passes with the file still named `baton.log`
   in test fixtures, proving the fallback works before anything is renamed.
2. Make `appendRecord` migrate before appending, per Decision 1. Keep
   `os.O_APPEND|os.O_WRONLY` with no `O_CREATE`.
   Verify: a new test asserts that appending to a project holding only
   `baton.log` leaves one file named `BATON-LOG.txt` containing the prior
   lines followed by the new one.
3. Make the both-present case an error in the resolver.
   Verify: a new test asserts that `status` and `append` both fail and that
   the message names both paths, and that neither file is modified.
4. Update `internal/baton/lint.go`: use the resolver for `checkLog` and for
   the required-file list, add the legacy-name notice, and add the per-path
   ignore check with the `bin/` carve-out from Decision 5.
   Verify: new tests for an ignored `GUIDANCE.md`, an ignored `runs/`, and an
   ignored `bin/` that must still pass.
5. Add the rename to `update`'s apply phase as its first mutation, add the
   both-present refusal to `preflightUpdate`, and correct the dry-run
   "Preserved" list.
   Verify: a test asserts the rename happens, and an existing-style test
   asserts a dry run still changes nothing.
6. Rename the seed file `bootstrap/.baton/baton.log` to
   `bootstrap/.baton/BATON-LOG.txt` and update `testutil_test.go` and the
   remaining test fixtures.
   Verify: `go test ./...`.
7. Sweep the paired protocol documents, then the top-level documents.
   Verify: the paired-copy diff under Validation reports no differences.
8. `git mv .baton/baton.log .baton/BATON-LOG.txt`.
   Verify: `git log --follow -- .baton/BATON-LOG.txt` reaches commits made
   under the old name, and `git status` shows a rename, not a delete plus an
   untracked file.
9. Run every command under Validation and record the output in the RUN.

## Success Criteria

- With `BATON-LOG.txt` present, every command behaves as it does on `main`.
  Observable: `go test ./...` passes and `go run ./cmd/baton status` output is
  unchanged against `main` for this repository's timeline.
- With only `baton.log` present, every read succeeds and the first append
  leaves exactly one file, `BATON-LOG.txt`, whose earlier lines are byte-identical
  to the ones that were in `baton.log`.
- With both files present, `status`, `append`, `gate` and `update` each fail
  with a message naming both paths, and neither file is modified.
- An old binary meeting only `BATON-LOG.txt` fails loudly, never with an empty
  timeline. Observable: it prints `open ....baton/baton.log: no such file or
  directory` and exits non-zero.
- `lint` fails when any of `GUIDANCE.md`, `LESSON-LEARNED.md`, a managed
  document, `VERSION`, the timeline, `templates/`, `runs/` or
  `lesson-learned/` is individually ignored, naming the ignored path.
- `go run ./cmd/baton lint` passes in this repository, whose `.baton/bin/` is
  ignored on purpose.
- The Go source contains no string literal `"baton.log"` outside
  `legacyTimelineFile` and test fixtures that deliberately construct the
  legacy state.
- No file under `.baton/runs/` and no line inside the timeline file is
  modified. Observable: `git diff --name-only` lists nothing under
  `.baton/runs/`, and the diff for the renamed timeline is a pure rename.

## Validation

- `go test ./...`: passes. Baseline on this branch before the change: `ok github.com/grollcake/baton/internal/baton 7.540s`.
- `go vet ./...`: no findings.
- `gofmt -l .`: prints nothing.
- `go run ./cmd/baton lint`: passes, and prints the per-path ignore results.
- Paired-copy check: `for f in PROTOCOL.md DIRECTOR.md PLANNER.md EXECUTOR.md HOW-TO-UPDATE.md GUIDANCE.md LESSON-LEARNED.md VERSION BATON-LOG.txt; do diff -q ".baton/$f" "bootstrap/.baton/$f"; done; diff -rq .baton/templates bootstrap/.baton/templates` — no differences reported. Note `BATON-LOG.txt` is the one pair that is expected to differ, since this repository has real history and the seed file has two placeholder lines; exclude it and say so in the RUN.
- Leftover-reference check: `grep -rn "baton\.log" . | grep -v /.git/ | grep -v '\.baton/runs/' | grep -v '\.baton/BATON-LOG\.txt:'` returns only the deliberate legacy-name occurrences: `legacyTimelineFile`, the fallback tests, the `lint` legacy notice and its test, and any document sentence that explains the old name to a migrating reader.
- Migration scenarios, each run on a scratch copy of `bootstrap/.baton/` and never on the working tree: (a) legacy only, read; (b) legacy only, append; (c) both present, refused; (d) new name only with a binary built from `main`, fails loudly.
- History check: `git log --follow --oneline -- .baton/BATON-LOG.txt` reaches commits predating this round.
- `go run ./cmd/baton check-artifact` on this round's own artifacts as the last step before reporting.

## Report To Director

- Suggested summary: Rename the event timeline to `.baton/BATON-LOG.txt`, migrate on append and on update, keep reading the legacy name, and make lint check each required path for an individual ignore rule
- Artifact path: `.baton/runs/20260920-1531-timeline-rename-PLAN.md`

## Risks Or Questions

- **R1, blocking a Director action.** This repository's own timeline is renamed
  mid-round, while Director is appending events to it. The installed
  `.baton/bin/baton` predates this change and appends with
  `os.O_APPEND|os.O_WRONLY` and no `O_CREATE`, so after step 8 every Director
  append with the installed binary fails with `open .baton/baton.log: no such
  file or directory`. Director must use `go run ./cmd/baton` for every append
  from the moment the Executor reports step 8 until the binary is rebuilt at
  close. This failure is loud, not silent, so the risk is an interrupted round,
  not a lost timeline.
- **R2.** Editing the managed documents makes the installed binary report
  document drift for the rest of the round, by design. Every check that must
  reflect the working tree uses `go run ./cmd/baton`. Director rebuilds at
  close.
- **R3.** The per-path ignore check will fail `lint` for real installed
  projects that ignore `runs/` or a managed document. That is the defect
  surfacing, and the message names the offending path, but Director should
  expect it to be reported as a new failure by the first project that updates.
- **R4, accepted.** The permanent fallback in Decision 2 means the legacy name
  stays in the source indefinitely. The alternative is a removal date with no
  evidence behind it; see Decision 2 for why no such date is defensible here.
- **Question for Director, non-blocking.** The `REQUEST` line already recorded
  for this task says `Rename baton.log to TIMELINE.txt`. It is append-only
  history and this plan leaves it alone; if Director wants the final name on
  the record, that belongs in a later event's summary or in `CLOSE`, not in an
  edit to the recorded line.
