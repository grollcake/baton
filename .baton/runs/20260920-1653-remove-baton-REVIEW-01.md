# REVIEW-01: Let a project remove Baton and keep its records

Task ID: wbqt
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1653-remove-baton-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

Director narrowed this review to three reproductions plus one judgement on
`artifact.go`. Everything else in the round was read from the diff and RUN-01
and recorded as a nit rather than re-driven. `remove` was never run against
this repository, in any form. Every fixture below is a fresh directory under
the session scratchpad, built by copying `bootstrap/` into it; the binary under
test was built with `go build ./cmd/baton` into the scratchpad, never
`.baton/bin/baton`.

## Reproduction 1: the purge guard

### 1a. `.baton/` holds an uncommitted change and an untracked file

Before (fixture committed, then dirtied):

```
$ git status --porcelain --untracked-files=all -- .baton
 M .baton/GUIDANCE.md
?? .baton/runs/scratch.md
```

Run and after:

```
$ baton remove --apply --purge
not committed: baton/GUIDANCE.md
not committed: .baton/runs/scratch.md
cannot --purge: 2 path(s) under .baton/ are not committed and clean
exit=1

$ diff before.sha256 after.sha256      # sha256 of all 24 files in the tree
(no output)                            # TREE BYTE-IDENTICAL
$ test -e .baton && echo YES
YES
```

The guard refuses and deletes nothing, and it refuses before the instruction
files are spliced, so `AGENTS.md`/`CLAUDE.md` are untouched as well. But the
first path it names is wrong: `baton/GUIDANCE.md`, not `.baton/GUIDANCE.md`.
See nit N1.

### 1b. fully committed `.baton/` is allowed through

Before: `.baton/` with 21 files, `AGENTS.md`, `CLAUDE.md`, all committed clean.

```
$ baton remove --apply --purge
Baton 0.32.0 purged; .baton/ deleted in full (kept at commit 4286cc7b7e5d...)
exit=0

$ ls -A .
.git
$ git show HEAD:.baton/BATON-LOG.txt
2026-01-01T09:00:00 | aaaa | REQUEST  | Director | Bootstrap Baton
2026-01-01T09:00:01 | aaaa | RUN_DONE | Director | Baton initialized
```

The guard is a guard, not a blanket refusal: the working tree loses `.baton/`
and both instruction files, and `HEAD` still holds the record.

### 1c. not a Git repository

```
before: .baton exists? YES
$ baton remove --apply --purge
cannot --purge: this project is not a Git repository
exit=1
after: .baton exists? YES; AGENTS.md exists? YES
```

Refuses, as Decision 2 states. Nothing is deleted, including the instruction
files.

### 1d. `.baton/` ignored by Git

Two shapes, and they behave differently.

`.gitignore` containing `.baton/` (the whole directory ignored):

```
$ git ls-files -- .baton
(empty)
$ baton remove --apply --purge
cannot --purge: no path under .baton/ is tracked by Git
exit=1
after: .baton exists? YES; AGENTS.md exists? YES
```

`.gitignore` containing `/.baton/bin/` (this repository's own shape), with an
uncommitted change inside the ignored subtree:

```
$ git status --porcelain --untracked-files=all -- .baton
(empty)
$ baton remove --apply --purge
Baton 0.32.0 purged; .baton/ deleted in full (kept at commit 2c3c140c65a4...)
exit=0
after: .baton exists? NO
```

So an ignored `.baton/` can never purge, and an ignored *subtree* of a tracked
`.baton/` is deleted with the rest without being named, dirty or not. That is
exactly Decision 2's declared boundary ("a file Git ignores was declared by the
project itself not to be part of its history"), reproduced rather than assumed.
It is worth the user seeing it stated, so it is nit N2.

## Reproduction 2: a hand-edited block refuses

A sentence of a person's own was placed inside the markers of `AGENTS.md`;
`CLAUDE.md` was left as shipped.

Before, `tail -8 AGENTS.md`:

```
read next. Until you have read it, do not run a `baton` command and do not
write under `.baton/`.


Our team also reviews every PR twice.
</baton-rules>
```

Three runs:

```
$ baton remove --apply
AGENTS.md: Baton block matches neither the active nor the paused block this
binary ships; move your text outside the <baton-rules> markers and re-run
exit=1

$ baton remove --apply --force
(same message)                                        exit=1

$ baton remove
(same message)                                        exit=1
```

After, `tail -8 AGENTS.md`: identical to the block quoted above, and

```
$ diff before.sha256 after.sha256     # every file in the tree
(no output)                           # WHOLE TREE BYTE-IDENTICAL
```

`--force` does not get past it; neither does the dry run, which reports the
refusal instead of a plan. `CLAUDE.md`, whose block was untouched and
removable, is also left alone — the refusal is raised over the whole operation
before any instruction file is written, which is what keeps the two files from
disagreeing.

## Reproduction 3: install then remove, byte-exact

Each case: write the original, `baton merge-agent-block AGENTS.md
bootstrap/AGENTS.md`, `baton remove --apply`, compare bytes.

### 3a. project text above the block, file ends in a newline

```
original (od -c tail):  o   w   n   .  \n          36 bytes
after remove:           o   w   n   .  \n          36 bytes
$ cmp original after  ->  BYTE-IDENTICAL
```

### 3b. the file is nothing but the block

The original was an empty `AGENTS.md` (0 bytes); after the merge it held the
block and nothing else.

```
$ baton remove --apply
Baton 0.32.0 removed; records kept under .baton/          exit=0
after: (AGENTS.md DELETED)
```

Not byte-exact in the strict sense: the round trip ends with no file where the
project had a zero-byte one. This is Decision 3's stated "the file is deleted"
branch and is the right behaviour for the common case (Baton created the file),
but the plan's Success Criterion says "install round trip", and a pre-existing
empty file is the one shape where install-then-remove does not restore the
starting state. Nit N3.

### 3c. no final newline before the block

```
original (od -c tail):  o   w   n   .                36 bytes
after remove:           o   w   n   .  \n            37 bytes
$ cmp original after
cmp: EOF on original after byte 36, in line 3         DIFFERS by exactly one \n
```

What actually happens matches R3's claim precisely: one newline is gained, no
other byte changes, and the added byte is at the end of the file. The round
does not overclaim here.

## Judged from the code, not re-driven

### Director's fourth item: the `removed bool` on `artifact.go`

Judged from the code, then confirmed with one fixture run because the code left
one question open (whether a removed project's own `check-artifact` relaxes).

The relaxed branch is reached only when **both** conditions hold:
`templatePlaceholderTokens(removed=true)` **and** `.baton/templates/` does not
exist (`artifact.go:37-41`). Everything else in that function is unchanged, so
wherever `removed` is false the function is byte-for-byte its former self,
including the "empty directory" and "no tokens" errors, which the code reaches
regardless of `removed`.

`removed` has exactly two sources. The public `CheckArtifact`
(`artifact.go:117-119`) hardcodes `false`, and it is the entry point for
`check-artifact`, `append` (`events.go:249`), `await` (`await.go:57`) and
`events.go:736`. The other is `lintState.removed`, set once in `Lint()`
(`lint.go:380`) as `statusErr == nil && status == blockAbsent`, and threaded to
one call site (`lint.go:249`).

Can a project that is not removed reach it? `blockAbsent` requires that no
present instruction file carries a block. A partially removed project cannot:
removal writes the instruction files **first** (`remove.go`, Decision 4 order),
so any state in which a block still exists gives `removed == false`, and any
state in which the blocks are gone is a removal past its first step. The one
non-removed project that qualifies is one that never had instruction files at
all and also has no `templates/` — a broken or half-bootstrapped tree, which
`Lint()`'s removed mode already treats as removed by Decision 6's widened
`blockAbsent`. That is a lint-reporting question, not a gate question.

Reproduced, in a removed fixture holding an artifact with an unfilled
placeholder:

```
before removal:  $ baton check-artifact PLANNED .../demo-PLAN.md bbbb
  artifact-check: PLANNED artifact contains an unresolved placeholder ... exit=1
after removal:   $ baton check-artifact PLANNED .../demo-PLAN.md bbbb
  artifact-check: no template files found under .../templates           exit=1
```

The fail-closed behaviour the earlier round established holds everywhere
`removed` is false, including inside a removed project when the user reaches
the check through the public command. Not a blocker. What the flag does cost is
recorded as nit N4.

### The rest of the narrowed group

- **`lint`'s removed mode** (`lint.go:378-431`): the skipped checks are exactly
  the ones whose subject `remove` deletes, and every record check survives.
  Reproduced as a by-product above: a removed fixture with a deleted `runs/`
  artifact still fails, naming the path.
- **The fourth classifier state** (`pause.go:74-127`): per-file
  classification, `blockAbsent` for a present blockless file, and an error
  naming which file lacks a block when two present files disagree. Reproduced
  as a by-product: after removal, `resume` and `pause` both exit non-zero with
  the Decision 9 messages, and `update --apply` refuses (via the `VERSION`
  check in the default removal, via `removedUpdateMessage` when `VERSION`
  survives). The two instruction files cannot be left disagreeing by this
  round's code: `buildInstructionPlan` plans `unchanged` for a blockless file
  and splices the other, so removal *heals* that state rather than creating it.
- **Removal of a paused project**: reproduced as a by-product. `pause`, then
  `remove --apply`, exits 0 without resuming first; the paused `AGENTS.md` is
  normalised to the active block before the whole-file comparison, so it is
  deleted exactly as an active one is; `resume` afterwards exits non-zero.
- **The protocol sentence**: `diff .baton/PROTOCOL.md
  bootstrap/.baton/PROTOCOL.md` is empty, so the two copies cannot disagree.
  The sentence names `--apply`, says records are kept unless `--purge`, and
  says it cannot be undone — the three things an agent reading only this needs.

Nothing in this group looks like it could lose a record or leave the two
instruction files disagreeing, so nothing here is escalated to a blocker.

## Findings

### Blockers

- none

### Nits

- **N1 — the refusal names a path that does not exist.** `runGit`
  (`app.go:208`) returns `strings.TrimSpace(output)`, which strips the leading
  status-column space from the **first** line of `git status --porcelain`.
  `purgePrecondition` (`remove.go`) then takes `line[3:]`, so the first
  offending path loses its first character whenever its status code is an
  unstaged one (` M`, ` D`): reproduced above as `not committed:
  baton/GUIDANCE.md` for `.baton/GUIDANCE.md`, while the second line, `??
  .baton/runs/scratch.md`, printed correctly. The guard still refuses
  correctly; only the message is wrong, and it sends the user looking for a
  path that is not there. A one-line fix (split first, or match on the
  two-column prefix rather than slicing after trimming).
- **N2 — `--purge` deletes ignored content under a tracked `.baton/` without
  naming it.** Reproduced in 1d: with `/.baton/bin/` ignored and dirty,
  `--purge` succeeded and took the ignored subtree with it. This follows from
  Decision 2 and is correct as designed, but the dry run's `--purge` line names
  only the commit, so the one category of bytes Git does not hold is the one
  category the output does not mention.
- **N3 — a pre-existing empty instruction file is not restored by the round
  trip.** Reproduced in 3b. Correct for the case Decision 3 aimed at, and
  vanishingly rare otherwise; naming it so the Success Criterion's
  "byte-identical to the original" is read with its two exceptions (this and
  R3) rather than one.
- **N4 — `lint` in a removed project no longer reports an unfilled
  placeholder.** The cost of the `removed` bypass, confirmed above: the
  placeholder in the fixture's recorded PLAN was flagged before removal and not
  after, while the other artifact checks still ran and still failed. Records
  cannot be lost by this and the gate is unaffected, but a removed project's
  `lint` answers a slightly smaller question about its history than an
  installed one's.
- **N5 — `removedUpdateMessage` is reachable in a never-installed project.**
  `projectBlockState` returns `blockAbsent` for a project with no instruction
  files at all, so `update --apply` there would report "Baton has been removed
  from this project". Unreachable in practice (`update` stops at a missing
  `.baton/VERSION` first, reproduced), and the plan's R6 already names the
  widened `blockAbsent`; recorded so the wording is a known consequence rather
  than a surprise.
- **N6 — the round changed a file outside the plan's named scope.**
  `internal/baton/artifact.go` gained the `removed` parameter. The Executor
  flagged it rather than hiding it, and the judgement above finds it safe, but
  a plan-scope deviation is the Planner's to record.
- **N7 — carried, not introduced.** R4 (`PROTOCOL-GUIDE.md` now lags three
  features) and R5 (`VERSION`/`.baton/VERSION` bump and the `.baton/bin/baton`
  rebuild) remain Director's at close, as in the previous three rounds.

## Suggested User Checks

1. In a throwaway copy of any Baton project, run `baton remove` with no flags.
   Read the whole dry run and confirm the "Keep:" list contains everything you
   would be upset to lose, and that the timeline and `runs/` counts match what
   you expect. Nothing is changed by this.
2. In that same copy, run `baton remove --apply`, then confirm your own
   judgement of the result: `git diff --stat` shows only the instruction files
   and Baton's installed files, and `.baton/runs/` plus `.baton/BATON-LOG.txt`
   are still there with two new lines at the end.
3. Decide whether `--purge`'s Git precondition is the confirmation you want for
   the one command that can delete a record, given that it is checked rather
   than asked, and that with `.baton/bin/` ignored it takes the ignored files
   too (nit N2).
4. Put a sentence of your own inside the `<baton-rules>` markers of a copy's
   `AGENTS.md` and run `baton remove --apply --force`. Confirm the refusal is
   the behaviour you want from `--force`, since no flag overrides it.
5. Read the one new sentence in `.baton/PROTOCOL.md` under "Git And Updates"
   and judge whether an agent told "we're done with Baton" would act on it
   correctly with nothing else to go on.

## Evidence Reviewed

- `.baton/runs/20260920-1653-remove-baton-PLAN.md`,
  `.baton/runs/20260920-1653-remove-baton-RUN-01.md`
- `internal/baton/remove.go`, `pause.go`, `lint.go`, `artifact.go`,
  `update.go`, `app.go`, `merge.go`, `docs.go`
- `diff .baton/PROTOCOL.md bootstrap/.baton/PROTOCOL.md` (empty)
- Eight scratchpad fixtures built from `bootstrap/`, driven with a binary built
  from the working tree: purge dirty / purge clean / no Git / fully ignored /
  partially ignored, hand-edited block, three round-trip shapes, removed-project
  lint, paused-then-removed.

## Report To Director

- Suggested summary: REVIEW-01 reproduced the purge guard, the hand-edited refusal and the install round trip; no blockers, seven nits
- Artifact path: `.baton/runs/20260920-1653-remove-baton-REVIEW-01.md`

## Required Next Step

- ask Director to request user approval
