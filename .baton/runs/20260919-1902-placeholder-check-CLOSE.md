# CLOSE: Stop rejecting command syntax as an unfilled placeholder

Task ID: wqkb
Date: 2026-09-19
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260919-1902-placeholder-check-REVIEW-01.md

## Acceptance

- An artifact may now describe Baton's own command syntax without being refused.
  The check judges provenance rather than shape: it refuses only text that
  appears verbatim as a placeholder in the project's own templates.
- Every unmodified template is still refused, and so is a complete artifact with
  a single placeholder reinserted, for all four artifact kinds. The reviewer
  built these cases by hand rather than trusting the run report.
- When the template vocabulary cannot be read, the check fails closed. A missing
  directory, an empty one, an unreadable file and a placeholder-free template
  set all return an error rather than accepting everything.
- PLANNER.md and EXECUTOR.md now tell a delegate to run check-artifact on its own
  artifact as the last step before reporting, and say that lint is not a
  substitute because lint inspects only artifacts already recorded in the log.
  Both copies of each file are byte-identical.
- One round, no blockers.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass
- Reviewer fixtures on a freshly built binary: four templates refused, four
  single-placeholder artifacts refused after confirming each baseline passed,
  the two previously refused lines accepted, four vocabulary-failure modes
  refused

## Plan Deviations

- The REVIEW event could not be recorded with the installed binary. It predates
  this change, so its blanket rule refused a REVIEW artifact that discusses the
  notation. Director appended the event by running the working-tree binary
  instead. No artifact content changed for that reason, and the plan's scope and
  success criteria are unchanged.

## Remaining Nits

- RUN-01 claimed EXECUTED still catches an invented marker through its unresolved
  task marker check. That is overstated and is corrected here: the check matches
  one literal word, so a bracketed marker of an author's own invention is now
  accepted for all four artifact kinds, not three.
- A test doc comment describes an unreadable-file subtest that does not exist.
- The single-placeholder test uses a token that several templates share, so it
  does not pin which template supplied it.
- The new role-file sentences name the binary without saying that a repository
  which is modifying Baton itself must use a freshly built one.

## Residual Risks

- The narrowing gives up catching markers an author invents rather than copies
  from a template. Baton can tell that an artifact still carries its template's
  words; it cannot tell that an author left a note to themselves.
- A stale installed binary now misleads three steps, not one. Lint reports
  document drift, append refuses artifacts written under newer rules, and await
  polls an artifact it cannot recognise until it times out. The await case is the
  worst because it fails silently, and the timeout guidance in DIRECTOR.md does
  not list a stale binary among the causes to check.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Judge placeholders by template provenance, and have delegates self-check
- Artifact path: .baton/runs/20260919-1902-placeholder-check-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the corrected
claim about unresolved task markers, the three nits, and the stale-binary
finding.
