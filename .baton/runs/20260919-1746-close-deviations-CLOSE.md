# CLOSE: Record plan deviations in CLOSE

Task ID: jiaq
Date: 2026-09-19
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260919-1746-close-deviations-REVIEW-01.md

## Acceptance

- New CLOSE artifacts must carry `## Plan Deviations` with at least one list
  item, and `- none` is the accepted way to say the work matched its PLAN.
- The rule sits behind `allowLegacy`, so the two CLOSE artifacts already in this
  repository keep passing lint while `append` and `check-artifact` hold new ones
  to the current contract. Both files are byte-identical to before this task.
- The contract is stated where it is enforced: `PROTOCOL.md` in both copies, the
  `close.md` template in both copies, and the Korean artifact list in
  `PROTOCOL-GUIDE.md`.
- One round, no blockers. This artifact is the first CLOSE written under the new
  rule and was validated by a binary built from this change.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass, with both older CLOSE artifacts
  unmodified
- `baton check-artifact CLOSE` on this file with the rebuilt binary: pass
- Reviewer CLI runs: a CLOSE missing the heading and a CLOSE whose section has
  no item were both refused; `- none` was accepted

## Plan Deviations

- none. The Executor implemented the PLAN's Executor Prompt as written. Director
  added a note about the embedded-document drift the documentation edits would
  cause in the installed binary, which set expectations for validation and
  reporting but changed no scope, no file list, and no success criterion.

## Lesson Candidates

- Requiring a heading is not requiring content. A section that may be empty is
  a section every artifact passes by having.

## Remaining Nits

- `validClose` in `coverage_test.go` and `writeClose` in `testutil_test.go` both
  build a CLOSE body, with slightly different content. The PLAN asked for both;
  recorded as duplication to watch.
- The template ships the section with an angle-bracket placeholder, so an
  unfilled CLOSE is caught by the existing unresolved-placeholder check first.
  The new non-empty rule only applies once an author deletes the placeholder and
  leaves the section bare.

## Residual Risks

- An author who did deviate can still write `- none`, and no check can detect
  that. The rule turns an invisible omission into an explicit claim that is
  wrong on the record and reviewable, which is the part a file-based protocol
  can enforce.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Require Plan Deviations in new CLOSE artifacts
- Artifact path: .baton/runs/20260919-1746-close-deviations-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming both nits, the
`none` escape hatch, and the reviewer's judgement on it.
