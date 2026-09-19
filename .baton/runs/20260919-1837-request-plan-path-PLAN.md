# PLAN: Record the PLAN path on REQUEST so the round key survives

Task ID: vxym
Date: 2026-09-19
Planner: Claude Opus 5
Status: complete

## Director Brief

- Goal: `new-round` records the future PLAN path on its REQUEST event, and `status` offers `baton prompt plan` while a task's last event is REQUEST.
- Scope: `internal/baton/events.go`, `internal/baton/artifact.go`, `internal/baton/lint.go`, plus tests in `internal/baton/status_test.go`, `lint_test.go`, `commands_test.go`. No managed document changes, so no binary drift at close.
- Success Criteria: a fresh `new-round` writes a REQUEST line ending in `.baton/runs/KEY-PLAN.md` and `status --task-id` prints `next_command: baton prompt plan --task-id ID --key KEY`; every pathless REQUEST already in `baton.log` still lints and still prints no `next_command`.
- Risks: relaxing the lint existence check too widely would silently drop the guarantee for FEEDBACK and RUN_DONE paths; `TestLintDoesNotExecutePaths` currently relies on the REQUEST branch and must keep failing (through the new shape check) while the stat branch keeps its own coverage.
- Required Checks: `go test ./...`, `go vet ./...`, `gofmt -l .` (silent), `go run ./cmd/baton lint` in this repository.
- Executor Prompt: Implement §Plan steps 1-6 verbatim in `internal/baton/`. Only REQUEST is exempted from the lint file-existence check, and only REQUEST gains a shape-only path validation; PLANNED, EXECUTED, REVIEW and CLOSE keep full `checkArtifact` validation (which includes existence), and FEEDBACK and RUN_DONE keep the bare `os.Stat` existence check. `--path` stays optional on `append REQUEST`. Do not touch `.baton/PROTOCOL.md`, `.baton/DIRECTOR.md`, `bootstrap/`, or any line already in `.baton/baton.log`. Use `go run ./cmd/baton` for any manual check; the installed `.baton/bin/baton` predates this change.

## Goal

- `runNewRound` appends its REQUEST record with `Path` set to `.baton/runs/KEY-PLAN.md`, a forward reference to an artifact that does not exist yet.
- `nextCommand` recovers the key from that path and returns `baton prompt plan --task-id ID --key KEY` when the last event is REQUEST.
- `lint` accepts that forward reference, still rejects a malformed REQUEST path, and keeps today's existence guarantee for every other event.
- REQUEST records already in `baton.log` (bootstrap lines, four closed tasks, the `vxym` REQUEST itself) keep linting and keep producing no `next_command`.

## Scope

In scope:

- `internal/baton/events.go`: `runNewRound`, `runAppend`, `nextCommand`.
- `internal/baton/artifact.go`: one new REQUEST path validator.
- `internal/baton/lint.go`: the path branch of `checkLog`.
- Tests in `internal/baton/status_test.go`, `internal/baton/lint_test.go`, `internal/baton/commands_test.go`.

Out of scope:

- Managed documents (`PROTOCOL.md`, `DIRECTOR.md`, `PLANNER.md`, `EXECUTOR.md`, `HOW-TO-UPDATE.md`) and `bootstrap/`. `DIRECTOR.md` already documents the log line with an optional trailing path, and `new-round` is the only writer of this path, so no document needs to change. Leaving them alone also avoids the managed-document drift error against the installed binary.
- Rewriting history in `.baton/baton.log`.
- The REQUEST path written by `runUpdate` (`update.go:172`). Baton update is Direct work (`REQUEST -> RUN_DONE`) with no PLAN, so its REQUEST stays pathless and correctly yields no `next_command`.
- Cross-checking that a later PLANNED path matches the key the REQUEST announced. It is a new enforcement, not needed for key recovery, and a mismatch is self-correcting once PLANNED is logged.
- `status --open`. Decision and reasoning in §Decisions.

## Decisions

1. Narrowest lint relaxation. In `checkLog` the non-artifact path branch (`lint.go:157`) stats the path and errors when it is missing. Exempt exactly one event, REQUEST, and give it a shape-only check instead. Events that keep the existence check: PLANNED, EXECUTED, REVIEW and CLOSE through `state.app.checkArtifact` (which reads the file), and FEEDBACK and RUN_DONE through the unchanged `os.Stat` branch. REQUEST is the only event whose path is a forward reference, so it is the only one that can legitimately point at a file that does not exist yet.

2. `append REQUEST --path`. `runAppend` must accept it and validate it the same way, otherwise `new-round` writes a record the CLI would reject. Do not make `--path` required on REQUEST: the Direct flow and `runUpdate` both append a pathless REQUEST, and requiring it would break them. So: path optional; when present, shape-validated exactly as lint validates it, with no existence check. This makes `new-round`, `append` and `lint` agree on one rule.

3. Validate the REQUEST path for shape. Yes. It costs one reuse of `validateArtifactPath`, it keeps `TestLintDoesNotExecutePaths` meaningful (its injected `$(touch ...)` path is now rejected by shape rather than by a missing file), and an unvalidated free-text path would let `nextCommand` emit a garbage `--key`. The rule is: under `.baton/runs/`, no `..`, allowed characters only, and the name ends in `-PLAN.md` — exactly `validateArtifactPath(eventPlanned, path)`, minus existence.

4. `status --open`. Leave unchanged. `reportOpenTasks` prints one fixed-shape line per task and exists to let Director pick a task; `status --task-id ID` then gives the command for the one that was picked. Appending a variable-length field to that line changes an output shape other callers parse, for no capability that is not already one command away. Not nearly free enough to bundle.

## Plan

1. `internal/baton/artifact.go`: add a validator next to `validateArtifactPath`:

   ```go
   // validateRequestPath validates the forward reference a REQUEST may carry:
   // the PLAN artifact the task will produce. It checks shape only, because the
   // file does not exist when new-round records it.
   func validateRequestPath(path string) error {
       if validateArtifactPath(eventPlanned, path) != nil {
           return fmt.Errorf("artifact-check: REQUEST path must be a PLAN artifact path under .baton/runs/: %s", path)
       }
       return nil
   }
   ```

2. `internal/baton/events.go`, `runNewRound`: compute `planPath := ".baton/runs/" + key + "-PLAN.md"` after the key-collision check, pass `Path: planPath` in the `appendRecord` call, and reuse `planPath` in the `plan_path=` field of the existing `Fprintf` so the printed path and the recorded path cannot diverge. The rest of the `new-round` output stays byte-identical.

3. `internal/baton/events.go`, `runAppend`: after the existing artifact-bearing-event block and before the record is built, add

   ```go
   if event == eventRequest && path != "" {
       if err := validateRequestPath(path); err != nil {
           return err
       }
   }
   ```

4. `internal/baton/events.go`, `nextCommand`: handle REQUEST before the `lastRecord(..., eventPlanned)` lookup, which returns early today:

   ```go
   if last == eventRequest {
       request, found := lastRecord(records, taskID, eventRequest)
       if !found || request.Path == "" {
           return ""
       }
       key := filepath.Base(artifactKey(request.Path, eventPlanned))
       return fmt.Sprintf("baton prompt plan --task-id %s --key %s", taskID, key)
   }
   ```

   Update the function's doc comment, which currently claims the command stays empty before PLANNED: it now stays empty only when the REQUEST carries no path (an older record or Direct work).

5. `internal/baton/lint.go`, `checkLog`: replace the `if requiresArtifact { ... } else if info, statErr := os.Stat(...)` pair inside `if record.Path != ""` with a three-way switch: `requiresArtifact` keeps `state.app.checkArtifact(...)`; `record.Event == eventRequest` calls `validateRequestPath` and reports `state.err("Baton log line %d: %s", lineNumber, pathErr)`; the default keeps the current `os.Stat` check and its `artifact path not found on line %d` message verbatim.

6. Tests, in the existing harness style:
   - `status_test.go`: after `new-round`, assert the logged line contains `.baton/runs/KEY-PLAN.md`, that `status --task-id` prints `next_command: baton prompt plan --task-id ID --key KEY`, and that `lint` passes with that forward reference in the log and no PLAN file on disk.
   - `lint_test.go`: a log whose REQUEST has no path (followed by a PLANNED whose artifact exists) lints clean, and `status --task-id` prints no `next_command:` line.
   - `lint_test.go`: `lint` fails for a PLANNED whose artifact file is absent, and fails for a `RUN_DONE` carrying a path to a file that does not exist. The second case keeps coverage on the `os.Stat` branch, which `TestLintDoesNotExecutePaths` no longer exercises.
   - `commands_test.go`: `append REQUEST --path .baton/runs/20260711-1001-slug-PLAN.md` succeeds with no such file on disk; `append REQUEST --path .baton/runs/notes.txt` and `append REQUEST --path ../outside-PLAN.md` fail.

## Success Criteria

- A new task's REQUEST line ends in `.baton/runs/KEY-PLAN.md`, and `status --task-id` for it prints `next_command: baton prompt plan --task-id ID --key KEY`.
- A REQUEST with no path lints clean and produces no `next_command`, exactly as today; no existing line in `.baton/baton.log` is modified.
- `lint` still fails when a PLANNED, EXECUTED, REVIEW or CLOSE path is missing from disk, and still fails when a FEEDBACK or RUN_DONE path is missing from disk.
- `lint` fails on a REQUEST path that is not a `.baton/runs/` PLAN path, and never executes it.
- `append REQUEST --path` accepts exactly what `new-round` writes and rejects other shapes; `--path` remains optional on REQUEST.

## Validation

- `go test ./...`: passes, including the four new or extended tests.
- `go vet ./...`: no output.
- `gofmt -l .`: no output.
- `go run ./cmd/baton lint`: `baton-lint passed` in this repository, with the six existing pathless REQUEST lines untouched.
- `go run ./cmd/baton status --task-id vxym`: still prints no `next_command`, because this task's REQUEST predates the change. Expected, not a regression.

## Report To Director

- Suggested summary: Record the PLAN path on REQUEST and offer the planner command at REQUEST
- Artifact path: .baton/runs/20260919-1837-request-plan-path-PLAN.md

## Risks Or Questions

- The REQUEST path is a promise, not a fact: nothing forces the later PLANNED to use that key. Deliberately left unenforced (see §Scope); the worst case is one wrong `--key` suggestion before PLANNED is logged.
- `pendingArtifacts` still globs every `*-PLAN.md` at REQUEST rather than the announced key. Narrowing it is now possible but is a behavior change to pending detection, so it is not part of this task.
- This task's own REQUEST (`vxym`) stays pathless, so Director will not see a `next_command` for it even after the change lands.
