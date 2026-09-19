# PLAN: Narrow in-flight detection to PLANNED and FEEDBACK

Task ID: uonl
Date: 2026-09-19
Planner: Planner
Status: complete

## Director Brief

- Goal: `inFlightTasks` treats only `PLANNED` and `FEEDBACK` as in flight, so the concurrency approval gate stops firing for tasks whose delegate has already finished editing.
- Scope: `internal/baton/concurrency.go` (the `switch` in `inFlightTasks` and its doc comment) plus one new test in `internal/baton/commands_test.go`.
- Success Criteria: `gate before-execute` for a new task succeeds with no `CONCURRENCY.md` while another open task sits at `EXECUTED` or at `REVIEW`; it still fails while another open task sits at `PLANNED`.
- Risks: detection stays approximate — `PLANNED` cannot tell "delegated and running" from "planned, not yet delegated", and a task at `REVIEW` whose `before-execute` gate already passed is genuinely editing but will no longer be counted. See Risks Or Questions.
- Required Checks: `go test ./...`
- Executor Prompt: In `/Users/rollcake/lab/baton`, edit `internal/baton/concurrency.go` so the `switch` in `inFlightTasks` matches only `eventPlanned` and `eventFeedback` (drop `eventExecuted` and `eventReview`), and update the function's doc comment to say editing happens between the `before-execute` gate passing and the `EXECUTED` event being appended. Add one test to `internal/baton/commands_test.go` proving that an open task whose last event is `EXECUTED`, and separately one whose last event is `REVIEW`, no longer blocks `gate before-execute` for another task with no `CONCURRENCY.md` present. Use the existing `newHarness`, `writePlan`, `writeRun`, `writeReview` helpers; give each case its own harness so no other task is left at `PLANNED`. Do not add log events, change the gate transition table in `internal/baton/events.go`, or touch `CONCURRENCY.md` format or the approval checks in `requireConcurrencyApproval`. `go test ./...` must pass and `TestConcurrentEditingNeedsUserApproval` must stay unchanged and green.

## Goal

- `inFlightTasks` counts a task as in flight only when its last event is `PLANNED` or `FEEDBACK`, the window in which a delegate may hold the checkout.
- A second task can start while an earlier task waits at `EXECUTED` (Executor finished) or at `REVIEW` (awaiting the user's decision), with no `CONCURRENCY.md` record.
- The gate still refuses an unapproved second task when a real editing window is open.

## Scope

In scope:

- `internal/baton/concurrency.go`: the event `switch` inside `inFlightTasks` and the function's doc comment, which currently says "already have a PLAN" and stops being accurate.
- `internal/baton/commands_test.go`: one new test covering the `EXECUTED` and `REVIEW` cases.

Out of scope:

- New log events or any change to `internal/baton/events.go`, including the `before-execute` / `before-review` / `before-approval` transition table.
- `templates/concurrency.md`, the `CONCURRENCY.md` format, and every check inside `requireConcurrencyApproval`.
- The `closed` handling in `inFlightTasks` (`CLOSE` / `RUN_DONE`), which stays as it is.
- `DIRECTOR.md` and `PROTOCOL.md` wording; both say "in flight" without enumerating events, so they remain correct.

## Plan

1. In `inFlightTasks`, reduce the `switch` cases to `eventPlanned` and `eventFeedback`. -> verify: `go build ./...` and `go vet ./...` are clean.
2. Rewrite the doc comment to state the real window: a delegate may be editing between `gate before-execute` passing and the `EXECUTED` event being appended, which is last event `PLANNED` or `FEEDBACK`. -> verify: comment no longer claims "already have a PLAN".
3. Add `TestFinishedTasksDoNotBlockNewWork` (or a similarly named test) to `internal/baton/commands_test.go` with two independent harnesses: one driving a task to `EXECUTED`, one driving a task to `REVIEW`, each then opening a second task and asserting `gate before-execute` succeeds with no `CONCURRENCY.md` on disk. -> verify: the new test fails against the pre-change `switch` and passes after it.
4. Run the full suite. -> verify: `go test ./...` passes, including the untouched `TestConcurrentEditingNeedsUserApproval`.

## Success Criteria

- `inFlightTasks` returns only tasks whose last event is `PLANNED` or `FEEDBACK`.
- New test: with an open task at `EXECUTED`, `gate before-execute` for a second task succeeds and no `CONCURRENCY.md` exists.
- New test: with an open task at `REVIEW`, `gate before-execute` for a second task succeeds and no `CONCURRENCY.md` exists.
- `TestConcurrentEditingNeedsUserApproval` is unmodified and still proves the gate refuses an unapproved second task, refuses approval that omits a running task, and refuses approval not signed by the user.
- No file outside `internal/baton/concurrency.go` and `internal/baton/commands_test.go` is changed.

## Validation

- `go test ./...`: all packages pass; `internal/baton` is `ok`.
- `go test ./internal/baton -run 'ConcurrentEditing|FinishedTasks' -v`: both the existing and the new test pass.
- `git diff --stat`: exactly two files changed.

## Report To Director

- Suggested summary: Narrow in-flight detection to PLANNED and FEEDBACK
- Artifact path: `.baton/runs/20260919-1700-concurrency-inflight-PLAN.md`

## Risks Or Questions

- Detection stays approximate, not exact. The log cannot distinguish "PLAN appended and Executor is running" from "PLAN appended, nobody delegated yet", so a task parked at `PLANNED` still trips the gate even when no one holds the checkout. This change removes two clear false positives; it does not make detection correct.
- New false negative, accepted within Director's scope: the transition table allows `gate before-execute` after `REVIEW` (a further round without an intervening `FEEDBACK` event, see `internal/baton/events.go:303`). A task in that state whose gate already passed is genuinely editing, and after this change it will no longer block another task. Making that case safe would need a new event or a gate-passed marker, which Director excluded.
- Loosening a safety gate cannot be proven safe by tests; the tests can only prove the gate still refuses the unapproved-`PLANNED` case. The residual risk above is a judgment call for the user.
