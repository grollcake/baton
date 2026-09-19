# RUN-02: Fail closed on unreadable REVIEW artifacts

Task ID: uonl
Date: 2026-09-19
Executor: Executor
Plan: .baton/runs/20260919-1700-concurrency-inflight-PLAN.md
Status: complete

## Changes

- `internal/baton/concurrency.go`: the REVIEW branch in `inFlightTasks` now
  reads `a.reviewResult(record.Path) != "ready-for-user-decision"` instead of
  `== "blockers"`. This matches lint's reading of the same helper
  (`internal/baton/lint.go:211`) and fails closed: `reviewResult` returns `""`
  when the artifact cannot be read, and `"" != "ready-for-user-decision"` is
  true, so a missing/unreadable REVIEW artifact now counts as in flight.
  Doc comment updated to state the new condition and that it fails closed.
- `internal/baton/commands_test.go`: added a fourth subtest to
  `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview`,
  "REVIEW blockers with missing artifact still blocks" — drives a task to
  REVIEW with `Result: blockers`, deletes the logged REVIEW artifact from
  disk, then asserts a second task's `gate before-execute` is refused with no
  `CONCURRENCY.md` present. This reproduces the reviewer's exact repro.
- `internal/baton/commands_test.go`: removed the `os.Stat` check on
  `CONCURRENCY.md` inside `assertNoConcurrencyNeeded` (the REVIEW-01 nit). No
  harness in these subtests ever writes that file, so the stat could never
  fail; it asserted nothing and only added noise around the one load-bearing
  assertion (`harness.run(t, "gate", ...)`, which already fails the test on a
  non-nil error). Removed rather than made load-bearing: making it load-bearing
  would mean writing a `CONCURRENCY.md` in these subtests, which is outside
  this round's scope (approval-file content is not under test here).

No changes to the transition table, `templates/concurrency.md`, or
`requireConcurrencyApproval`.

## Validation

- Evidence collected before fixing a user-reported defect: REVIEW-01 already
  reproduced the bug (deleting a logged `Result: blockers` REVIEW artifact
  made `gate before-execute` for a second task succeed); this round encodes
  that repro as `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview/REVIEW
  blockers with missing artifact still blocks`.
- Verified the new subtest is load-bearing: temporarily reverted the
  one-token fix (`== "blockers"`) and reran
  `go test ./internal/baton -run TestInFlightNarrowsToPlannedFeedbackAndBlockedReview -v`.
  Only the new "missing artifact" subtest failed
  (`gate started a second task while another's blocked REVIEW artifact is
  missing from disk`); the other three subtests still passed. Restored the
  fix and reran the same command: all four subtests PASS.
- `go build ./...` and `go vet ./...`: clean.
- `gofmt -l .`: no output.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton` (all
  packages pass).
- `go test ./internal/baton -run 'ConcurrentEditing|InFlight' -v`:
  `TestConcurrentEditingNeedsUserApproval` PASS (unmodified);
  `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview` and all four
  subtests PASS.
- `git diff --stat`: `internal/baton/concurrency.go`,
  `internal/baton/commands_test.go`, `internal/baton/events.go` (untouched by
  this round, carried from RUN-01), and `.baton/baton.log` (Director-owned,
  untouched by this Executor).
- Self smoke test after fixing a user-reported defect: n/a (this is a
  reviewer-identified gate defect on unreleased work, not a user-reported
  defect; the reproduction-then-fix verification above serves the same
  purpose).

## Report To Director

- Suggested summary: Fail closed on unreadable REVIEW artifacts in the
  concurrency gate; nit resolved by removing a non-load-bearing assertion
- Artifact path: .baton/runs/20260919-1700-concurrency-inflight-RUN-02.md

## Success Criteria Status

- `internal/baton/concurrency.go:39`'s REVIEW check fails closed and matches
  lint's reading of `reviewResult`: met.
- New regression test proves a second task's `gate before-execute` is
  refused when another open task's last event is REVIEW and its logged
  `Result: blockers` artifact is missing from disk: met.
- The three existing subtests in
  `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview` still pass: met.
- `TestConcurrentEditingNeedsUserApproval` unchanged and green: met.
- `assertNoConcurrencyNeeded`'s non-load-bearing `os.Stat` nit addressed:
  met (removed, per stated rationale above).
- No change to the transition table, `CONCURRENCY.md` format, or
  `requireConcurrencyApproval`: met.

## Unresolved Risks

- None new. The residual risk already named in the PLAN and RUN-01 (in-flight
  detection for `PLANNED`/`FEEDBACK` stays approximate) is unchanged by this
  round.

## Out Of Scope Returned To Director

- None.
