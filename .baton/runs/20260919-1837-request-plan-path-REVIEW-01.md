# REVIEW-01: Record the PLAN path on REQUEST so the round key survives

Task ID: vxym
Date: 2026-09-19
Planner: Claude Opus 5
Run: .baton/runs/20260919-1837-request-plan-path-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- None.

### Nits

- `validateRequestPath` (artifact.go) discards the error returned by
  `validateArtifactPath` and substitutes one generic message. A user who
  mistypes a REQUEST path is told only that it "must be a PLAN artifact path
  under .baton/runs/", never which of the four rules it broke (prefix, dot-dot,
  character set, name suffix). Wrapping the inner error would keep the
  diagnostics the underlying validator already produces.
- Only the PLANNED case of the shared `requiresArtifact` lint branch is pinned
  by a test. EXECUTED, REVIEW and CLOSE ride the same switch arm, so they are
  covered in practice, but a change that exempted one of them individually
  would not be caught by a failing test.
- RUN-01's "Unresolved Risks" section closes with a note explaining that the
  artifact avoids angle-bracket notation because `check-artifact` rejects it.
  That is process commentary, not a risk of the delivered change, and it sits
  in the section a reader scans for residual danger. Director has already
  recorded the underlying `placeholderPattern` defect separately; no fix here.

## Evidence Reviewed

Loosening scope (verified by reading `internal/baton/lint.go` lines 148-168):
the path branch is now a three-way switch. `requiresArtifact`, defined on line
148 as PLANNED, EXECUTED, REVIEW or CLOSE, is the first case and still calls
`state.app.checkArtifact(...)`, which reads the file and therefore still
enforces existence. Only `record.Event == eventRequest` reaches the shape-only
case. The `default` arm, which now receives exactly FEEDBACK and RUN_DONE,
keeps the original `os.Stat` check and its "artifact path not found on line"
message byte for byte. Nothing else lost a guarantee.

How a future widening is caught: the exemption is a single equality test on
`eventRequest`, so widening it requires editing that one case, and two new
tests fail the moment it is. `TestLintRejectsMissingPlannedArtifact` fails if
PLANNED is moved into the exempt arm; `TestLintRejectsMissingFeedbackOrRunDonePath`
fails if either FEEDBACK or RUN_DONE is. Both tests pass against pre-change
source, confirming they are pure guards on the preserved behaviour rather than
assertions about the new behaviour.

Shell-injection guard, `TestLintDoesNotExecutePaths`: I reproduced its fixture
on a scratch copy and captured what lint reports. It fails with exactly one
error, and that error is
`Baton log line 3: artifact-check: REQUEST path must be a PLAN artifact path
under .baton/runs/: $(touch ...)`. It fails through the new shape check, not
incidentally through some other rule and not through a missing file. The marker
file is never created.

Adversarial case, constructed directly against `validateRequestPath`: a REQUEST
path that passes the shape check yet carries a shell metacharacter cannot
exist. `artifactPathPattern` is the allowlist `^[A-Za-z0-9._/-]+$`; every shell
metacharacter is outside it. I probed dollar-paren command substitution,
backticks, semicolon, pipe, ampersand and a bare space, each wrapped in an
otherwise perfectly formed `.baton/runs/NAME-PLAN.md` path, and every one was
refused. Dot-dot traversal is refused by its own explicit check. The one path
that passes while looking unusual is `.baton/runs/-rf-PLAN.md`, a leading-dash
segment; it is inert because the mandatory `.baton/runs/` prefix means the
string can never be read as a leading option, and because nothing in this
package passes a log path to a shell. `os/exec` appears only in `app.go` for
`git` invocations, never in the lint or artifact path.

Test load-bearingness, on a copied tree at the scratchpad with the working tree
untouched: reverting only `artifact.go`, `events.go` and `lint.go` and keeping
all test files produces exactly three failures, matching RUN-01's claim.
`TestAppendRequestPathValidation` fails with "append accepted a REQUEST path
that is not a PLAN artifact path", because the old `runAppend` has no REQUEST
path validation. `TestStatusNextCommand` and `TestStatusOffersPlanCommandAtRequest`
fail because the old `runNewRound` never wrote `Path` and `nextCommand` had no
REQUEST branch. The three new lint tests pass on both sides, as guards should.

`TestStatusNextCommand` in coverage_test.go: the change is the intended
inversion, not a weakened test. The old two lines asserted the absence of a
substring; the new three lines assert the presence of the full expected
command, built from the task id and key that `new-round` just printed. The
REQUEST-stage case is still present and is now strictly more specific than it
was, since absence of "next_command:" was satisfied by any output whereas the
new assertion pins the exact command text. The rest of the test, covering
PLANNED onward, is unchanged.

Backward compatibility: all six REQUEST lines in `.baton/baton.log` are
pathless, confirmed by field extraction. `go run ./cmd/baton lint` prints
`baton-lint passed`, "Baton tasks: 6 total, 5 closed, 1 open". No task in this
log currently rests at REQUEST, so I exercised the pathless case on the scratch
fixture instead: a hand-appended pathless REQUEST produced `last_event: REQUEST`
with no `next_command` line at all, and added no lint error. The unit test
`TestLintAcceptsPathlessRequest` pins the same behaviour. The three added log
lines are appends; no existing line was modified.

Real CLI on a scratch fixture built from `bootstrap/.baton` with a freshly
built binary. `new-round demo-fixture` printed
`plan_path='.baton/runs/20260919-1850-demo-fixture-PLAN.md'` and wrote the log
line `... | itfv | REQUEST  | Director | Demo fixture run |
.baton/runs/20260919-1850-demo-fixture-PLAN.md` with no `.baton/runs/`
directory on disk. `status --task-id itfv` printed `last_event: REQUEST` and
`next_command: baton prompt plan --task-id itfv --key 20260919-1850-demo-fixture`.
`append REQUEST --path .baton/runs/20260711-1001-slug-PLAN.md` was accepted and
echoed its record although no such file exists. `append REQUEST --path
.baton/runs/notes.txt` and the metacharacter path both exited 1 with the
shape-check message, and the command-substitution probe created no file.
`lint` on the fixture reported three errors, all of them about the absent
`.baton/bin/baton` binary that the fixture never installed, and zero errors
about any log path.

RUN-01 accuracy under the no-angle-bracket constraint: every factual claim I
re-ran reproduced. The substitutions it made, concrete example values in place
of placeholder notation, read clearly and lost no precision; the revert-check
paragraph in particular names the three failures and their causes correctly.
The only cost is the trailing meta-note recorded as a nit above.

Scope and gates: `git status` shows changes confined to `.baton/baton.log`
(three appended lines), the three source files, four test files, and the two
new round artifacts. No managed document, no `VERSION`, no `.baton/bin/`.
`go test ./...` passes, `go vet ./...` is silent, `gofmt -l .` is silent.

## Suggested User Checks

- Open a throwaway round with `baton new-round scratch-check --summary "Check"`,
  then run `baton status --task-id` with the id it printed. Confirm the
  `next_command` line gives `baton prompt plan` with the same key `new-round`
  printed, and that no PLAN file exists on disk yet. Do not append anything
  further to this task.
- Run `baton lint` in this repository and confirm `baton-lint passed`, so the
  forward reference on this branch's own REQUEST-style lines and the six
  pathless historical REQUESTs all still pass.
- Run `baton status --task-id jiaq` (a closed task whose REQUEST is pathless)
  and confirm nothing changed for it: no `next_command`, same event list.
- Try `baton append REQUEST --task-id zzzz --role Director --summary "Probe"
  --path .baton/runs/notes.txt` and confirm it is refused, then confirm no new
  line reached `.baton/baton.log`.
- Confirm you are comfortable with the intended trade: a REQUEST path is a
  promise about a file that does not exist yet, and nothing forces the later
  PLANNED to use that key. The worst case is one wrong key suggestion before
  PLANNED is logged.

## Report To Director

- Suggested summary: Review round 01
- Artifact path: .baton/runs/20260919-1837-request-plan-path-REVIEW-01.md

## Required Next Step

- ask Director to request user approval
