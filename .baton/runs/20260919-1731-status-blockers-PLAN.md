# PLAN: Make status reflect a REVIEW that reports blockers

Task ID: qstp
Date: 2026-09-19
Planner: Planner
Status: complete

## Director Brief

- Goal: When the task's latest REVIEW is not `Result: ready-for-user-decision`, `baton status` must report the next step as another EXECUTED round and offer the exec prompt for the next run number, instead of pointing Director at a `before-approval` gate that is guaranteed to refuse.
- Scope: `internal/baton/events.go` (`runStatus`, `nextCommand`) plus tests in `internal/baton/`; no change to the transition table, `runGate`, or artifact checks.
- Success Criteria: With the latest REVIEW recording `Result: blockers`, `status` prints `next_gate: EXECUTED (delegate Executor)` and `next_command: baton prompt exec --task-id qstp --key KEY --run-number 02`; with `Result: ready-for-user-decision`, `status` still prints `next_gate: user approval -> CLOSE` and no `next_command`.
- Risks: The readability trap in `reviewResult` (returns `""` when the artifact cannot be read) must be handled by comparing `!= "ready-for-user-decision"`, so an unreadable REVIEW fails closed toward another EXECUTED round rather than claiming approval-readiness.
- Required Checks: `go test ./...`, `go vet ./...`, `gofmt -l .`
- Executor Prompt: In `/Users/rollcake/lab/baton/internal/baton/events.go`, make `runStatus` and `nextCommand` consult the latest REVIEW's result. Add one unexported helper on `*App` (e.g. `reviewReadyForUser(records []Record, taskID string) bool`) that returns true only when the latest `eventReview` record exists and `a.reviewResult(record.Path) == "ready-for-user-decision"`. In `runStatus`, when `last == eventReview` and the helper is false, replace the `next` value with the existing EXECUTED wording `"EXECUTED (delegate Executor)"`. In `nextCommand`, add `eventReview` to the existing `case eventPlanned, eventFeedback` branch, but only when the helper is false; when it is true, keep returning `""`. Do not add a new reviewResult variant, do not touch the transition table, `runGate`, or `internal/baton/artifact.go`. Add tests covering both the blockers path and the ready-for-user-decision path, then run `go test ./...`, `go vet ./...`, and `gofmt -l .`.

## Goal

- `baton status` for a task whose latest REVIEW records blockers tells Director to run another EXECUTED round, and hands over the exec prompt command for the next round number.
- `baton status` for a task whose latest REVIEW records `Result: ready-for-user-decision` behaves exactly as today: `next_gate: user approval -> CLOSE` and no `next_command` line.
- An unreadable or missing REVIEW artifact is treated as "not ready", never as ready-for-user-decision.

## Scope

In scope:

- `internal/baton/events.go`: `runStatus`'s `next` map handling for `eventReview`, and `nextCommand`'s `eventReview` case, plus one shared unexported helper.
- Tests under `internal/baton/` (extend `status_test.go` or `coverage_test.go`; `writeReview(t, project, path, taskID, round, result)` already takes the result).

Out of scope:

- The transition table and `runGate` (`before-approval` already refuses correctly; nothing there changes).
- Artifact checks in `internal/baton/artifact.go`.
- Refactoring `reviewResult` to `(string, error)` — see Risks Or Questions.
- `pendingArtifacts`, which already routes `eventReview` to the next RUN and needs no change.

## Plan

1. Add an unexported helper on `*App` in `internal/baton/events.go`, near `nextCommand`, that answers whether the latest REVIEW for a task is ready for the user's decision: look up `lastRecord(records, taskID, eventReview)` and return true only when the record is found and `a.reviewResult(record.Path) == "ready-for-user-decision"`. → verify: one helper, both call sites use it; no second copy of the comparison.
2. In `runStatus`, after computing `next` from the map, override it to the existing EXECUTED wording `"EXECUTED (delegate Executor)"` when `last == eventReview` and the helper reports not-ready. → verify: `status` prints `next_gate: EXECUTED (delegate Executor)` on the blockers path and the unchanged `user approval -> CLOSE` on the ready path.
3. In `nextCommand`, let `eventReview` fall into the existing `case eventPlanned, eventFeedback` round-increment branch when the helper reports not-ready, and keep returning `""` when it reports ready. Update the function's doc comment, which currently claims REVIEW always stays empty. → verify: blockers path prints `next_command: baton prompt exec ... --run-number 02` after a logged `RUN-01`; ready path prints no `next_command`.
4. Add a test that drives `new-round` → PLANNED → EXECUTED(`RUN-01`) → REVIEW(`REVIEW-01`, `blockers`) and asserts both the `next_gate` and `next_command` lines, and a sibling case with `ready-for-user-decision` asserting `next_gate: user approval -> CLOSE` and the absence of `next_command:`. → verify: both assertions fail when the `events.go` change is reverted.
5. Run the required checks. → verify: `go test ./...` passes, `go vet ./...` and `gofmt -l .` produce no output.

## Success Criteria

- With the latest REVIEW recording `Result: blockers` and `RUN-01` logged, `baton status --task-id qstp` contains `next_gate: EXECUTED (delegate Executor)` and `next_command: baton prompt exec --task-id qstp --key KEY --run-number 02`.
- With the latest REVIEW recording `Result: ready-for-user-decision`, `baton status --task-id qstp` still contains `next_gate: user approval -> CLOSE` and contains no `next_command:` line.
- The not-ready decision is written as `!= "ready-for-user-decision"` (equivalently, ready is `== "ready-for-user-decision"`), matching `internal/baton/concurrency.go:41` and `internal/baton/lint.go:211`, so an unreadable REVIEW (`reviewResult` returns `""`) lands on the EXECUTED path.
- Reverting only the `internal/baton/events.go` change makes the new tests fail.
- The existing `TestStatusNextCommand` and `TestStatusReportsPendingArtifacts` continue to pass unchanged.

## Validation

- `go test ./...`: passes, including the new blockers and ready-for-user-decision cases.
- `go vet ./...`: no output.
- `gofmt -l .`: no output.
- Manual revert check: undo the `events.go` edit, run `go test ./internal/baton/`, confirm the new tests fail, then restore.

## Report To Director

- Suggested summary: Make status report another EXECUTED round when the latest REVIEW records blockers
- Artifact path: `.baton/runs/20260919-1731-status-blockers-PLAN.md`

## Risks Or Questions

- Direction chosen, and why: the check is `== "ready-for-user-decision"` for the ready path, so everything else — including `reviewResult`'s `""` for an unreadable artifact — falls to the EXECUTED path. Comparing `== "blockers"` would fail open: an unreadable REVIEW would keep `status` pointing Director at a `before-approval` gate, which is the exact defect this task fixes, and is the failure mode already corrected in `concurrency.go` and `lint.go` today. Failing closed here is also cheap: the worst outcome is `status` suggesting an extra EXECUTED round for a review it could not read, which Director can see and correct, whereas failing open re-creates a guaranteed gate refusal.
- `reviewResult`'s `(string, error)` refactor (recommended in REVIEW-02 of task `uonl`) is deliberately **not** folded in. It is not necessary for correctness here: the `== "ready-for-user-decision"` direction makes the `""` case behave correctly at this call site, and the refactor would touch `lint.go` and `concurrency.go`, which Director placed out of scope. Recommend Director track it as a separate follow-up task covering all three call sites at once.
- Wording risk: `next_gate` reuses the existing string `"EXECUTED (delegate Executor)"` rather than inventing a new phrase, so Director's vocabulary stays consistent. Note that after a blockers REVIEW the protocol's normal path is user FEEDBACK opening `the next RUN round`; the round number this plan offers (`lastRun+1`) is the same one `feedback` would produce, so the suggestion stays correct whether or not FEEDBACK is appended first.
- No ambiguity requiring Director's decision beyond the follow-up above.
