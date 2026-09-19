# CLOSE: Record the PLAN path on REQUEST

Task ID: vxym
Date: 2026-09-19
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260919-1837-request-plan-path-REVIEW-01.md

## Acceptance

- `new-round` records the PLAN path the task will produce on its REQUEST event,
  so the round key survives in the log and `status` offers the planner command
  while the last event is REQUEST. Director hit the missing key in real use on
  task uonl and copied it by hand from terminal output.
- The lint relaxation reaches only REQUEST. PLANNED, EXECUTED, REVIEW and CLOSE
  keep full artifact validation including existence, and FEEDBACK and RUN_DONE
  keep the bare existence check unchanged.
- Every REQUEST already in the log is pathless, still lints, and still produces
  no command. Nothing in history was rewritten.
- One round, no blockers.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass
- Reviewer CLI runs on a scratch fixture: new-round wrote the forward path,
  status printed the planner command, and append REQUEST accepted a well-formed
  path while rejecting two malformed ones
- Shell-injection guard still refuses its fixture, now through the path shape
  check; the reviewer probed six metacharacter variants against that check and
  every one was refused, because the path pattern is an allowlist that excludes
  them

## Plan Deviations

- none in scope or approach. The round took three attempts to record the
  EXECUTED event: check-artifact twice refused RUN-01 because it treats any
  angle-bracketed token as an unfilled template field, and the artifact quoted
  Baton's own command syntax. The second refusal was caused by Director asking
  for a note that itself used the notation it described. Only artifact wording
  changed; no source, test, or success criterion moved.

## Remaining Nits

- validateRequestPath discards the error the underlying validator produced and
  reports one generic message, so a mistyped path does not say which rule it
  broke.
- Only the PLANNED case pins the shared lint branch by test. Exempting EXECUTED,
  REVIEW or CLOSE individually would not fail a test today.
- RUN-01 ends its risk section with commentary about the angle-bracket
  constraint, which is process rather than residual danger.

## Residual Risks

- REQUEST now carries a path to a file that does not exist yet, which is a
  meaning the log did not previously have. It is confined to one event and
  guarded by shape validation, but a future reader of the log format should be
  told that a REQUEST path is a forward reference.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Record the PLAN path on REQUEST so the key survives
- Artifact path: .baton/runs/20260919-1837-request-plan-path-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the three
nits, the adversarial check on the loosened validation, and the backward
compatibility evidence.
