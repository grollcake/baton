# REVIEW-01: Let a project pause Baton without removing it

Task ID: srkz
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1618-pause-baton-RUN-01.md
Status: complete

## Decision

Result: blockers

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

The two things Director weighed above everything else both hold, and I
established each myself rather than taking the RUN's word for it. The round
trip is byte-exact on project shapes this repository does not have, and the
partial-write state is unreachable through a failure I constructed in the real
call shape, not the test's. The one blocker is not a defect in behaviour: it is
that the single guard protecting the second of those two properties has no
regression test, and the RUN reports it as covered by a test that does not
cover it.

## Findings

### Blockers

- **B1. `update --apply`'s pause refusal is untested, and the RUN says it is
  tested.** I deleted the check from `runUpdate` on a scratch copy of the tree
  (the three lines `if pausedMessage != "" { return errors.New(pausedMessage) }`)
  and `go test ./internal/baton/` still passed. The RUN's Success Criteria line
  claims this criterion is "met (V3,
  `TestUpdateDryRunAllowedWhilePausedAndMentionsResume`)", but that test runs
  `update` without `--apply` and only asserts the output mentions `baton
  resume`; nothing exercises `--apply`. The behaviour today is correct — I ran
  `update --upstream <repo> --apply` against a paused fixture, it exited 1 with
  the Decision 4a message, and the whole-tree checksum was identical before and
  after — so this is a coverage hole, not a live bug. It is the hole that
  matters most: `update --apply` is the one command in the codebase that
  rewrites the rules block, and Decision 4a exists solely to stop it silently
  un-pausing a project. Every other guard in the round is load-bearing under
  the same mutation test (see Evidence); this one is not. Fix is one test in
  `pause_test.go` in the shape of `TestUpdateDryRunAllowedWhilePausedAndMentionsResume`,
  with `--apply`, asserting a non-zero exit and that `AGENTS.md`, `CLAUDE.md`
  and `.baton/VERSION` are byte-identical after. Correct the Success Criteria
  line in the RUN at the same time.

### Nits

- **N1. The paused block text is pinned to itself.** I changed one character of
  Decision 2's text inside the `pausedBlock` constant
  (`paused.` -> `paused!`) and the full suite still passed.
  `TestPauseWritesTheExactPausedBlock` compares the written block against the
  constant, so it proves pause writes the constant to both files — which is
  load-bearing — but it cannot catch the constant drifting from the text
  Decision 2 fixed. Not a runtime risk while pause, resume and lint all read
  the same constant; it does mean a future reword of the paused block is a
  silent change. A golden-file or literal-string assertion would close it.

- **N2. A file carrying two `<baton-rules>` blocks pauses only the first.**
  `batonBlockPattern.Find` takes the first match, so on a fixture whose
  `AGENTS.md` held the active block twice, `pause` exited 0, rewrote the first
  block to the paused text, left the second block active, and `lint` then
  printed `OK: Baton is paused`. An agent reading that file is told Baton is
  paused and then, a few lines later, told to follow the protocol. Only
  reachable by hand-editing, and `lint` before this round reported the same
  file as `blocks match`, so this is pre-existing regexp behaviour rather than
  something the round introduced. It is worth naming because this round is what
  makes the block the state: a half-paused file now passes `lint` as paused.

- **N3. Decision 6's prose and its table disagree, and the code follows the
  table.** The prose says "error only when neither exists"; the code returns
  silently when neither `AGENTS.md` nor `CLAUDE.md` is present, so a project
  with `.baton/` intact and no instruction file at all prints no block line and
  `lint` passes. That is a project where Baton is in effect nowhere an agent
  will read. Executor followed the table, which is the normative half of
  Decision 6, so this is a plan ambiguity rather than a deviation; Director
  should decide whether that state should error.

- **N4. `batonBlockPattern` now exists twice, kept in sync by hand.** `docs.go`
  carries its own copy because the module root cannot import `internal/baton`.
  The comment says both copies must stay identical; nothing enforces it. A
  cheap assertion in a root-package test would.

- **N5. The restore path ignores its own errors.** In `writeInstructionBlocks`,
  the rollback is `_ = atomicWrite(done.path, done.original, done.mode)`. If
  the restore itself fails, the caller sees only the original write error and
  the two instruction files are left disagreeing with nothing said about it.
  The window is narrow and I could not construct it, but it is the one path
  where the "files can never disagree" property is on trust rather than on a
  check.

### Judgement asked for, not a blocker: will the removal round reuse this?

Partly, and the part it most needs is the part it will have to rework.

`replaceBlock` and `computeReplacedContent` are clean and removal reuses them
unchanged. `classifyBlock` and the `blockStatus` type are reusable as written.

`blockAbsent`, though, does not mean what removal needs it to mean. In
`projectBlockState` it is returned only when *neither instruction file exists*.
The state removal actually produces — files present, block gone — returns an
*error* (`AGENTS.md missing <baton-rules> block`), not `blockAbsent`. I
confirmed this on a fixture with both files present and no block: `lint` prints
`ERROR: AGENTS.md missing <baton-rules> block` and `resume` exits 1 with
`cannot resume: AGENTS.md missing <baton-rules> block`. So a Baton-removed
project fails `lint`, and the removal round will have to change
`projectBlockState`'s error path into the `absent` return, change
`checkAgentBlocks`'s two `missing block` branches, and decide what
`refuseIfPaused` and `pausedUpdateMessage` do with the new state. That is a
contained rework of one function and one check, not a redesign — but it is
rework, and the four-state classifier as written does not hand removal a
finished state to switch on.

## Suggested User Checks

- In a throwaway copy of a real project (not this repository), run
  `pause`, open `AGENTS.md` and `CLAUDE.md`, and read the paused block as an
  agent would: does it tell you clearly enough that Baton is off, that
  `.baton/` must be left alone, and how to come back? This text is the whole
  feature and it is the one thing no test can judge.
- In that same copy, run `resume` and then `git diff`: confirm the only change
  is four lines appended to `.baton/BATON-LOG.txt`, and that you are content
  with pause and resume writing to the timeline at all (PLAN R1 — Director
  chose recording; this is your last cheap moment to choose otherwise).
- While paused, try the thing you would actually do by accident: run
  `baton update --upstream UPSTREAM-DIR --apply`, confirm it refuses, then follow the
  printed recipe (`resume`, update, `pause`) and confirm you end up paused
  again on the new version.
- Run `baton lint` in an older project of yours that has not been updated to
  the current block text. It will now report `ERROR: Baton block matches
  neither ...` where it passed before (PLAN R2). Confirm that message tells you
  what to do, and that you are willing to have every such project go red until
  it is updated.
- Run `baton pause` in a project with an open task, confirm it names the task
  and refuses, then `--force` and check the `RUN_DONE` line names the task.

## Evidence Reviewed

- `.baton/runs/20260920-1618-pause-baton-PLAN.md`,
  `.baton/runs/20260920-1618-pause-baton-RUN-01.md`, and the full branch diff
  against `main` (13 files, +173/-38) plus untracked `internal/baton/pause.go`
  and `internal/baton/pause_test.go`.
- Checks re-run on the working tree: `go vet ./...` clean, `gofmt -l .` empty,
  `go test ./...` ok, `go build ./cmd/baton` ok.
- **Round trip, odd-shaped project (Director's item 1).** Fixture with
  `AGENTS.md` carrying a preamble, trailing-space lines, a tab-terminated line
  and **no final newline**, and `CLAUDE.md` with entirely different surrounding
  content and trailing blank/whitespace lines. `shasum` of every file before,
  `pause`, `resume`, `shasum` again: every file identical except
  `.baton/BATON-LOG.txt`, which gained exactly four lines with its two original
  lines unchanged. Verified separately on a **`CLAUDE.md`-only** project: same
  result, byte-identical.
- **Partial state (item 2).** Making `CLAUDE.md` read-only does *not* fail —
  `atomicWrite` renames a temp file within a writable directory. Constructed a
  genuine failure instead with `chflags uchg CLAUDE.md`: `pause` exited 1 with
  `rename ...: operation not permitted`, `AGENTS.md` was restored
  byte-identical, `CLAUDE.md` untouched, `BATON-LOG.txt` still two lines, no
  stray `.baton-*` temp files, and `lint` reported `OK: ... blocks match
  (active)`. Note the round's own
  `TestWriteInstructionBlocksRestoresOnPartialFailure` uses two separate
  directories, a shape `instructionFiles()` never produces; the property holds
  in the real shape too, which is what I checked.
- **Refuse list (item 3).** On a paused fixture, `append`, `new-round`,
  `feedback`, `gate`, `prompt`, `subagent-prompt`, `await` and
  `merge-agent-block` each exited 1. `shasum` over **every file in the tree**
  (not only the timeline) was identical before and after the whole sweep.
  `append`, `new-round`, `feedback`, `gate`, `prompt` and `await` share the
  `App.Run` gate; `merge-agent-block` gives the more specific update-trap
  message, as Decision 4a intends.
- **Update trap, both paths (item 4).** `update --apply` exited 1 with the
  Decision 4a message and left the whole tree byte-identical.
  `merge-agent-block` — the route `HOW-TO-UPDATE.md` step 7 sends a manual
  updater down — exited 1 with the same message, so the documented manual route
  cannot un-pause a project. The other half holds: `resume`, `update --upstream
  <repo> --apply` (0.0.1 -> 0.31.0), `pause` ran clean end to end and left the
  project paused on the new version.
- **Lint classification (item 5).** Seven fixtures: active-both ->
  `OK: ... (active)`; paused -> `OK: Baton is paused; ...`; identically emptied
  -> `ERROR: ... matches neither ...`; identically hand-edited -> same ERROR;
  a block from the previous release (`54d8926~1`) -> same ERROR;
  `CLAUDE.md`-only -> fully checked (gap A closed); `AGENTS.md` present with no
  block -> `ERROR: AGENTS.md missing <baton-rules> block`; blocks differ ->
  `ERROR: ... blocks differ`; no instruction files -> no block line, passes
  (N3). Both gaps the plan set out to close are closed. On R2: the state a
  reasonable existing project can be in and now fails is **a block from an
  earlier release**, which is wider than the PLAN's "hand-edited inside the
  markers". In a normally installed project the binary and the block are
  updated together by `update --apply`, so they stay consistent; the exposure
  is a project whose binary was replaced without running `update`. The message
  names the fix ("run an update or restore it"), which I consider adequate, but
  it is a broader class than R2 as written.
- **Documents (item 6).** `.baton/PROTOCOL.md` and `bootstrap/.baton/PROTOCOL.md`
  each gained exactly the one Decision 8 sentence (three wrapped lines) and
  nothing else; `.baton/HOW-TO-UPDATE.md` and
  `bootstrap/.baton/HOW-TO-UPDATE.md` each gained exactly the one Step 0 (three
  wrapped lines) and nothing else. `diff` between the two copies of each file is
  empty. No rule is stated in a tool and a document both: the paused block text,
  the refusal lists and the classification live only in the binary, and the two
  document additions describe where to find the commands and which order to run
  them in, which is the one thing the CLI cannot tell an agent that has not run
  it yet.
- **Repository's own blocks.** All four `<baton-rules>` copies
  (`AGENTS.md`, `CLAUDE.md`, `bootstrap/AGENTS.md`, `bootstrap/CLAUDE.md`)
  hash identically to `30702eb1...` and are the active text; the round did not
  touch them.
- **Load-bearing check (mutation on a scratch copy).** Removing the `App.Run`
  pause gate -> `TestRefusalSweepWhilePaused` fails. Removing the
  `refusePausedForUpdate` call in `runMergeAgentBlock` -> same test fails.
  Removing the restore loop in `writeInstructionBlocks` ->
  `TestWriteInstructionBlocksRestoresOnPartialFailure` fails. Removing the
  `update --apply` refusal -> **suite passes** (B1). Altering the paused block
  text by one character -> **suite passes** (N1).
- No file under `.baton/runs/` was created, modified or deleted by any of the
  above; every experiment ran on scratch fixtures outside the repository.

## Report To Director

- Suggested summary: Review round 01 — pause round trip and partial-failure safety verified on shapes beyond this repository; one blocker, the `update --apply` refusal has no regression test
- Artifact path: `.baton/runs/20260920-1618-pause-baton-REVIEW-01.md`

## Required Next Step

- request RUN-02: add the `update --apply` refusal test and correct the RUN's
  Success Criteria line (B1). N1-N5 are Director's call to fold in or carry.
