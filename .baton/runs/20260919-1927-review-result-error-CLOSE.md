# CLOSE: Make an unreadable review a state callers must handle

Task ID: obbv
Date: 2026-09-19
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260919-1927-review-result-error-REVIEW-01.md

## Acceptance

- `reviewResult` returns a typed outcome and an error instead of signalling an
  unreadable artifact as the empty string. The unknown outcome is the zero
  value, so a caller that ignores the error still lands on the safe side. The
  mistake that shipped earlier today, comparing the old string against
  "blockers" and thereby opening a safety gate on an unreadable review, can no
  longer be written.
- The three call sites keep three explicit decisions rather than one shared
  comparison, because safe means a different sentence at each of them.
- `status` now names a REVIEW artifact it could not read, on its own line, and
  leaves the next-gate wording unchanged.
- The reviewer established equivalence rather than accepting it: with the three
  source files reverted on a copied tree and the new tests kept, exactly one
  test failed, and only on its assertion about the new status line. Every
  call-site guard passed against both old and new code, which is what proves
  behaviour did not move.
- One round, no blockers.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass
- Careless-caller probe: ignoring the error and using the outcome falls closed
  at every call site
- Scratch fixture with a freshly built binary: ready and blocked reviews print
  byte-identical output to before; a removed review artifact adds exactly one
  line and the approval gate still refuses

## Plan Deviations

- none. The fourth call of the helper, in runStatus, looked like scope growth
  against the Director Brief's count of three, but the plan's step five
  specifies it: status calls the helper directly so it can see the error, while
  nextCommand keeps using the ready check. The reviewer confirmed it is not a
  converted external reader.

## Lesson Candidates

- Write the equivalence tests before a refactor and run them against the old
  code. Tests written after only prove the new code agrees with itself.

## Remaining Nits

- The new runStatus branch guards on finding a REVIEW record and has no else
  arm, so if the record were absent the next gate would stay at user approval
  where the old code fell to another executed round. It is unreachable, since
  the branch runs only when the last event is REVIEW, but it is the one path in
  the change that is not equivalent and it leans the unsafe way.
- One status run now reads the review artifact twice, once for the gate wording
  and once through the command suggestion. Harmless today, and the two must
  stay in agreement.
- The new status line carries two colons, unlike every other line in that
  output, which is a key and a value.
- requireReviewReady still parses the result line itself with its own file
  read. It was out of scope here and fails closed on its own.

## Residual Risks

- Baton still has two readers of the same line in the artifact. They agree
  today and each fails closed, but the second one is not covered by the type
  that now protects the first.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Type the review outcome so an unreadable review cannot pass as an answer
- Artifact path: .baton/runs/20260919-1927-review-result-error-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the four
nits, including the non-equivalent branch and the double colon in the new
output line.
