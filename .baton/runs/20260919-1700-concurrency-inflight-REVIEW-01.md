# REVIEW-01: Narrow in-flight detection to PLANNED, FEEDBACK, and blocked REVIEW

Task ID: uonl
Date: 2026-09-19
Planner: Planner
Run: .baton/runs/20260919-1700-concurrency-inflight-RUN-01.md
Status: complete

## Decision

Result: blockers

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- `internal/baton/concurrency.go:39` fails open on a safety gate. The REVIEW
  branch tests `a.reviewResult(record.Path) == "blockers"`, but `reviewResult`
  (`internal/baton/lint.go:234`) returns `""` when the artifact cannot be read,
  and `"" != "blockers"`, so a blocked task whose REVIEW artifact is missing,
  renamed, or unreadable stops counting as in flight and a second task starts
  with no user concurrency approval. I reproduced this: with the shipped code,
  deleting a logged `REVIEW` artifact that records `Result: blockers` makes
  `gate before-execute` for a second task succeed. The rest of this codebase
  reads the same sentinel the other way: lint at `internal/baton/lint.go:211`
  uses `result != "ready-for-user-decision"`, so an unreadable artifact counts
  as blockers there. Two call sites of one helper disagreeing on its `""`
  sentinel is a latent trap independent of this round. Fix is one token —
  `a.reviewResult(record.Path) != "ready-for-user-decision"` — which matches
  lint, fails closed, and leaves the whole suite green (verified: `go test
  ./...` passes with that variant, and the deleted-artifact probe then refuses
  the gate). Lint does verify every logged artifact exists, but lint is a
  separate command that need not have run at gate time, so it is not a
  substitute. `requireConcurrencyApproval` failing closed on a missing
  `CONCURRENCY.md` is not a mitigation either: this hole means
  `requireConcurrencyApproval` is never reached. A wrong block is recoverable
  (record the approval or close the task); a wrong pass is the silent
  cross-task overwrite the gate exists to prevent.

### Nits

- `internal/baton/events.go:313` (out of the PLAN's named scope) is acceptable:
  `inFlightTasks` needs the `App` receiver to reach `a.reviewResult`, this is
  the only call site in the tree (`grep -rn inFlightTasks internal/` returns
  the definition, its doc comment, and this one line), and the edit is the
  mechanical `inFlightTasks(...)` -> `a.inFlightTasks(...)` change with no
  transition-table or gate-flow edit. The PLAN's scope line should have been
  amended with the rest of the amendment; the RUN reports the deviation
  honestly, which is the right handling.
- `assertNoConcurrencyNeeded`'s `os.Stat` on `CONCURRENCY.md` can never fail in
  these subtests, since no harness writes that file. It documents intent rather
  than testing anything; the load-bearing assertion is `harness.run(t, "gate",
  ...)`, which fails the test on a non-nil error.
- The RUN says `.baton/baton.log` carries "a pre-existing 2-line modification";
  it is 3 added lines (the `uonl` REQUEST/PLANNED/EXECUTED events). Director-
  owned, untouched by Executor, no action needed.

## Suggested User Checks

- Decide the blocker: should an unreadable or missing REVIEW artifact make the
  concurrency gate refuse (fail closed, matching lint) or allow (current
  behavior)? This is the only open question in the round.
- Sanity-check the intended flow by hand in a scratch checkout: drive a task to
  `EXECUTED`, open a second task, and confirm `baton gate before-execute`
  now succeeds with no `.baton/CONCURRENCY.md` present.
- Confirm the amendment's rule reads correctly to you: a task parked at
  `REVIEW` with `Result: ready-for-user-decision` may be joined by a second
  task, while one at `REVIEW` with `Result: blockers` still demands approval.
- Re-read the new doc comment on `inFlightTasks`
  (`internal/baton/concurrency.go:12-20`) and confirm it describes the editing
  window you actually want enforced.
- Accept or reject the residual risk the PLAN already named: a task sitting at
  `PLANNED` with nobody delegated still trips the gate, so detection remains
  approximate in the conservative direction.

## Evidence Reviewed

- `.baton/runs/20260919-1700-concurrency-inflight-PLAN.md` (as amended by
  Director) and `.baton/runs/20260919-1700-concurrency-inflight-RUN-01.md`.
- `git diff --stat` / `git diff internal/`: four files touched —
  `internal/baton/concurrency.go` (+19/-5 region), `internal/baton/events.go`
  (one line), `internal/baton/commands_test.go` (+58, all appended after line
  260), and `.baton/baton.log` (Director's events). `git diff main...HEAD` is
  empty; all work is in the working tree.
- `TestConcurrentEditingNeedsUserApproval` is unchanged: the diff adds only
  lines after its closing brace, and it passes.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton`. `go vet
  ./...` clean, `gofmt -l internal/` empty.
- `go test ./internal/baton -run 'ConcurrentEditing|InFlight' -v`: both tests
  and all three subtests PASS.
- Negative control, run against a copy of the tree in a scratch directory (the
  working tree was not modified): with the pre-change switch
  (`eventPlanned, eventExecuted, eventReview, eventFeedback`) the `EXECUTED`
  and `REVIEW ready-for-user-decision` subtests FAIL and the `blockers` subtest
  passes; with the PLAN's original switch (`eventPlanned, eventFeedback` only)
  those two pass and the `blockers` subtest FAILS. Each subtest is therefore
  load-bearing and discriminates the behavior it names, and the test genuinely
  fails without the source change.
- Fail-open probe, same scratch copy: a throwaway test that logs a `REVIEW`
  with `Result: blockers`, deletes the artifact, then gates a second task —
  the gate allowed it under the shipped code and refused it under the
  `!= "ready-for-user-decision"` variant, with `go test ./...` green either way.

## Report To Director

- Suggested summary: Review found one blocker — the REVIEW in-flight check
  fails open when the REVIEW artifact is unreadable
- Artifact path: `.baton/runs/20260919-1700-concurrency-inflight-REVIEW-01.md`

## Required Next Step

- Ask Director for a decision on the single blocker; if the user wants the
  gate to fail closed, request `RUN-02` for the one-token change in
  `internal/baton/concurrency.go` plus a subtest covering the unreadable-
  artifact case.
