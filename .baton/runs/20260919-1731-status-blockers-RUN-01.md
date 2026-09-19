# RUN-01: Make status reflect a REVIEW that reports blockers

Task ID: qstp
Date: 2026-09-19
Executor: Executor
Plan: .baton/runs/20260919-1731-status-blockers-PLAN.md
Status: complete

## Changes

- `internal/baton/events.go`: added `(a *App) reviewReadyForUser(records []Record, taskID string) bool`, a single unexported helper that looks up the latest `eventReview` record for a task and returns true only when `a.reviewResult(record.Path) == "ready-for-user-decision"` (false when no REVIEW is logged or the REVIEW artifact cannot be read).
- `internal/baton/events.go`: `runStatus` now overrides the computed `next` value to `"EXECUTED (delegate Executor)"` when `last == eventReview` and `reviewReadyForUser` is false; the ready path (`next_gate: user approval -> CLOSE`) is unchanged.
- `internal/baton/events.go`: `nextCommand` folds `eventReview` into the existing `case eventPlanned, eventFeedback` round-increment branch; when `last == eventReview` and `reviewReadyForUser` is true it returns `""` (unchanged behavior), otherwise it falls through to the same round-increment logic used for PLANNED/FEEDBACK. Updated the function's doc comment to describe this.
- `internal/baton/status_test.go`: added a `statusThroughReview` test helper (drives `new-round` -> PLANNED -> EXECUTED(RUN-01) -> REVIEW-01 with a given `Result`, optionally deleting the REVIEW artifact from disk after logging it) and three tests: `TestStatusAfterReviewBlockersOffersNextExecutedRound`, `TestStatusAfterReviewReadyOffersUserApproval`, `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound`.

## Validation

- Evidence collected before fixing a user-reported defect: n/a (feature/plan work, not a user-reported defect).
- `go build ./...`: clean, no output.
- `go test ./...`: passes (`ok github.com/grollcake/baton/internal/baton`), including the three new tests and the unchanged `TestStatusNextCommand` / `TestStatusReportsPendingArtifacts`.
- `go vet ./...`: no output.
- `gofmt -l .`: no output.
- Self smoke test after fixing a user-reported defect: n/a.
- Load-bearing verification: copied the working tree to a scratch directory outside the repo (`/private/tmp/.../scratchpad/baton-revert-check`), ran `git checkout -- internal/baton/events.go` there to revert only the source change (keeping the new tests and `.baton/baton.log` state), then ran `go test ./internal/baton/ -run 'TestStatusAfterReviewBlockersOffersNextExecutedRound|TestStatusAfterReviewReadyOffersUserApproval|TestStatusAfterMissingReviewArtifactOffersNextExecutedRound' -v`. Result: `TestStatusAfterReviewBlockersOffersNextExecutedRound` and `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound` FAILED (both showed `next_gate: user approval -> CLOSE` instead of `EXECUTED (delegate Executor)`, confirming they are load-bearing for the fix), while `TestStatusAfterReviewReadyOffersUserApproval` still PASSED (the ready path is unchanged by this fix, as expected). The scratch copy was then deleted; the real working tree was never touched by the revert.

## Report To Director

- Suggested summary: status now points at another EXECUTED round (with the correct next-round exec prompt) when the latest REVIEW reports blockers or its artifact is unreadable, and still points at user approval only when the latest REVIEW records ready-for-user-decision.
- Artifact path: .baton/runs/20260919-1731-status-blockers-RUN-01.md

## Success Criteria Status

- `status` with latest REVIEW `Result: blockers` prints `next_gate: EXECUTED (delegate Executor)` and `next_command: baton prompt exec --task-id qstp --key KEY --run-number 02`: met.
- `status` with latest REVIEW `Result: ready-for-user-decision` still prints `next_gate: user approval -> CLOSE` and no `next_command:` line: met.
- Not-ready decision is written as `!= "ready-for-user-decision"` (via `== "ready-for-user-decision"` in `reviewReadyForUser`), so an unreadable/missing REVIEW artifact lands on the EXECUTED path: met.
- Reverting only the `internal/baton/events.go` change makes the new tests fail: met (verified on a scratch copy; see Validation).
- Existing `TestStatusNextCommand` and `TestStatusReportsPendingArtifacts` continue to pass unchanged: met.
- No changes to the transition table, `runGate`, `internal/baton/artifact.go`, `internal/baton/lint.go`, or `internal/baton/concurrency.go`: met (only `internal/baton/events.go` and `internal/baton/status_test.go` were touched).
- No second way to read the review result added; `a.reviewResult` is reused by the new helper: met.

## Unresolved Risks

- None.

## Out Of Scope Returned To Director

- The `reviewResult` `(string, error)` refactor recommended in REVIEW-02 of task `uonl` remains out of scope per the PLAN's explicit decision; not touched here.
