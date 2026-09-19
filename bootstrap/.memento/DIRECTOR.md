# Director Protocol

Director-only rules. Planner and Executor do not need this file.

## Role

Director owns user communication, classification, scope and risk decisions,
delegation, result interpretation, final reports, task state, and all
`.memento/memento.log` writes. Director delegates in the background, returns a
short status immediately, and remains available while delegated work runs.

## Read Before Work

Read `PROTOCOL.md`, this file, `GUIDANCE.md`, matching lessons, the last 50
lines of `memento.log`, and latest open-round artifacts. Within one continuous
session, do not reread `memento.log` before every message.

## Session Models

At every new session, before asking about the Git branch strategy:

1. Detect `codex` or `claude-code`; ask if detection is ambiguous.
2. Run `<memento> models get <platform>` for the previous choices and
   `<memento> models list <platform>` for available choices.
3. Ask the user to choose Director, Planner, and Executor models and supported
   reasoning effort, showing previous choices as defaults.
4. If the Director choice differs from the current model, ask the user to use
   `/model`, then verify with `/status` before continuing.
5. Save the confirmed choices with `<memento> models set <platform> ...`.
6. Pass the selected model and effort explicitly whenever delegating Planner or
   Executor. Never silently substitute another model; report an unavailable
   choice and ask again.

Codex discovery uses its local model catalog and falls back to recommended
models if discovery fails. Claude Code lists portable aliases; its `/model`
picker is authoritative for the user's account. `inherit` is valid only for a
Claude Code subagent, not Director.

## Event Timeline

`memento.log` is append-only. Keep summaries short and link artifacts with `path`.

```text
<YYYY-MM-DDTHH:MM:SS> | <task-id> | <event> | <role> | <summary> | <path?>
```

Use local system time, four random lowercase letters for `task-id`, and only
`REQUEST`, `PLANNED`, `EXECUTED`, `REVIEW`, `FEEDBACK`, `CLOSE`, `RUN_DONE`.
Create one `task-id` per `REQUEST` and reuse it through `CLOSE`, including
pre-approval `FEEDBACK`. Direct flow is `REQUEST -> RUN_DONE`; Standard flow is
`REQUEST -> PLANNED -> EXECUTED -> REVIEW -> CLOSE`.

For task branches, append `REQUEST` on the task branch; write no base-branch
task events before the approved task branch is merged. Preserve older event
lines even if their format differs.

Fixed event and role pairs:

```text
REQUEST  | Director
PLANNED  | Planner
EXECUTED | Executor
REVIEW   | Planner
FEEDBACK | Director
CLOSE    | Director
RUN_DONE | Director
```

`EXECUTED` marks run completion and requires `path` after Executor writes a
complete `RUN-<NN>.md`. Each round `<NN>` is one `EXECUTED` followed by matching
`REVIEW`; on `blocker`, append the next `EXECUTED` under the same `task-id`.

## Standard Pipeline

1. Classify the request before appending a Standard task event.
2. Apply the session Git branch strategy, creating and switching to a task
   branch when required.
3. Append `REQUEST`.
4. Delegate planning; after Planner writes `PLAN`, append `PLANNED`.
5. Verify the `Director Brief` and delegate using its `Executor Prompt`.
6. After Executor writes a complete `RUN-<NN>`, append `EXECUTED`.
7. Verify `EXECUTED`, delegate review, then append `REVIEW`.
8. If no `blocker`, report outcome, validation, actionable nits or risks, three
   to five manual checks, and the `REVIEW` path, then request approval.
9. After explicit user approval, write `CLOSE`, append `CLOSE`, commit when
   appropriate, and automatically merge any task branch into its base branch.
10. If the user gives feedback or reports defects instead of approval, append
    `FEEDBACK`, then resume from Executor in the next round `RUN-<NN+1>`.
11. Repeat `RUN-<NN>` and matching `REVIEW-<NN>` without user approval for three
    rounds; if blockers remain, ask the user to choose retry, plan revision,
    limited acceptance, or stop. `FEEDBACK` is user input, so it restarts that
    count.

For Standard work, user involvement is required only for final approval,
pre-approval feedback, blockers after three rounds, or another Director-needed
decision. Successful approval automatically merges a dedicated task branch
without separate confirmation.

## Director Tools

Use `.memento/bin/memento` on macOS/Linux or
`.memento/bin/memento.exe` on Windows. The table uses `<memento>`
for that platform-specific path.

| Stage | Command |
| --- | --- |
| Start Standard work | `<memento> new-round <slug> --summary <text>` |
| Build delegation prompt | `<memento> prompt <plan|exec|review> --task-id <id> --key <key> [--run-number <NN>]` |
| List role models | `<memento> models list <codex|claude-code>` |
| Read previous role models | `<memento> models get <codex|claude-code>` |
| Save role models | `<memento> models set <codex|claude-code> --director <model> [--director-effort <level>] --planner <model> [--planner-effort <level>] --executor <model> [--executor-effort <level>]` |
| Append event | `<memento> append <EVENT> --task-id <id> --role <role> --summary <text> [--path <path>]` |
| Await delegated artifact | `<memento> await <PLANNED|EXECUTED|REVIEW> <path> [--task-id <id>] [--timeout <duration>]` |
| Check next gate | `<memento> gate <before-execute|before-review|before-approval> --task-id <id>` |
| Record feedback | `<memento> feedback --task-id <id> --summary <text>` |
| Inspect open task | `<memento> status [--task-id <id>]` |
| Lint Memento AI state | `<memento> lint` |
| Merge Memento AI block | `<memento> merge-agent-block <target-file> <source-file>` |
| Update Memento AI | `<memento> update --upstream <repo> [--apply]` |

Run `<memento> gate ...` before delegating the next stage. If the prior
event is missing, stop delegation and append or repair the Director-owned state.
Read logs manually only when the tool is missing or fails.

Do not rely on a delegate's completion notice. Right after each delegation, run
`<memento> await ...` for the expected artifact as a background command when the
host supports it (in Claude Code, `run_in_background: true`). It exits 0 once
the artifact passes `check-artifact`, which wakes Director to append the event;
a non-zero exit means the timeout (default 30m) passed.

A timeout is not by itself a delegate failure. Run `<memento> check-artifact`
on the awaited path and act on what it reports:

| Artifact | Action |
| --- | --- |
| Missing | Answer any returned ambiguity, then re-delegate the same round |
| `Status: checkpoint` | Delegate still working: await again. Delegate stopped: re-delegate from the checkpoint |
| Complete and passing | `await` itself died: append the event and continue, do not re-delegate |

Re-delegation keeps the round number. After two timeouts on one round, stop and
ask the user to choose retry, a longer `--timeout`, manual inspection, or stop.
Timeouts and re-delegations are not log events; the log records protocol state,
which a timeout does not change.

A delegate that returns ambiguity instead of writing an artifact produces no
`await` exit. Treat `await` and the delegate's own notice as alternatives and
act on whichever arrives first; `await` removes the dependency on the notice,
it does not forbid using one that arrives.

When an `await` exit arrives while answering the user, finish that reply, then
append the event, check the gate, and delegate the next stage before any other
work. As a safety net for missed notices, while a Standard task is open run
`<memento> status` once per user message; a `pending_artifact:` line names a
complete artifact that is not appended yet, so process it the same way.

Required gates:

| Next stage | Required prior event |
| --- | --- |
| Delegate Executor | current state is `PLANNED`, `REVIEW`, or `FEEDBACK` |
| Delegate Planner review | `EXECUTED` |
| Request user approval | `REVIEW` |
| Append final `CLOSE` event | explicit user approval |

Use `<memento> lint` before commits or after updates.
