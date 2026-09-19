# RUN-01: Placeholder check accepts documented command syntax

Task ID: wqkb
Date: 2026-09-19
Executor: Claude Sonnet 5
Plan: .baton/runs/20260919-1902-placeholder-check-PLAN.md
Status: complete

## Changes

- `internal/baton/artifact.go`: added `templatePlaceholderTokens`, which globs
  `.baton/templates/*.md`, extracts every angle-bracket token with the
  existing placeholder regexp, and returns an error if the glob fails, no
  template file is found, or the resulting token set is empty. Chosen
  direction: fail closed. An empty vocabulary would make `checkArtifact`
  accept every artifact silently, turning the check off without anything
  failing — the same failure-open shape flagged twice today in this
  codebase — so a missing directory, an unreadable file, or a template set
  with no placeholder token now makes `check-artifact` return an error
  instead of passing.
- `internal/baton/artifact.go`: `checkArtifact` no longer refuses on any
  angle-bracket match. It now calls `templatePlaceholderTokens` and refuses
  only when the artifact contains one of those tokens as a literal
  substring, naming the offending token in the error. A `templatePlaceholderTokens`
  error is returned as-is.
- `internal/baton/commands_test.go`: added
  `TestPlaceholderCheckAcceptsDocumentedSyntax` (the two vxym fixture lines
  pass for both EXECUTED and PLANNED),
  `TestPlaceholderCheckRefusesSinglePlaceholder` (a complete artifact of each
  of the four kinds with one template token reinserted is still refused),
  and `TestPlaceholderCheckFailsClosedWithoutTemplates` (missing templates
  directory, empty templates directory, and templates with no placeholder
  token all refuse). `TestTemplatesAreRejected`, which already covers all
  four templates (plan.md, run.md, review.md, close.md) rather than one
  representative, was left unchanged and still passes.
- `.baton/PLANNER.md`, `.baton/EXECUTOR.md`,
  `bootstrap/.baton/PLANNER.md`, `bootstrap/.baton/EXECUTOR.md`: added a
  sentence instructing the delegate to run `baton check-artifact` on its own
  artifact as the last step before reporting completion, and a sentence
  stating lint is not a substitute because lint only inspects artifacts
  already recorded in `baton.log`. Command examples use literal uppercase
  `KEY`, `TASK-ID`, and `NN` rather than angle brackets. Both PLANNER.md
  copies are byte-identical and both EXECUTOR.md copies are byte-identical
  (verified with `diff`, not by eye).

## Validation

- Evidence collected before fixing a user-reported defect: n/a, this is planned feature work, not a defect fix.
- `go test ./...`: passes (`ok github.com/grollcake/baton/internal/baton`).
- `go vet ./...`: no findings.
- `gofmt -l .`: prints nothing.
- `go run ./cmd/baton lint`: passes, `baton-lint passed`.
- `diff .baton/PLANNER.md bootstrap/.baton/PLANNER.md`: silent.
- `diff .baton/EXECUTOR.md bootstrap/.baton/EXECUTOR.md`: silent.
- `git status --porcelain`: shows only `.baton/EXECUTOR.md`, `.baton/PLANNER.md`,
  `.baton/baton.log`, `bootstrap/.baton/EXECUTOR.md`, `bootstrap/.baton/PLANNER.md`,
  `internal/baton/artifact.go`, `internal/baton/commands_test.go`, and the
  untracked PLAN artifact; no file under `.baton/runs/` was modified.
- Load-bearing check: copied the working tree to `/tmp/baton-revert-check`,
  ran `git checkout -- internal/baton/artifact.go` there to revert only the
  source change, and reran the new tests. Result:
  `TestPlaceholderCheckAcceptsDocumentedSyntax` fails (the old blanket
  angle-bracket scan refuses the vxym fixture lines, confirming the fix is
  load-bearing for that regression). `TestPlaceholderCheckFailsClosedWithoutTemplates`
  fails on all three subcases (the old code path never calls
  `templatePlaceholderTokens`, so it never fails closed, confirming that test
  is load-bearing for the fail-closed direction). `TestPlaceholderCheckRefusesSinglePlaceholder`
  still passes on the reverted tree, because the old blanket regexp already
  caught a leftover template token; this test guards that this partial-fill
  behavior is preserved by the change, not that the change is what causes it.
- Self smoke test after fixing a user-reported defect: n/a, not a defect fix.
- `go run ./cmd/baton check-artifact EXECUTED .baton/runs/20260919-1902-placeholder-check-RUN-01.md wqkb`: exits 0, run as the last step before reporting.

## Report To Director

- Suggested summary: Placeholder check now refuses only text copied from `.baton/templates/*.md`, fails closed when it cannot derive that vocabulary, and PLANNER/EXECUTOR are told to self-check their artifacts before reporting.
- Artifact path: .baton/runs/20260919-1902-placeholder-check-RUN-01.md

## Success Criteria Status

- An artifact containing angle-bracket command syntax passes `check-artifact`: met.
- An unmodified copy of each artifact template is still refused: met.
- An artifact with a single template placeholder left in place is still refused, for each of the four kinds: met.
- Both role files instruct self-check and state lint is not a substitute, with byte-identical bootstrap copies: met.
- Missing, empty, or placeholder-free templates cause the check to fail closed rather than silently accept everything: met.

## Unresolved Risks

- As the PLAN's Risks Or Questions section already states: the new rule keys
  on provenance, so a hand-invented unfilled marker (for example a bracketed
  TBD an author writes themselves) is no longer caught for PLANNED, REVIEW,
  and CLOSE artifacts; EXECUTED artifacts keep the separate unresolved-task-
  marker check as a second line of defense. This tradeoff was Director's
  chosen direction in the PLAN, not something this RUN introduces.
- The check now reads the project's `.baton/templates/*.md` files on every
  call; no caching was added, matching the PLAN's note that none should be
  added without a measurement.

## Out Of Scope Returned To Director

- none.
