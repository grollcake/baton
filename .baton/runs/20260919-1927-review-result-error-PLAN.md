# PLAN: make an unreadable REVIEW a state callers must handle

Task ID: obbv
Date: 2026-09-19
Planner: Planner
Status: complete

## Director Brief

- Goal: replace `reviewResult`'s bare-string result so an unreadable REVIEW artifact is a distinct, unmissable state, keep all three call sites failing closed exactly as they do today, and let `status` say when the REVIEW artifact could not be read.
- Scope: `internal/baton/lint.go`, `internal/baton/concurrency.go`, `internal/baton/events.go`, and their tests.
- Success Criteria: `reviewResult` returns `(reviewOutcome, error)` with a zero value that is neither ready nor blockers; one test per call site pins the unreadable case; `status` prints one extra `review_artifact: unreadable:` line naming the REVIEW path for a REVIEW whose artifact cannot be read while `next_gate` stays `EXECUTED (delegate Executor)`.
- Risks: this is a safety-relevant refactor; a silent flip at one call site would let an unreadable REVIEW open the approval gate (it already shipped once as `== "blockers"`). The pinning tests are the mitigation and are mandatory, not optional.
- Required Checks: `go test ./...`, `go vet ./...`, `gofmt -l .` (silent), `go run ./cmd/baton lint`.
- Executor Prompt: In `/Users/rollcake/lab/baton`, change `internal/baton/lint.go`'s `reviewResult` to `func (a *App) reviewResult(path string) (reviewOutcome, error)` with `type reviewOutcome int` and constants `reviewUnknown` (iota zero), `reviewReady`, `reviewBlockers`; return `(reviewUnknown, err)` on read failure, `(reviewReady, nil)` for `Result: ready-for-user-decision`, `(reviewBlockers, nil)` otherwise. Update the three call sites so behaviour is unchanged: lint.go:213 stores the outcome and still errors on CLOSE unless the outcome is `reviewReady`; concurrency.go:41 still counts the task as in flight unless the outcome is `reviewReady`; events.go:581 `reviewReadyForUser` still returns false unless the outcome is `reviewReady`. In `runStatus`, when the last event is REVIEW and `reviewResult` returned an error, print one extra line `review_artifact: unreadable:` followed by the REVIEW path, after the `branch:` line and before `next_command:`; leave `next_gate` unchanged. Refresh the now-stale doc comments on `inFlightTasks` and `reviewReadyForUser` that describe the old empty-string sentinel. Add or strengthen one test per call site for the unreadable case as described in the PLAN's Plan section. Do not touch the transition table, artifact contracts, templates, managed documents, `VERSION`, or `.baton/bin/`. Run the required checks; use `go run ./cmd/baton` rather than the installed binary.

## Goal

- `reviewResult` no longer signals "could not read the artifact" as a value that reads like a normal answer. The unreadable case is carried by a returned `error` and by a named zero value (`reviewUnknown`) that matches neither `reviewReady` nor `reviewBlockers`, so a call site that ignores it still falls to the safe side instead of silently reading as "ready".
- All three existing call sites keep exactly today's behaviour for an unreadable artifact, and a test at each call site pins that behaviour.
- `status` distinguishes "REVIEW reports blockers" from "REVIEW artifact could not be read", closing the diagnosability gap a reviewer recorded in task qstp.

## Scope

In scope:

- `internal/baton/lint.go`: the `reviewResult` signature, the `reviewOutcome` type and constants, and the `reviewResults` map plus the CLOSE check in `checkLog`.
- `internal/baton/concurrency.go`: the `eventReview` branch of `inFlightTasks` and its doc comment.
- `internal/baton/events.go`: `reviewReadyForUser`, its doc comment, and the REVIEW branch of `runStatus`.
- `internal/baton/lint_test.go`, `internal/baton/commands_test.go`, `internal/baton/status_test.go`: the three pinning tests.

Out of scope:

- `requireReviewReady` (events.go:335). It reads the same artifact with its own `os.ReadFile` and already reports the underlying error, which is the behaviour Defect D wants elsewhere. Folding it into `reviewResult` is a plausible follow-up, not this task; leaving it alone keeps this diff to the three named call sites. Flagged to Director as a possible later round.
- The transition table, artifact contracts, `.baton/templates/`, managed documents, `VERSION`, `.baton/bin/`.
- Any wording change to lint's `CLOSE ... follows a REVIEW reporting blockers` message or to the concurrency gate message. Defect D asks only for `status`; widening the wording change widens the blast radius for no behavioural gain.

## Plan

1. Design choice. In `lint.go`, add `type reviewOutcome int` with `reviewUnknown reviewOutcome = iota`, `reviewReady`, `reviewBlockers`, and change the helper to `func (a *App) reviewResult(path string) (reviewOutcome, error)`. Reasons, against the alternatives Director listed:
   - A second return value is the only option that makes the compiler participate: `x := a.reviewResult(p)` stops compiling, so every existing and future call site must write the two-value form and visibly decide what to do with the error. A single named type with an `unknown` member compiles unchanged at a call site that forgets it.
   - Making `reviewUnknown` the zero value is the belt to that brace. `go` still permits `result, _ := ...`; when someone does, `result` is `reviewUnknown`, which equals neither `reviewReady` nor `reviewBlockers`, so a comparison written either way round still refuses to treat the unreadable artifact as ready. This is what makes the earlier `== "blockers"` bug unwritable: there is no value that an unreadable read can produce which compares equal to a normal answer.
   - A boolean pair `(ready, known bool)` gives the same information but invites the same silent drop (`ready, _ :=`) with `ready == false` carrying no trace of why, and it reads worse at the lint call site, which stores the value in a map across log lines.
   - `(string, error)` keeps the stringly-typed comparison that produced the original bug (`"blockers"` vs `"ready-for-user-decision"` is still a typo away from a safety flip), so the named type is worth the few extra lines.
   - Keep the existing rule that a readable artifact without the exact `Result: ready-for-user-decision` line is `reviewBlockers`; only the read failure changes shape.
   - Verify: `go build ./...` fails until every call site is updated, then passes.
2. Update `lint.go`'s `checkLog`. Change `reviewResults` to `map[string]reviewOutcome`, and at line 213 store the outcome, discarding the error explicitly (`outcome, _ := state.app.reviewResult(record.Path)` with a short comment saying lint already reports the unreadable artifact through `checkArtifact` on the same line). At the CLOSE branch keep the guard as `found && result != reviewReady`. Safe side here: unknown counts as blockers, so a CLOSE that follows an unreadable REVIEW is still a lint error.
   - Verify: `TestLintRejectsCloseAfterBlockers` still passes unchanged.
3. Update `concurrency.go`'s `inFlightTasks`. Read the outcome and keep the guard as `outcome != reviewReady`, ignoring the error explicitly. Safe side here: unknown counts as in flight, so the concurrency gate still refuses to start a second task beside a task whose REVIEW artifact cannot be read. Rewrite the doc comment sentence that describes the empty-string sentinel so it describes `reviewUnknown` instead.
   - Verify: `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview` passes.
4. Update `events.go`'s `reviewReadyForUser` to `outcome, err := a.reviewResult(review.Path)` and `return err == nil && outcome == reviewReady`. Safe side here: unknown is not ready, so `next_gate` and `nextCommand` still point at another EXECUTED round. Rewrite its doc comment for the new shape.
   - Verify: `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound` passes.
5. Defect D. In `runStatus`, the REVIEW branch currently calls `a.reviewReadyForUser`. Give it the outcome and the error: look up the last REVIEW record, call `reviewResult` once, set `next = "EXECUTED (delegate Executor)"` when the outcome is not `reviewReady` (unchanged), and remember whether the read failed. After the existing `next_gate`/`branch` line and before `next_command`, print one extra line only in the read-failure case:

   `review_artifact: unreadable:` followed by the REVIEW artifact path

   This matches the existing `snake_case_key: value` vocabulary already used by `last_event`, `next_gate`, `next_command`, and `pending_artifact`, adds no new line in any other state, and leaves `next_gate` wording untouched so no existing assertion moves. Keep `reviewReadyForUser` as the helper `nextCommand` uses, so only `runStatus` grows the extra read.
   - Verify: the status test below asserts the new line, and the ready/blockers status tests assert it is absent.
6. Pin the unreadable case at each call site. Each test constructs the unreadable artifact the same portable way: write a well-formed REVIEW artifact, log the REVIEW event, then `os.Remove` the artifact file before the command under test runs, so `os.ReadFile` inside `reviewResult` takes its error branch. (Chmod-based unreadability is avoided: it is a no-op for a root test runner and behaves differently on Windows.) Crucially, each test writes the REVIEW with `Result: ready-for-user-decision`, so the safe outcome can only come from the unreadable branch and not from a `blockers` line that would produce the same result anyway.
   - lint call site, new test in `lint_test.go` (model it on `TestLintRejectsCloseAfterBlockers`, which uses `writeLog` directly and so bypasses `append`'s own validation): write PLAN, RUN-01, REVIEW-01 with `ready-for-user-decision`, and CLOSE; write the log through all five events; `os.Remove` the REVIEW artifact; assert `harness.fail("lint")` returns an error and that `harness.err` contains `CLOSE on line`/`follows a REVIEW reporting blockers`. Assert on that substring rather than on failure alone, because lint independently reports the missing artifact on the REVIEW line and a bare failure would pass even if the CLOSE guard were removed.
   - concurrency call site, in `commands_test.go`: change the existing subtest `REVIEW blockers with missing artifact still blocks` to drive the first task to REVIEW with `ready-for-user-decision` (not `blockers`) before removing the artifact, and rename it to `REVIEW with unreadable artifact still blocks`. As written today the subtest passes even if the unreadable branch were treated as ready, because `blockers` blocks on its own; after the change only the unreadable branch can produce the block. Assert `harness.fail("gate", "before-execute", ...)` returns an error for the second task.
   - status call site, in `status_test.go`: keep `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound` (it already uses `statusThroughReview(..., "ready-for-user-decision", true)`, the exact construction above) and extend it to also assert the new `review_artifact: unreadable:` line carrying the REVIEW path. Add an assertion to the existing blockers-path status test that the `review_artifact:` line is absent, so the blockers and unreadable states stay distinguishable in both directions.
   - Verify: each of the three tests fails when its guard is inverted by hand, then passes as written. The executor should actually try the inversion once per test before finishing, and report that it did.
7. Run the required checks and record their output in the RUN.

## Success Criteria

- `reviewResult` has signature `(reviewOutcome, error)`; `reviewOutcome`'s zero value is `reviewUnknown` and compares equal to neither `reviewReady` nor `reviewBlockers`; no call site can obtain the old bare string.
- An unreadable REVIEW artifact still: makes lint reject a following CLOSE; keeps the task counted as in flight by the concurrency gate; and keeps `status`'s `next_gate` at `EXECUTED (delegate Executor)`.
- One test per call site pins that unreadable behaviour, each built by removing a REVIEW artifact that records `ready-for-user-decision`, and each fails if its guard is inverted.
- `status` for a REVIEW whose artifact cannot be read prints `review_artifact: unreadable:` followed by the REVIEW path; `status` for a readable REVIEW (ready or blockers) prints no such line, and no existing `status` line changes wording or order.
- `go test ./...` passes, `go vet ./...` is clean, `gofmt -l .` prints nothing, `go run ./cmd/baton lint` passes.

## Validation

- `go test ./...`: all packages pass.
- `go vet ./...`: no findings.
- `gofmt -l .`: no output.
- `go run ./cmd/baton lint`: exits 0 (use `go run`, not `.baton/bin/baton`, which may lag the working tree).
- Manual: drive a scratch task to REVIEW, delete the REVIEW artifact, run `go run ./cmd/baton status --task-id` for that task and confirm both the unchanged `next_gate` and the new `review_artifact: unreadable:` line.

## Report To Director

- Suggested summary: Plan to replace `reviewResult`'s bare string with `(reviewOutcome, error)`, keep all three call sites failing closed, and have `status` name an unreadable REVIEW artifact.
- Artifact path: `.baton/runs/20260919-1927-review-result-error-PLAN.md`

## Risks Or Questions

- Safety-relevant refactor. The failure mode is a silent behaviour flip at one of the three call sites, which is how the `== "blockers"` bug shipped earlier today. Mitigation: the three pinning tests, plus the requirement that the executor verifies each test fails under an inverted guard.
- `reviewUnknown` as the zero value means a future `result, _ :=` still fails closed. This is deliberate defence in depth, not permission to drop the error; the intended form at every call site is the two-value one.
- The lint call site legitimately discards the error, because lint already reports the unreadable artifact through `checkArtifact` on the same log line. Executor should keep that discard explicit and commented so it does not read as an oversight.
- Out of scope but noted for Director: `requireReviewReady` in `events.go` is a fourth reader of the same `Result:` line with its own `os.ReadFile`. Consolidating it onto `reviewResult` would remove the last duplicate parse, but it is not one of the three call sites Director named and is left for a later round.
