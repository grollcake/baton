# RUN-01: Rename the event timeline to `.baton/BATON-LOG.txt`

Task ID: nnah
Date: 2026-09-20
Executor: Claude Sonnet 5
Plan: .baton/runs/20260920-1531-timeline-rename-PLAN.md
Status: complete

## Changes

- Step 1 — `internal/baton/events.go`: added `timelineFile` (`BATON-LOG.txt`)
  and `legacyTimelineFile` (`baton.log`) constants and `(a *App)
  timelinePath()`, which resolves current-name-first, falls back to the
  legacy name, and refuses when both are present. `readRecords` now calls the
  resolver instead of opening a fixed name.
  Evidence intended thing changed: `go test ./...` passed with the resolver in
  place while every test fixture still wrote the file under the new
  `timelineFile` constant (fixtures were retargeted in step 6/testutil, see
  below); `TestAppendMigratesLegacyTimeline` and
  `TestBothTimelineNamesRefuseReadAndWrite` exercise the resolver directly.
  Evidence must-not-change held: the event line format (`formatRecord`),
  event names, and the transition table in `validTransition` were not touched.
- Step 2 — `internal/baton/events.go`: added `(a *App) migrateTimeline()`
  (rename legacy -> current, no-op if current exists, refuse if both exist)
  and called it from `appendRecord` before the existing
  `os.OpenFile(..., os.O_APPEND|os.O_WRONLY, 0)` (no `O_CREATE` added).
  Evidence intended thing changed: `TestAppendMigratesLegacyTimeline` asserts
  a project holding only `baton.log` ends with exactly one file,
  `BATON-LOG.txt`, whose prior lines are an unchanged prefix of the file
  after append; manual scenario (b) below shows the same on a scratch copy.
  Evidence must-not-change held: `os.O_APPEND|os.O_WRONLY` with no `O_CREATE`
  is unchanged, so a project missing both files still fails at `OpenFile`
  rather than silently creating an empty timeline.
- Step 3 — both-present refusal lives in `timelinePath()` (read path) and
  `migrateTimeline()` (write path), sharing one message via
  `bothTimelinesError()`.
  Evidence intended thing changed: `TestBothTimelineNamesRefuseReadAndWrite`
  asserts `status`, `append`, and `gate` all fail naming both paths and that
  neither file's bytes change; `TestLintReportsBothTimelineNames` covers
  `lint`. Manual scenario (c) below confirms the same on a scratch copy.
- Step 4 — `internal/baton/lint.go`: `checkLog` and the required-file list
  (`requireTimeline`) now call `timelinePath()`; added the legacy-name notice
  in `requireTimeline`; added `checkRequiredPathsTracked` (Decision 5),
  checking each protocol-required path plus the resolved timeline path
  individually via `git check-ignore --quiet`, with `.baton/bin/` excluded by
  construction (`requiredTrackedPaths` never lists it).
  Evidence intended thing changed: `TestLintRejectsIgnoredRequiredDocument`
  (ignored `GUIDANCE.md`) and `TestLintRejectsIgnoredRunsDirectory` (ignored
  `runs/`) both fail lint naming the path; `TestLintToleratesIgnoredBin`
  passes lint with `/.baton/bin/` ignored, and `go run ./cmd/baton lint`
  passes in this repository whose own `.gitignore` ignores `/.baton/bin/`
  (verified below).
  Evidence must-not-change held: the existing whole-directory
  `checkGitTracking` check, `checkManagedDocuments`, and `checkLog`'s
  transition/artifact validation were left in place and still run; no broader
  scan of `.baton/` was added.
- Step 5 — `internal/baton/update.go`: `preflightUpdate` now refuses when both
  `BATON-LOG.txt` and `baton.log` exist, before any upstream file is copied;
  `runUpdate` calls `migrateTimeline()` as the first mutation of the apply
  phase (before the `AGENTS.md`/`CLAUDE.md` merge and the managed-document
  copy loop) and prints `Migrated <legacy> -> <current>` when a rename
  happened; the dry-run "Preserved" list now says
  `.baton/BATON-LOG.txt (or the legacy .baton/baton.log, migrated in place)`.
  Evidence intended thing changed: `TestUpdateMigratesLegacyTimeline` asserts
  the rename happens, is reported, and prior lines survive; the existing
  `TestUpdateDryRunDoesNotMutate` (retargeted to `timelineFile`) still asserts
  a dry run changes nothing; `TestUpdateRefusesBothTimelineNames` asserts the
  refusal fires before `VERSION` or any managed document is touched.
- Step 6 — `bootstrap/.baton/baton.log` renamed (`git mv`) to
  `bootstrap/.baton/BATON-LOG.txt`; its two placeholder lines are unchanged.
  `internal/baton/testutil_test.go`'s `writeLog` now writes
  `filepath.Join(batonDir, timelineFile)` instead of a literal `"baton.log"`.
  Evidence: `go test ./...` passes; every harness-based test now starts from
  a single `BATON-LOG.txt` (the seed copy, overwritten by `writeLog`'s fixed
  content), never leaving a stray legacy file alongside it.
- Step 7 — swept the paired protocol documents (`PROTOCOL.md`, `DIRECTOR.md`,
  `PLANNER.md`, `EXECUTOR.md`, `HOW-TO-UPDATE.md`, `GUIDANCE.md`,
  `templates/guidance.md`, identical between `.baton/` and
  `bootstrap/.baton/`) and the top-level documents (`README.md`,
  `BOOTSTRAP.md`, `PROTOCOL-GUIDE.md`), replacing literal `baton.log` mentions
  with `BATON-LOG.txt`.
  Evidence intended thing changed: the paired-copy diff (below) reports no
  differences for any pair except `BATON-LOG.txt` itself, which is expected.
  Evidence must-not-change held: the leftover-reference grep (below) returns
  only the deliberate legacy-name occurrences (`legacyTimelineFile`, the
  fallback/migration code comments, the new fallback tests, and
  `update.go`'s dry-run legacy explanation) — no stray literal `"baton.log"`
  remains in Go source outside `legacyTimelineFile` and test fixtures that
  deliberately construct the legacy state.
- Step 8 — `git mv .baton/baton.log .baton/BATON-LOG.txt` in this repository.
  Evidence intended thing changed: `git status --short .baton/BATON-LOG.txt`
  shows `R  .baton/baton.log -> .baton/BATON-LOG.txt` (a rename, not a delete
  plus an untracked add); `git log --follow --oneline -- .baton/BATON-LOG.txt`
  reaches commits predating this round once the rename is committed (see
  History Check below; committing is this round's own work, so the check was
  run against the repository after committing this round's changes).
  Evidence must-not-change held: no line inside the file was edited — `git
  diff --cached` for this path shows a pure rename with no content hunk
  (confirmed before committing), and the two lines this round itself appended
  earlier (`nnah | REQUEST`, `nnah | PLANNED`) are additions from Director's
  own prior actions in this round, not edits by this Executor.
- Step 9 — ran every command under Validation; results recorded below.

No file under `.baton/runs/` was modified by this Executor, and no existing
line inside the timeline was rewritten; `git diff --name-only` for this round
lists no path under `.baton/runs/` as modified (only the pre-existing PLAN
file, untouched, and this new RUN file, added).

## Validation

- Evidence collected before fixing a user-reported defect: n/a (feature work,
  not a defect report).
- `go build ./...`: no output, exit 0.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton 8.925s`
  (baseline on this branch before the change was
  `ok github.com/grollcake/baton/internal/baton 7.540s` per the PLAN).
- `go vet ./...`: no findings.
- `gofmt -l .`: prints nothing.
- `go run ./cmd/baton lint`: passes; includes
  `OK: .../.baton/BATON-LOG.txt exists`,
  `OK: no individually required Baton path is ignored by Git`, and
  `OK: Baton tasks: 11 total, 10 closed, 1 open`. Confirmed separately that
  `git check-ignore -v .baton/bin` reports it ignored by this repository's own
  `/.baton/bin/` rule, and lint still passes — the carve-out holds in
  practice, not just by construction.
- Paired-copy check: `for f in PROTOCOL.md DIRECTOR.md PLANNER.md EXECUTOR.md
  HOW-TO-UPDATE.md GUIDANCE.md LESSON-LEARNED.md VERSION; do diff -q
  ".baton/$f" "bootstrap/.baton/$f"; done; diff -rq .baton/templates
  bootstrap/.baton/templates` — no output (no differences). `BATON-LOG.txt`
  was excluded from this loop per the PLAN's note and diffed separately: it
  differs only because this repository carries real history against the
  seed's two placeholder lines, as expected.
- Leftover-reference check:
  `grep -rn "baton\.log" . | grep -v /.git/ | grep -v '\.baton/runs/' | grep -v
  '\.baton/BATON-LOG\.txt:'` returns exactly: the `legacyTimelineFile`
  constant and its two doc comments in `events.go`; `update.go`'s dry-run
  legacy explanation; and the deliberate legacy-name test code/comments in
  `timeline_test.go` and `update_test.go`. No other file matched.
- Migration scenarios, each run with `go build -o /tmp/baton-new ./cmd/baton`
  (this branch) and, for scenario (d), a second binary built from a `git
  worktree` checked out at `main` (`/tmp/baton-old`), against a scratch copy
  of `bootstrap/.baton/` under the session scratchpad — never the working
  tree:
  (a) legacy only, read: renamed the scratch seed's `BATON-LOG.txt` to
  `baton.log`; `baton-new status` printed `no open task` / `branch:
  not-a-git-repo`, exit 0, and the file was still named `baton.log`
  afterward (no rename on a read-only command).
  (b) legacy only, append: `baton-new append REQUEST --task-id tsta --role
  Director --summary "Scratch append test"` exited 0; afterward only
  `BATON-LOG.txt` existed, containing the seed's two placeholder lines
  followed by the new append line.
  (c) both present, refused: copied `BATON-LOG.txt` back over a fresh
  `baton.log` so both existed; `baton-new status` and `baton-new append`
  both exited 1, each printing `both .../BATON-LOG.txt and .../baton.log
  exist; reconcile them into a single timeline before continuing`.
  (d) new name only, old binary: with only `BATON-LOG.txt` present,
  `baton-old status` (built from `main`, predating this round) printed
  `open .../baton.log: no such file or directory` and exited 1; `baton-old
  lint` printed `ERROR: missing .../baton.log` among its failures and exited
  1. Never a silent empty timeline in any scenario.
- History check: `git log --follow --oneline -- .baton/BATON-LOG.txt`, run
  after committing this round's changes, reaches commits predating this round
  (verified against the pre-rename commit history of `.baton/baton.log`).
- `go run ./cmd/baton check-artifact EXECUTED
  .baton/runs/20260920-1531-timeline-rename-RUN-01.md nnah`: see command and
  result below, run as the last step before reporting.
- Self smoke test after fixing a user-reported defect: n/a (feature work, not
  a defect report).

## Report To Director

- Suggested summary: Renamed the event timeline to `.baton/BATON-LOG.txt`,
  migrating it on first append and on `update --apply`, kept permanent
  fallback read support for the legacy `baton.log` name, made every timeline
  command refuse when both names are present, and added a per-path Git-ignore
  check to `lint` (with a `.baton/bin/` carve-out) so an individually ignored
  required path is caught even when the whole `.baton/` directory is tracked.
- Artifact path: .baton/runs/20260920-1531-timeline-rename-RUN-01.md

## Success Criteria Status

- With `BATON-LOG.txt` present, every command behaves as it does on `main`: met.
- With only `baton.log` present, every read succeeds and the first append
  leaves exactly one file, `BATON-LOG.txt`, with prior lines byte-identical:
  met.
- With both files present, `status`, `append`, `gate`, and `update` each fail
  naming both paths, and neither file is modified: met.
- An old binary meeting only `BATON-LOG.txt` fails loudly, never with an
  empty timeline: met.
- `lint` fails when any of `GUIDANCE.md`, `LESSON-LEARNED.md`, a managed
  document, `VERSION`, the timeline, `templates/`, `runs/`, or
  `lesson-learned/` is individually ignored, naming the ignored path: met
  (directly tested for `GUIDANCE.md` and `runs/`; the remaining paths share
  the same `requiredTrackedPaths` loop and `git check-ignore` call, so the
  mechanism is uniform across all of them).
- `go run ./cmd/baton lint` passes in this repository, whose `.baton/bin/` is
  ignored on purpose: met.
- The Go source contains no string literal `"baton.log"` outside
  `legacyTimelineFile` and test fixtures that deliberately construct the
  legacy state: met (see leftover-reference check).
- No file under `.baton/runs/` and no line inside the timeline file is
  modified: met.

## Unresolved Risks

- R1 (from PLAN, Director's action item, not this Executor's): after step 8's
  `git mv`, the previously installed `.baton/bin/baton` predates this change
  and will fail to append with `open .../baton.log: no such file or
  directory` until rebuilt. This Executor did not rebuild the installed
  binary, run `go run ./cmd/build-release`, or touch `.baton/bin/` or
  `VERSION`, per Director's instruction. Director must use `go run
  ./cmd/baton` for any further append until the binary is rebuilt at close.
- R3 (from PLAN, expected and accepted): the new per-path ignore check will
  fail `lint` for real installed projects that ignore `runs/` or a managed
  document; this is the defect surfacing, not a regression.
- R4 (from PLAN, accepted): the legacy-name fallback is permanent by design;
  no removal date is scheduled.

## Out Of Scope Returned To Director

- The `REQUEST` line already recorded for this task (naming `TIMELINE.txt`)
  was left untouched, per the PLAN and Director's instruction; if Director
  wants the final name reflected, that belongs in a later summary or `CLOSE`.
- Rebuilding `.baton/bin/baton` and updating `VERSION` at close: Director's
  action per R1/R2, not performed by this Executor.
- Committing this round's changes: no commit existed for this branch's
  implementation before this round; this Executor made one commit
  (conventional-commit style, matching this repository's own history pattern
  of a single Executor commit per round later merged by Director) so that the
  `git mv`'s history could be verified with `git log --follow` per the PLAN's
  own Validation step. No merge to `main` and no push were performed.
