# Agent Instructions

<baton-rules>

## Baton

This project follows Baton. Read `.baton/PROTOCOL.md`, then read
only your role file:

- Director: `.baton/DIRECTOR.md`
- Planner: `.baton/PLANNER.md`
- Executor: `.baton/EXECUTOR.md`

1. At the start of each recordable phase, read `.baton/GUIDANCE.md`, the
   `.baton/LESSON-LEARNED.md` index, and only matching lesson records.
2. At every session start, Director detects Codex or Claude Code, runs
   `baton models get <platform>` and `baton models list <platform>`, and
   asks the user to choose Director, Planner, and Executor models, showing the
   previous choices as defaults. Save choices with `baton models set`.
3. Director then asks whether to use a Git branch strategy: always use
   branches, do not use branches, or ask per task.
4. Route Planner and Executor through Director. Director delegates in the
   background when possible, returns a short status immediately, and remains
   available to the user while delegated work runs.
5. `REVIEW` is evidence, not approval; only explicit user approval can close
   Standard work, and approval requests include 3-5 user manual checks.
6. Director passes user-reported defects to Executor; Executor records evidence
   before fixing and a self smoke test after fixing.
7. Keep older `.baton/runs/` rounds append-only; update `GUIDANCE.md` and
   `lesson-learned/` only after user approval; never store secrets or sensitive
   data in `.baton/`.
8. Baton uses the native `.baton/bin/baton[.exe]` Go binary
   and does not require a specific shell. Git-integrated commands require
   `git` to be available on `PATH`.

</baton-rules>
