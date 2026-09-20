# REVIEW-02: Let a project pause Baton without removing it

Task ID: srkz
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1618-pause-baton-RUN-02.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

REVIEW-01's B1 is closed. I judged both tests by mutation rather than by
reading, on a scratch copy of the tree, and both fail when the thing they
claim to guard is removed. The round stayed inside its scope: the only
content change since RUN-01 is `internal/baton/pause_test.go`. RUN-02 states
the correction to RUN-01's Success Criteria line plainly and accurately.

## Findings

### Blockers

- None.

### Nits

- **N6. The new test's failure mode without the guard is the weaker of the
  two assertions.** With the three-line `--apply` guard deleted,
  `TestUpdateApplyRefusesWhilePaused` fails at `refusal did not mention the
  pause`, because the minimal synthetic upstream makes `preflightUpdate`
  error on a missing `bootstrap/AGENTS.md` before anything is written. The
  three byte-identity assertions — the ones that would catch a silent
  un-pause — are never reached in the mutant. The test is load-bearing, and
  it fails for a defensible reason (without the guard, `update --apply` does
  not refuse *for the pause*), but it detects the guard's absence rather than
  the damage the guard prevents. I confirmed the damage is real: pointing the
  same test's `--upstream` at a complete tree with the guard removed,
  `update --apply` returned `nil` against a paused project and the test
  failed at `update --apply was accepted while paused`. The Executor found
  and reported exactly this and chose the synthetic fixture to avoid coupling
  a unit test to a real repository — a judgement I agree with, since a
  complete synthetic upstream needs a real binary and a matching
  `SHA256SUMS` (`preflightUpdate` verifies the checksum), which is a lot of
  fixture for one assertion. Recording it so the weaker failure mode is a
  known property of the test, not a surprise later.

- **N7. `TestRefusalSweepWhilePaused`'s own `update --apply` line is still not
  load-bearing.** With the guard removed, the full suite fails on exactly one
  test — the new one. The sweep's `harness.fail("update", "--upstream",
  t.TempDir(), "--apply")` still passes, because an empty `t.TempDir()` makes
  the call fail inside `preflightUpdate` whether or not the pause guard
  exists. It asserts only `err != nil`, so it cannot tell a pause refusal
  from a missing-file error. This is round-01 code and outside round 02's
  scope, but it is the same fault B1 named — a test that exists and guards
  nothing — and it sits four lines from the fix. Director may want it folded
  in with N1–N5 rather than left as a line that reads like coverage.

- **N8. RUN-02 cites evidence that cannot show what it is cited for.** Its
  last Success Criteria line says `git diff --stat` for this round "touches
  only `internal/baton/pause_test.go`". `pause_test.go` is untracked, so it
  never appears in `git diff --stat` at all. The claim itself is true — I
  verified it independently, see Evidence — but the command named does not
  establish it.

## Suggested User Checks

- Open `internal/baton/pause_test.go` at `TestUpdateApplyRefusesWhilePaused`
  and read it as the thing that now stands between you and a silent un-pause.
  Satisfy yourself that "exits non-zero, says `paused`, and leaves
  `AGENTS.md`, `CLAUDE.md` and `.baton/VERSION` byte-identical" is the whole
  of what you want guaranteed there, given N6 — that the last three
  assertions only run when the guard is present.

- Read the skip conditions on `TestWriteInstructionBlocksRestoresOnPartialFailure`
  (no `chflags`, or running as root) and decide whether you are content that
  this test is Darwin/BSD-only. On Linux CI it will skip with its reason
  printed and the partial-write property will go unchecked there. The
  Executor looked for a portable shape and reported not finding one; the
  question is whether a silent-by-design skip in CI is acceptable to you or
  whether it should fail loudly instead.

- Run `go test ./internal/baton/ -run 'TestUpdateApplyRefusesWhilePaused|TestWriteInstructionBlocksRestoresOnPartialFailure' -v`
  yourself and confirm both report PASS rather than SKIP on your machine.

- Do the one-line mutation yourself if you want it first-hand: delete the
  three lines `if pausedMessage != "" { return errors.New(pausedMessage) }`
  from `runUpdate` in `internal/baton/update.go`, run `go test ./...`, watch
  the new test fail, then `git checkout internal/baton/update.go`.

- The five nits in REVIEW-01 (N1–N5) and the three above are all still open
  by design — round 02 was scoped to B1 only. Decide with Director which are
  folded in before close and which are carried; N3 (a project with `.baton/`
  and no instruction file passes `lint`) is the one that asks for a product
  decision rather than a code change.

## Evidence Reviewed

- `.baton/runs/20260920-1618-pause-baton-PLAN.md`,
  `-RUN-01.md`, `-REVIEW-01.md`, `-RUN-02.md`; the working tree's
  `internal/baton/pause_test.go`, `pause.go`, `update.go`.
- All experiments ran on a scratch copy of the tree outside the repository
  (`tar`-cloned, `.git` excluded), never on the working tree. Baseline on
  that copy: `go test ./...` ok, `go vet ./...` clean, `gofmt -l .` empty,
  `go run ./cmd/baton lint` → `baton-lint passed` with `OK: AGENTS.md and
  CLAUDE.md Baton blocks match (active)`. Every validation RUN-02 reports
  reproduces.

- **Item 1 — `update --apply` guard, mutation.** Deleted the three-line
  guard (`if pausedMessage != "" { return errors.New(pausedMessage) }` before
  `preflightUpdate`) on the copy. `go test ./internal/baton/ -run
  TestUpdateApplyRefusesWhilePaused` → FAIL at `pause_test.go:299`, `refusal
  did not mention the pause: missing upstream file: .../bootstrap/AGENTS.md`.
  Full suite with the guard removed → FAIL, and the new test is the **only**
  failure (this is N7). Restored `update.go` from a saved copy, confirmed the
  restored `internal/baton` tree is byte-identical to the working tree
  (`diff -r --brief`), and `go test ./...` → ok. So the test REVIEW-01 asked
  for exists, and unlike the state at RUN-01 the suite no longer passes with
  the guard gone.

- **Item 1 — the damage the guard prevents, verified directly.** RUN-02
  claims that with a complete upstream and no guard, `update --apply`
  succeeds against a paused project. Reproduced: on the mutated copy I
  repointed the same test's `--upstream` at the copy's own root, and the test
  failed at `pause_test.go:290`, `update --apply was accepted while paused` —
  `err == nil`. RUN-02's account of this, including its reason for not
  committing that variant, is accurate.

- **Item 2 — partial-failure test, mutation.** Deleted the restore loop from
  `writeInstructionBlocks` (`for _, done := range written { if done.hadFile
  { _ = atomicWrite(...) } }`) on the copy. `go test ./internal/baton/ -run
  TestWriteInstructionBlocksRestoresOnPartialFailure -v` → FAIL at
  `pause_test.go:381`, `the first file was not restored after the second
  write failed`, printing `AGENTS.md` left carrying the paused block while
  `CLAUDE.md` still holds the old one — precisely the disagreement the
  restore exists to prevent, and precisely the reason the code was written.
  Restored `pause.go`, suite ok.

- **Item 2 — is the shape the one that occurs?** Yes. `instructionFiles()`
  (pause.go:56) joins `AGENTS.md` and `CLAUDE.md` onto `a.ProjectDir` in that
  order, so the two paths are always siblings in the project root. The
  reworked test now uses one `t.TempDir()` with `AGENTS.md` and `CLAUDE.md`
  inside it, in that order — the real shape. The two-separate-directories
  shape REVIEW-01 flagged is gone. The failure is induced with `chflags uchg`
  on the second file, which makes `atomicWrite`'s `os.Rename` fail onto an
  existing target while the directory stays writable; this is the same
  mechanism I used independently in REVIEW-01 to construct a genuine `pause`
  failure, so I can confirm it is a real failure and not a stub. The test ran
  (did not skip) on this machine.

- **Item 2 — the honesty question.** The Executor could not make the write
  fail portably and said so, in the test's own comment and in RUN-02:
  `chflags` is BSD/Darwin-only, and Linux's `chattr +i` needs
  `CAP_LINUX_IMMUTABLE`, which would test "root cannot write" rather than
  "an ordinary write fails". It chose a stated skip over a shape that does
  not occur. I consider that the right call and honestly reported — a test
  that skips loudly is better than one that passes against fiction, which is
  the fault this round was opened to fix. The residual cost is that the
  property goes unchecked on non-Darwin platforms. Raised as a user check
  rather than a nit, because it is a choice, not an error.

- **Item 3 — scope.** Only `internal/baton/pause_test.go` (mtime 16:47:34)
  and the new `RUN-02.md` (16:48:24) carry content from this round. Every
  production file, document, `README.md`, `.baton/VERSION` (16:17:24) and
  `.baton/bin/baton` (16:17:03, older than the PLAN itself) is untouched
  since RUN-01 was written at 16:36:42. `internal/baton/update.go` has a
  later mtime (16:47:38) because RUN-02 restored it from its saved copy after
  the mutation demo; its content is unchanged — `git diff` shows exactly the
  seven-line pause guard RUN-01 describes ("`runUpdate` computes
  `pausedUpdateMessage()` once; the dry run prints it and `--apply` returns
  it as an error before `preflightUpdate`") and nothing else. `git status`
  lists no file beyond the RUN-01 set plus `RUN-02.md`; nothing was deleted.
  `RUN-01.md` (16:36:42) and `REVIEW-01.md` (16:44:11) both predate this
  round's first write and were not edited. Note the evidence here is mtime
  plus content inspection, not a commit diff: round 01 was never committed,
  so no "state RUN-01 left" exists in Git to diff against. That is a limit of
  this check, stated rather than papered over.

- **Item 4 — the correction.** RUN-02's third Changes bullet quotes RUN-01
  line 106 verbatim, states it is wrong, and gives the reason: the cited
  `TestUpdateDryRunAllowedWhilePausedAndMentionsResume` runs `update` without
  `--apply` and asserts only that the output mentions `baton resume`. I
  confirmed line 106 reads as quoted and that the cited test is as described
  (pause_test.go:257). The correction is in RUN-02 and RUN-01 is unmodified,
  as required.

- No file under `.baton/runs/` was created, modified or deleted by any
  experiment above; every mutation ran on the scratch copy.

## Report To Director

- Suggested summary: Review round 02 — B1 closed; both tests verified load-bearing by mutation on a scratch copy, round stayed test-only, RUN-01's Success Criteria correction recorded; no blockers, three new nits
- Artifact path: `.baton/runs/20260920-1618-pause-baton-REVIEW-02.md`

## Required Next Step

- ask Director to request user approval. N6-N8 join REVIEW-01's N1-N5 as
  Director's call to fold in before close or carry.
