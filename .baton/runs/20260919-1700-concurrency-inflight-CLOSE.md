# CLOSE: Narrow in-flight detection to PLANNED and FEEDBACK

Task ID: uonl
Date: 2026-09-19
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260919-1700-concurrency-inflight-REVIEW-02.md

## Acceptance

- The PLAN's success criteria hold as amended: a second task's `gate
  before-execute` now succeeds with no `CONCURRENCY.md` while another open task
  sits at `EXECUTED` or at `REVIEW` with `Result: ready-for-user-decision`, and
  still fails while another sits at `PLANNED` or at `REVIEW` with blockers.
- The Director amendment made after the PLAN was written, that `REVIEW` counts
  as in flight only when the task's latest REVIEW does not record
  `ready-for-user-decision`, is implemented and covered by tests.
- REVIEW-01 raised one blocker, a fail-open on an unreadable REVIEW artifact.
  Round 02 closed it, and REVIEW-02 confirmed the fix by an independent
  reproduction rather than by trusting the Executor's report.
- No new log events, no transition-table change, no change to the
  `CONCURRENCY.md` format or to `requireConcurrencyApproval`.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- Reviewer reproduction, three damage modes plus a readable control: the gate
  refuses a second task whenever the REVIEW artifact cannot be read, and allows
  it in the control

## Plan Deviations

- none recorded at the time. This section was added to the CLOSE contract by a
  later task on the same day, and this artifact predates it. Director filled it
  in when backward compatibility was removed, because the repository had never
  been installed anywhere and carrying a legacy path for a file written hours
  earlier cost more than correcting it.

## Lesson Candidates

- A helper that signals failure as an empty string lets each call site pick
  which way the unknown falls. One picked wrong and opened a safety gate.

## Remaining Nits

- `reviewResult` reports an unreadable artifact as `""`, which both call sites
  now read the same way but which every future caller must interpret by
  convention. Recommended as a follow-up task, changing it to return an error
  or an explicit unknown value, because the fix touches `lint.go`.
- `inFlightTasks` drops a task when `lastRecord` returns `ok == false`. That
  path is currently unreachable, since the branch runs only when the task's
  last event is `REVIEW`, so it is a latent trap rather than a defect.

## Residual Risks

- `PLANNED` still cannot distinguish a task that has been delegated from one
  that has only been planned. The log records completed phases, so it has no
  representation for work in progress. Detection is narrower than before, not
  exact.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Narrow in-flight detection, failing closed on unreadable reviews
- Artifact path: .baton/runs/20260919-1700-concurrency-inflight-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved this task after reading the REVIEW-02
report, its five manual checks, and the fail-closed trade it names.
