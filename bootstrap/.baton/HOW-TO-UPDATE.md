# Baton Lifecycle: Update, Pause, Resume, Remove

Use this when the user asks to update, sync, or refresh Baton; to stop using
it for a while; or to take it out of a project that already has `.baton/`.

## Which Of These Is It?

| The user wants | Do this |
| --- | --- |
| Not to use it for a while | `<baton> pause` |
| Just this one task without the protocol | Nothing; short Q&A is not a recording target |
| Baton out of this repository entirely | Remove, below |
| A refresh in place | Update, below |

When a user says something like "stop using Baton" or "turn this off," they
usually mean pause, not removal: `pause` is reversible and removal is not.
Ask when the request is ambiguous, and offer `pause` first rather than
defaulting to removal.

## Update

0. If the project is paused, run `<baton> resume` first, update, then
   `<baton> pause` again. `update` and `merge-agent-block` both refuse while
   paused.
1. Read `.baton/VERSION`; stop and report if it is missing.
2. Fetch or copy the latest upstream `main` from `https://github.com/grollcake/baton` into a temporary location.
3. Select and checksum-verify the current platform binary under upstream `bootstrap/.baton/bin/<os>-<arch>/`.
4. Use the installed binary as `<baton>`. On Windows, run the new upstream binary from its upstream or a temporary path while the current directory is the target project.
5. Run `<baton> update --upstream <repo>` for a dry-run and inspect managed files for local customizations.
6. If managed files can be replaced, run `<baton> update --upstream <repo> --apply`.
7. If updating manually, use `<baton> merge-agent-block` for `AGENTS.md` and `CLAUDE.md` blocks.
8. Run the installed `<baton> lint` after the update.
9. Record `REQUEST -> RUN_DONE` in `BATON-LOG.txt` with the before/after `VERSION` in the summary if the update was manual. The update command records this automatically.

### Preserve

Do not overwrite:

- non-Baton instructions in `AGENTS.md` or `CLAUDE.md`
- `.baton/GUIDANCE.md`
- `.baton/LESSON-LEARNED.md`
- `.baton/lesson-learned/`
- `.baton/BATON-LOG.txt`
- `.baton/runs/`

If a Baton block contains project-specific instructions that cannot be
separated safely, leave the file unchanged and report the conflict.

### Report

Keep the final report short: version change, updated categories, and any manual
checks or conflicts.

## Pause And Resume

```text
<baton> pause      # turns the rules off only
<baton> resume     # reverses it
```

Both change only the instruction-file `<baton-rules>` block, never `.baton/`
records; a paused project's history and installed files are untouched. While
paused, commands that would write a record or advance a round refuse, and
status commands still work and report the paused state.

`pause` refuses if the timeline has open tasks, since a paused project stops
recording progress on them; `--force` overrides this and pauses anyway. A
paused project must `resume` before it can update, so that update does not
silently lift the pause by rewriting the rules block.

## Remove

### 1. Show what would be deleted, first

```text
<baton> remove
```

Without `--apply` this changes nothing and only prints the plan. Show this
output to the user as-is and get approval; do not decide on the user's
behalf.

The output lists files to delete, files kept, files Baton does not recognise
as its own and therefore leaves behind, and any open tasks.

### 2. Run it after approval

```text
<baton> remove --apply
```

This cannot be undone. In a Git repository, check for uncommitted changes
before running it.

### 3. Confirm the result

The project is removed once `.baton/` still holds its records and
`AGENTS.md`/`CLAUDE.md` no longer carry a `<baton-rules>` block. `<baton>
lint` still works in a removed project; it only checks that the remaining
record is intact.

When an instruction file had no content of its own and Baton had written the
whole file, the file is deleted outright, not just the block. The dry run
marks this `deleted (Baton wrote the whole file)`, so tell the user before
approval if they meant to keep using that file.

### Refusals

`remove` refuses whenever proceeding could cause an unrecoverable loss.
Explain the refusal to the user before working around it.

| Refusal | Meaning | What to do |
| --- | --- | --- |
| N open tasks | In-progress work would be abandoned with only its record left | Close the tasks, or pass `--force` if abandoning them is intended |
| Block matches neither the active nor the paused text | A human wrote text inside the markers | Move that text **outside** the `<baton-rules>` markers and re-run; `--force` does not override this |
| `--purge` refused: not committed | The records being deleted are not in Git history | Commit first |
| `--purge` refused: not a Git repository | Deleting would leave no way to recover | Make your own backup, then delete manually |

There is no `--force` override for user text inside the markers, by design:
those bytes cannot be reconstructed from what Baton ships.

### Removing The Record Too

```text
<baton> remove --apply --purge
```

`--purge` also deletes `BATON-LOG.txt`, `runs/`, `GUIDANCE.md`,
`LESSON-LEARNED.md`, and `lesson-learned/`. It is **not the default** because
these files are the project's own record, not Baton's output; `runs/` is
often the only document explaining why the code in the tree ended up the way
it did.

`--purge` only proceeds when every path under `.baton/` that Git does not
ignore is tracked and clean. This is a condition, not a confirmation prompt,
because in an agent pipeline the prompt would be answered by the agent that
wants to proceed, not the user. When the condition holds, what gets deleted
can still be recovered with `git show`.

Before suggesting `--purge`, tell the user concretely what disappears: how
many timeline lines, how many run records — both are in `remove`'s dry-run
output.

### After Removal

- **The `baton` command no longer works.** The binary is gone.
- **`resume` cannot bring it back.** This was removal, not a pause.
- **To reinstall**, follow `BOOTSTRAP.md`. Its first step stops if `.baton/`
  already exists, so to reinstall while keeping the existing record, confirm
  the scope with the user first.
- **The remaining record stays readable.** `BATON-LOG.txt` and `runs/` are
  plain text and need no Baton binary to read.
