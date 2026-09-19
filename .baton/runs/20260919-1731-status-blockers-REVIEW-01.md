# REVIEW-01: Make status reflect a REVIEW that reports blockers

Task ID: qstp
Date: 2026-09-19
Planner: Planner
Run: .baton/runs/20260919-1731-status-blockers-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- none

### Nits

- The fix reaches the repo's own `.baton/bin/baton` only after a release build.
  A fresh `go build ./cmd/baton` binary shows the new behavior; the currently
  installed `.baton/bin/baton` (built 17:30, before this change) still does not.
  This is the normal install lifecycle owned by task `veee`, not a defect here,
  but Director should not test the new behavior with the installed binary.
- `TestStatusAfterReviewReadyOffersUserApproval` passes both before and after
  the source change, so it is a regression guard rather than a load-bearing
  test for this fix. That is correct and intended by the PLAN; noted only so
  the round's "reverting makes the new tests fail" claim is read precisely: two
  of the three new tests fail on revert, and the third is guarding the path the
  change must not disturb.
- On the missing-artifact path, `status` gives exactly the same output as the
  blockers path, with no hint that the REVIEW file could not be read, while
  `gate before-approval` on the same state does print the underlying
  `no such file or directory`. Fail-closed is the right behavior; the silent
  conflation is a diagnosability nit, and is the strongest argument for the
  `reviewResult` follow-up (see Risks).

## Suggested User Checks

- Build a fresh binary (`go build -o /tmp/baton ./cmd/baton`), then in a scratch
  project drive `new-round` -> PLANNED -> EXECUTED(RUN-01) -> REVIEW-01 with
  `Result: blockers`, and confirm `status --task-id TASK` prints
  `next_gate: EXECUTED (delegate Executor)` and
  `next_command: baton prompt exec ... --run-number 02`.
- In the same scratch project, repeat with `Result: ready-for-user-decision` and
  confirm `status` still prints `next_gate: user approval -> CLOSE` and prints
  no `next_command:` line at all.
- Delete the logged REVIEW-01 file from disk and re-run `status`: it must fall
  back to `next_gate: EXECUTED (delegate Executor)`, never to approval-ready.
- On the blockers state, run `baton gate before-approval --task-id TASK` and
  confirm it still refuses with the existing `REVIEW reports blockers` message,
  i.e. `status` and `gate` now agree instead of contradicting each other.
- Append a `feedback` after the blockers REVIEW and re-run `status`: the
  `next_command` run number must still be `02`, and `next_gate` becomes the
  pre-existing `EXECUTED (resume Executor work)` wording.

## Evidence Reviewed

- Diff against `main`: only `internal/baton/events.go`, `internal/baton/status_test.go`,
  and `.baton/baton.log` (three event lines) changed. `git diff main --name-only`
  confirms no change to the `runStatus` transition/`next` map keys beyond the
  added override, and no change to `runGate`, `internal/baton/artifact.go`,
  `internal/baton/lint.go`, or `internal/baton/concurrency.go`. Untracked files
  are only this task's PLAN and RUN-01.
- Comparison direction, checked explicitly: `internal/baton/events.go:575` reads
  `return a.reviewResult(review.Path) == "ready-for-user-decision"`. A repo-wide
  grep for `reviewResult` shows the other two call sites
  (`internal/baton/concurrency.go:41`, `internal/baton/lint.go:211`) both use
  `!= "ready-for-user-decision"`. Nothing in the diff compares against
  `== "blockers"`; the only occurrences of the string `blockers` in the diff are
  the literal test input and a test failure message in `status_test.go`. The
  fail-open mistake that shipped earlier today in this codebase is therefore not
  repeated: `reviewResult`'s `""` (read error) is not equal to
  `"ready-for-user-decision"` and lands on the EXECUTED path.
- Real CLI exercise (not just Go tests). Built `go build -o SCRATCH/batonbin ./cmd/baton`,
  created a scratch git repo with a copy of `.baton/` (empty log, no runs),
  and drove it with that binary: `new-round demo-blockers` (task `lctn`, key
  `20260919-1737-demo-blockers`), `append PLANNED`, `append EXECUTED` (RUN-01),
  then `append REVIEW` with a REVIEW-01 written from `templates/review.md`.
  - `Result: blockers` -> `status --task-id lctn` printed
    `next_gate: EXECUTED (delegate Executor)` and
    `next_command: baton prompt exec --task-id lctn --key 20260919-1737-demo-blockers --run-number 02`.
    `gate before-approval` on the same state printed
    `gate failed: REVIEW reports blockers ...; run another EXECUTED -> REVIEW round` (rc=1),
    so `status` and `gate` now agree.
  - `Result: ready-for-user-decision` (fresh copy of the same fixture) ->
    `status` printed `next_gate: user approval -> CLOSE`, no `next_command:`
    line, no `pending_artifact:` line; `gate before-approval` succeeded (rc=0).
  - Logged REVIEW whose artifact was then deleted from disk -> `status` printed
    `next_gate: EXECUTED (delegate Executor)` and the same `--run-number 02`
    command, i.e. it failed closed even though the file had said
    `ready-for-user-decision` before removal. `gate before-approval` on that
    state failed with `no such file or directory` (rc=1).
- Round-number correctness in both orderings, on the real CLI: after the
  blockers REVIEW with no FEEDBACK, `next_command` offered `--run-number 02`;
  after `baton feedback --task-id lctn --summary ...` was appended, `status`
  offered the same `--run-number 02` (with `next_gate: EXECUTED (resume Executor work)`
  from the pre-existing FEEDBACK mapping). `lastRun + 1` is used in both paths
  because `eventReview` joins the existing `eventPlanned, eventFeedback` branch,
  so FEEDBACK does not double-increment.
- Load-bearing test check, on a copy outside the working tree: extracted the
  tree to a scratch directory, overwrote only `internal/baton/events.go` with
  `git show main:internal/baton/events.go`, and ran the three new tests.
  `TestStatusAfterReviewBlockersOffersNextExecutedRound` and
  `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound` both FAILED,
  and for the right reason: the dumped status showed
  `next_gate: user approval -> CLOSE` with no `next_command` line, which is
  exactly the defect this task fixes, not an unrelated panic or setup error.
  `TestStatusAfterReviewReadyOffersUserApproval` still passed, as expected for
  a guard on the unchanged path. The real working tree was not modified.
- Required checks re-run by the reviewer in `/Users/rollcake/lab/baton`:
  `go test ./...` -> `ok github.com/grollcake/baton/internal/baton`, other
  packages have no test files; `go vet ./...` -> no output; `gofmt -l .` -> no
  output.
- RUN-01 honesty: every claim in its Changes, Validation, and Success Criteria
  Status sections was independently reproduced above, including the scratch-copy
  revert method and the "real working tree was never touched" claim (working
  tree still shows only the three expected modified files). The RUN correctly
  reports `None` unresolved risks and correctly returns the `reviewResult`
  refactor as out of scope. No overclaiming found.

## Report To Director

- Suggested summary: status now sends Director back to another EXECUTED round with the correct next run number when the latest REVIEW reports blockers or its artifact cannot be read, and still points at user approval only for ready-for-user-decision
- Artifact path: .baton/runs/20260919-1731-status-blockers-REVIEW-01.md

## Required Next Step

- ask Director to request user approval

## Risks Or Questions

- Judgment requested on the `reviewResult` `(string, error)` follow-up, now that
  three call sites read its `""` sentinel by convention: **slightly more
  urgent**, not blocking.
  - More urgent because this round is the first call site where the `""` case
    produces *user-facing advice* rather than an internal guard. `concurrency.go`
    and `lint.go` use the sentinel to refuse or warn, and a refusal carries its
    own message; `status` instead prints an ordinary "run another EXECUTED
    round" recommendation that is indistinguishable from a genuine blockers
    result. Confirmed on the real CLI above: the missing-artifact run and the
    blockers run produce byte-identical `next_gate`/`next_command` lines, while
    `gate` on the same state names the missing file. With an `error` return,
    `status` could say the REVIEW artifact is unreadable, which is a materially
    better message for the one case where the user's own files are broken.
  - Also more urgent because the convention now has three copies, so the fourth
    reader is likelier to reproduce the fail-open direction that shipped in this
    codebase earlier today. That risk is a function of how many places restate
    the rule, and this round added one.
  - Not *much* more urgent because the new helper `reviewReadyForUser` is the
    only place in `events.go` that touches the sentinel, both new call sites go
    through it, and it carries a doc comment stating the fail-closed intent. The
    round did not multiply raw comparisons inside `events.go`; it added exactly
    one, in the same direction as the existing two.
  - No action needed inside this round. The follow-up should convert
    `reviewResult` to `(string, error)` and update all three call sites at once,
    keeping every one of them fail-closed on error.
- No ambiguity requiring a Director decision.
