# PLAN: Let a project remove Baton and keep its records

Task ID: wbqt
Date: 2026-09-20
Planner: Claude Opus 5
Status: complete

## Director Brief

- Goal: Add `baton remove`, a dry run by default, that takes Baton out of a project — the rules block out of the instruction files and the files Baton installed out of `.baton/` — while every record the project made under Baton stays exactly where it is.
- Scope: New `internal/baton/remove.go`; `removeBlock`/deletion support added to the existing block writer in `pause.go` and `merge.go`; a fourth state in `projectBlockState`; a removed mode in `lint.go`; an absent-state refusal in `update.go`; `bootstrap/CLAUDE.md` added to the embed in `docs.go`; one sentence in both `PROTOCOL.md` copies; a short README section; tests.
- Success Criteria: Install-then-remove is byte-exact — splicing the active block into a project file and then running `remove --apply` returns that file to its original bytes (V1); every kept path under `.baton/` is byte-identical before and after, with `BATON-LOG.txt` gaining exactly two appended lines and no existing line changed (V2); `remove` refuses and writes nothing over a hand-edited block, over open tasks without `--force`, and over `--purge` in a project whose `.baton/` is not fully committed (V3); `lint` in a removed project exits 0 and still fails on a damaged record (V4); `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...`, `gofmt -l .` all clean.
- Risks: R1 removal cannot be undone and `bootstrap` refuses a project that still has `.baton/`, so re-installing later needs the user's explicit re-initialisation path; R2 `--purge` is the first command in Baton that can delete a record, guarded by a Git precondition rather than by a prompt; R3 a file whose Baton block sat last and which had no final newline gains one.
- Required Checks: `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...`, `gofmt -l .`, plus V1-V6 below.
- Executor Prompt: Implement `.baton/runs/20260920-1653-remove-baton-PLAN.md` steps 1-11. Removal deletes only bytes Baton wrote and can still recognise (Decision 1); when a file is not recognisable, keep it and say so in the report rather than deleting it. Reuse `writeInstructionBlocks` for the instruction-file change by extending it with a delete case — do not write a second atomic multi-file writer. Do not change the log format, the event names, the transition table, or any artifact contract; `remove` records itself with the existing `REQUEST -> RUN_DONE` pair, exactly as `pause` and `update` do. Do not put `remove` in `refusesWhilePaused`. Keep the `PROTOCOL.md` addition to the one sentence in Decision 8 and apply it identically to the `.baton/` and `bootstrap/.baton/` copies. Never run `remove` against this repository; every removal test uses a `t.TempDir()` fixture. Run every check with `go run ./cmd/baton`, not `.baton/bin/baton`.

## Goal

- A project can say "we are done with Baton" in one command, and afterwards no
  agent opening the project is told to follow the protocol, no `baton` command
  can be run from it, and nothing under `.baton/` that the project itself wrote
  has changed.
- Removal cannot destroy a record that Git holds, by construction rather than
  by care.
- Removal cannot destroy text a human wrote, including text a human wrote
  inside the `<baton-rules>` markers.
- `AGENTS.md` and `CLAUDE.md` come out of removal agreeing: either both carry
  no Baton block, or neither file was touched.
- A project Baton has been removed from is a state the classifier and `lint`
  name, not a state they report as damage.

## Scope

In scope:

- `internal/baton/remove.go` (new), and the parts of `pause.go`, `merge.go`,
  `lint.go`, `update.go`, `app.go`, `docs.go` the decisions below name.
- `.baton/PROTOCOL.md` and `bootstrap/.baton/PROTOCOL.md`: one sentence.
- `README.md`: one short section.
- Tests.

Out of scope:

- The log format, event names, transition table, and artifact contracts
  (constraint).
- The active and paused block texts. Removal deletes a block; it never writes
  one.
- `BOOTSTRAP.md` and `PROTOCOL-GUIDE.md` (Decision 9), and `HOW-TO-UPDATE.md`,
  which covers updating and not removal.
- REVIEW-01/02 nits N1-N8 from the pause round, except N3, which this round
  resolves as a by-product (Decision 6).
- `.baton/VERSION` and `VERSION` bumps, and rebuilding `.baton/bin/baton`:
  Director's at close, as in the previous three rounds.

## Findings: what the previous round left, and what the code does today

Read rather than assumed.

1. **The classifier does not have the state removal produces.** REVIEW-01's
   judgement section and the CLOSE's Residual Risks both say so, and I
   confirmed it in `pause.go:75-93`: `projectBlockState` returns `blockAbsent`
   only when *neither* `AGENTS.md` nor `CLAUDE.md` exists. A present file
   carrying no block returns an *error* (`AGENTS.md missing <baton-rules>
   block`). `checkAgentBlocks` (`lint.go:88-100`) has the same two branches.
   So today a removed project makes `lint` fail and `resume` report an error
   that reads like damage.
2. **`replaceBlock` and `computeReplacedContent` (`merge.go:22-63`) are
   reusable unchanged**, and `writeInstructionBlocks` (`pause.go:105-146`)
   already carries the property this round needs most: compute every file's
   new content first, write in sequence, restore the already-written files if
   a later write fails. REVIEW-02 verified that property by mutation. It has
   no delete case, which is the one thing removal adds.
3. **`MergeAgentBlock` appends when the target has no block**
   (`merge.go:50-60`). In a removed project, `update --apply` would therefore
   splice Baton straight back in — the mirror of the pause round's update
   trap, in the state this round creates (Decision 6).
4. **Nothing in the repository writes `.gitignore`.** The only mentions are
   `lint`'s two checks that `.baton/` is *not* ignored (`lint.go:120-170`),
   `BOOTSTRAP.md:107` and `PROTOCOL-GUIDE.md:429`, which both tell an installer
   to *remove* an existing `.baton/` rule, and this repository's own
   `/.baton/bin/` rule, which exists because this repository is Baton's source
   and ships six binaries under `bootstrap/`. Bootstrap adds no entry to a
   project's `.gitignore` (Decision 7).
5. **`bootstrap/AGENTS.md` and `bootstrap/CLAUDE.md` are byte-identical to each
   other**: a `# Agent Instructions` heading, a blank line, the block, and a
   final newline. A project bootstrapped into an empty slot gets that file
   verbatim, so "the block is all the file holds" is not the only shape in
   which Baton wrote the whole file (Decision 3).
6. **`Lint()` (`lint.go:376-400`) requires the installed machinery by name**:
   the five managed documents, `VERSION`, the binary, `bin/SHA256SUMS`, and the
   `templates`, `runs`, `lesson-learned`, `bin` directories. Every one of those
   is a file removal deletes, so without Decision 6 a removed project produces
   a dozen ERROR lines describing an intended state.

## Decision 1: the rule that decides every deletion

**Removal deletes only bytes Baton wrote and can still recognise as its own.
Anything else is kept, and the run says it was kept.**

Applied, this gives one table and no judgement calls at the call site:

| Path | Removal | Why it is on this side of the line |
|---|---|---|
| the `<baton-rules>` block in each instruction file | delete, only when it is the active or the paused text this binary ships | Baton's bytes, recognised by exact comparison; anything else is a human's (Decision 3) |
| `AGENTS.md` / `CLAUDE.md` as whole files | delete only when Baton wrote the whole file (Decision 3) | otherwise the project's file, which merely carries a Baton block |
| `.baton/PROTOCOL.md`, `DIRECTOR.md`, `PLANNER.md`, `EXECUTOR.md`, `HOW-TO-UPDATE.md` | delete each one whose bytes equal the copy this binary ships (`docs.Managed`, the same comparison `checkManagedDocuments` already makes) | shipped documents; a locally modified one is not recognisable, so it stays |
| `.baton/templates/*.md` | delete each file whose name matches a shipped template; `rmdir templates/` only if it is then empty | a project's own template is not Baton's |
| `.baton/VERSION` | delete | written by Baton, meaningless without it; the version it held is preserved in the removal's `RUN_DONE` summary (Decision 5) |
| `.baton/bin/baton[.exe]`, `.baton/bin/SHA256SUMS`, then `rmdir bin/` if empty | delete | the program and its checksum |
| `.baton/BATON-LOG.txt` (or the legacy `baton.log`), `runs/`, `GUIDANCE.md`, `LESSON-LEARNED.md`, `lesson-learned/`, `CONCURRENCY.md`, anything else under `.baton/` | keep | the record, and files the project wrote |
| `.gitignore`, and every other file in the project | keep, never read for writing | Decision 7 |

Nothing is deleted by glob or by "everything under this directory except".
Every deletion is an enumerated path, and an unrecognised file is a file that
survives. That is what makes "removal cannot lose a record" a property of the
code rather than of the plan.

**I agree with Director's position on the records, and add one reason to it.**
`runs/` and `BATON-LOG.txt` are the handoff record; they are committed to the
project's history, and they are the only place where the reasoning behind
changes that are still in the codebase is written down. Deleting them removes
an explanation of code that stays. So: records stay by default, and deleting
them is possible only under Decision 2.

**The alternative I rejected: delete the block and nothing else.** It is
tempting — smallest possible change, nothing irreversible. But a project that
asked to remove Baton and still carries `.baton/PROTOCOL.md`, a full role-file
set and a runnable binary has not removed it: the next agent that greps the
repository finds a protocol telling it to take a role, and `baton new-round`
still works. "Not in effect" and "not installed" would differ, and only the
instruction file would say which one is true.

## Decision 2: deleting the records is possible, behind `--purge`, and the confirmation is Git

`baton remove --apply --purge` deletes `.baton/` in full, records included.
It refuses unless **every path under `.baton/` that Git does not ignore is
tracked and clean** — `git status --porcelain --untracked-files=all -- .baton`
empty, and `git ls-files -- .baton` non-empty. Otherwise it names each
offending path and writes nothing. In a project that is not a Git repository,
`--purge` always refuses.

This is the constraint stated as a precondition: *removal must not be able to
lose a record that was committed*, so `--purge` deletes only bytes `HEAD` still
holds. A file Git ignores was declared by the project itself not to be part of
its history, so it is outside the promise; `.baton/bin/` in this repository is
exactly such a file.

**The confirmation is the precondition, the dry run, and `--apply` — not a
prompt.** Baton's commands are run by agents inside a delegation pipeline where
stdin is not a terminal; an interactive "type YES to continue" would either
hang or be answered by the agent that wanted to proceed, which is confirmation
in appearance only. A machine-checkable precondition cannot be talked around,
the dry run prints the commit that holds what would be deleted, and
`PROTOCOL.md` already reserves destructive decisions for the user.

**Why offer it at all**, given that `rm -rf .baton` is one line: because that
is precisely what an agent will run when the user says "get rid of it
completely", and it is the one action in this area that cannot be undone. The
same argument the pause round made for putting the command in the protocol
applies to the flag: the guarded version exists so the unguarded one is not
invented. A user who wants the directory gone gets it gone, and gets told first
which commit still holds it.

With `--purge` the timeline is one of the files being deleted, so **removal
records nothing**: there is nothing left to append to. The dry run says so in
one line and names the commit; the removal then exists in Git as the deletion
commit the user makes. Appending two lines to a file that the same command
deletes moments later would be a record of nothing.

## Decision 3: the block comes out, the surrounding content does not move

`removeBlock` computes a file's new content by cutting the block's bytes and
closing the seam:

- **Block last in the file** (the shape `MergeAgentBlock` produces when it
  appends, and therefore the shape every bootstrapped and updated project has):
  cut the block and everything after it, then trim the remainder to exactly one
  trailing newline. `merge` writes `<original>` + `\n\n` + `<block>` + `\n`, so
  this returns the file to its original bytes whenever the original ended with
  a newline. That round trip is Success Criterion 1.
- **Block in the middle** (only reachable by hand placement): cut the block and
  collapse the run of newlines spanning the seam to exactly one blank line
  (`\n\n`). Everything before and after is byte-identical.
- **Nothing but the block**, i.e. the remainder is empty or whitespace only:
  the file is deleted.

**A file whose whole content Baton wrote is deleted too.** Compare the file,
with its block replaced by the *active* block, against `bootstrap/AGENTS.md`
(for `AGENTS.md`) or `bootstrap/CLAUDE.md` (for `CLAUDE.md`) as this binary
ships them; if they are equal, delete the file. Without the normalisation step
a paused project would keep a stub that an active project deletes, which is an
inconsistency with no defensible reason. This covers the most common install of
all — the one where Baton created the instruction file — and leaves no trace
where it wrote everything. A project that added one line of its own to that
file is no longer byte-identical, so its file survives with its line intact.
The shipped file's own `# Agent Instructions` heading is not reproduced or
pattern-matched anywhere; only whole-file equality is used, so a heading a
future release changes cannot be half-recognised.

**Someone's own text inside the markers stops the removal.** If a present block
is neither the active nor the paused text this binary ships, `remove` refuses,
writes nothing, names the file, and says to move that text outside the markers
and re-run. No flag overrides it. The markers are Baton's, but bytes inside
them that Baton did not write are a human's, they are not recoverable from
upstream, and a command whose entire promise is "nothing you wrote is lost"
cannot make an exception for the one case where it cannot tell what it is
deleting. This is the same refusal `pause` already makes over a drifted block
(`pause.go:229`), for a different reason: `pause` refuses so that `resume` can
be lossless; `remove` refuses because there is nothing to restore from.

Both instruction files are handled by one call to the existing
`writeInstructionBlocks`, extended with a delete case (planned content `nil`
means delete, and the rollback restores a deleted file from the bytes it
already reads). Reusing it, rather than writing a second multi-file writer, is
what carries "the two files can never end up disagreeing" into this round:
REVIEW-02 verified that property by deleting the restore loop and watching the
test fail, and removal inherits the same test's guarantee.

## Decision 4: `remove` is a CLI command shaped like `update`

`baton remove [--apply] [--force] [--purge]`. Without `--apply` it prints what
it would do and changes nothing — the same shape `update` already has, for the
same reason, and here the reason is stronger: this is the only destructive
command Baton has.

Why a command rather than a documented procedure, in the order the reasons
carry weight:

1. **It must refuse.** Refusing over a hand-written block (Decision 3), over
   open tasks (Decision 5), and over an uncommitted `.baton/` (Decision 2) is
   most of what makes removal safe, and prose cannot refuse anything.
2. **It must change two files atomically.** Hand-editing `AGENTS.md` and
   `CLAUDE.md` is exactly the operation that produces the disagreement the
   constraint forbids.
3. **It must recognise its own bytes.** Decision 1's line between "Baton wrote
   this" and "the project wrote this" is drawn by comparison against copies the
   binary embeds. A human following a document cannot make that comparison, and
   would delete `.baton/` wholesale instead.

**The dry run must print, in this order:** the version being removed; for each
instruction file, whether its block is spliced out or the whole file is
deleted; every path under `.baton/` that will be deleted; every path under
`.baton/` that will be kept, with the timeline's line count and the number of
files under `runs/`, so the user sees the size of what survives; any file
removal declined to recognise and therefore kept; the open tasks that would be
abandoned, or `none`; and one closing line stating that after `--apply` no
`baton` command can be run from this project and that re-installing means
bootstrapping again. With `--purge`, the "keep" section is replaced by the
delete list plus the commit that holds it, or by the refusal and the list of
paths that are not committed.

Running `remove` in a project that has no block and nothing left to delete
prints `Baton is not installed in this project` and exits 0. If the block is
gone but machinery remains — an interrupted removal — the dry run shows the
remaining deletions and `--apply` finishes them. Removal is therefore
idempotent and resumable, which matters because it is the one command that can
fail halfway through a set of deletions.

**Order of operations**, chosen so that every reachable partial state is one an
agent reads correctly: instruction files first, then the `.baton/` deletions,
then the timeline append, then `bin/` last. If the run dies after the block is
gone, agents already treat the project as Baton-free and a re-run finishes the
job; the reverse order would leave instruction files pointing at a
`.baton/PROTOCOL.md` that no longer exists. `bin/` goes last because the
running program is inside it: on Unix a process survives unlinking its own
binary, and on Windows the delete fails, in which case `remove` prints
`left .baton/bin/baton.exe in place; delete it yourself (Windows cannot delete
a running program)` and still exits 0 — everything that defines the removal has
already happened, and failing the command over one leftover file would send the
user looking for damage that is not there.

## Decision 5: open work refuses, exactly as pause does, and the timeline ends with the removal

`remove` refuses when the timeline leaves any task open, listing each open task
and its last event, and telling the user to close them or re-run with
`--force`. `--force` proceeds and names them in the `RUN_DONE` summary. Same
shape as `pause`, sharing `openTaskList` — a third copy of open-task detection
would be the fault Decision 5 of the pause round set out to avoid.

**Why identical rather than stricter**, which Director invited me to consider:
a user who has decided to stop using Baton is, by definition, not going to
close those rounds, so a refusal with no escape would be a refusal to obey a
decision the user has already made — and the escape they would then reach for
is `rm -rf .baton`, which loses the record. The default refusal is what
matters: it is the moment the user finds out that an open round is about to be
abandoned, and it is cheap to produce. Removal differs from pause only in the
weight of the warning, not in its shape, because the same `--force` that
abandons a task for a week here abandons it for good.

**The timeline ends with the removal itself**, as the existing Direct-work
pair with a fresh task id:

```
<ts> | <id> | REQUEST  | Director | Remove Baton
<ts> | <id> | RUN_DONE | Director | Baton 0.33 removed; records kept under .baton/
```

and with `--force`, `Baton 0.33 removed with 1 open task: wbqt; records kept
under .baton/`. No new event name, no new transition, no format change. The
version comes from `.baton/VERSION` before it is deleted, which is what makes
deleting that file lossless: the last line of the record says which Baton
produced everything above it. An abandoned task keeps its last event and gains
nothing; the transition table is untouched, so a project that ever reinstalls
finds that task exactly where it stopped.

With `--purge`, nothing is appended (Decision 2).

## Decision 6: what changes in `projectBlockState`, `checkAgentBlocks`, and `update`

**`projectBlockState`**: classify each *present* instruction file independently
as `active`, `paused`, `absent` (the file exists and carries no block), or
`unknown`, then require the present files to agree and return that state. Two
consequences:

- A present file with no block no longer produces an error. `blockAbsent` comes
  to mean "no instruction file in this project carries a Baton block", which
  covers both a project with no instruction files (today's meaning) and a
  project whose files carry none (removal's result). That is the single change
  REVIEW-01 said this round would have to make.
- One file carrying a block and the other not is an error —
  `AGENTS.md carries a Baton block and CLAUDE.md does not` — because it is
  precisely the disagreement the constraint forbids, and it is the state a
  half-finished hand removal produces.

Callers: `refuseIfPaused` and the pause gate compare against `blockPaused`, so
`absent` reads as "not paused" and they are unchanged. `runPause`'s and
`runResume`'s `blockAbsent` branches already exist; `runResume`'s message
becomes `cannot resume: no instruction file carries a <baton-rules> block;
Baton was removed from this project — re-install it with bootstrap to bring it
back` (Decision 9 covers pause's).

**`checkAgentBlocks`** gains the fourth row and loses its two `missing block`
errors:

| Block content | lint output |
|---|---|
| active | `OK: AGENTS.md and CLAUDE.md Baton blocks match (active)` (unchanged) |
| paused | `OK: Baton is paused; records under .baton/ are unchanged by a pause` (unchanged) |
| **no block in any present file** | **`OK: Baton is not in effect here; .baton/ holds the records of work done under it`** |
| one file has a block and another does not | `ERROR: AGENTS.md carries a Baton block and CLAUDE.md does not` |
| present files' blocks differ | `ERROR: AGENTS.md and CLAUDE.md Baton blocks differ` (unchanged) |
| anything else | `ERROR: ... matches neither the active nor the paused block ...` (unchanged) |

**`Lint()` gains a removed mode**, and this is my answer to Director's question
of whether `lint` should say anything at all there: **yes, and only about the
record.** When the state is `absent` and `.baton/` exists, `lint` skips the
checks that require the installed machinery — the five managed documents,
`VERSION`, the binary, its executable bit, its checksum, `bin/`, `templates/`,
and `checkManagedDocuments` — and keeps `GUIDANCE.md`, `LESSON-LEARNED.md`,
`runs/`, `lesson-learned/`, the timeline, the legacy `scripts`/`protocol-guard`
checks, both Git-tracking checks (over the kept subset of
`requiredTrackedPaths`), and `checkLog`. A removed project that still carries
records has exactly one question worth asking — are they intact and still
committed — and `lint` should keep answering it. The alternative, which is what
today's code does, is a dozen ERROR lines describing an intended state, which
teaches the user to ignore `lint` output.

This reaches a user only through a binary outside the project (`BATON_DIR`, a
parent checkout, or `go run ./cmd/baton` in this repository), since the default
removal deletes the project's own binary. It is still worth the twenty lines:
it is how the tests for this round check a removed project at all, and reporting
"removed" instead of "broken" is the same distinction the pause round made
between a pause and damage.

Note this also settles REVIEW-01's N3 — a project with `.baton/` and no
instruction file at all passed `lint` silently. It now gets the removed line,
which is a true statement about it.

**`update --apply` refuses in a removed project** with
`Baton has been removed from this project; update would restore the rules
block. Re-install it with bootstrap if you want Baton back.` `MergeAgentBlock`
appends when it finds no block (Finding 3), so without this an update would
silently reinstall Baton — the pause round's update trap in the state this
round creates. In practice `update` cannot get that far in a default removal,
because `.baton/VERSION` is gone and `runUpdate` stops at it; the check is
there because that is an accident of the deletion list, not a guarantee, and
because the message it prints is the one the user needs.

**`merge-agent-block` is deliberately left alone here.** It refuses while
paused, but it must keep working against a file with no block: that is the
install primitive, the operation `BOOTSTRAP.md` step 2 and `HOW-TO-UPDATE.md`
step 7 both depend on, and it cannot tell "not installed yet" from "removed".
Refusing on `absent` would break bootstrap to close a trap that `update`
already closes one level up.

## Decision 7: `.gitignore` is never touched

Bootstrap adds no `.gitignore` entry, and no code in the repository writes one
(Finding 4). The only Baton-related rule a project can have is one it wrote
itself, and if it covers `.baton/` then `lint` has been failing all along. Under
Decision 1 a file Baton did not write is a file removal does not touch, so
`.gitignore` is untouched, and the dry run does not pretend otherwise. A user
whose `.gitignore` mentions Baton removes that line themselves; it is one line,
it is theirs, and guessing which of their rules were "for Baton" is exactly the
recognition Decision 1 refuses to fake.

This repository's own `/.baton/bin/` rule is a property of Baton's source
checkout, not of an installed project, and nothing in this round reads it — with
one exception worth naming: it is what `--purge`'s precondition treats as
outside Git's promise (Decision 2).

## Decision 8: the smallest protocol statement

One sentence, added to the existing pause sentence's paragraph in
`PROTOCOL.md` "Git And Updates", in both copies:

> A project can take Baton out with `<baton> remove --apply`, which deletes the
> rules block and the files Baton installed, keeps `.baton/` records unless
> `--purge` is given, and cannot be undone.

Why a tool cannot carry it: the user asks the agent, not the CLI. An agent in a
Baton project that is told "we're done with this, take it out" has read
`PROTOCOL.md` and its role file and nothing else, and every command it does
know about refuses to help. It will then invent an answer, and the two obvious
inventions are `rm -rf .baton`, which destroys the committed record this round
exists to protect, and hand-editing the two instruction files, which produces
the disagreement the constraint forbids. `baton help` lists `remove`, but only
an agent that already suspects the command exists will look there. One sentence
in the document every role reads first is the smallest thing that turns a
destructive invention into a guarded command — and it is the same argument the
pause round made, which is why it joins that sentence rather than opening a new
section. The clause about `--purge` is in it because the sentence has to say
that the safe default exists, or an agent reading only this will assume
"remove" already means "delete everything".

Nothing else goes in a protocol document. `HOW-TO-UPDATE.md` is about updating.
The removal's own dry run carries every detail a user needs at the moment they
need it.

## Decision 9: the rest of the sweep

Searched every `.md` and `.go` for `baton-rules`, `pause`, `resume`,
`merge-agent-block`, and the managed-document list, and read every hit outside
`.baton/runs/`.

| Location | Action |
|---|---|
| `README.md` "일시 중지와 재개" | **Add** a short "제거" subsection after it: the command, the dry run, what is deleted and what is kept, `--purge` and its Git precondition, and that there is no `resume` from it. User-facing, not a protocol document. |
| `pausedBlock` text (`pause.go:18-25`) | **None.** It says "Resume only when the user asks for it"; adding removal to it would put a second rule in a text that four success criteria pin byte-for-byte, for a case the user reaches through Director, not through the paused file. |
| `runPause`'s `blockAbsent` message | **Change** to match `runResume`'s (Decision 6): `cannot pause: no instruction file carries a <baton-rules> block; Baton is not installed here`. It is now a reachable state with a name, not only a corner case. |
| `usage()` | **Add** `remove [--apply] [--force] [--purge]` under "Configure and maintain", beside `pause`/`resume`. |
| `refusesWhilePaused` (`app.go:86-94`) | **None.** `remove` must not be in it (Decision 6 / Q4). |
| `BOOTSTRAP.md` step 1 | **None**, and it leaves a rough edge (R1): bootstrap stops when `.baton/` exists, which a removed project still has. Its own text already routes that to "user explicitly asks to re-initialise", so the path exists; documenting removal there is a Korean-document round of its own. |
| `PROTOCOL-GUIDE.md` (Korean) | **None**, same as the pause round, and it now lags by two features (R4). |
| `internal/baton/lint_test.go`, `coverage_more_test.go` | Fixtures asserting `AGENTS.md missing <baton-rules> block` will change meaning under Decision 6. Executor updates them to the new expectation; that is part of step 9, not an incidental edit. |
| `docs.go` | **Add** `bootstrap/CLAUDE.md` to the embed and a `ShippedInstructionFile(name)` accessor for Decision 3's whole-file comparison. N4 from REVIEW-01 (the duplicated `batonBlockPattern`) is untouched and stays open. |

## Plan

1. `docs.go`: embed `bootstrap/CLAUDE.md` alongside `bootstrap/AGENTS.md`; add
   `ShippedInstructionFile(name string) ([]byte, error)` returning the shipped
   bytes for `AGENTS.md` or `CLAUDE.md`. `ActiveBlock()` is unchanged.
2. `merge.go`: add `computeRemovedContent(target string) ([]byte, os.FileMode,
   bool, error)` next to `computeReplacedContent` — the seam rules of
   Decision 3, with the bool reporting "nothing but the block remains".
   `MergeAgentBlock` and `replaceBlock` are unchanged.
3. `pause.go`: extend `writeInstructionBlocks` to take a per-file planned
   content where `nil` means delete, reading and restoring deleted files in the
   same rollback loop. Existing callers pass the same block for every file and
   keep their behaviour.
4. `pause.go`: rewrite `projectBlockState` per Decision 6 — per-file
   classification with `absent` for a present file carrying no block, a
   disagreement error when present files differ in whether they carry one — and
   update `runPause`/`runResume`'s `blockAbsent` messages.
5. `internal/baton/remove.go` (new): `runRemove(args)` per Decisions 1-5 —
   argument parsing (`--apply`, `--force`, `--purge`), the state and drift
   refusals, the open-task refusal, the deletion plan built from Decision 1's
   table, the dry-run report of Decision 4, the `--purge` Git precondition of
   Decision 2, the apply path in the order of Decision 4, and the
   `REQUEST`/`RUN_DONE` pair of Decision 5.
6. `app.go`: dispatch `remove`; add it to `usage()`; do not add it to
   `refusesWhilePaused`.
7. `update.go`: refuse `--apply` when the state is `absent`, with the
   Decision 6 message, beside the existing paused refusal; print it from the
   dry run the same way.
8. `lint.go`: the fourth row in `checkAgentBlocks` and the removed mode in
   `Lint()`, per Decision 6. Leave every other check untouched.
9. Tests, per Success Criteria; update the existing fixtures named in
   Decision 9.
10. Documents: the sentence in both `PROTOCOL.md` copies (identical —
    `checkManagedDocuments` compares them), the README subsection.
11. Run every Required Check and record V1-V6 output in the RUN.

## Success Criteria

- **Install round trip**: for a project file with its own content ending in a
  newline, `merge-agent-block` the active block in, then `remove --apply`, and
  the file is byte-identical to the original. Verified for `AGENTS.md`-only,
  `CLAUDE.md`-only, and both-files projects.
- A file Baton wrote in full (byte-identical to `bootstrap/AGENTS.md` or
  `bootstrap/CLAUDE.md`, after normalising its block to the active text) is
  deleted; a file with one line of its own added survives with that line
  byte-identical; a file whose remainder is whitespace only is deleted. All
  three hold for a paused project as well as an active one.
- A block with a hand-written sentence inside the markers makes `remove` exit
  non-zero, name the file, and leave every file in the tree byte-identical.
  `--force` does not override it.
- After `remove --apply`: `.baton/BATON-LOG.txt` has exactly two new lines,
  every existing line unchanged, and the `RUN_DONE` line names the removed
  version; `.baton/runs/`, `GUIDANCE.md`, `LESSON-LEARNED.md`,
  `lesson-learned/` are byte-identical to before; `PROTOCOL.md`, `DIRECTOR.md`,
  `PLANNER.md`, `EXECUTOR.md`, `HOW-TO-UPDATE.md`, `VERSION`, `templates/`,
  `bin/` are gone; a locally modified managed document, a project's own
  template and an unrecognised file under `.baton/` all survive and are named
  in the output as kept.
- `remove` with an open task exits non-zero, names the task, and writes
  nothing; with `--force` it exits 0 and the `RUN_DONE` summary names the task.
- `remove` without `--apply` exits 0 and the whole tree is byte-identical
  afterwards; its output contains each section of Decision 4.
- A paused project removes without resuming first: `remove --apply` exits 0,
  and afterwards `resume` exits non-zero saying Baton was removed.
- `--purge` in a project whose `.baton/` has an uncommitted change or an
  untracked file exits non-zero, names that path, and deletes nothing; in a
  project with no Git repository it exits non-zero. With `.baton/` committed
  and clean it exits 0, `.baton/` is gone, and
  `git show HEAD:.baton/BATON-LOG.txt` still prints the timeline.
- `lint` in a removed project exits 0 and prints
  `OK: Baton is not in effect here; ...`; in the same project with a `runs/`
  artifact deleted it exits non-zero naming that artifact; in a project where
  `AGENTS.md` carries a block and `CLAUDE.md` does not it exits non-zero.
- `update --apply` in a removed project exits non-zero and leaves the
  instruction files byte-identical.
- `merge-agent-block` still splices a block into a file that has none.
- `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...` pass and
  `gofmt -l .` is empty.
- This repository's four `<baton-rules>` copies are byte-identical and active
  after the round, and nothing under this repository's `.baton/` is deleted by
  any test.

## Validation

- V1 install round trip: in a `t.TempDir()` fixture and again by hand,
  `shasum` the instruction files, merge the block in, `remove --apply`,
  `shasum` again — identical to the pre-merge hashes.
- V2 record integrity: `shasum` every file under `.baton/` before and after
  `remove --apply`; only the timeline differs, and
  `diff <(head -n -2 new) old` is empty.
- V3 refusal sweep: hand-edited block, open task without `--force`, `--purge`
  with a dirty `.baton/`, `--purge` outside Git — each exits non-zero with a
  `shasum` over every file in the tree identical before and after.
- V4 `lint` on a removed fixture: exits 0 with the removed line; then with one
  `runs/` artifact deleted: exits non-zero naming it.
- V5 `--purge` on a fixture that is a real Git repository with `.baton/`
  committed: exits 0, `.baton/` gone, `git show HEAD:.baton/BATON-LOG.txt`
  prints the timeline.
- V6 paused-then-removed: `pause`, then `remove --apply`, then `resume` —
  exits 0, 0, non-zero, with the removed message.
- `go test ./...`, `go vet ./...`, `gofmt -l .`, `git diff --check`.
- Use `go run ./cmd/baton` throughout; `.baton/bin/baton` lags the working tree
  and will report managed-document drift until Director rebuilds at close.
- Every fixture is a `t.TempDir()` or a scratch copy outside the repository.
  `remove` is never run against this repository.

## Report To Director

- Suggested summary: Add `baton remove`, a guarded uninstall that keeps the project's records
- Artifact path: `.baton/runs/20260920-1653-remove-baton-PLAN.md`

## Risks Or Questions

- **R1** (Decision 9): removal cannot be undone, and `BOOTSTRAP.md` step 1
  stops a bootstrap when `.baton/` exists — which a removed project still has.
  Re-installing therefore needs the user to ask for re-initialisation
  explicitly, which that document already covers but does not connect to
  removal. Deliberate; a candidate for a Korean-document round with R4.
- **R2** (Decision 2): `--purge` is the first command in Baton that can delete
  a record. Its guard is the Git precondition, not a prompt. A project that
  keeps `.baton/` untracked on purpose can therefore never purge through
  Baton — correct, in my reading, but it means the flag is unavailable exactly
  where a user might most want it.
- **R3** (Decision 3): a file whose Baton block sat last and which had no final
  newline before the block was merged in gains one newline on removal. One
  byte, in a shape the pause round's reviewer deliberately constructed, so it
  will be seen; naming it rather than claiming byte-exactness in all shapes.
- **R4** (Decision 9): `PROTOCOL-GUIDE.md` now describes neither pause nor
  removal.
- **R5**, for Director at close: `VERSION`, `.baton/VERSION` and the installed
  `.baton/bin/baton` are untouched by this plan, so the installed binary
  reports managed-document drift until it is rebuilt. Director's at close, as
  in the previous three rounds.
- **R6**: this round makes `projectBlockState` return `absent` where it used to
  return an error, which is a behaviour change for `pause` and `resume` as well
  as for `lint`. Both already had a `blockAbsent` branch, so the change is a
  message and a reachable path rather than new logic, but it widens beyond
  removal and Director should see it named.
