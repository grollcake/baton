# REVIEW-01: make an unreadable REVIEW a state callers must handle

Task ID: obbv
Date: 2026-09-19
Planner: Planner
Run: .baton/runs/20260919-1927-review-result-error-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- none

### Nits

- `internal/baton/events.go:513`: the new `runStatus` branch is
  `if review, found := lastRecord(...); found { ... }` with no `else`. If
  `found` were ever false, `next` would keep `user approval -> CLOSE`, where
  the old `!a.reviewReadyForUser(...)` form fell to
  `EXECUTED (delegate Executor)`. Unreachable in practice, because `last`
  comes from `lastEvent(records, taskID)`, so `last == eventReview` implies a
  REVIEW record exists. This is the only place in the diff where the new code
  is not literally equivalent to the old, and it is on the unsafe side, so a
  one-line comment (or setting `next` outside the `found` guard) would be
  worth having.
- Four `reviewResult` call sites now exist where the PLAN named three:
  `concurrency.go:46`, `lint.go:218`, `events.go:595` (`reviewReadyForUser`),
  and `events.go:514` (`runStatus`). The fourth is not a converted external
  reader: `runStatus` previously reached the same decision indirectly through
  `reviewReadyForUser` and now calls the helper itself so it can see the
  error. It is in scope — PLAN step 5 specifies exactly this, including that
  `nextCommand` keeps using `reviewReadyForUser`. Consequence: one `status`
  run reads the artifact twice (once for `next_gate`, once via `nextCommand`).
  Harmless today; a note that the two must stay in agreement would help.
- `requireReviewReady` (`events.go:341`) is still a separate reader of the
  `Result:` line with its own `os.ReadFile` and `hasExactLine`. It was
  explicitly out of scope in the PLAN and it fails closed on its own (the
  `before-approval` gate returned the raw `no such file or directory` error
  in the scratch fixture). Confirmed as the only remaining out-of-helper
  reader; `artifact.go:180` is a format validator, not a decision. A
  follow-up round to fold it onto `reviewResult` is reasonable and is not a
  blocker for this round.
- The new line reads `review_artifact: unreadable: REVIEW-PATH`, two colons in one
  `snake_case_key: value` line. It matches what the PLAN specified, so it is
  recorded as a wording preference only.

## Suggested User Checks

- Drive a scratch task to REVIEW with `Result: ready-for-user-decision`, run
  `baton status --task-id TASK-ID` (expect `next_gate: user approval -> CLOSE`
  and no `review_artifact:` line), then delete the REVIEW artifact and run
  `status` again: `next_gate` must become `EXECUTED (delegate Executor)` and
  a `review_artifact: unreadable: REVIEW-PATH` line must appear between `branch:`
  and `next_command:`.
- With the REVIEW artifact still deleted, run
  `baton gate before-approval --task-id TASK-ID` and confirm it refuses, so an
  unreadable REVIEW cannot reach the approval gate.
- With the REVIEW artifact present but recording `Result: blockers`, confirm
  `status` prints no `review_artifact:` line, so "blockers" and "unreadable"
  stay visibly different states.
- With one task at REVIEW whose artifact has been deleted, open a second task
  and run `baton gate before-execute --task-id SECOND-TASK-ID`: it must still
  refuse, because the unreadable task is still counted in flight.
- Run `baton lint` on a log where a CLOSE follows a REVIEW whose artifact is
  missing from disk: lint must still reject the CLOSE.

## Evidence Reviewed

- `.baton/runs/20260919-1927-review-result-error-PLAN.md`,
  `.baton/runs/20260919-1927-review-result-error-RUN-01.md`.
- `git diff main` over `internal/baton/{lint,concurrency,events}.go` and
  `internal/baton/{lint,commands,status}_test.go`.
- Equivalence at every call site, established independently: a copied tree
  (scratchpad `oldsrc`) with `lint.go`, `concurrency.go` and `events.go`
  restored to `HEAD` while keeping the new tests. It compiles, and
  `go test ./...` there produced exactly one failure,
  `TestStatusAfterMissingReviewArtifactOffersNextExecutedRound`, and only on
  its new `review_artifact: unreadable: ...` assertion. Everything else
  passed on old and new code, including
  `TestLintRejectsCloseAfterUnreadableReview`, the renamed
  `REVIEW with unreadable artifact still blocks` subtest, the
  `next_gate: EXECUTED (delegate Executor)` assertion of the missing-artifact
  status test, and the two new `review_artifact:` absence assertions in the
  ready and blockers status tests. So the tests split as:
  equivalence-proving (pass on both) for all three call-site guards across
  readable-and-ready, readable-and-blocked and unreadable; pinning
  (fail only on old) for the one genuinely new behaviour, the `status` line.
  This matches RUN-01's claim that the call-site tests were written and run
  before the refactor and that the status-line assertions were added after.
  It also answers the load-bearing check: reverting only the source makes the
  status-line assertion fail, and nothing else.
- Unsafe-by-default direction, tested rather than argued: a scratch test in a
  copy of the working tree wrote the careless caller,
  `outcome, _ := app.reviewResult(a missing path)`, and asserted `outcome ==
  reviewUnknown`, that `reviewUnknown` is the zero value, that it compares
  equal to neither `reviewReady` nor `reviewBlockers`, and that both call-site
  shapes (`!= reviewReady` at lint and concurrency, `== reviewReady` at
  `reviewReadyForUser` and `runStatus`) land on the safe side. It passed. No
  call site compares against `reviewBlockers`, so there is no value an
  unreadable read can produce that reads as a normal answer.
- Status output exercised on a real scratch fixture (`bootstrap/` copied to a
  scratchpad project) with a freshly built binary, against a binary built
  from the pre-change source for comparison. Readable-and-ready and
  readable-and-blocked output is byte-identical old vs new, with no
  `review_artifact:` line in either. Unreadable adds exactly one line,
  `review_artifact: unreadable: .baton/runs/KEY-REVIEW-01.md`, after
  `branch:` and before `next_command:`, with `next_gate: EXECUTED (delegate
  Executor)` unchanged. `gate before-approval` on the same unreadable REVIEW
  refuses.
- Doc comments: `grep` for `sentinel`/`empty string` over `internal/baton/*.go`
  returns nothing; the `inFlightTasks` and `reviewReadyForUser` comments and
  the new `reviewOutcome`/`reviewResult` comments describe the typed outcome
  and the returned error, and match the code as written.
- Scope: `git diff main --name-only` lists only the six Go files plus
  `.baton/baton.log` (Director-owned). No transition table, artifact
  contract, template, managed document, `VERSION` or `.baton/bin/` change.
- Required checks on the working tree: `go vet ./...` clean, `gofmt -l .`
  silent, `go test ./...` passes, `go run ./cmd/baton lint` prints
  `baton-lint passed`.

## Report To Director

- Suggested summary: Review verified the `reviewResult` refactor is
  behaviour-identical at every call site by running the new tests against the
  pre-change source on a copied tree; only the new `status` line is new
  behaviour. `reviewUnknown` fails closed for a careless caller, the fourth
  call site is the in-scope `runStatus` read, and all required checks pass.
  No blockers; four nits.
- Artifact path: .baton/runs/20260919-1927-review-result-error-REVIEW-01.md

## Required Next Step

- ask Director to request user approval
