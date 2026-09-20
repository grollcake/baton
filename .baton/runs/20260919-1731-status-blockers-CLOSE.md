# CLOSE: Make status reflect a REVIEW that reports blockers

Task ID: qstp
Date: 2026-09-19
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260919-1731-status-blockers-REVIEW-01.md

## Acceptance

- The PLAN's success criteria hold. With the latest REVIEW recording
  `Result: blockers`, `status` prints `next_gate: EXECUTED (delegate Executor)`
  and a `next_command` whose run number is the previous RUN plus one. With
  `ready-for-user-decision`, the previous output is unchanged.
- A logged REVIEW whose artifact cannot be read now falls to the EXECUTED path
  instead of claiming the task is ready for approval.
- `status` and `gate before-approval` agree. The defect this task fixes was
  found in real use during task `uonl`, where `status` sent Director at a gate
  that was guaranteed to refuse.
- One round, no blockers. The reviewer exercised the real CLI against a scratch
  `.baton` fixture rather than reading only the Go tests.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- Reviewer CLI runs on a scratch fixture: blockers, ready-for-user-decision,
  and deleted-artifact states each produced the expected `status` output, and
  `gate before-approval` agreed in all three
- Round number verified as `02` both with and without a preceding `FEEDBACK`

## Plan Deviations

- none recorded at the time. This section was added to the CLOSE contract by a
  later task on the same day, and this artifact predates it. Director filled it
  in when backward compatibility was removed, because the repository had never
  been installed anywhere and carrying a legacy path for a file written hours
  earlier cost more than correcting it.

## Remaining Nits

- The installed `.baton/bin/baton` carries this change only after a release
  build. That refresh is task `veee`'s behaviour, not a defect here.
- `TestStatusAfterReviewReadyOffersUserApproval` passes with or without the
  source change. It guards the path this task must not disturb rather than
  proving the fix; two of the three new tests are the load-bearing ones.
- `status` gives the same output for a blocked REVIEW and an unreadable one,
  with no hint that the file could not be read, while `gate` reports the
  underlying error. Fail-closed is correct; the conflation is a diagnosability
  gap.

## Residual Risks

- Three call sites now read `reviewResult`'s `""` sentinel by convention, and
  this is the first one where that sentinel drives user-facing advice rather
  than a refusal message. The reviewer judged the `(string, error)` refactor
  slightly more urgent because of it. Director accepted it as a separate
  follow-up covering all three call sites at once.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Status agrees with the approval gate on a blocked review
- Artifact path: .baton/runs/20260919-1731-status-blockers-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the three
nits, the diagnosability gap, and the accepted follow-up.
