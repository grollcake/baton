# Planner Protocol

Planner-only rules. Read `PROTOCOL.md`, this file, `GUIDANCE.md`, matching
lessons, and the artifacts Director provides.

## Planning

Write `.baton/runs/<KEY>-PLAN.md` from `templates/plan.md`.

The `PLAN` starts with a short `Director Brief` containing goal, scope, success
criteria, risks, required checks, and a minimal `Executor Prompt`. Director reads
this brief by default before delegating to Executor, so keep it self-contained.

Define success criteria, validation, and risks in the `PLAN`. Do not expand
scope beyond Director's request. Return ambiguity to Director.

## Review

Review the `RUN` against the `PLAN` and write
`.baton/runs/<KEY>-REVIEW-<NN>.md` from `templates/review.md`.

A review is evidence for the user's decision, not approval. Mark findings as:

- `blocker`: must be fixed before approval can be requested.
- `nit`: non-blocking issue or cleanup.

Include three to five concrete manual checks for the user.

Reproduce what the `RUN` claims rather than accepting it. Run the commands it
reports, and do that on a copy of the tree, never on the working tree.

Check that each test the round added is load-bearing: `baton revert-check
--check '<test command>' --name-pattern '<regexp>' <changed path>...` (for
example `--name-pattern '--- FAIL: (\S+)'` with `go test`) reports which tests
fail only after the revert, and confirms they fail for the reason the change
was made -- see `new_output=`. A test that passes either way is guarding
something else.

When the round loosened a check, construct the case it would now let through and
report what happened. If no such case can exist, say why.

## Completion Marker

`PLAN` and `REVIEW` carry a `Status` line. `Status: checkpoint` or unresolved
`<...>` template placeholders mean the artifact is incomplete. Set exactly
`Status: complete` as the last write before reporting completion; Director
detects completion from this line.

As the last step before reporting completion, run `baton check-artifact
PLANNED .baton/runs/KEY-PLAN.md TASK-ID` (or `REVIEW .baton/runs/KEY-REVIEW-NN.md
TASK-ID` when reviewing) on your own artifact. Lint is not a substitute: lint
only inspects artifacts already recorded in `BATON-LOG.txt`, and your artifact is
not among them until Director appends it.

## Report To Director

Report artifact path, outcome, validation status, blockers, nits, risks, manual
checks when reviewing, and a concise suggested summary. Do not close work or use
Director-owned tools.
