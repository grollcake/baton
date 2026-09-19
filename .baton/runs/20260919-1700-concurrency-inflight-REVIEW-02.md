# REVIEW-02: Fail closed on unreadable REVIEW artifacts

Task ID: uonl
Date: 2026-09-19
Planner: Planner
Run: .baton/runs/20260919-1700-concurrency-inflight-RUN-02.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- None. The REVIEW-01 blocker is closed. `internal/baton/concurrency.go:41`
  now reads `a.reviewResult(record.Path) != "ready-for-user-decision"`, which
  fails closed on the `""` that `reviewResult` (`internal/baton/lint.go:234`)
  returns for an unreadable artifact, and matches lint's reading of the same
  sentinel at `internal/baton/lint.go:211`. I re-ran my own reproduction from
  scratch rather than trusting the Executor's — see Evidence Reviewed — with
  three independent damage modes and a control; all four behaved as the
  amended PLAN requires.

### Nits

- `reviewResult` still signals "could not read" as `""`, a value that means
  nothing on its own and that both call sites now interpret by convention.
  This round's fix makes the two call sites agree (both treat `""` as "not
  ready"), so nothing is broken, but the helper's contract still depends on
  every future caller guessing the safe direction. Recommended as a separate
  follow-up task, not a blocker here: change the signature to
  `(string, error)` (or return a typed result with an explicit `unknown`
  value) so an unreadable artifact is a state the compiler forces a caller to
  handle. That touches `lint.go`, which this round's scope excludes, so it
  should not be folded in now.
- `internal/baton/concurrency.go:41` drops the task when `lastRecord` returns
  `ok == false` — the one remaining fail-open path in the branch. It is
  currently unreachable (the surrounding `switch` only runs when the task's
  last event *is* `REVIEW`, so a `REVIEW` record must exist), so this is a
  latent trap, not a defect. Worth a comment or an explicit
  `if !ok { tasks = append(...) }` if the helper is ever refactored.
- The `os.Stat` nit from REVIEW-01 was resolved by deletion rather than by
  making it load-bearing, and RUN-02 states why: no harness in these subtests
  writes `CONCURRENCY.md`, so the check asserted nothing, and making it
  meaningful would require putting approval-file content under test, which is
  outside this round. I verified the claim independently — my own probe kept
  an `os.Stat` precondition on `.baton/CONCURRENCY.md` and it held in every
  case — so the removal loses no coverage. Defensible; no action.
- `internal/baton/events.go:313` (the `inFlightTasks` -> `a.inFlightTasks`
  receiver change) is carried unchanged from RUN-01 and was already judged
  acceptable in REVIEW-01. Round 02 did not touch it. Restated only so the
  four-file diff is not mistaken for new scope.

## Suggested User Checks

- In a scratch checkout, drive a task to `REVIEW` with `Result: blockers`,
  delete or `chmod 000` the REVIEW artifact, then run `baton gate
  before-execute` for a second task. Confirm it is refused and asks for
  concurrency approval rather than starting.
- Confirm the fail-closed direction is the trade you want: a REVIEW artifact
  that is missing, renamed, or unreadable now blocks a second task until you
  either restore the file or record approval in `.baton/CONCURRENCY.md`. A
  wrong block is recoverable; a wrong pass is a silent cross-task overwrite.
- Re-read the doc comment on `inFlightTasks`
  (`internal/baton/concurrency.go:12-23`) and confirm it describes the
  editing window you want enforced, including the sentence about failing
  closed.
- Confirm the happy path is not over-tightened: with a first task parked at
  `EXECUTED`, and separately at `REVIEW` with `Result:
  ready-for-user-decision`, a second task's `baton gate before-execute` must
  still succeed with no `CONCURRENCY.md` present.
- Decide whether the `reviewResult` `""` sentinel (first nit) should become a
  separate follow-up task, since it is a trap for future callers even though
  both current callers now agree.

## Evidence Reviewed

- `.baton/runs/20260919-1700-concurrency-inflight-PLAN.md` (as amended by
  Director), `-REVIEW-01.md`, `-RUN-01.md`, `-RUN-02.md`.
- Independent reproduction of the REVIEW-01 blocker, written from scratch in
  a copy of the tree under the session scratchpad (the working tree was not
  modified). A throwaway test drives a first task to `REVIEW`, damages its
  logged artifact, asserts `.baton/CONCURRENCY.md` does not exist, then gates
  a second task. Three damage modes all now REFUSE the gate: artifact deleted
  with `Result: blockers`; artifact deleted with `Result:
  ready-for-user-decision`; artifact `chmod 000` with `Result:
  ready-for-user-decision`. The control (artifact readable, `Result:
  ready-for-user-decision`) still ALLOWS the gate. All four PASS.
- Load-bearing check for the new subtest, on the same copy: reverting the
  comparison to `== "blockers"` makes exactly
  `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview/REVIEW blockers with
  missing artifact still blocks` FAIL ("gate started a second task while
  another's blocked REVIEW artifact is missing from disk") while the other
  three subtests PASS. The new test therefore discriminates this round's
  change alone.
- Discrimination check for the three earlier subtests, same copy: with the
  pre-change switch (`eventPlanned, eventExecuted, eventReview,
  eventFeedback`) the `EXECUTED` and `REVIEW ready-for-user-decision`
  subtests FAIL and both `blockers` subtests PASS; with the PLAN's original
  switch (`eventPlanned, eventFeedback` only) those two PASS and both
  `blockers` subtests FAIL. Every subtest still asserts what its name claims.
- Scope: `git diff --stat` shows four files —
  `internal/baton/concurrency.go` (+21/-5 region, confined to `inFlightTasks`
  and its doc comment), `internal/baton/commands_test.go` (+70, all appended
  after line 260, no deleted lines), `internal/baton/events.go` (the single
  RUN-01 receiver line), `.baton/baton.log` (Director-owned). No change to
  the transition table in `internal/baton/events.go`, to
  `templates/concurrency.md` or the `CONCURRENCY.md` format, or to
  `requireConcurrencyApproval`. `TestConcurrentEditingNeedsUserApproval` ends
  at line 260 and the diff adds nothing before it, so it is unmodified.
- `.baton/runs/` mtimes: RUN-01 17:11, REVIEW-01 17:15, RUN-02 17:17. RUN-01
  and REVIEW-01 both predate RUN-02 and were not rewritten.
- Working tree checks, all clean: `go build ./...`, `go vet ./...`,
  `gofmt -l .` (no output), `go test ./...` (`ok
  github.com/grollcake/baton/internal/baton`), and `go test ./internal/baton
  -run 'ConcurrentEditing|InFlight' -v` — `TestConcurrentEditingNeedsUserApproval`
  PASS and all four subtests of
  `TestInFlightNarrowsToPlannedFeedbackAndBlockedReview` PASS.

## Report To Director

- Suggested summary: Round 02 closes the fail-open blocker; gate now refuses
  a second task when a REVIEW artifact is unreadable, verified by independent
  reproduction
- Artifact path: `.baton/runs/20260919-1700-concurrency-inflight-REVIEW-02.md`

## Required Next Step

- Ask Director to request user approval. No blockers remain. Separately,
  surface the `reviewResult` `""` sentinel as a candidate follow-up task.
