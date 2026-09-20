# How To Update Baton

Use this when the user asks to update, sync, or refresh Baton in a project
that already has `.baton/`.

## Steps

0. If the project is paused, run `<baton> resume` first, update, then
   `<baton> pause` again. `update` and `merge-agent-block` both refuse while
   paused.
1. Read `.baton/VERSION`; stop and report if it is missing.
2. Fetch or copy the latest upstream `main` from `https://github.com/grollcake/baton` into a temporary location.
3. Select and checksum-verify the current platform binary under upstream `bootstrap/.baton/bin/<os>-<arch>/`.
4. Use the installed binary as `<baton>` when available. For legacy script-only installs and all Windows updates, run the new upstream binary from its upstream or a temporary path while the current directory is the target project.
5. Run `<baton> update --upstream <repo>` for a dry-run and inspect managed files for local customizations.
6. If managed files can be replaced, run `<baton> update --upstream <repo> --apply`.
7. If updating manually, use `<baton> merge-agent-block` for `AGENTS.md` and `CLAUDE.md` blocks.
8. Run the installed `<baton> lint` after the update.
9. Record `REQUEST -> RUN_DONE` in `BATON-LOG.txt` with the before/after `VERSION` in the summary if the update was manual. The update command records this automatically.

## Preserve

Do not overwrite:

- non-Baton instructions in `AGENTS.md` or `CLAUDE.md`
- `.baton/GUIDANCE.md`
- `.baton/LESSON-LEARNED.md`
- `.baton/lesson-learned/`
- `.baton/BATON-LOG.txt`
- `.baton/runs/`

If a Baton block contains project-specific instructions that cannot be
separated safely, leave the file unchanged and report the conflict.

## Report

Keep the final report short: version change, updated categories, and any manual
checks or conflicts.
