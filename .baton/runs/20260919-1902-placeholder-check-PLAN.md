# PLAN: Placeholder check accepts documented command syntax

Task ID: wqkb
Date: 2026-09-19
Planner: Planner (Claude Opus 5)
Status: complete

## Director Brief

- Goal: an artifact may describe CLI syntax in angle-bracket notation without being refused, while an unfilled template is still refused; and Planner and Executor are told to run `check-artifact` on their own artifact before reporting completion.
- Scope: `internal/baton/artifact.go`, its tests, and the Completion Marker sections of `PLANNER.md` and `EXECUTOR.md` in both `bootstrap/.baton/` and `.baton/`.
- Success Criteria: an artifact containing angle-bracket command syntax passes `check-artifact`; an unmodified copy of each artifact template, and an artifact with even one template placeholder left in place, is still refused; both role files tell the delegate to self-check and say lint is not a substitute.
- Risks: the new rule accepts angle-bracket text an author invented rather than copied from a template, so a hand-written unfilled marker is no longer caught; artifacts can no longer quote template placeholder text verbatim.
- Required Checks: `go test ./...`, `go vet ./...`, `gofmt -l .` (silent), `go run ./cmd/baton lint`.
- Executor Prompt: Replace the blanket angle-bracket scan in `checkArtifact` with a check against the exact placeholder tokens found in the project's own `.baton/templates/*.md`, per section Plan below. Then add the two self-check sentences to `PLANNER.md` and `EXECUTOR.md`, keeping the `bootstrap/.baton/` and `.baton/` copies byte-identical. Use `go run ./cmd/baton` for any check that must reflect the working tree. Managed-document drift from the role-file edits is expected until Director rebuilds at close, but `go run ./cmd/baton lint` must be clean because it builds from the working tree.

## Goal

- `checkArtifact` refuses an artifact only when it still carries text that came from a Baton template, not whenever it contains angle brackets.
- A delegate has a documented way to verify its own artifact before reporting `Status: complete`.

## Scope

In scope:

- `internal/baton/artifact.go`: the placeholder rule inside `checkArtifact`.
- `internal/baton/commands_test.go`: new and existing placeholder tests.
- `.baton/PLANNER.md`, `.baton/EXECUTOR.md`, `bootstrap/.baton/PLANNER.md`, `bootstrap/.baton/EXECUTOR.md`: Completion Marker guidance only.

Out of scope:

- `PROTOCOL.md`, `DIRECTOR.md`, `GUIDANCE.md`, and the templates themselves. `check-artifact` is read-only and already outside the Director-owned command list, so no protocol change is needed.
- Any change to `allowLegacy`, to lint, or to existing artifacts under `.baton/runs/`.
- Bumping `.baton/VERSION`; Director handles that at close as in previous rounds.

## Chosen Rule

Refuse an artifact when it contains, as a literal substring, any angle-bracket
token that occurs literally in a file under `.baton/templates/`. The token set
is the union of every `[^<>]` run enclosed in angle brackets across those
files, read from the project at check time.

Why the alternatives were rejected:

- Excluding code spans: the templates put placeholders inside backticks, so an
  unfilled template would pass. Director ruled this out and the templates
  confirm it.
- Requiring the artifact to differ from its template: changing one word makes a
  near-empty artifact pass, and it says nothing about an artifact that filled
  three sections out of nine. The failure it must catch is the partly filled
  artifact, which this rule cannot see.
- Restricting to placeholder-shaped tokens by shape: no shape rule separates
  the two populations. Template placeholders are English phrases such as the
  one on the Acceptance line of `close.md`, and the syntax that was wrongly
  refused in task `vxym` included a phrase of the same shape. Word count,
  punctuation, and spacing all fail on these two examples.

The chosen rule separates them by provenance rather than by shape, which is the
property the check actually cares about, and it makes the template-copy
criterion true by construction: a template copy consists of exactly these
tokens.

## Plan

1. In `internal/baton/artifact.go`, keep the existing angle-bracket regexp but
   repurpose it as the extractor, and add a method on `App` that globs
   `.baton/templates/*.md`, reads each file, and returns the set of tokens the
   regexp matches. If the glob fails, matches no file, or yields no token,
   return an error naming the problem so the check fails closed rather than
   silently disappearing. Verify: `go build ./...`.
2. In `checkArtifact`, replace the `placeholderPattern.MatchString(content)`
   branch with a loop over that set using `strings.Contains`, keeping the error
   prefix `artifact-check:` and the event and path, and naming the offending
   token so an author can find it. Leave the position of the check, the TODO
   check, and every other rule unchanged. Verify: `go test ./...` still passes
   `TestTemplatesAreRejected`.
3. In `internal/baton/commands_test.go`, next to `TestTemplatesAreRejected`,
   add a test that builds a complete EXECUTED artifact containing both lines
   quoted below and asserts `CheckArtifact` accepts it, and a second case that
   does the same for a complete PLANNED artifact. Verify: `go test ./...`.
4. In the same file, add a test that takes a complete artifact of each of the
   four kinds, reinserts exactly one placeholder token drawn from the matching
   template, and asserts `CheckArtifact` refuses it. This is the criterion a
   whole-file comparison would miss. Verify: `go test ./...`.
5. Append to the Completion Marker section of `PLANNER.md` and to the matching
   paragraph of `EXECUTOR.md`: an instruction to run `check-artifact` on the
   delegate's own artifact as the last step before reporting, and one sentence
   saying lint is not a substitute because lint only inspects artifacts already
   recorded in `baton.log`, and an artifact not yet appended is not among them.
   Write command examples with literal uppercase KEY and TASK-ID rather than
   angle brackets. Apply the identical text to the `bootstrap/.baton/` copy.
   Verify: `diff bootstrap/.baton/PLANNER.md .baton/PLANNER.md` and the same
   for `EXECUTOR.md` are both silent.
6. Run every required check. Verify: all four are clean.

### Test fixture lines

These are the lines task `vxym` was refused for. The angle brackets are written
as HTML entities here only so that this PLAN passes today's check; substitute
the literal characters in the fixture.

    returns `baton prompt plan --task-id &lt;id&gt; --key &lt;key&gt;`
    `--path &lt;valid PLAN path with no file&gt;`

## Success Criteria

- An otherwise-valid artifact containing both fixture lines passes
  `check-artifact` for EXECUTED, and a valid PLAN containing them passes for
  PLANNED. Covered by step 3.
- An unmodified copy of `plan.md`, `run.md`, `review.md`, and `close.md` is
  still refused for its matching event. Covered by the existing
  `TestTemplatesAreRejected`, which must keep passing unchanged.
- An otherwise-complete artifact of each of the four kinds that still carries a
  single template placeholder is refused. Covered by step 4.
- `.baton/PLANNER.md` and `.baton/EXECUTOR.md` each instruct the delegate to run
  `check-artifact` on its own artifact before reporting, and each states that
  lint does not cover an artifact that has not been appended yet. Their
  `bootstrap/.baton/` copies are byte-identical.
- Every artifact already under `.baton/runs/` still lints.

## Validation

- `go test ./...`: passes.
- `go vet ./...`: no findings.
- `gofmt -l .`: prints nothing.
- `go run ./cmd/baton lint`: passes, including managed-document comparison,
  because it builds from the working tree.
- `diff` of each role file against its bootstrap copy: silent.

## Report To Director

- Suggested summary: Placeholder check matches template tokens instead of any angle brackets; PLANNER and EXECUTOR self-check their artifacts
- Artifact path: .baton/runs/20260919-1902-placeholder-check-PLAN.md

## Risks Or Questions

- Stated plainly, as Director asked: the rule keys on provenance, so an author
  who invents an unfilled marker of their own, for example a bracketed TBD, is
  no longer caught for PLANNED, REVIEW, and CLOSE. EXECUTED artifacts keep the
  separate TODO check. A careless author who deletes the template's prompt text
  and leaves the section empty was never caught by the old rule either.
- Artifacts can no longer quote template placeholder text verbatim, because
  doing so is indistinguishable from leaving it unfilled. A future PLAN that
  discusses the templates has to paraphrase, as this one does.
- Director's note that existing artifacts under `.baton/runs/` pass only
  because lint uses `allowLegacy` is not accurate: the placeholder check runs
  before the `allowLegacy` branches and is never relaxed, and no file under
  `.baton/runs/` currently contains an angle-bracket token at all. Those
  artifacts therefore pass the new rule for the same reason they pass today,
  and nothing in this plan depends on `allowLegacy`.
- The check now reads the project's templates on every call. Lint calls it once
  per logged artifact, which is a handful of small files per record; no caching
  is planned, and none should be added without a measurement.
- If a project has customized its templates, the token set follows the
  customization, which is the intended behavior.
