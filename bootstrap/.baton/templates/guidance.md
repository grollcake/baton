# Guidance

Use this file for durable project guidance that should survive across agent
sessions.

Do not use this file for progress tracking, current task status, temporary
plans, or next steps. Use `baton.log` and `.baton/runs/` artifacts for
that.

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
  suggested event summary; only Director writes `baton.log`.
- The `baton` binary is Director-owned; delegates must not use its
  state-mutating commands.
- Keep delegation prompts brief: pass explicit constraints and source/artifact
  paths, not copied context or derived criteria. Planner defines criteria and
  risks in `PLAN`; Executor and the reviewer follow `PLAN`.
