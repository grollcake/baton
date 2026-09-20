# REVIEW-01: Rename the event timeline to `.baton/BATON-LOG.txt`

Task ID: nnah
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1531-timeline-rename-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

Every claim below was reproduced on scratch installations seeded from
`bootstrap/.baton/` and on two throwaway `git worktree` checkouts (this branch
and `main`), never on the working tree. No project reached a state with no
timeline or with two timelines, and no scenario produced a silent empty
timeline.

## Findings

### Blockers

- None.

### Nits

- `internal/baton/update.go:52-56` re-implements the two-name `os.Stat` pair
  inline instead of calling the shared resolver, which is the one place
  Decision 3's "one resolver, no site spells a name" rule is not followed.
  It is currently correct, but it is the seed from which the two-name pattern
  spreads. Evidence: deleting that block alone leaves the whole test suite
  green, because `migrateTimeline`'s own refusal still fires before any copy.
- `migrateTimeline`'s both-present branch is unreachable through today's
  callers: `readRecords` and `preflightUpdate` refuse first. Evidence: turning
  that branch into a silent no-op leaves the whole suite green, including
  `TestBothTimelineNamesRefuseReadAndWrite`. The RUN's statement that this test
  covers the write-path refusal overstates what it proves; it proves the read
  path's refusal.
- `TestUpdateRefusesBothTimelineNames` asserts `VERSION` and `PROTOCOL.md` are
  unchanged, but the harness upstream is byte-identical to the installed state,
  so those assertions hold even if the refusal fired after the copy loop. The
  property itself is true — verified separately against a genuinely differing
  upstream, where every file's checksum was unchanged after the refusal — but
  the test is not what establishes it.
- The manual-update path lost the legacy name. `update.go`'s dry-run
  "Preserved" list says `.baton/BATON-LOG.txt (or the legacy .baton/baton.log,
  migrated in place)`, but `HOW-TO-UPDATE.md`'s Preserve list and
  `BOOTSTRAP.md`'s do-not-overwrite list now name only `BATON-LOG.txt`. A human
  following step 7 ("if updating manually") in a project still holding
  `baton.log` has nothing telling them to preserve it.
- In a project that is not a Git repository, `checkRequiredPathsTracked`
  returns silently: no `OK`, no `ERROR`. This matches `checkGitTracking`'s
  existing style exactly and is the right call, but it means lint's output says
  nothing at all about ignore status there. Noted, not a change request.

### Judgment requested by Director: is the permanent fallback cheap to keep?

Cheap, and as written it does not spread. The two names exist as two constants
in `events.go`, and every behavioural site — `readRecords`, `appendRecord`,
`lint.requireTimeline`, `lint.checkLog`, `lint.checkRequiredPathsTracked`,
`update`'s apply phase — goes through `timelinePath()` or `migrateTimeline()`.
A future caller copying the nearest example copies a resolver call, not a name
pair. Runtime cost is two `os.Stat` calls on paths that were about to be
opened anyway. The single exception is the inline stat pair in
`preflightUpdate` named in the first nit; that is the one line of code a future
author could plausibly copy into a third site. Folding it into a shared helper
would make the fallback genuinely single-sited.

## Suggested User Checks

- In a real project still holding `.baton/baton.log`, run `baton status`
  (it should read normally and leave the file named `baton.log`), then record
  one real Director event, then confirm the directory holds exactly one
  `BATON-LOG.txt` whose earlier lines are unchanged and whose last line is the
  new event.
- In that same project, add `*.log` and then `runs/` to `.gitignore` and run
  `baton lint`. Confirm it fails and names the offending path, where the old
  binary reported `OK: .baton is not ignored by Git`.
- Deliberately create both `.baton/BATON-LOG.txt` and `.baton/baton.log` in a
  scratch copy and run `status`, `append`, `gate`, `update --apply` and `lint`.
  Confirm each refuses, names both paths, and that neither file's contents
  change.
- After Director rebuilds `.baton/bin/baton` at close, confirm
  `git log --follow --oneline -- .baton/BATON-LOG.txt` reaches commits from
  before this round, and that the next real Director append lands in
  `BATON-LOG.txt`.
- Confirm the recorded `REQUEST` line for `nnah` still reads
  `Rename baton.log to TIMELINE.txt ...`, and that no line under
  `.baton/runs/` from an earlier round was rewritten.

## Evidence Reviewed

- `.baton/runs/20260920-1531-timeline-rename-PLAN.md`,
  `.baton/runs/20260920-1531-timeline-rename-RUN-01.md`, and the branch diff
  against `main`.
- `internal/baton/events.go`, `internal/baton/lint.go`,
  `internal/baton/update.go`, `internal/baton/timeline_test.go`,
  `internal/baton/lint_test.go`, `internal/baton/update_test.go`.
- Reproduced checks on a throwaway worktree of this branch: `go build ./...`,
  `go vet ./...` (no findings), `gofmt -l .` (no output), `go test ./...`
  (`ok github.com/grollcake/baton/internal/baton 9.349s`), and
  `go run ./cmd/baton lint` in this repository (exit 0, printing
  `OK: .../.baton/BATON-LOG.txt exists`, `OK: .baton is not ignored by Git`,
  `OK: no individually required Baton path is ignored by Git`).
- Migration paths, each on a fresh scratch installation with a binary built
  from this branch and, where stated, one built from `main`:
  - legacy only, read: `status` exits 0, and the file is still named
    `baton.log` afterwards. A read stayed a read.
  - legacy only, append: exits 0; afterwards exactly one `BATON-LOG.txt`,
    holding the prior lines followed by the new one.
  - legacy only, `update --apply`: prints
    `Migrated .../baton.log -> .../BATON-LOG.txt`; prior lines survive.
  - both present: `status`, `append`, `gate`, `update --apply` and `lint` each
    exit 1 naming both paths, and a checksum of every file in the project is
    identical before and after all five commands. Repeated with an upstream
    whose managed documents genuinely differ from the installed ones: still
    refused with no file changed, so the refusal precedes the copy loop.
  - neither present: `status` and `append` fail with
    `missing timeline: .../BATON-LOG.txt (or legacy .../baton.log)` and create
    no file. `appendRecord` still opens with `os.O_APPEND|os.O_WRONLY` and no
    `O_CREATE`.
  - new name only, binary built from `main`: `status` prints
    `open .../baton.log: no such file or directory` and exits 1; `lint` prints
    `ERROR: missing .../baton.log` and exits 1; `append` fails the same way and
    creates nothing. Loud in every case.
  - the real-world upgrade path, end to end: a legacy installation updated by
    its own old installed binary against this branch's upstream. The old binary
    has no migration code, so it left `baton.log` in place and created no
    second file; the first append with the new binary then renamed it, and the
    resulting `BATON-LOG.txt` held the bootstrap lines, the update's own two
    lines and the new append, in order. This is the path `HOW-TO-UPDATE.md`
    step 4 actually sends users down, and it loses nothing.
- Per-path ignore check, each in a scratch Git repository, comparing the
  binary from `main` against this branch's:
  - `*.log` with a legacy-name timeline — the case that motivated this round.
    `git check-ignore -q .baton` exits 1 and the old binary printed
    `OK: .baton is not ignored by Git`; the new binary prints
    `ERROR: Git ignores required Baton path(s): .../.baton/baton.log` and
    exits 1.
  - `*.txt`, `runs/`, `.baton/GUIDANCE.md`, `.baton/templates/` and
    `.baton/lesson-learned/`: each caught and named individually.
  - `/.baton/bin/`: the per-path check still prints
    `OK: no individually required Baton path is ignored by Git`. The carve-out
    is exactly `.baton/bin/` and nothing wider — `requiredTrackedPaths` lists
    the eight required documents plus `templates`, `runs` and `lesson-learned`,
    which matches Decision 5's list item for item, and the resolved timeline is
    appended separately. Adding `"bin"` to that list makes
    `TestLintToleratesIgnoredBin` fail, so the carve-out is guarded.
  - not a Git repository: the check returns without printing, exactly as
    `checkGitTracking` already does.
- Append-only history: `git diff -M --name-status main...HEAD` reports
  `R094 .baton/baton.log -> .baton/BATON-LOG.txt` and
  `R100 bootstrap/.baton/baton.log -> bootstrap/.baton/BATON-LOG.txt`, so both
  show in Git as renames, not delete-plus-add. The old file's content is a
  strict line-for-line prefix of the new file's; the only difference is two
  appended `nnah` lines from Director's own events. `git diff --name-status
  main...HEAD -- .baton/runs/` lists only two additions and no modification.
  The recorded `REQUEST` line still carries the superseded name `TIMELINE.txt`.
- Reference sweep: the leftover grep returns only deliberate occurrences —
  the `legacyTimelineFile` constant, three doc comments in `events.go`, the
  dry-run legacy explanation in `update.go`, and legacy-name comments and
  assertions in `timeline_test.go` and `update_test.go`. The paired-copy diff
  across `PROTOCOL.md`, `DIRECTOR.md`, `PLANNER.md`, `EXECUTOR.md`,
  `HOW-TO-UPDATE.md`, `GUIDANCE.md`, `LESSON-LEARNED.md`, `VERSION` and
  `templates/` reports no differences between `.baton/` and
  `bootstrap/.baton/`.
- Load-bearing test check, by reverting one behaviour at a time on a throwaway
  worktree:
  - `appendRecord` stops migrating -> `TestAppendMigratesLegacyTimeline` fails.
  - `timelinePath` prefers the current name instead of refusing both ->
    `TestBothTimelineNamesRefuseReadAndWrite` and
    `TestLintReportsBothTimelineNames` fail.
  - `timelinePath` loses the legacy fallback ->
    `TestAppendMigratesLegacyTimeline` fails.
  - `Lint` stops calling `checkRequiredPathsTracked` ->
    `TestLintRejectsIgnoredRequiredDocument` and
    `TestLintRejectsIgnoredRunsDirectory` fail.
  - `requiredTrackedPaths` gains `"bin"` -> `TestLintToleratesIgnoredBin`
    fails.
  - `runUpdate` stops migrating -> `TestUpdateMigratesLegacyTimeline` fails.
  - Each failing test failed for the reason its change was made.
  - Two reverts left the suite green and are recorded as nits above:
    `migrateTimeline`'s both-present refusal, and `preflightUpdate`'s
    both-present refusal (each is masked by the other).

## Report To Director

- Suggested summary: Review round 01 — no blockers; every migration path,
  binary combination and ignore case reproduced on scratch installations
  without losing, truncating, merging or silently emptying a timeline; five
  nits recorded, all about test coverage and the manual-update docs
- Artifact path: .baton/runs/20260920-1531-timeline-rename-REVIEW-01.md

## Required Next Step

- ask Director to request user approval
