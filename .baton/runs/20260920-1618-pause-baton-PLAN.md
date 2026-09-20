# PLAN: Let a project pause Baton without removing it

Task ID: srkz
Date: 2026-09-20
Planner: Claude Opus 5
Status: complete

## Director Brief

- Goal: Add `baton pause` and `baton resume`, expressed as a paused `<baton-rules>` block in the instruction files, so a project can set Baton aside and come back to it with every record intact and no way to un-pause by accident.
- Scope: New `internal/baton/pause.go`; pause gate and usage in `app.go`; a three-way block check in `lint.go`; pause refusals in `update.go` and `merge.go`; a shared `replaceBlock` split out of `MergeAgentBlock`; `bootstrap/AGENTS.md` added to the embed in `docs.go`; one sentence in both `PROTOCOL.md` copies; one line in both `HOW-TO-UPDATE.md` copies; a README section; tests.
- Success Criteria: `pause` then `resume` leaves the working tree byte-identical except for the four timeline lines the two commands append (Validation V1); `append`, `new-round`, `feedback`, `gate`, `prompt`, `await`, `merge-agent-block`, and `update --apply` all refuse while paused and write nothing; `status`, `lint`, `version`, `guide`, `check-artifact`, `models`, and `update` dry run all work and report the pause; `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...`, `gofmt -l .` all clean.
- Risks: R1 the timeline-append reading of the constraint (Decision 7, Director decision requested before Executor starts); R2 lint gets stricter for every project, not only paused ones (Decision 6); R3 `PROTOCOL-GUIDE.md` is left stale (Decision 9).
- Required Checks: `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...`, `gofmt -l .`, plus V1-V4 below.
- Executor Prompt: Implement `.baton/runs/20260920-1618-pause-baton-PLAN.md` steps 1-10. The paused block text in Decision 2 is exact, including blank lines; do not reword it. Put the pause gate in one place (`App.Run`'s dispatch), not in each command. Do not change the log format, the event names, the transition table, or any artifact contract; `pause` and `resume` record themselves with the existing `REQUEST -> RUN_DONE` pair, the same way `runUpdate` already does. Do not write anything under `.baton/runs/`, and do not modify any existing line of `BATON-LOG.txt`. Keep the `PROTOCOL.md` addition to the one sentence in Decision 8 and the `HOW-TO-UPDATE.md` addition to the one line there, and apply each to both the `.baton/` and the `bootstrap/.baton/` copy identically. Run every check with `go run ./cmd/baton`, not `.baton/bin/baton`.

## Goal

- A project can stop Baton for a week with one command and restart it with one
  command, and neither command changes a single byte under `.baton/runs/` or
  rewrites a single existing line of `BATON-LOG.txt`.
- An agent that opens the project while it is paused learns that Baton is not in
  effect from the file its tool already reads, without opening `.baton/`.
- A paused project cannot be un-paused by a command that was not asked to
  un-pause it, in particular `update`.
- `lint` still verifies a paused project, and tells a pause apart from damage.
- The way "Baton is not in effect here" is expressed is decided once here, and
  the removal round reuses it rather than inventing a second one.

## Scope

In scope:

- `internal/baton/pause.go` (new), `app.go`, `lint.go`, `update.go`, `merge.go`,
  `docs.go`, and the matching tests.
- `.baton/PROTOCOL.md` and `bootstrap/.baton/PROTOCOL.md`: one sentence.
- `.baton/HOW-TO-UPDATE.md` and `bootstrap/.baton/HOW-TO-UPDATE.md`: one line.
- `README.md`: one short section.

Out of scope:

- Removing Baton (round two).
- The log format, event names, transition table, and artifact contracts
  (constraint).
- The active `<baton-rules>` text in any of the four copies. It is unchanged;
  this round only adds a second, paused text the CLI can swap in.
- `BOOTSTRAP.md` and `PROTOCOL-GUIDE.md` (Decision 9).
- `.baton/VERSION` and `VERSION` (Director's at close, noted in Risks).

## Finding: what the code does today

Read rather than assumed. Four facts drive every decision below.

1. `batonBlockPattern` is `(?s)<baton-rules>.*?</baton-rules>`
   (`internal/baton/lint.go:62`). The tag is literal, with no attribute
   position. An attribute such as `<baton-rules state="paused">` would not
   match, and `MergeAgentBlock` would then *append a second block* instead of
   replacing the first (`merge.go:24-42`). This rules out expressing the pause
   on the tag.
2. `MergeAgentBlock` splices the source block over the target block in place and
   is called by `runUpdate` for `AGENTS.md` and, when present, `CLAUDE.md`
   (`update.go:132-145`). Nothing in that path looks at what the target block
   says. This is Director's "update trap", confirmed.
3. `checkAgentBlocks` returns silently when either instruction file is missing
   (`lint.go:69-71`) and otherwise only requires the two blocks to be
   byte-equal. Two identically emptied blocks pass. `BOOTSTRAP.md:60` tells
   Claude users they may keep only `CLAUDE.md`, so the early return is on a
   reachable path, not a corner case.
4. `docs.go` embeds the five managed `.baton/*.md` documents but **not**
   `bootstrap/AGENTS.md`. An installed project therefore holds no copy of the
   active block anywhere except in its own instruction files. Resume needs one.

## Decision 1: the block is the paused state, and it is the only copy of it

**The project is paused when, and only when, every `<baton-rules>` block present
in the project's instruction files is byte-identical to the paused block in
Decision 2.** No marker file, no field in `VERSION`, no file under `.baton/`.

Weighed against the alternatives Director named:

- *A marker file under `.baton/`*: trivially readable by the CLI, invisible to
  the agent. The agent is the reader that matters: the block exists precisely
  because `.baton/` is not read automatically. A marker alone would pause the
  CLI and leave every agent behaving as if Baton were live.
- *Marker file **and** block*: two sources of truth that can disagree, and the
  disagreement that matters is silent — marker present, block active, agent
  runs the protocol and the CLI refuses every command it issues. A pause whose
  two halves can drift is not a pause.
- *A `state=` attribute on the tag*: ruled out by Finding 1; it would break
  splicing and duplicate the block on the next update.
- *Block text only* (chosen): one place, read by the agent automatically,
  readable by the CLI with the regexp it already has, and diffable by a human.
  Its only cost is that detection is a text comparison — the same technique
  `hasExactLine` already carries for the REVIEW result gate
  (`events.go:412`, `lint.go:327`), so it is a technique this codebase already
  trusts for a safety-relevant decision.

For the removal round: "Baton is not in effect here" is stated in the
instruction block and nowhere else. The classifier this round adds
(`blockState`, Decision 5) returns `active`, `paused`, `absent`, or `unknown`;
removal is the transition to `absent`, and the removal round should reuse
`blockState` and `replaceBlock` rather than add a second notion of the state.

## Decision 2: the paused block text

Byte-identical in `AGENTS.md` and `CLAUDE.md`, exactly as below, including the
blank lines after `<baton-rules>` and before `</baton-rules>`:

```
<baton-rules>

## Baton

Baton is installed in this project and is paused. Do not follow
`.baton/PROTOCOL.md`, do not run a `baton` command, and do not write under
`.baton/`: work as if Baton were not installed. `.baton/` holds this project's
records; leave every file in it unchanged. Resume only when the user asks for
it, with `.baton/bin/baton resume`.

</baton-rules>
```

Four sentences, the same shape and the same `## Baton` heading as the active
block, so a diff between the two states is one hunk in one paragraph.

Why each sentence:

- Sentence 1 states the state, not a prohibition, so an agent that reads only
  the first line already knows.
- Sentence 2 names the three Baton-shaped things an agent might otherwise do,
  in the same order the active block names them, and then gives the positive
  instruction ("work as if Baton were not installed") so the agent is not left
  guessing whether it may work at all. A paused project is a working project.
- Sentence 3 is the one that protects the records. Without it, "Baton is not in
  effect" invites an agent to tidy `.baton/` away.
- Sentence 4 names the exit, and conditions it on the user asking, so an agent
  that finds the pause inconvenient does not resolve it by resuming.

The paused text lives as a Go constant in `internal/baton/pause.go`, not as a
file under `bootstrap/`. It is a rule the CLI enforces and the only thing that
ever writes it is the CLI, which matches Director's standing instruction that
rules belong in the CLI. The *active* block keeps living in
`bootstrap/AGENTS.md`, because `update` splices it from the upstream checkout on
disk, not from the binary.

## Decision 3: what the CLI does while paused

Two categories, one sentence each.

**Refuse, writing nothing, exit non-zero**: `append`, `new-round`, `feedback`,
`gate`, `prompt`/`subagent-prompt`, `await`, `merge-agent-block`, and
`update --apply`. These either write state or advance a round.

**Work normally, and say the project is paused**: `status`, `lint`, `version`,
`guide`, `check-artifact`, `models`, and `update` without `--apply`. These only
report.

`append` during a pause **refuses and writes nothing.** This is the case
Director singled out and it is the one case where no other answer is available:
`BATON-LOG.txt` is append-only, so a warning that still appended would leave a
line belonging to no live round permanently in the record, and removing it later
would violate this round's own constraint. Refusing costs an agent one error
message; warning costs the project a line it can never take back.

Why `gate` and `prompt` refuse although they do not write: they exist only to
launch the next delegation. Letting them run means a delegate is spawned, does
the work, and discovers at `append` that it cannot record it — work done and
unrecordable, which is strictly worse than refusing up front. I considered the
simpler rule "refuse only the commands that write" and rejected it for that
reason; the cost is one extra name in one map.

Why the read-only commands do not refuse: a project pauses Baton precisely
because it wants to stop working through it, not because it wants to stop
checking that its records survived. `lint` in particular must keep working, or a
project cannot verify before resuming that nothing was damaged during the pause.
`update` dry run stays available because it is where the user learns the
resume/update/pause detour (Decision 4).

Implementation: one map of command names and one check at the top of `App.Run`'s
dispatch, before the command runs. `update` is not in the map; it checks inside
`runUpdate` after parsing, so the dry run stays allowed. `runPause` and
`runResume` call `a.appendRecord` directly, which the dispatch gate never sees.

The refusal message names the state and the exit, once:
`baton is paused in this project; run "baton resume" first (records under .baton/ are unchanged)`.

## Decision 4: `pause` and `resume` are commands, not a documented edit

`baton pause [--force]` and `baton resume`. Three reasons, in order of weight:

1. **Resume must restore the active block byte-for-byte**, and after this round
   `lint` says so (Decision 6). The only copy of that text in an installed
   project is the block the pause is about to overwrite; a human resuming from
   a written procedure would have to retype it. The binary can hold it
   (Decision 5) and write it exactly.
2. **The blocks must stay byte-identical across two files.** Editing two files
   identically by hand is exactly the operation that produces the "blocks
   differ" lint error, and pause is the moment a project is least likely to be
   watching.
3. **A documented edit cannot refuse anything.** Refusing while a task is open
   (Decision 5) and refusing an `update` that would un-pause (Decision 4a) are
   the two behaviours that make the pause trustworthy, and prose cannot perform
   either.

`pause` refuses, and writes nothing, when:

- an instruction file's block matches neither the active block this binary ships
  nor the paused block — the block has drifted, and pausing over drift would
  make the later resume lossy;
- `AGENTS.md` and `CLAUDE.md` are both present and their blocks differ;
- no instruction file carries a block at all (nothing to pause);
- any task is open and `--force` was not given (Decision 5).

`resume` refuses, and writes nothing, when a present block is not the paused
block — that is either "not paused" or "edited while paused", and in the second
case splicing over the edit would destroy it silently.

Running `pause` in an already-paused project, or `resume` in an active one,
prints the current state and exits 0. Nothing is ambiguous and nothing is
written, so a non-zero exit would only break the scripts and agents that check
the state by running the command.

### Decision 4a: the update trap

`runUpdate` refuses before `preflightUpdate` when the project is paused, with:

```
baton is paused in this project; update would restore the active rules block.
Run "baton resume", update, then "baton pause" again.
```

The same check goes in `runMergeAgentBlock`, **not** in `MergeAgentBlock`
itself. `HOW-TO-UPDATE.md` step 7 tells a manual updater to run
`merge-agent-block` directly, so leaving that path open would reopen the trap
one document away from where it was closed. Keeping the check in the command
wrapper leaves the library function usable by `runUpdate`, which has already
done its own check by then, and by `runResume`.

`update` without `--apply` still runs and prints the same two-line recipe, so a
paused project finds out before it tries.

Why not "update, then re-apply the pause automatically": it means a paused
project runs a state-mutating command that also appends two timeline records
while paused, contradicting Decision 3, and it adds a partial-failure state
(managed files updated, pause not restored) with no obvious recovery. Refusing
keeps exactly one code path that writes the block and keeps a paused project
able to take any update it wants, at the cost of two extra commands.

## Decision 5: open work blocks the pause by default

`pause` refuses when the timeline leaves any task open, listing each open task
and its last event the way `status --open` does, and telling the user to close
them or re-run with `--force`. An open Standard task with no Director is the
state that rots: the next session sees a `next_gate` with no context and no
record of why nothing happened.

`--force` proceeds and records the open tasks in the `RUN_DONE` summary, e.g.
`Baton paused with 1 open task: srkz`. So the answer across Director's three
options is: refuse by default, record when forced, never warn-and-continue.

`resume` does nothing to an open task. It prints the open tasks with their last
events and points at `baton status`. `.baton/runs/` and every existing timeline
line are untouched by both commands, so the task simply continues from its last
event under the existing transition table — no new transition, no recovery path,
no code.

Detection reuses the `closed[taskID]` logic already in `reportOpenTasks`
(`events.go:637-660`); Executor should extract it rather than write a second
copy.

## Decision 6: what `lint` reports, and both gaps close

`checkAgentBlocks` becomes a three-way classification over the blocks actually
present:

| Block content | lint output |
|---|---|
| matches the active block this binary ships | `OK: AGENTS.md and CLAUDE.md Baton blocks match (active)` |
| matches the paused block | `OK: Baton is paused; records under .baton/ are unchanged by a pause` |
| anything else | `ERROR: Baton block matches neither the active nor the paused block this binary ships; run an update or restore it` |
| no block in a present file | `ERROR: <file> missing <baton-rules> block` (unchanged) |
| present files' blocks differ | `ERROR: AGENTS.md and CLAUDE.md Baton blocks differ` (unchanged) |

Every other check runs unchanged in a paused project, and a pause on its own
never makes `lint` fail. That is the point: the check a paused project most
wants is "are my records still intact", and it must keep answering.

**Both existing gaps have to close, and both close as a by-product.**

- *The missing-file gap* (Finding 3) must close, or the single source of truth
  this design depends on goes unchecked in the configuration `BOOTSTRAP.md:60`
  explicitly invites. Fix: check whichever instruction files exist, require
  equality only when both exist, and error only when neither exists. A
  Claude-only paused project is then fully verified.
- *The identically-emptied-block gap* (Finding 3) must close, because "emptied
  in both files" is a plausible hand-rolled pause, and a project that paused
  that way would pass `lint`, keep an agent-visible nothing instead of an
  agent-visible pause, and be un-paused silently by the next `update`. The
  three-way table closes it: an emptied block matches neither shipped text.

Closing the second gap is also what makes `resume` provably lossless: since
`pause` refuses any block that is not the shipped active block, the block
`resume` writes back is the block that was there.

**R2, stated plainly**: this makes `lint` stricter for every project, not only
paused ones. A project that hand-edited its block inside the markers passes
today and fails after this round. I judge that a true positive — it is the same
posture `checkManagedDocuments` already takes for the five managed documents
(`lint.go:134-152`), and `PROTOCOL.md:34-35` already requires the blocks to stay
identical — but it is a behaviour change beyond the pause feature and Director
should see it named rather than discover it in a user report.

Note that the previous round (`20260920-1605-rules-one-line-PLAN.md`,
Decision 3) explicitly rejected pinning the block against an embedded copy,
because it would stop a project from disabling the block in one edit. This round
supersedes that on purpose: a project that wants the block disabled now has a
supported way to do it, so pinning costs it nothing and buys the pause its
integrity.

## Decision 7: `pause` and `resume` record themselves in the timeline

Both append the existing Direct-work pair with a fresh `task-id`, exactly as
`runUpdate` already does (`update.go:180-190`):

```
<ts> | <id> | REQUEST  | Director | Pause Baton
<ts> | <id> | RUN_DONE | Director | Baton paused
```

No new event name, no new transition, no format change — all three are
forbidden by the constraint, and the existing `REQUEST -> RUN_DONE` pair already
means "Director did this directly", which is what a pause is.

Order of operations: **write the two blocks first, then append.** If the append
then fails, the project is correctly paused and only the record is missing,
which `lint` and a `git diff` both show. The reverse order would leave a
permanent timeline line claiming a pause that did not happen, and an append-only
record that overstates cannot be corrected.

The two block writes are computed in memory first and written in sequence; if
the second write fails, Executor restores the first from the original bytes, so
the "files disagree" state is not reachable through a partial pause.

**R1, and the one question I want Director to answer before Executor starts.**
The constraint reads "no line inside the timeline is ever touched by pausing". I
read that as "no existing line is modified", which appending does not do. If
Director meant "pausing adds nothing to the timeline either", then `pause` and
`resume` write only the blocks and the timeline says nothing about the pause. I
recommend recording, because `BATON-LOG.txt` is the handoff record and a project
returning after a week should be able to see when it stopped and whether a task
was open at the time; without it, `status` shows a stale `next_gate` with no
explanation. But the reading is Director's, it changes two lines of code and
four lines of the round-trip criterion, and it is cheaper to settle now than
after the RUN.

## Decision 8: the smallest protocol statement

One sentence, added to `PROTOCOL.md` "Git And Updates", in both copies:

> A project can set Baton aside with `<baton> pause` and restore it with
> `<baton> resume`; both change only the instruction-file rules block, never
> `.baton/` records, and a paused project must resume before it can update.

Why a tool cannot carry this: the user asks the agent, not the CLI. An agent in
an *active* project that is asked "can we stop using Baton for a while?" has
read `PROTOCOL.md` and its role file and nothing else. If neither names the
command, the agent will invent a way — hand-edit the block, add `.baton/` to
`.gitignore` (which `lint` then fails), or delete the directory. `baton help`
lists the command, but only someone who already suspects it exists will look.
One sentence in the document Director reads first is the smallest thing that
turns an invented answer into a correct one. The clause about updating is in the
same sentence because that is the only trap a Director can walk into without the
CLI stopping it first — and the CLI does stop it (Decision 4a), so this half is
convenience, not the guard.

One line in `HOW-TO-UPDATE.md`, in both copies, under Steps:

> 0. If the project is paused, run `<baton> resume` first, update, then
>    `<baton> pause` again. `update` and `merge-agent-block` both refuse while
>    paused.

That document is what a manual updater follows, and its step 7 sends them
straight into the refusal. One line turns a mid-procedure error into a step.

Nothing else goes in a protocol document. The paused block itself carries every
instruction a paused project's agent needs, and it is not a protocol document.

## Decision 9: the rest of the sweep

Searched every `.md` and `.go` for `baton-rules`, `merge-agent-block`, and the
managed-document list, and read every hit outside `.baton/runs/`.

| Location | Action |
|---|---|
| `README.md` "설치와 업데이트" | **Add** a short "일시 중지와 재개" section: the two commands, what they change (the block only), and that records are preserved. This is the documentation gap the task names; README is user-facing, not a protocol document, so it does not conflict with the standing instruction. |
| `BOOTSTRAP.md` | None. It covers install and update; pausing an installed project is neither, and round two will revisit this file anyway. |
| `PROTOCOL-GUIDE.md` (Korean) | **None, and this leaves it incomplete** (R3). It restates the protocol at length; adding pause there is a translation task with no code consequence, and the previous round left it untouched on the same reasoning. Flagged for Director as a possible follow-up, not planned. |
| `internal/baton/commands_test.go:304-320`, `coverage_more_test.go:92-95` | These use synthetic block contents (`old`, `new`, `same`). The new `lint` classification will fail them. Executor updates them to use the real shipped blocks; that is a required part of step 8, not an incidental edit. |
| `.baton/CONCURRENCY.md` handling (`concurrency.go`) | None. Read-only, and only reached from `gate`, which refuses while paused. |

## Plan

1. `docs.go`: add `bootstrap/AGENTS.md` to the embed and an `ActiveBlock()
   ([]byte, error)` that returns the `<baton-rules>` block from it. Fail loudly
   if the file holds no block.
2. `merge.go`: extract `replaceBlock(target string, block []byte) error` from
   `MergeAgentBlock` — the splice-or-append half, unchanged in behaviour — and
   have `MergeAgentBlock` call it. No behaviour change in this step.
3. `internal/baton/pause.go` (new): the paused-block constant from Decision 2;
   `blockState()` returning `active | paused | absent | unknown` plus the block
   bytes, over whichever of `AGENTS.md`/`CLAUDE.md` exist, erroring when two
   present blocks differ; `runPause(args)` and `runResume(args)` per Decisions
   4, 5, and 7.
4. `app.go`: dispatch `pause` and `resume`; add the refusal gate (Decision 3) as
   one map plus one check before the switch; add both commands to `usage()`
   under "Configure and maintain".
5. `update.go`: refuse `--apply` while paused with the Decision 4a message, and
   print the same recipe from the dry run.
6. `merge.go`: refuse `runMergeAgentBlock` while paused, with the same message.
7. `lint.go`: rewrite `checkAgentBlocks` as the Decision 6 table; leave every
   other check untouched.
8. Tests, per the list in Success Criteria; update the two existing tests named
   in Decision 9.
9. Documents: the sentence in both `PROTOCOL.md` copies, the line in both
   `HOW-TO-UPDATE.md` copies (identically — `checkManagedDocuments` compares
   them), the README section.
10. Run every Required Check and record V1-V4 output in the RUN.

## Success Criteria

- **Round trip**: in a fixture project, `pause` then `resume` leaves every file
  byte-identical to its pre-pause content, except `BATON-LOG.txt`, which has
  exactly four new lines appended and no existing line changed.
- The paused blocks written to `AGENTS.md` and `CLAUDE.md` are byte-identical to
  each other and to Decision 2 exactly.
- `append` during a pause exits non-zero and `BATON-LOG.txt` is byte-identical
  before and after. Same for `new-round`, `feedback`, `gate`, `prompt`, `await`,
  and `merge-agent-block`.
- `update --apply` during a pause exits non-zero, and both instruction files and
  `.baton/VERSION` are byte-identical before and after. `update` without
  `--apply` exits 0 and its output contains `baton resume`.
- `status`, `lint`, `version`, `guide protocol`, and `check-artifact` all exit 0
  during a pause, and `lint`'s output contains the paused OK line.
- `pause` with an open task exits non-zero, names the task, and writes nothing;
  with `--force` it exits 0 and the `RUN_DONE` summary names the task.
- `pause` over a block that is neither shipped text exits non-zero and writes
  nothing. `resume` over a block that is not the paused block does the same.
- `lint` in a project with only `CLAUDE.md` still checks that file's block (gap
  A closed). `lint` in a project whose two blocks are identically emptied now
  errors (gap B closed).
- No file under `.baton/runs/` is created, modified, or deleted by any pause or
  resume test.
- `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...` pass and
  `gofmt -l .` is empty.
- `bootstrap/AGENTS.md`, `bootstrap/CLAUDE.md`, `AGENTS.md`, `CLAUDE.md` still
  carry four byte-identical active blocks after the round (this repository is
  not paused).

## Validation

- V1 round trip, the single strongest check:
  `cp -R` a fixture project to a scratch copy, `shasum` every file, run
  `go run ./cmd/baton pause` then `go run ./cmd/baton resume`, `shasum` again:
  every hash identical except `BATON-LOG.txt`, and
  `diff <(head -n -4 new) old` is empty.
- V2 `for f in AGENTS.md CLAUDE.md bootstrap/AGENTS.md bootstrap/CLAUDE.md; do sed -n '/<baton-rules>/,/<\/baton-rules>/p' "$f" | shasum; done`:
  four identical hashes, unchanged from before the round.
- V3 refusal sweep, on a paused scratch copy: run each of the eight refusing
  commands, record exit codes (all non-zero) and `shasum .baton/BATON-LOG.txt`
  before and after the whole sweep (identical).
- V4 `go run ./cmd/baton lint` on a paused scratch copy: exits 0 and prints the
  paused line; then on the same copy with one block hand-edited: exits non-zero
  with the "matches neither" error.
- `go test ./...`, `go vet ./...`, `gofmt -l .`, `git diff --check`.
- Use `go run ./cmd/baton` throughout; `.baton/bin/baton` lags the working tree
  and will report managed-document drift until Director rebuilds at close.

## Report To Director

- Suggested summary: Add `baton pause`/`resume` with the rules block as the paused state
- Artifact path: `.baton/runs/20260920-1618-pause-baton-PLAN.md`

## Risks Or Questions

- **R1, question for Director, wanted before Executor starts** (Decision 7):
  whether `pause`/`resume` may append `REQUEST -> RUN_DONE` to `BATON-LOG.txt`.
  I read the constraint as forbidding changes to existing lines only, and I
  recommend recording; if Director meant the timeline gains nothing at all, say
  so and the round-trip criterion becomes "byte-identical, no exception".
- **R2** (Decision 6): `lint` becomes stricter for every project. A block
  hand-edited inside the markers passes today and errors after this round. I
  believe that is correct and it is what makes `resume` lossless, but it is a
  behaviour change wider than the feature.
- **R3** (Decision 9): `PROTOCOL-GUIDE.md` will not describe pause. Korean
  readers of that guide will not find it. Deliberate, to keep the round's
  document edits to two sentences; a candidate for its own small round.
- R4: `resume` writes the block the *binary* ships. If a project's binary is
  older than its documents, `resume` restores that binary's block, not
  necessarily the bytes that were there. `pause` refusing any non-shipped block
  closes this for every project whose `lint` passes; a project whose `lint`
  already fails cannot pause at all, which I consider correct.
- R5, for Director at close: `VERSION` and `.baton/VERSION` are untouched by
  this plan, and the installed binary will report managed-document drift until
  it is rebuilt. Both are Director's at close, as in the previous two rounds.
