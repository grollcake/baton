# PLAN: Record plan deviations in the CLOSE artifact

Task ID: jiaq
Date: 2026-09-19
Planner: Planner
Status: complete

## Director Brief

- Goal: Every newly written CLOSE artifact must carry a `## Plan Deviations` section stating how the delivered work differed from its PLAN, with `- none` as the accepted way to say it did not, so a scope change decided after PLANNED lands in one place at close instead of scattering across RUN prose and REVIEW.
- Scope: `internal/baton/artifact.go` (CLOSE branch of `checkArtifact`), `bootstrap/.baton/templates/close.md` and its installed copy `.baton/templates/close.md`, the CLOSE contract sentences in `bootstrap/.baton/PROTOCOL.md` + `.baton/PROTOCOL.md` and in `PROTOCOL-GUIDE.md` section 6, and tests under `internal/baton/`.
- Success Criteria: `check-artifact CLOSE` refuses a CLOSE that lacks the heading and refuses one whose heading has no list item under it; `baton lint` still passes on this repository with the two existing CLOSE artifacts byte-identical; `go test ./...`, `go vet ./...`, `gofmt -l .` all clean.
- Risks: The two existing CLOSE artifacts are append-only history and must not be edited; the new rule must therefore sit behind the existing `allowLegacy` flag. Test helpers `writeClose` and any new `validClose` must both carry the section or unrelated append and lint tests start failing for the wrong reason.
- Required Checks: `go test ./...`, `go vet ./...`, `gofmt -l .`, `.baton/bin/baton lint`, and `go run ./cmd/baton lint` (the installed binary predates this change, so only the freshly built one proves the legacy path).
- Executor Prompt: In `/Users/rollcake/lab/baton/internal/baton/artifact.go`, extend the `eventClose` branch of `checkArtifact`. Keep the five existing checks unconditional. When `allowLegacy` is false, additionally require the line `^## Plan Deviations[[:space:]]*$` with the message `must include Plan Deviations`, and after `checkArtifactLines` succeeds require `countSectionItems(content, "## Plan Deviations") >= 1`, returning an error naming the path when it is zero. Do not touch the PLANNED, EXECUTED, or REVIEW branches, the transition table, `baton.log` events, or the two CLOSE files under `.baton/runs/`. Add the section `## Plan Deviations` and a single placeholder bullet worded "deviation from the PLAN, or none", written in the same angle-bracket placeholder style the other sections of that template use, to both `bootstrap/.baton/templates/close.md` and `.baton/templates/close.md`, placed after `## Validation Summary` and before `## Remaining Nits`, keeping the two files identical. Add one sentence to the `## Round Artifacts` section of `bootstrap/.baton/PROTOCOL.md` and `.baton/PROTOCOL.md` (keep them identical) stating that each `CLOSE` records how the delivered work differed from its `PLAN`, or `none`, and add the matching Korean bullet to the artifact bullet list in section 6 of `PROTOCOL-GUIDE.md`. In tests: add the new section to `writeClose` in `internal/baton/testutil_test.go`, add a `validClose(taskID string)` helper in `internal/baton/coverage_test.go` returning the same body, extend `TestArtifactRequiresProtocolSections` (or add a sibling test) so a CLOSE with the heading removed and a CLOSE whose section has no list item are both refused by `check-artifact` while the valid one passes, and add a lint test mirroring `TestLintToleratesLegacyRunSections` that logs a CLOSE without the section (behind a REVIEW with `Result: ready-for-user-decision`), asserts `lint` passes, and asserts `check-artifact CLOSE` on the same file fails. Confirm each new assertion fails when the `artifact.go` change is reverted. Then run `go test ./...`, `go vet ./...`, `gofmt -l .`, `go run ./cmd/baton lint`, and `.baton/bin/baton lint`, and confirm `git status` shows no modification to the two existing CLOSE artifacts.

## Goal

- A CLOSE artifact written from now on states, in one named section, how the delivered work differed from the PLAN it closes, or records that it did not.
- The two CLOSE artifacts already in this repository stay byte-identical and keep passing `baton lint`.

## Scope

In scope:

- `internal/baton/artifact.go`: the `eventClose` branch of `checkArtifact` only.
- `bootstrap/.baton/templates/close.md` and `.baton/templates/close.md`, kept identical.
- CLOSE contract wording in `bootstrap/.baton/PROTOCOL.md`, `.baton/PROTOCOL.md` (kept identical) and `PROTOCOL-GUIDE.md` section 6.
- Tests in `internal/baton/`: `testutil_test.go`, `coverage_test.go`, `lint_test.go`.

Out of scope:

- New `baton.log` events, transition table changes, PLAN round numbers, any PLAN amendment mechanism.
- Editing `.baton/runs/20260919-1700-concurrency-inflight-CLOSE.md` and `.baton/runs/20260919-1731-status-blockers-CLOSE.md`.
- The PLANNED, EXECUTED, and REVIEW branches of `checkArtifact`.
- Bumping `VERSION` / `.baton/VERSION` / `bootstrap/.baton/VERSION` and refreshing `.baton/bin/baton`; both are Director release steps, not this task.
- `README.md`, `DIRECTOR.md`, and prompt text in `internal/baton/prompt.go` (Director writes CLOSE from the template, which now carries the section).

## Decisions

These are the judgement calls Director asked to see stated, not deferred:

1. **The `allowLegacy` pattern applies.** The new requirement goes inside an
   `if !allowLegacy` guard in the `eventClose` branch, exactly as `## Changes`
   and `## Unresolved Risks` are handled for `EXECUTED`. `lint.go` passes
   `true`, so history stays valid; `CheckArtifact` passes `false`, so `append`
   and `check-artifact` hold every new CLOSE to the rule.
2. **`baton lint` must keep passing on this repository with both existing CLOSE
   files untouched, and must report no warning about them.** They are
   append-only history. The legacy path is the whole reason the rule is not
   retroactive.
3. **The section must be non-empty: at least one list item.** A bare heading
   cannot distinguish "there were no deviations" from "nobody filled this in",
   which is the exact failure this task exists to prevent. The existing
   `countSectionItems` helper already does this counting for REVIEW, so the
   rule costs one call and no new machinery. This is stricter than the
   heading-presence checks used elsewhere, and deliberately so.
4. **`none` is expressible and is the convention.** `- none` is one list item
   and satisfies the rule, matching `## Unresolved Risks` in `run.md` and
   `## Remaining Nits` in `close.md`. The template ships a placeholder bullet,
   so the existing unresolved-placeholder check already catches a CLOSE copied
   from the template and left unfilled.

## Plan

1. Extend the `eventClose` branch of `checkArtifact` in
   `internal/baton/artifact.go`: keep the five current checks unconditional;
   under `if !allowLegacy` append the `## Plan Deviations` heading check, and
   after `checkArtifactLines` returns nil require
   `countSectionItems(content, "## Plan Deviations") >= 1`. → verify:
   `go build ./...` succeeds and the function still returns early for the other
   three events.
2. Add the `## Plan Deviations` section to `bootstrap/.baton/templates/close.md`
   and `.baton/templates/close.md`, after `## Validation Summary` and before
   `## Remaining Nits`. → verify: `diff bootstrap/.baton/templates/close.md
   .baton/templates/close.md` prints nothing.
3. Add the CLOSE contract sentence to both `PROTOCOL.md` copies and the Korean
   bullet to `PROTOCOL-GUIDE.md` section 6. → verify: `diff
   bootstrap/.baton/PROTOCOL.md .baton/PROTOCOL.md` prints nothing.
4. Update `writeClose` in `testutil_test.go` and add `validClose` in
   `coverage_test.go` so both emit the section with `- none.`. → verify:
   `go test ./...` passes with no unrelated failures.
5. Add the refusal tests: heading removed is refused by `check-artifact`,
   heading present with no list item is refused, unmodified valid CLOSE passes.
   → verify: each assertion fails when step 1 is reverted.
6. Add the lint legacy test modelled on `TestLintToleratesLegacyRunSections`:
   a CLOSE without the section, logged behind a REVIEW with
   `Result: ready-for-user-decision`, passes `lint` and fails `check-artifact`.
   → verify: passes, and its `check-artifact` half fails when step 1 is
   reverted.
7. Run the required checks and confirm the two existing CLOSE artifacts are
   unmodified. → verify: `git status --porcelain .baton/runs/` lists neither
   file.

## Success Criteria

- `baton check-artifact CLOSE` on a path fails for a CLOSE without a
  `## Plan Deviations` heading, and fails for one whose heading has no list item
  beneath it.
- The same CLOSE files are accepted by `baton lint` through the `allowLegacy`
  path, and `lint` reports no error for them.
- `baton lint` passes in this repository while
  `.baton/runs/20260919-1700-concurrency-inflight-CLOSE.md` and
  `.baton/runs/20260919-1731-status-blockers-CLOSE.md` remain byte-identical.
- Every new test assertion fails when the `internal/baton/artifact.go` change is
  reverted, proving the section's presence is load-bearing.
- `bootstrap/.baton/templates/close.md` and `.baton/templates/close.md` are
  identical and both carry the section; the two `PROTOCOL.md` copies are
  identical.
- `go test ./...`, `go vet ./...`, and `gofmt -l .` are clean.

## Validation

- `go test ./...`: pass, including the new refusal and legacy tests.
- `go vet ./...`: no output.
- `gofmt -l .`: no output.
- `go run ./cmd/baton lint`: passes in this repository with the two existing
  CLOSE artifacts untouched. This is the check that proves the legacy path,
  because the installed binary predates the change.
- `.baton/bin/baton lint`: passes. The installed binary is refreshed only by a
  release build, so this run is a regression guard, not evidence of the new
  rule.
- `git status --porcelain .baton/runs/`: shows no modification to either
  existing CLOSE artifact.
- Reverting the `artifact.go` change makes the new tests fail.

## Report To Director

- Suggested summary: Require a Plan Deviations section in new CLOSE artifacts
- Artifact path: .baton/runs/20260919-1746-close-deviations-PLAN.md

## Risks Or Questions

- The non-empty rule in Decision 3 is stricter than any other section check in
  the codebase. If Director prefers heading presence only, that is a one-line
  removal, but the section then permits a silent empty answer.
- `countSectionItems` stops at the next `## ` heading, so the new section must
  keep a `## ` heading after it in the template. Placing it before
  `## Remaining Nits` satisfies this; moving it to the end of the file would
  not change counting behaviour but would separate it from the nits and risks
  cluster it belongs with.
- A version bump and an `.baton/bin` refresh are customary in this repository's
  release commits but were excluded from the prior task as well. Director
  decides whether they ride along with this change.
- Nothing forces a Director to write a truthful deviation entry; this task
  makes the question unavoidable, not the answer correct.
