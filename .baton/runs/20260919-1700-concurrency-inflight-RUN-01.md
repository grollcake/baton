# RUN-01: Narrow in-flight detection to PLANNED, FEEDBACK, and blocked REVIEW

Task ID: uonl
Date: 2026-09-19
Executor: Executor
Plan: .baton/runs/20260919-1700-concurrency-inflight-PLAN.md
Status: complete

## Changes

- `internal/baton/concurrency.go`: `inFlightTasks` is now `(a *App) inFlightTasks`. The switch counts `eventPlanned` and `eventFeedback` as in flight (as the PLAN specified), and per the Director amendment also counts `eventReview` only when that task's latest REVIEW record's artifact reports `Result: blockers` (reused via `a.reviewResult`, no new Result parser). `eventExecuted` is dropped, as planned. Doc comment rewritten to explain the real editing window and why REVIEW is conditional (REVIEW:EXECUTED is a valid transition, so a blocked review is about to re-execute).
- `internal/baton/events.go`: updated the one call site (`runGate`) from `inFlightTasks(records, taskID)` to `a.inFlightTasks(records, taskID)`.
- `internal/baton/commands_test.go`: added `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview` with three subtests, each on its own harness: (1) a task at `EXECUTED` does not block `gate before-execute` for a new task with no `CONCURRENCY.md`; (2) a task at `REVIEW` with `Result: ready-for-user-decision` does not block it; (3) a task at `REVIEW` with `Result: blockers` does block it. `TestConcurrentEditingNeedsUserApproval` is unmodified.

No changes to `internal/baton/events.go`'s transition table, `templates/concurrency.md`, or `requireConcurrencyApproval`. No new log events added.

## Validation

- Evidence collected before fixing a user-reported defect: n/a (scoped feature narrowing per Director amendment, not a user-reported defect).
- `go build ./...` and `go vet ./...`: clean.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton` (all packages pass).
- `go test ./internal/baton -run 'ConcurrentEditing|InFlight' -v`: `TestConcurrentEditingNeedsUserApproval` PASS; `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview` and all three subtests PASS.
- `gofmt -l .`: no output (nothing unformatted).
- `git diff --stat`: only `internal/baton/concurrency.go`, `internal/baton/events.go`, and `internal/baton/commands_test.go` changed by this work (`.baton/baton.log` shows a pre-existing 2-line modification from before this task started, untouched by this Executor).
- Self smoke test after fixing a user-reported defect: n/a.

## Report To Director

- Suggested summary: Narrow in-flight detection to PLANNED, FEEDBACK, and REVIEW-with-blockers
- Artifact path: .baton/runs/20260919-1700-concurrency-inflight-RUN-01.md

## Success Criteria Status

- `inFlightTasks` returns only tasks whose last event is `PLANNED`, `FEEDBACK`, or `REVIEW` with `Result: blockers`: met.
- New test proves EXECUTED does not block: met.
- New test proves REVIEW ready-for-user-decision does not block: met.
- New test proves REVIEW blockers does block: met (this is the Director amendment's added criterion, beyond the original PLAN).
- `TestConcurrentEditingNeedsUserApproval` unmodified and green: met.
- No third Result parser written; reused `a.reviewResult`: met.
- Only `internal/baton/concurrency.go` and `internal/baton/commands_test.go` changed per the original PLAN scope: not met as originally scoped — `internal/baton/events.go` also changed, by necessity, to update the one call site to the now-method-receiver `inFlightTasks` (`a.inFlightTasks(...)`), which the Director amendment implicitly required by asking for the App receiver. No behavior in `events.go` beyond that one-line call-site update was touched.

## Unresolved Risks

- Same residual risk the PLAN already named: in-flight detection stays approximate for `PLANNED`/`FEEDBACK` (cannot distinguish "delegated and running" from "not yet delegated"). The Director amendment closes the specific false negative it targeted (a REVIEW with blockers is now caught), so no new gap is introduced by this round.
- If a task's latest REVIEW record's `--path` points to a missing or unreadable artifact, `a.reviewResult` returns `""`, which this code treats as not blocking (not equal to `"blockers"`). This matches `reviewResult`'s existing default-safe behavior elsewhere (e.g. lint) and was not flagged as a concern in the PLAN or amendment, so it was not treated as in scope to change.

## Out Of Scope Returned To Director

- None.
