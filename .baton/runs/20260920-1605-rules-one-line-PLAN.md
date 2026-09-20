# PLAN: Reduce the agent rules block to a pointer

Task ID: yitg
Date: 2026-09-20
Planner: Claude Opus 5
Status: complete

## Director Brief

- Goal: Replace the 36-line `<baton-rules>` block in all four instruction-file copies with a short pointer to `.baton/PROTOCOL.md`, so the protocol is stated once and the block becomes one editable unit a project can disable.
- Scope: The `<baton-rules>` block in `AGENTS.md`, `CLAUDE.md`, `bootstrap/AGENTS.md`, `bootstrap/CLAUDE.md`. No Go code, no `PROTOCOL.md`, no other document.
- Success Criteria: The four blocks are byte-identical to each other and to the text in Decision 2; `go run ./cmd/baton lint` still reports `AGENTS.md and CLAUDE.md Baton blocks match` and passes; `go test ./...`, `go vet ./...`, `gofmt -l .` unchanged; no file outside those four contains a sentence that only the old block supplied.
- Risks: R1, the accepted all-or-nothing risk, is Director's and is recorded below, not mitigated by code. R2 is a pre-existing `update` behaviour that this change makes louder: a project that put its own instructions *inside* the block loses 34 lines on the next update instead of nothing noticeable. Neither is an Executor blocker.
- Required Checks: `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...`, `gofmt -l .`, plus the four-way byte-identity diff in Validation.
- Executor Prompt: Implement `.baton/runs/20260920-1605-rules-one-line-PLAN.md` steps 1-4. Replace the `<baton-rules>` block in `bootstrap/AGENTS.md` with the exact text in Decision 2, then make `bootstrap/CLAUDE.md`, `AGENTS.md`, and `CLAUDE.md` carry that block byte-for-byte. Change nothing else in those files and no other file at all: no Go code, no `.baton/PROTOCOL.md`, no `BOOTSTRAP.md`, no `PROTOCOL-GUIDE.md`. Run every check under Required Checks with `go run ./cmd/baton`, not the installed binary. Record in the RUN the byte-identity evidence and the evidence that the non-block content of each of the four files is unchanged.

## Goal

- `.baton/PROTOCOL.md` is the only place the protocol's rules are written. The instruction-file block states no rule of its own.
- The block still does the one job nothing else can do: an agent whose tool auto-reads `AGENTS.md` or `CLAUDE.md` learns that `.baton/PROTOCOL.md` exists and must be read.
- Adding or changing a protocol rule is one edit to `PROTOCOL.md`, not three.
- The block is small enough that removing or commenting it out is a single deliberate edit.

## Scope

In scope:

- The `<baton-rules>...</baton-rules>` block in `AGENTS.md`, `CLAUDE.md`, `bootstrap/AGENTS.md`, `bootstrap/CLAUDE.md`.

Out of scope:

- `.baton/PROTOCOL.md` and the `bootstrap/.baton/` copy (constraint; see Finding 1).
- `internal/baton/lint.go` and `internal/baton/merge.go` (see Decisions 4 and 5).
- `BOOTSTRAP.md`, `PROTOCOL-GUIDE.md`, `README.md`, `.baton/HOW-TO-UPDATE.md` (see Decision 6).
- The log format, event names, transition table, and artifact contracts.

## Decision 1: the chain was verified, and it closes

Director asked for this to be checked rather than assumed. I read the current
`.baton/PROTOCOL.md` line by line against the eight rules and the three
role-classification bullets the block carries today.

The chain an agent must walk is: instruction file -> pointer -> `PROTOCOL.md`
tells it which role it holds -> `PROTOCOL.md` names the role file to read.

- Role classification: `PROTOCOL.md` "Know Your Role" (lines 17-27) states all
  three cases, in the same order and with the same outcomes as the block's
  three bullets, including the resumed-without-a-prompt case and the
  Director-owned command list.
- Role file mapping: stated in the document intro, lines 3-4 ("Role-specific
  operational details live in `DIRECTOR.md`, `PLANNER.md`, and `EXECUTOR.md`;
  read only the file for your role"), and again in "Read Before Work" (lines
  31-35, "your role-specific file"). "Know Your Role" itself names roles, not
  files, so the mapping is two sentences apart from the classification rather
  than in one place. Director -> `DIRECTOR.md` needs no inference, so I call
  this a readability nit in `PROTOCOL.md`, not a break in the chain. Per the
  constraint I am reporting it rather than planning an edit.

Rule-by-rule, where each block rule already lives in `PROTOCOL.md`:

| Block rule | Stated in `PROTOCOL.md` |
|---|---|
| Role bullets (3) | lines 17-27, plus 3-4 for the file mapping |
| 1. Reread guidance and matching lessons per phase | lines 37-47 |
| 2. Session-start model selection with `models get/list/set` | lines 58-63 (all three commands named) |
| 3. Branch strategy question | line 63 |
| 4. Route through Director, delegate in the background | line 15, lines 54-56 |
| 5. REVIEW is evidence; approval closes; 3-5 manual checks | lines 99-104, line 116 |
| 6. Defect evidence before fix, smoke test after | lines 86-87 |
| 7. Append-only rounds, approval for guidance, no secrets | line 75, lines 104, 134-136 |
| 8. Native `.baton/bin/baton[.exe]`, `git` on `PATH` | lines 144-147 |

**Nothing the block carries today is absent downstream except the existence and
path of `.baton/PROTOCOL.md` itself.** That single fact is what the pointer must
carry. Director's premise held for all six sampled rules and for the other two.

## Decision 2: the pointer text, and why it is three lines and not one

Director's "one line" is intent. Measured against the chain, the pointer must
carry three things, and only the first two are strictly the chain:

1. That this project follows Baton and `.baton/PROTOCOL.md` exists.
2. That the agent must read it, and what reading it will settle (which role it
   holds, which role file to open next). Without the second clause a hurried
   agent has no reason to think the file is about itself.
3. A fail-safe: do nothing Baton-shaped until it has been read.

The exact block, to be byte-identical in all four files:

```
<baton-rules>

## Baton

This project follows Baton. Before doing any work here, read
`.baton/PROTOCOL.md`: it decides which role you hold and which role file to
read next. Until you have read it, do not run a `baton` command and do not
write under `.baton/`.

</baton-rules>
```

Seven lines including the markers and blanks, four of them prose, against 36
today. One paragraph, no list, so there is nothing to append a ninth rule to
without noticing that the block is not a rule list.

Why each piece:

- "Before doing any work here", not "before Baton work". An agent cannot judge
  whether its task is Baton work until it has read the classification rules,
  which are inside the file the pointer points to. Conditioning the read on a
  judgement that requires the read is the one way this pointer could fail
  silently, so the read is unconditional.
- The `## Baton` heading stays. It costs one line, it matches every existing
  copy, and it gives the merge conflict in a hand-edited file something to
  align on.
- The third sentence duplicates one clause of `PROTOCOL.md` line 26-27, and I
  am flagging that honestly rather than hiding it. It is there as a blast
  radius limit, not as a rule: see Decision 3. If Director judges any
  duplication unacceptable, drop that sentence and the block is five lines;
  the chain still closes, and R1 gets correspondingly worse.

## Decision 3: the accepted cost, stated plainly (R1)

Today an agent that reads `CLAUDE.md` and never opens `PROTOCOL.md` still
obeys eight real rules: it routes through Director, it treats REVIEW as
evidence, it does not overwrite rounds, it keeps secrets out of `.baton/`.
After this change that agent obeys nothing. The whole protocol now rests on one
pointer being followed, and a pointer that is skipped produces an agent that
works normally and records nothing, which is quieter than a violation.

Director has accepted this. What can reduce it without restoring duplication:

- **The fail-safe sentence (in the proposed text).** It does not make a
  skipping agent follow the protocol. It makes a skipping agent *harmless*:
  no `baton` command, no write under `.baton/`, so the timeline and the round
  artifacts stay consistent and Director sees the gap instead of a corrupted
  record. This is the only mitigation I recommend, and it costs one sentence.
- **`lint` already detects a removed block** in a project that has both files
  (Decision 4). It cannot detect a followed-but-ignored pointer; nothing
  textual can.
- **Rejected: keeping two or three "most important" rules inline.** That is
  the duplication this task exists to remove, and the selection would drift
  first. Rejected explicitly so it is not rediscovered as an improvement.
- **Rejected: making the block a managed document compared byte-for-byte
  against a copy embedded in the binary** (the `checkManagedDocuments`
  treatment). It would make an edited pointer detectable, but it directly
  contradicts the goal that a project can disable or remove the block in one
  edit: such a project would then fail `lint` forever. See Decision 4.

## Decision 4: `lint` needs no change

`checkAgentBlocks` (`internal/baton/lint.go:62-83`) does exactly three things:
require the block in `AGENTS.md`, require it in `CLAUDE.md`, and require the
two to be byte-equal. All three remain the right checks, because the
constraint this round must hold is unchanged: the two blocks stay identical.
The check is content-agnostic, so a 7-line block satisfies it exactly as a
36-line one does. **No code change, no test change.**

On "should a pointer that has been edited away be detectable": partially, and
that is the correct amount.

- Deleting the block from both files, in a project that has both, already
  fails `lint` with `AGENTS.md missing <baton-rules> block`.
- Deleting it from one file already fails with `blocks differ`.
- Rewording it in both files identically is undetectable, today and after this
  change. Making it detectable means pinning the text, which means the block
  can never be legitimately customised or removed, which is the opposite of
  this round's goal.

Pre-existing hole, reported not fixed: `checkAgentBlocks` returns early when
either file is absent (`lint.go:69-71`), so a project that keeps only
`CLAUDE.md` gets no block check at all. `BOOTSTRAP.md:60` explicitly tells
Claude users they may delete `AGENTS.md`, so this is a reachable state, and
shrinking the block to a pointer raises what it costs to lose it unnoticed. It
is out of scope here; Director may want a separate round.

## Decision 5: an updating project gets the new block automatically

Checked against the code rather than assumed. `runUpdate`
(`internal/baton/update.go:132-145`) calls `MergeAgentBlock` for `AGENTS.md`
and, when present, `CLAUDE.md`. `MergeAgentBlock` (`internal/baton/merge.go:28-44`)
finds the target's existing block with the same regexp and **replaces it in
place**, splicing the upstream block between the surrounding bytes; only when
no block is found does it append. So a project carrying the 36-line block ends
the update carrying the pointer, with its own non-Baton content untouched.
`update --apply` already warns that it replaces the `AGENTS.md`/`CLAUDE.md`
Baton block. **No migration for Director to plan, and no code change.**

The one real consequence, R2: a project that edited its own instructions
*inside* the `<baton-rules>` markers has always had them replaced on update;
now 34 lines vanish instead of a few words, which is more likely to be noticed
and more likely to be reported as a defect. `BOOTSTRAP.md:183` tells a human
bootstrapper to stop and report such a file as a conflict, but the binary does
not implement that check. Pre-existing, out of scope, recorded so the first
report is recognised rather than debugged.

## Decision 6: the sweep found nothing else to change

I searched every `.md` and `.go` file for `baton-rules` and for references to
the block, and read every hit outside `.baton/runs/`.

| Location | What it says | Action |
|---|---|---|
| `BOOTSTRAP.md:44` | the block's source is `bootstrap/AGENTS.md`, and `BOOTSTRAP.md` deliberately does not duplicate its text | none; already correct |
| `BOOTSTRAP.md:45,59,61,148,157,181-183` | merge behaviour, described generically as "the block" | none; all still true |
| `PROTOCOL-GUIDE.md:280,285,371,386-387,397,405,462` | the two blocks stay identical, and an agent that meets the Baton notice reads `PROTOCOL.md` and its role file | none; line 280 already describes the pointer behaviour this change makes literal |
| `.baton/PROTOCOL.md:34-35` | the two blocks must stay identical | none; still true, and out of scope by constraint |
| `.baton/HOW-TO-UPDATE.md:14` | use `merge-agent-block` for both files | none |
| `internal/baton/commands_test.go:304-320`, `coverage_more_test.go:92-95` | use synthetic block contents (`old`, `new`, `same`) | none; content-agnostic |

**No document anywhere quotes or restates a rule from the block, and none cites
its length or rule count.** The four block copies are the entire change.

Noted but deliberately not touched: `BOOTSTRAP.md:45` and `:182` distinguish "a
Baton block" from "only a Baton pointer", meaning a hand-written notice with no
markers. After this round the word *pointer* describes both, which reads
oddly. The instruction still executes correctly, since the distinction it turns
on is the presence of the `<baton-rules>` markers, so under the surgical-change
rule I am reporting the wording rather than planning an edit to it.

## Plan

1. Replace the `<baton-rules>` block in `bootstrap/AGENTS.md` with the text in
   Decision 2, leaving the rest of the file byte-identical.
2. Apply the identical replacement to `bootstrap/CLAUDE.md`, `AGENTS.md`, and
   `CLAUDE.md`. Copy the block, do not retype it.
3. Verify the four blocks are byte-identical to each other and that the
   non-block content of each file is unchanged (`git diff` shows only the block
   hunk in each of the four files, and no fifth file).
4. Run the Required Checks and record the output.

## Success Criteria

- The block extracted from each of `AGENTS.md`, `CLAUDE.md`, `bootstrap/AGENTS.md`,
  `bootstrap/CLAUDE.md` is byte-identical to the other three and matches
  Decision 2 exactly, including the `## Baton` heading and the blank lines.
- `git diff --stat` lists exactly those four files.
- In each of the four files, `git diff` contains exactly one hunk, and it lies
  inside the `<baton-rules>` markers.
- `go run ./cmd/baton lint` prints `AGENTS.md and CLAUDE.md Baton blocks match`
  and exits `baton-lint passed`.
- `go test ./...`, `go vet ./...` pass and `gofmt -l .` prints nothing, with no
  test edited.
- The block contains no imperative that is not "read `.baton/PROTOCOL.md`" or
  the fail-safe: no models command, no branch question, no REVIEW rule, no
  secrets rule, no binary path.

## Validation

- `for f in AGENTS.md CLAUDE.md bootstrap/AGENTS.md bootstrap/CLAUDE.md; do sed -n '/<baton-rules>/,/<\/baton-rules>/p' "$f" | shasum; done`: four identical hashes.
- `git diff --stat`: exactly the four files, nothing else.
- `git diff -- AGENTS.md CLAUDE.md bootstrap/AGENTS.md bootstrap/CLAUDE.md`: every removed line is inside the old block; no line outside the markers is added or removed.
- `go run ./cmd/baton lint`: passes, including the blocks-match line. Use `go run`, not `.baton/bin/baton`, which lags the working tree.
- `go test ./...`: passes with no test file modified (`git diff --name-only` lists no `_test.go`).
- `go vet ./...`: clean. `gofmt -l .`: empty output.
- `git diff --check`: clean.
- Read-through, recorded in the RUN: starting from the new block alone, name the
  file it sends you to, the `PROTOCOL.md` section that settles your role
  ("Know Your Role"), and the line that names your role file (lines 3-4). If any
  of the three cannot be named, stop and report to Director instead of editing
  `PROTOCOL.md`.

## Report To Director

- Suggested summary: Reduce the agent rules block to a pointer to `.baton/PROTOCOL.md`
- Artifact path: `.baton/runs/20260920-1605-rules-one-line-PLAN.md`

## Risks Or Questions

- R1 (accepted by Director, recorded in Decision 3): the protocol becomes
  all-or-nothing on one pointer being followed. An agent that skips the pointer
  obeys no rule at all, where today it would obey eight, and its non-compliance
  looks like ordinary work rather than a violation. The fail-safe sentence
  bounds the damage; nothing textual can force the read.
- R2 (pre-existing, Decision 5): a project that kept its own instructions inside
  the `<baton-rules>` markers loses 34 lines on its next `update` instead of a
  few. The behaviour is unchanged; only its visibility is. `BOOTSTRAP.md:183`
  describes a conflict check the binary does not implement.
- R3 (pre-existing, Decision 4): `lint` skips the block check entirely when
  `AGENTS.md` or `CLAUDE.md` is absent, which `BOOTSTRAP.md:60` invites. A
  Claude-only project can lose the pointer without `lint` noticing. Out of
  scope; a candidate for its own round.
- Question for Director, not blocking: `PROTOCOL.md` "Know Your Role" names the
  three roles but not their three files, which are named two sentences earlier
  in the intro. The chain closes without an edit (Decision 1). If Director wants
  the classification and the file mapping in one place, that is a separate
  one-line decision about `PROTOCOL.md`, which this round is constrained not to
  touch.
- Question for Director, not blocking: whether to keep the fail-safe sentence.
  It is the round's only surviving duplication, one clause of `PROTOCOL.md`
  line 26-27, and it is the only thing standing between a skipped pointer and a
  corrupted timeline (Decision 3).
