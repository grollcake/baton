# RUN-01: Record plan deviations in the CLOSE artifact

Task ID: jiaq
Date: 2026-09-19
Executor: Executor
Plan: .baton/runs/20260919-1746-close-deviations-PLAN.md
Status: complete

## Changes

- `internal/baton/artifact.go`: in the `eventClose` branch of `checkArtifact`, added an `if !allowLegacy` guard requiring the line `^## Plan Deviations[[:space:]]*$`, then (after `checkArtifactLines` succeeds and only when `!allowLegacy`) requiring `countSectionItems(content, "## Plan Deviations") >= 1`. The five prior CLOSE checks and the PLANNED/EXECUTED/REVIEW branches, transition table, and `baton.log` events are untouched.
- `bootstrap/.baton/templates/close.md` and `.baton/templates/close.md`: added a `## Plan Deviations` section with a placeholder bullet worded "deviation from the PLAN, or none" (in the template's angle-bracket style), placed after `## Validation Summary` and before `## Remaining Nits`. Both files remain byte-identical (`diff` empty).
- `bootstrap/.baton/PROTOCOL.md` and `.baton/PROTOCOL.md`: added one sentence to `## Round Artifacts` stating each `CLOSE` records how the delivered work differed from its `PLAN`, or `none`. Both files remain byte-identical (`diff` empty).
- `PROTOCOL-GUIDE.md` section 6: added the matching Korean bullet to the artifact bullet list.
- `internal/baton/testutil_test.go`: `writeClose` now emits `## Plan Deviations` / `- none.`.
- `internal/baton/coverage_test.go`: added `validClose(taskID string)` helper (same body as `writeClose`); extended `TestArtifactRequiresProtocolSections` with CLOSE cases: heading removed is refused, heading present with no list item is refused, valid CLOSE passes.
- `internal/baton/lint_test.go`: added `TestLintToleratesLegacyCloseSections`, modelled on `TestLintToleratesLegacyRunSections` — a CLOSE without `## Plan Deviations`, logged behind a REVIEW with `Result: ready-for-user-decision`, passes `lint` and is refused by `check-artifact`.
- No changes to `.baton/runs/20260919-1700-concurrency-inflight-CLOSE.md` or `.baton/runs/20260919-1731-status-blockers-CLOSE.md` (confirmed via `git status --porcelain`, both clean).

## Validation

- Evidence collected before fixing a user-reported defect: n/a (feature work, not a defect).
- `go build ./...`: clean.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton`.
- `go vet ./...`: no output.
- `gofmt -l .`: no output.
- `go run ./cmd/baton lint`: `baton-lint passed` (freshly built binary embeds the edited PROTOCOL.md, so no drift).
- `.baton/bin/baton lint`: fails with `ERROR: PROTOCOL.md differs from the copy this baton binary shipped with; run an update`. This is expected per the Director's sequencing note: the installed binary predates this change and embeds the old PROTOCOL.md text. Refreshing `.baton/bin/baton` is Director's release/close-out step, not part of this task's scope.
- `git status --porcelain` on `.baton/runs/20260919-1700-concurrency-inflight-CLOSE.md` and `.baton/runs/20260919-1731-status-blockers-CLOSE.md`: both empty (unmodified).
- Reverting `internal/baton/artifact.go` on a scratch copy of the tree (`/private/tmp/.../scratchpad/baton-revert-check`, not the working tree) while keeping all other changes: `go test ./internal/baton/... -run 'TestArtifactRequiresProtocolSections|TestLintToleratesLegacyCloseSections'` then fails both tests with "check-artifact accepted a CLOSE without ## Plan Deviations" / "...without Plan Deviations", confirming the new assertions are load-bearing on the `artifact.go` change. The scratch copy was deleted after verification.
- Self smoke test after fixing a user-reported defect: n/a.

## Report To Director

- Suggested summary: Require a Plan Deviations section in new CLOSE artifacts
- Artifact path: .baton/runs/20260919-1746-close-deviations-RUN-01.md

## Success Criteria Status

- `check-artifact CLOSE` refuses a CLOSE lacking the `## Plan Deviations` heading: met.
- `check-artifact CLOSE` refuses a CLOSE whose heading has no list item under it: met.
- `baton lint` still passes on this repository with the two existing CLOSE artifacts byte-identical: met (via `go run ./cmd/baton lint`; the installed `.baton/bin/baton` reports PROTOCOL.md drift until Director rebuilds it, as expected and documented above).
- `go test ./...`, `go vet ./...`, `gofmt -l .` all clean: met.

## Unresolved Risks

- `.baton/bin/baton lint` reports PROTOCOL.md drift until Director runs a release build to refresh the installed binary; this is expected and out of this task's scope per the Director's brief.
- The non-empty Plan Deviations rule (Decision 3 in the PLAN) is stricter than any other section check in the codebase; Director already flagged this as a deliberate, reviewable choice.

## Out Of Scope Returned To Director

- none
