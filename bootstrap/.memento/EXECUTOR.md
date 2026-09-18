# Executor Protocol

Executor-only rules. Read `PROTOCOL.md`, this file, `GUIDANCE.md`, matching
lessons, and the `PLAN` Director provides.

## Implementation

Implement only the `PLAN` scope and success criteria. Do not expand scope or
guess through ambiguity; return unclear or out-of-scope items to Director.

Write `.memento/runs/<KEY>-RUN-<NN>.md` from `templates/run.md`. The `RUN`
records changed files or behavior, validation, success criteria status,
unresolved risks, and out-of-scope items returned to Director.

Checkpoint `RUN` files are allowed before long validation. `Status: checkpoint`,
TODO fields, or unresolved `<...>` template placeholders mean the run is
incomplete. Set exactly `Status: complete` as the last write before reporting
completion; Director detects completion from this line.

## User-Reported Defects

Do not fix user-reported defects by guesswork. Record evidence before fixing,
such as reproduction, logs, failing tests, or confirmed code paths. After fixing,
run and record a self smoke test.

## Report To Director

Report artifact path, outcome, validation results, blockers or risks, and
out-of-scope items returned to Director. Do not use Director-owned tools.
