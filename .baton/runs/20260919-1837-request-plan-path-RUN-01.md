# RUN-01: Record the PLAN path on REQUEST so the round key survives

Task ID: vxym
Date: 2026-09-19
Executor: Claude Sonnet 5
Plan: .baton/runs/20260919-1837-request-plan-path-PLAN.md
Status: complete

## Changes

- `internal/baton/artifact.go`: added `validateRequestPath`, a shape-only wrapper around `validateArtifactPath(eventPlanned, path)` for the REQUEST forward reference (no existence check).
- `internal/baton/events.go`:
  - `runNewRound` now computes `planPath := ".baton/runs/" + key + "-PLAN.md"`, appends the REQUEST record with `Path: planPath`, and reuses `planPath` in the printed `plan_path=` output so the two cannot diverge.
  - `runAppend` validates `--path` on REQUEST with `validateRequestPath` when a path is given; `--path` stays optional on REQUEST.
  - `nextCommand` now handles `last == eventRequest` first: if the REQUEST has no path, returns `""` (unchanged behavior for pathless REQUEST records); if it has a path, returns `baton prompt plan` with the task id and key substituted in, e.g. `baton prompt plan --task-id bqav --key 20260919-1843-demo-plan-path`, recovered via `artifactKey(request.Path, eventPlanned)`. Updated the function doc comment accordingly.
- `internal/baton/lint.go`: `checkLog`'s path-present branch is now a three-way switch: `requiresArtifact` events keep `checkArtifact` (existence + shape); `eventRequest` uses `validateRequestPath` (shape only, no existence, errors reported the same way as other artifact-check failures); everything else (FEEDBACK, RUN_DONE) keeps the original `os.Stat` existence check verbatim.
- Tests:
  - `internal/baton/status_test.go`: added `TestStatusOffersPlanCommandAtRequest` — asserts the logged REQUEST line contains the forward `.baton/runs/KEY-PLAN.md` path with no file on disk, that `status --task-id` prints `next_command: baton prompt plan ...`, and that `lint` accepts the forward reference.
  - `internal/baton/lint_test.go`: added `TestLintAcceptsPathlessRequest` (pathless REQUEST lints clean and produces no `next_command`, and stays clean once followed by a real PLANNED), `TestLintRejectsMissingPlannedArtifact` (PLANNED path absent from disk still fails lint), and `TestLintRejectsMissingFeedbackOrRunDonePath` (both a RUN_DONE and a FEEDBACK path pointing at a missing file still fail lint, restoring coverage on the `os.Stat` branch after `TestLintDoesNotExecutePaths` moved to the shape check).
  - `internal/baton/commands_test.go`: added `TestAppendRequestPathValidation` — `append REQUEST --path` with a well-formed PLAN path that has no file on disk (e.g. `.baton/runs/20260711-1001-slug-PLAN.md`) succeeds; `--path .baton/runs/notes.txt` and `--path ../outside-PLAN.md` are both rejected.
  - `internal/baton/coverage_test.go`: updated `TestStatusNextCommand`, which previously asserted no `next_command` before PLANNED; that assertion described the old guarantee, which this task intentionally changes, so it now asserts the new `baton prompt plan` command instead.

## Validation

- Evidence collected before fixing a user-reported defect: n/a (feature work, not a defect report).
- `go build ./...`: no output, builds clean.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton 6.002s` (all tests, including the new/updated ones, pass).
- `go vet ./...`: no output.
- `gofmt -l .`: no output.
- `go run ./cmd/baton lint` (in this repository): `baton-lint passed`, including `Baton tasks: 6 total, 5 closed, 1 open` and no complaint about the six existing pathless REQUEST lines in `.baton/baton.log` (none were edited).
- Manual `go run ./cmd/baton status` demonstration on a scratch fixture (`/tmp/baton-scratch`, built from `bootstrap/.baton` with a freshly built binary, cleaned up after): `new-round demo-plan-path` logged `... | REQUEST | Director | Demo plan path | .baton/runs/20260919-1843-demo-plan-path-PLAN.md` with no such file on disk, and `status --task-id bqav` printed `next_command: baton prompt plan --task-id bqav --key 20260919-1843-demo-plan-path`.
- Deliberate revert check (Director emphasis): copied the working tree to `/tmp/baton-revert-check`, ran `git checkout -- internal/baton/artifact.go internal/baton/events.go internal/baton/lint.go` (source only, test files kept), then `go test ./internal/baton/...`. Three tests failed for the expected reasons: `TestAppendRequestPathValidation` (`append accepted a REQUEST path that is not a PLAN artifact path` — the old code has no REQUEST path validation at all), `TestStatusNextCommand` and `TestStatusOffersPlanCommandAtRequest` (missing the new `next_command: baton prompt plan ...` line, and the REQUEST line missing the forward PLAN path — the old `runNewRound` never wrote `Path`). This confirms the new assertions are load-bearing against the pre-change source. Scratch copy removed afterward.

## Report To Director

- Suggested summary: Record the PLAN path on REQUEST and offer `baton prompt plan` while a task's last event is REQUEST; REQUEST path validation is shape-only (no existence check), while PLANNED/EXECUTED/REVIEW/CLOSE keep full `checkArtifact` and FEEDBACK/RUN_DONE keep the bare `os.Stat` existence check.
- Artifact path: .baton/runs/20260919-1837-request-plan-path-RUN-01.md

## Success Criteria Status

- A new task's REQUEST line ends in `.baton/runs/KEY-PLAN.md`, and `status --task-id` prints `next_command: baton prompt plan --task-id ID --key KEY`: met.
- A REQUEST with no path lints clean and produces no `next_command`; no existing `.baton/baton.log` line modified: met.
- `lint` still fails when PLANNED/EXECUTED/REVIEW/CLOSE path is missing, and when FEEDBACK/RUN_DONE path is missing: met (new tests for PLANNED and for both FEEDBACK and RUN_DONE).
- `lint` fails on a REQUEST path that is not a `.baton/runs/` PLAN path, and never executes it: met — `TestLintDoesNotExecutePaths`'s injected `$(touch FILE)` path is now rejected by `validateRequestPath`'s shape check (it fails `artifactPathPattern`/`.baton/runs/` prefix), not by a missing-file stat; confirmed via the revert check that removing the new validation is what makes the injection case pass instead of getting refused for the right reason.
- `append REQUEST --path` accepts exactly what `new-round` writes and rejects other shapes; `--path` stays optional: met.

## Unresolved Risks

- None beyond those already recorded in the PLAN (`§Risks Or Questions`): the REQUEST path is an unenforced promise until PLANNED is logged, and `pendingArtifacts` still globs every `*-PLAN.md` rather than the announced key (both explicitly out of scope).
- This artifact avoids angle-bracket placeholder notation entirely when describing command syntax, because `check-artifact`/`checkArtifact` rejects any angle-bracketed token in a RUN as an unresolved template placeholder; wording here uses concrete example values or plain-language description instead, so a later reader should not mistake this for sloppy phrasing.

## Out Of Scope Returned To Director

- None.
