# REVIEW-01: Reduce the agent rules block to a pointer

Task ID: yitg
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1605-rules-one-line-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- none.

Every item Director asked to be weighed was reproduced independently, on a copy
of the tree, and each held.

1. **The chain closes.** Starting from the new block text alone: it names
   `.baton/PROTOCOL.md` and says that file settles the role and the next file.
   Opening `PROTOCOL.md`, the reader meets the role-file mapping at lines 3-4
   (`DIRECTOR.md`, `PLANNER.md`, `EXECUTOR.md`; read only your own) *before*
   reaching "Know Your Role" at lines 17-27, which settles which of the three
   cases the reader is in, including the resumed-without-a-prompt case and the
   Director-owned command list. The gap the PLAN reports is real but reads
   awkwardly rather than breaking the chain: the mapping is encountered first
   in reading order, and Director -> `DIRECTOR.md` needs no inference. Not a
   blocker, not even a nit against this round; it is a standing `PROTOCOL.md`
   readability question, and `PROTOCOL.md` is out of scope by constraint.

2. **No rule was lost.** I checked the PLAN's table myself with `grep -n`
   rather than trusting it. All eight numbered rules and all three role bullets
   are stated in `PROTOCOL.md`: role cases 17-27 and file mapping 4; rule 1 at
   37-47; rule 2 at 58-62 (all three of `models get`/`list`/`set` named
   literally); rule 3 at 63; rule 4 at 15 and 54-56; rule 5 at 10-11, 101, 116,
   123; rule 6 at 86-87; rule 7 at 75, 134, 135-136; rule 8 at 144-147.
   One clause of block rule 4, "returns a short status immediately", is not in
   `PROTOCOL.md`; it is in `.baton/DIRECTOR.md:10`, which is on the chain for
   the only role it binds. Nothing the block carried is now stated nowhere.

3. **The four blocks are byte-identical and match Decision 2 exactly.**
   Verified by hash, not by eye. The block extracted from each of `AGENTS.md`,
   `CLAUDE.md`, `bootstrap/AGENTS.md`, `bootstrap/CLAUDE.md` and the fenced text
   machine-extracted from PLAN Decision 2 all hash to
   `30702eb11211c8c2550310e2864258539924ac3e`; `diff` of Decision 2 against the
   live block is empty. The fail-safe sentence Director kept is present.

4. **Nothing outside the block changed.** `git diff main` touches five paths:
   the four instruction files plus `.baton/BATON-LOG.txt`, whose diff is three
   appended `yitg` event lines (REQUEST/PLANNED/EXECUTED) written by Director,
   not by this round's Executor. In each of the four files the diff is exactly
   one hunk and every `+`/`-` line lies strictly between the marker lines; the
   markers and the `## Baton` heading are untouched. The root `AGENTS.md`
   project instructions above the block (`# Repository Instructions`,
   Environment And Editing, lines 1-13) survive intact and appear only as diff
   context.

5. **`lint` needs no change, and its blind spots are exactly the ones the PLAN
   claims.** Reproduced on a copy with `go run ./cmd/baton lint`: baseline
   passes with `OK: AGENTS.md and CLAUDE.md Baton blocks match`. Block deleted
   from `CLAUDE.md` only -> `ERROR: CLAUDE.md missing <baton-rules> block`,
   fails. Deleted from both -> `ERROR: AGENTS.md missing <baton-rules> block`,
   fails. Emptied in one file (markers kept) -> `ERROR: ... blocks differ`,
   fails. Reworded identically in both -> passes, undetected, as Decision 4
   states and as was already true of the 36-line block. `checkAgentBlocks` is
   content-agnostic (`internal/baton/lint.go:62-83`), so a 7-line block
   satisfies it exactly as a 36-line one did.

6. **An installed project on the old block gets the pointer automatically.**
   Verified against the code and against a scratch project, not assumed.
   `MergeAgentBlock` (`internal/baton/merge.go:28-44`) splices the upstream
   block between `targetContent[:start]` and `targetContent[end:]`. I built a
   scratch `AGENTS.md`/`CLAUDE.md` carrying the 36-line block from `main` with
   the project's own content above and below it, ran
   `go run ./cmd/baton merge-agent-block` against the new `bootstrap/` sources,
   and both files came back with the 7-line pointer, hash
   `30702eb11211c8c2550310e2864258539924ac3e`, with the surrounding own content
   byte-for-byte preserved. No migration is needed.

Required Checks reproduced on the copy: `go run ./cmd/baton lint` passed;
`go test ./...` -> `ok .../internal/baton 9.146s`, rest no test files;
`go vet ./...` clean; `gofmt -l .` empty; `git diff --check` clean. No test file
was added or edited this round, so the "is each added test load-bearing" step is
vacuous here and is recorded as such rather than skipped.

### Nits

- On the fail-safe sentence, which Director asked me to judge without making it
  a blocker: **it earns the duplication.** It restates one clause of
  `PROTOCOL.md` lines 26-27, but its job is not to state a rule, it is to bound
  the blast radius of the failure this round creates. The sentence is the only
  text in the whole system that an agent which skips the pointer still reads,
  and it converts that agent from one that silently writes an inconsistent
  timeline into one that does nothing Baton-shaped, which Director can see.
  A rule restated where it can never be read would be waste; this one is
  restated at the exact place where the read fails. Keep it.
- `lint` passes when the markers are kept but the pointer prose between them is
  deleted in both files (reproduced: `OK: blocks match`). Identical to the
  rewording hole and equally pre-existing, but worth naming separately: the
  block is now short enough that "markers present" and "block says something"
  have come far apart. Not actionable this round.
- The PLAN's rule table credits block rule 4 entirely to `PROTOCOL.md` lines 15
  and 54-56. The "returns a short status immediately" clause is actually in
  `.baton/DIRECTOR.md:10`. The rule is not lost, only cited one file off.
- R3 restated as a live consequence, not a new finding: `checkAgentBlocks`
  returns early when either file is absent (`lint.go:69-71`), and
  `BOOTSTRAP.md:60` invites Claude-only projects to delete `AGENTS.md`. Such a
  project can now lose the entire protocol and still pass `lint`. Out of scope;
  a candidate for its own round.

## Suggested User Checks

- Open `AGENTS.md` and `CLAUDE.md` and read only the `<baton-rules>` block.
  Ask yourself, knowing nothing else: would you open `.baton/PROTOCOL.md` before
  starting work? That judgement is the whole bet this round makes, and it is
  yours to make, not mine.
- Read `.baton/PROTOCOL.md` top to bottom once, as a new agent would. Confirm
  you can name your role and the file to open next without going back to the
  instruction file.
- Confirm your own project instructions at the top of `AGENTS.md`
  (Environment And Editing) are untouched, and that `bootstrap/AGENTS.md`,
  `bootstrap/CLAUDE.md`, and `CLAUDE.md` show only the block change.
- Decide whether you want the fail-safe sentence kept. It is the round's only
  surviving duplication; dropping it shortens the block to five lines and
  removes the only guard against a skipped pointer corrupting the timeline.
- Run `.baton/bin/baton lint` (or `go run ./cmd/baton lint`) and confirm
  `AGENTS.md and CLAUDE.md Baton blocks match` and `baton-lint passed`.

## Evidence Reviewed

- `.baton/runs/20260920-1605-rules-one-line-PLAN.md`, Decisions 1-6.
- `.baton/runs/20260920-1605-rules-one-line-RUN-01.md`.
- `git diff main --stat` and per-file `git diff main` for the four instruction
  files and `.baton/BATON-LOG.txt`.
- Four-way block hash plus machine `diff` of the block against PLAN Decision 2.
- `grep -n` mapping of all eight block rules and three role bullets onto
  `.baton/PROTOCOL.md` line numbers; `.baton/DIRECTOR.md:10`.
- `internal/baton/lint.go:62-83`, `internal/baton/merge.go:28-44`.
- Five `lint` mutation experiments on a scratch copy of the tree.
- `merge-agent-block` upgrade experiment on a scratch project carrying the
  36-line block.
- `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...`, `gofmt -l .`,
  `git diff --check`, all on the copy.

## Report To Director

- Suggested summary: Review round 01: pointer block verified byte-identical, chain closes, no rule lost, no blockers
- Artifact path: .baton/runs/20260920-1605-rules-one-line-REVIEW-01.md

## Required Next Step

- ask Director to request user approval
