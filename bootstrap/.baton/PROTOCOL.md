# Baton Protocol

Common rules for all roles. Role-specific operational details live in
`DIRECTOR.md`, `PLANNER.md`, and `EXECUTOR.md`; read only the file for your role.

## Roles

- `Director`: communicates with the user, classifies work, owns coordination,
  delegates, interprets results, and closes work.
- `Planner`: writes `PLAN` and reviews `RUN`. A review is evidence for the
  user's decision, not approval.
- `Executor`: implements `PLAN`, validates work, and writes `RUN`. Return
  ambiguity to Director instead of expanding scope.

Planner and Executor communicate only through Director.

## Read Before Work

When joining or resuming, read the active instruction file (`AGENTS.md`,
`CLAUDE.md`, or both), this file, your role-specific file, `GUIDANCE.md`, the
`LESSON-LEARNED.md` index, matching lesson records, and latest open-round
artifacts if any. The `<baton-rules>...</baton-rules>` blocks in
`AGENTS.md` and `CLAUDE.md` must stay identical.

At the start of each recordable phase, its responsible role rereads:

1. `.baton/GUIDANCE.md`
2. the `.baton/LESSON-LEARNED.md` index
3. only records under `.baton/lesson-learned/` whose `Applies When` or
   `Trigger / Symptom` matches that phase's scope

Clearly excluded requests may skip this check. If work needs file changes,
investigation, design judgment, or project-specific guidance, Director checks
before classifying it. Planner and Executor repeat lesson selection for their
own scope instead of relying on an earlier role.

## Work Classes

- Excluded from records: simple Q&A, short explanation, or brainstorming.
- `Direct`: minor localized edit, Baton bootstrap, or Baton update
  sync. Director handles it directly; no completion approval is required.
- `Standard`: multi-file work, design judgment, or work needing verification.
  Director coordinates Planner -> Executor -> Planner review, preferably in the
  background, while staying available to the user.

At every session start, Director first detects Codex or Claude Code and asks the
user to choose the Director, Planner, and Executor models. Show the saved choices
from `baton models get <platform>` as defaults and the currently available
choices from `baton models list <platform>`. Save the result with `baton
models set`; preferences are user-local and separate for each platform. Then ask
whether to always use branches, not use branches, or ask per task.

## Round Artifacts

Store artifacts in `.baton/runs/` using one stable
`<YYYYMMDD>-<HHMM>-<SLUG>` key:

- `.baton/runs/<YYYYMMDD>-<HHMM>-<SLUG>-PLAN.md`
- `.baton/runs/<YYYYMMDD>-<HHMM>-<SLUG>-RUN-<NN>.md`
- `.baton/runs/<YYYYMMDD>-<HHMM>-<SLUG>-REVIEW-<NN>.md`
- `.baton/runs/<YYYYMMDD>-<HHMM>-<SLUG>-CLOSE.md`

`<NN>` starts at `01`. Never overwrite an artifact. An artifact is incomplete
while any template placeholder (`<...>`) remains; a `RUN` is also incomplete
while any TODO remains. A completed `PLAN`, `RUN`, or `REVIEW` must contain exactly
`Status: complete`, and a completed `REVIEW` must set `Result` to
`ready-for-user-decision` or `blockers`. Each `RUN` records changed files,
change summary, validation, and unresolved risks. Use the matching template in
`.baton/templates/` for every artifact. Artifact `Task ID` values must
match `baton.log`; all artifacts for a task use the PLAN key, and each REVIEW
round must match its immediately preceding RUN round.

For user-reported defects, Director passes the report to Executor; Executor
records evidence before fixing and a self smoke test after fixing.

## Approval And Feedback

Only explicit user approval can close Standard work. Any user feedback after a
review is a normal pipeline step, not a reversal of completed work. Feedback
opens the next round `RUN-<NN+1>`; Director never asks the user to choose
between reusing a round and starting a new one.

## Prompt And Report Contract

Director passes explicit task requirements and artifact paths. Planner defines
success criteria, validation, and risks in `PLAN`; Executor and review follow
`PLAN`. Delegates do not expand scope, guess through ambiguity, use
Director-owned tools, or coordinate outside Director.

Reports to Director include artifact path, outcome, validation status, blockers
or risks, nits when applicable, and any user decision required. Executor reports
also list out-of-scope items returned to Director. For reviews, include three to
five manual check cases for the user.

## User-Facing Reports

Default to short user-facing reports. For `Direct` work, report outcome, changed
scope, and validation in one to three sentences. For `Standard` approval
requests, expose only outcome, validation status, actionable nits or risks,
three to five user manual checks, and the `REVIEW` path.

## Guidance, Lessons, And Security

- `GUIDANCE.md`: durable instructions, constraints, preferences, conventions,
  security rules, and prohibitions only.
- `lesson-learned/`: reusable mistakes, solutions, and validation knowledge from
  completed work only. Use `templates/lesson-learned.md`; each record includes
  `Applies When` and `Trigger / Symptom`, and accepted records are indexed in
  `LESSON-LEARNED.md`.
- Task progress stays in Director-owned state and round artifacts.
- Guidance and lesson updates require user acceptance.
- Never store secrets, credentials, customer data, personal information,
  sensitive internal information, or production secrets under `.baton/`.

## Git And Updates

Commit `.baton/` to Git. Do not add it to `.gitignore`. When updating,
read `.baton/HOW-TO-UPDATE.md` first. Bootstrap and update are recording
targets, not excluded meta work.

Baton uses the native `.baton/bin/baton[.exe]` Go binary and
does not require Go or a specific shell in installed projects. On Windows, it
can run from PowerShell, cmd, or Git Bash. Git-integrated commands require
`git` to be available on `PATH`.
