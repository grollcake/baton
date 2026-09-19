# REVIEW-01: Record plan deviations in the CLOSE artifact

Task ID: jiaq
Date: 2026-09-19
Planner: Planner
Run: .baton/runs/20260919-1746-close-deviations-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- none

### Nits

- `validClose` in `coverage_test.go` duplicates the CLOSE body that `writeClose`
  in `testutil_test.go` already builds, with slightly different content (no list
  items under Acceptance or Validation Summary). The PLAN asked for both, so this
  is recorded as duplication to watch, not a defect.
- The template bullet "deviation from the PLAN, or none" is a placeholder in the
  repository's angle-bracket style, so a CLOSE copied and left unfilled is caught
  by the unconditional unresolved-placeholder check, not by the new rule. That is
  the intended layering, but it means the new non-empty rule only bites when an
  author deletes the placeholder and leaves the section bare.

## Judgement On Decision 3 And The `none` Escape Hatch

Requiring at least one list item holds up. It costs one call to the existing
`countSectionItems` helper, it mirrors the REVIEW branch's three-to-five checks
rule exactly, and it is the only thing that separates "there were no deviations"
from "nobody filled this in". Without it the section would be a heading that
every CLOSE passes by existing.

`none` does weaken the rule in the sense the PLAN itself admits: an author who
did deviate can still write `- none`, and no check can catch that. But the
weakening is about truthfulness, not about the mechanism. The rule converts an
omission, which is invisible, into an explicit claim that is wrong on the record
and reviewable. That is the achievable half of the goal, and the PLAN's own
"Risks Or Questions" already states it. No change requested.

## Verification Performed

### 1. Success criteria, exercised on the real CLI

Built `go build -o SCRATCH/baton ./cmd/baton` and ran `check-artifact CLOSE`
against a scratch CLOSE fixture outside the working tree:

- Valid CLOSE with `## Plan Deviations` and `- none`: exit 0, no output.
- Same file with the heading kept and the bullet deleted: exit 1,
  `artifact-check: CLOSE artifact must list at least one Plan Deviations item: ...`.
  The fixture kept a later `## Remaining Nits` section with a `- none` bullet, so
  this also confirms `countSectionItems` stops at the next heading.
- Same file with the heading removed: exit 1,
  `artifact-check: CLOSE artifact must include Plan Deviations: ...`.

### 2. Backward compatibility

- `go run ./cmd/baton lint` exits 0, ending in `baton-lint passed` and
  `Baton tasks: 5 total, 4 closed, 1 open`.
- `git status --porcelain` on the two existing CLOSE artifacts is empty; the only
  entries under `.baton/runs/` are the untracked PLAN and RUN for this task.
- `grep "Plan Deviations"` finds nothing in either existing CLOSE file, so the
  legacy path is genuinely exercised and not vacuously satisfied.
- `lint.go:154` calls `checkArtifact(..., true)` while `CheckArtifact` at
  `artifact.go:61` passes `false`. Both new rules sit behind `if !allowLegacy`
  and an early `return nil` on the legacy path, matching the REVIEW branch's
  existing shape.

### 3. The new assertions are load-bearing

On two separate copies of the tree in the scratch directory, with the working
tree untouched:

- Reverting the whole `internal/baton/artifact.go` change:
  `TestArtifactRequiresProtocolSections` fails at `coverage_test.go:279`
  ("check-artifact accepted a CLOSE without ## Plan Deviations") and
  `TestLintToleratesLegacyCloseSections` fails at `lint_test.go:183`
  ("check-artifact accepted a CLOSE without Plan Deviations").
- Because that first failure is a `t.Fatal`, the empty-section assertion was
  proven separately: removing only the `countSectionItems` block and keeping the
  heading check makes `TestArtifactRequiresProtocolSections` fail at
  `coverage_test.go:283` ("check-artifact accepted a CLOSE with an empty Plan
  Deviations section"). Both halves of the rule are independently covered.

### 4. Paired files

- `diff bootstrap/.baton/templates/close.md .baton/templates/close.md`: no output.
- `diff bootstrap/.baton/PROTOCOL.md .baton/PROTOCOL.md`: no output.
- `docs.go:12` embeds `bootstrap/.baton/PROTOCOL.md`, which is the copy the drift
  check compares against, so the identical pair means the drift error disappears
  once Director rebuilds. The current `.baton/bin/baton lint` failure is that
  stale embed, as Director noted, not a defect in this change.

### 5. Scope

The branch touches ten files. `internal/baton/artifact.go` changes only inside
the `eventClose` branch; the PLANNED, EXECUTED and REVIEW branches and the
unconditional prefix checks are byte-identical to `main`. No transition-table
change, no PLAN round-number change, no new event types. The three added
`baton.log` lines are Director's own REQUEST, PLANNED and EXECUTED records for
this task, which is normal Director bookkeeping rather than the new-event-type
work the PLAN excluded. Neither existing CLOSE artifact is modified.

### 6. Documentation

All three places carry matching wording: the `## Round Artifacts` sentence in
`.baton/PROTOCOL.md` and `bootstrap/.baton/PROTOCOL.md`, and the Korean bullet in
section 6 of `PROTOCOL-GUIDE.md`. The Korean bullet says each CLOSE records how
the delivered work differed from the PLAN, or `none` if it did not, which matches
both the English sentence and the code's two-part rule.

### 7. Required checks

- `go test ./...`: `ok github.com/grollcake/baton/internal/baton`.
- `go vet ./...`: no output.
- `gofmt -l .`: no output.
- `go run ./cmd/baton lint`: passes.

## Suggested User Checks

- Copy `.baton/templates/close.md` to a scratch path, fill in every section
  honestly, and confirm `go run ./cmd/baton check-artifact CLOSE PATH TASKID`
  accepts it; then delete just the `- none` bullet under Plan Deviations and
  confirm the same command refuses it.
- Read the new `## Plan Deviations` section in `.baton/templates/close.md` and
  decide whether the placeholder wording is the prompt you want a Director to see
  at close time, or whether it should name the PLAN section to compare against.
- Read the new sentence in `.baton/PROTOCOL.md` and the Korean bullet in
  `PROTOCOL-GUIDE.md` section 6 side by side and confirm they say the same thing
  in both languages.
- Confirm that requiring at least one list item, stricter than every other
  section check in the codebase, is the tradeoff you want; removing it is a
  three-line deletion in the `eventClose` branch of `internal/baton/artifact.go`.
- Confirm that `.baton/bin/baton lint` failing with a PROTOCOL.md drift error
  until Director runs the release build is the expected sequencing for this
  branch, since that binary refresh was deliberately left out of this task.

## Evidence Reviewed

- .baton/runs/20260919-1746-close-deviations-PLAN.md
- .baton/runs/20260919-1746-close-deviations-RUN-01.md
- git diff against main for all ten changed files
- internal/baton/artifact.go lines 60 to 195, internal/baton/lint.go line 154
- go test ./..., go vet ./..., gofmt -l ., go run ./cmd/baton lint
- check-artifact CLOSE runs on three scratch fixtures outside the working tree
- two reverted copies of the tree under the scratch directory

## Report To Director

- Suggested summary: Review 01 found no blockers for the CLOSE Plan Deviations rule
- Artifact path: .baton/runs/20260919-1746-close-deviations-REVIEW-01.md

## Required Next Step

- ask Director to request user approval
