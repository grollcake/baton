# RUN-01: make an unreadable REVIEW a state callers must handle

Task ID: obbv
Date: 2026-09-19
Executor: Executor
Plan: .baton/runs/20260919-1927-review-result-error-PLAN.md
Status: complete

## Changes

- `internal/baton/lint.go`: replaced `reviewResult`'s bare-string return with
  `func (a *App) reviewResult(path string) (reviewOutcome, error)`, added
  `type reviewOutcome int` and constants `reviewUnknown` (iota zero, the
  read-failure/zero value), `reviewReady`, `reviewBlockers`. `checkLog`'s
  `reviewResults` map is now `map[string]reviewOutcome`; the CLOSE guard reads
  `result != reviewReady` (unchanged meaning). The error from `reviewResult`
  is discarded explicitly at this call site with a comment explaining why
  (lint already reports the unreadable artifact via `checkArtifact` on the
  same log line, and `reviewUnknown` still fails the CLOSE guard closed).
- `internal/baton/concurrency.go`: `inFlightTasks`'s REVIEW branch now reads
  `outcome, _ := a.reviewResult(record.Path)` and guards on
  `outcome != reviewReady` (unchanged meaning). Refreshed the doc comment
  that described the old empty-string sentinel to describe `reviewUnknown`
  instead.
- `internal/baton/events.go`: `reviewReadyForUser` now calls
  `a.reviewResult(review.Path)` and returns `err == nil && outcome ==
  reviewReady` (unchanged meaning); refreshed its doc comment for the new
  return shape. `runStatus`'s REVIEW branch now calls `reviewResult` once
  directly (rather than through `reviewReadyForUser`) to get both the outcome
  and the error: `next_gate` still becomes `EXECUTED (delegate Executor)`
  whenever the outcome is not `reviewReady` (unchanged, covers both blockers
  and unreadable), and when the read failed it also prints one extra line,
  the `review_artifact: unreadable:` line, followed by the REVIEW path, after `branch:` and before
  `next_command:`. `nextCommand` still uses `reviewReadyForUser` as before, so
  only `runStatus` grew the extra read. (The line's format is
  `review_artifact: unreadable:` followed by the REVIEW artifact's path.)
- Tests: added `TestLintRejectsCloseAfterUnreadableReview`
  (`internal/baton/lint_test.go`); renamed and changed the concurrency
  subtest to `REVIEW with unreadable artifact still blocks`, driven with
  `ready-for-user-decision` instead of `blockers` before removing the artifact
  (`internal/baton/commands_test.go`); extended
  `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound` to assert the
  new `review_artifact: unreadable:` line and its position between `branch:`
  and `next_command:`, and added absence assertions to
  `TestStatusAfterReviewBlockersOffersNextExecutedRound` and
  `TestStatusAfterReviewReadyOffersUserApproval` (`internal/baton/status_test.go`).

## Validation

- Evidence collected before fixing a user-reported defect: n/a (planned
  refactor, not a user-reported defect).
- Order of work, as Director required: the three pinning tests (lint CLOSE
  guard, concurrency in-flight guard, and the existing
  `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound`, which
  exercises `reviewReadyForUser`) were written/updated first and run against
  the unmodified, pre-refactor `reviewResult` (still `(string, error)`-free,
  returning `""` on read failure). All passed against that code
  (`go test ./internal/baton/... -run
  'TestLintRejectsCloseAfterUnreadableReview|TestInFlightNarrowsToPlannedFeedbackAndBlockedReview'`
  and the existing status test passed unmodified). Only then was the
  refactor made (new `reviewOutcome` type, new signature, all three call
  sites, doc comments, and the `runStatus` line). All the same tests, plus
  the new status assertions added afterward, were run again post-refactor
  and passed.
- Load-bearing check by guard inversion (per-call-site, done on a scratch
  copy under `/private/tmp/.../scratchpad/baton-inv-check`, pre-refactor
  code): flipping the lint CLOSE guard from `result != "ready-for-user-decision"`
  to `result == "blockers"` made `TestLintRejectsCloseAfterUnreadableReview`
  fail (lint no longer rejected the CLOSE). Flipping the concurrency guard
  from `!= "ready-for-user-decision"` to `== "blockers"` made the
  `REVIEW with unreadable artifact still blocks` subtest fail (gate let the
  second task start). Flipping `reviewReadyForUser` from `== "ready-for-user-decision"`
  to `!= "blockers"` made `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound`
  fail (`next_gate` read `user approval -> CLOSE` instead of pointing at
  another EXECUTED round). All three inversions were reverted; none touched
  the real working tree.
- Post-refactor load-bearing check for the new status line, per Director's
  instruction to revert on a copied tree (not the working tree): copied the
  full tree to a scratch git repo, restored `lint.go`, `concurrency.go`, and
  `events.go` to their pre-refactor (`git show HEAD:...`) contents while
  keeping the new tests, and ran `go test ./internal/baton/...`. Result:
  `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound` failed with
  `missing "review_artifact: unreadable: ...-REVIEW-01.md" in ...`, confirming
  that assertion depends on the `runStatus` change and not merely on
  `reviewReadyForUser`'s existing behavior. Scratch copy deleted afterward.
- `go build ./...`: succeeds.
- `go vet ./...`: no findings.
- `gofmt -l .`: no output.
- `go test ./...`: all packages pass (`ok github.com/grollcake/baton/internal/baton`).
- `go run ./cmd/baton lint`: exits 0, `baton-lint passed`.
- Manual scratch-fixture check with a freshly built binary
  (`go build -o SCRATCH/baton-bin ./cmd/baton`, run against a scratch
  project seeded from `bootstrap/.baton`): drove a task through
  `new-round` -> `append PLANNED` -> `append EXECUTED` -> `append REVIEW`
  (REVIEW artifact recorded `Result: ready-for-user-decision`), ran `status
  --task-id`, then deleted the REVIEW artifact file and ran `status
  --task-id` again. Output before deletion:
  ```
  last_event: REVIEW
  next_gate: user approval -> CLOSE
  branch: not-a-git-repo
  ```
  Output after deleting the REVIEW artifact:
  ```
  last_event: REVIEW
  next_gate: EXECUTED (delegate Executor)
  branch: not-a-git-repo
  review_artifact: unreadable: .baton/runs/20260919-1935-demo-task-REVIEW-01.md
  next_command: baton prompt exec --task-id ffpw --key 20260919-1935-demo-task --run-number 02
  ```
  `next_gate` is unchanged from the ready-but-now-unreadable case's expected
  fail-closed behavior, and the new `review_artifact:` line names the path.
  Scratch project deleted afterward.
- Self smoke test after fixing a user-reported defect: n/a (not a
  user-reported defect).

## Report To Director

- Suggested summary: Replaced `reviewResult`'s bare string with
  `(reviewOutcome, error)` (zero value `reviewUnknown`), kept lint's CLOSE
  guard, the concurrency in-flight guard, and `reviewReadyForUser` each as
  their own explicit `!= reviewReady` / `== reviewReady` decision reading the
  typed outcome, and added a `review_artifact: unreadable:` line, naming the REVIEW path, to
  `status` for a REVIEW whose artifact cannot be read. Pinning tests added at
  all three call sites, run before and after the refactor, and each confirmed
  load-bearing by inverting its guard (and, for the new status line, by
  reverting the source on a scratch copy).
- Artifact path: .baton/runs/20260919-1927-review-result-error-RUN-01.md

## Success Criteria Status

- `reviewResult` has signature `(reviewOutcome, error)`; zero value `reviewUnknown` compares equal to neither `reviewReady` nor `reviewBlockers`: met.
- Unreadable REVIEW still makes lint reject a following CLOSE, keeps the task in flight for concurrency, and keeps `next_gate` at `EXECUTED (delegate Executor)`: met.
- One test per call site pins the unreadable behavior, each built by removing a REVIEW artifact recording `ready-for-user-decision`, each confirmed to fail under an inverted guard: met.
- `status` prints `review_artifact: unreadable:` followed by the REVIEW path only when that artifact cannot be read, with no other status line changed in wording or order: met.
- `go test ./...` passes, `go vet ./...` clean, `gofmt -l .` silent, `go run ./cmd/baton lint` passes: met.

## Unresolved Risks

- None identified. The three call sites keep three independent, explicit comparisons against the typed outcome, as directed; no shared helper was introduced.

## Out Of Scope Returned To Director

- `requireReviewReady` (events.go, the `before-approval` gate) still does its own `os.ReadFile` and `hasExactLine` check rather than calling `reviewResult`. This was explicitly out of scope per the PLAN (it already reports the underlying read error itself); flagged there as a possible later round to remove the last duplicate parse of the REVIEW `Result:` line.
