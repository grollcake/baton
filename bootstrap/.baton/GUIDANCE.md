# Guidance

Use this file for durable project guidance that should survive across agent sessions.

Do not use this file for progress tracking, current task status, temporary plans, or next steps. Use `BATON-LOG.txt` and `.baton/runs/` artifacts for that.

Update this file when the user gives a durable instruction, constraint, preference, convention, security rule, or "do not" rule that future agents should keep following.

Do not update this file for one-off task details, transient debugging notes, temporary plans, or work progress.

Examples to record:

- "Always keep the public API backward compatible."
- "Do not introduce new runtime dependencies without asking."
- "Use pnpm for package scripts in this repository."
- "Never store customer data in fixtures."

Examples not to record:

- "Fix this failing test."
- "Try the other implementation first."
- "The current debugging hypothesis is a race condition."
- "Next, update the button copy."

## Stable Project Context

- <long-lived project purpose or domain context>
- <important architectural or product context that is unlikely to change often>

## User Instructions

- <standing user preference or instruction>

## Constraints

- <hard technical, product, compatibility, or operational constraint>

## Do Not

- <thing future agents must not do>
- Do not fix user-reported defects by guesswork; gather evidence first.
- Do not finish a user-reported defect fix without recording a self smoke test.

## Security And Privacy

- <data, credential, privacy, or operational safety rule>

## Conventions

- When Director delegates to Planner or Executor with an agent/subagent tool, run the
  delegated agent in the background if the tool supports it. In Claude Code,
  set `run_in_background: true` for every Planner/Executor delegation, including
  review delegation and follow-up SendMessage calls, so Director can respond to
  the user while delegated work continues.
- Treat `REVIEW` as evidence for the user's decision, never as approval. Only
  explicit user approval allows Director to write and append `CLOSE`; user
  `FEEDBACK` after any review is a normal pipeline step.
- Planner and Executor notify Director when artifacts are complete and provide a
  suggested event summary; only Director writes `BATON-LOG.txt`.
- The `baton` binary is Director-owned; delegates must not use its
  state-mutating commands.
- Keep delegation prompts brief: pass explicit constraints and source/artifact
  paths, not copied context or derived criteria. Planner defines criteria and
  risks in `PLAN`; Executor and the reviewer follow `PLAN`.
