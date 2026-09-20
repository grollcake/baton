# RUN-02: Put the removal procedure where the agent asked to do it can read it

Task ID: kovh
Date: 2026-09-20
Executor: Claude Sonnet 5
Plan: .baton/runs/20260920-1821-uninstall-reachable-PLAN.md
Status: complete

## Changes

Round 02, per Director's four items (the fourth arrived as a follow-up
mid-round). RUN-01 and REVIEW-01 are unmodified.

1. **Restored `UNINSTALL.md` at the repository root, in Korean, cut down.**
   Recovered the original from Git (`git show HEAD:UNINSTALL.md`, 123 lines)
   rather than rewriting from memory, then rewrote it short for its actual
   audience: a person deciding whether to adopt Baton, reading the
   repository on the web, who has not installed anything and cannot see
   `.baton/` -- the same reason `BOOTSTRAP.md` lives at the root. It no
   longer restates the procedure (no dry-run/apply steps, no refusal table):
   it states that removal exists, that `remove` is the only irreversible
   one, what `remove` keeps by default and why, and points to
   `.baton/HOW-TO-UPDATE.md` and `<baton> guide remove` for the actual
   steps. After item 4 added `PAUSE.md` as its own document, trimmed
   `UNINSTALL.md` further: the paragraph that used to explain `pause` in
   full now just points to `PAUSE.md` for readers who only want to stop
   temporarily, so pause is explained in exactly one root document, not two.
   Final length: 34 lines (was 123).
   - `README.md`: re-added `UNINSTALL.md` to 더 읽기, kept the
     `HOW-TO-UPDATE.md` lifecycle-document link added in round 01.

2. **N3 -- restored the "usually means pause" reasoning, in English, in the
   lifecycle document.** `bootstrap/.baton/HOW-TO-UPDATE.md`'s "Which Of
   These Is It?" section (the first thing an agent reads, before any Update
   or Remove step) now reads: "When a user says something like 'stop using
   Baton' or 'turn this off,' they usually mean pause, not removal:
   `pause` is reversible and removal is not. Ask when the request is
   ambiguous, and offer `pause` first rather than defaulting to removal."
   Copied byte-for-byte to `.baton/HOW-TO-UPDATE.md`.

3. **N1 -- moved the binary/platform paragraph back above the new heading**
   in both `PROTOCOL.md` copies. It had slid under `## Pause, Remove, And
   Update` in round 01; it now sits at the end of `## Git And Updates`,
   where Decision 3 put it, immediately before the new heading. Moved the
   paragraph only; no word in it changed.

4. **Added `PAUSE.md` at the repository root, in Korean.** Same audience
   and split as `UNINSTALL.md`: what pausing is and when to reach for it,
   for a reader who cannot see `.baton/` yet, with no procedure repeated --
   `.baton/HOW-TO-UPDATE.md` and `<baton> guide pause` carry the steps.
   19 lines, shorter than `UNINSTALL.md`'s 34, since pausing is smaller:
   two commands, reversible, nothing deleted. States what pausing does
   (the rules block reads "paused"; agents work as if Baton were not
   installed; records and installed files untouched), that `resume`
   restores the block exactly, what the CLI does while paused (write/round
   commands refuse and change nothing; read-only status commands still
   work and report paused), and the two surprises: an open task refuses
   `pause` unless forced, and a paused project must `resume` before it can
   update, so update cannot silently switch Baton back on by rewriting the
   rules block.
   - `README.md`: added one 더 읽기 line for `PAUSE.md`, next to
     `UNINSTALL.md`.

No Go code, `VERSION`, or `.baton/bin/` touched. `pause` and `remove`
behavior and output are unchanged.

## Validation

Re-run after item 4 (`PAUSE.md`) landed, since the tree changed after this
artifact's first validation pass.

- Evidence collected before fixing a user-reported defect: n/a (not a defect
  fix; four Director-specified document corrections)
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton`.
- `go vet ./...`: clean, no output.
- `gofmt -l .`: clean, no output.
- `go run ./cmd/baton lint`: `baton-lint passed`, no drift, all managed
  documents present and matching.
- Self smoke test after fixing a user-reported defect: n/a

### Paired-copy diffs

```
diff .baton/HOW-TO-UPDATE.md bootstrap/.baton/HOW-TO-UPDATE.md   -> empty (identical)
diff .baton/PROTOCOL.md bootstrap/.baton/PROTOCOL.md              -> empty (identical)
```

### What the two root documents say that `HOW-TO-UPDATE.md` does not

Recorded so the three-way split (`UNINSTALL.md`, `PAUSE.md`,
`.baton/HOW-TO-UPDATE.md`) is explicit, not implicit.

**`UNINSTALL.md`:**

- That the reader may not have Baton installed yet and cannot see `.baton/`
  -- the reason this document exists at the root instead of being folded
  entirely into the installed one.
- That removal exists at all, and that it is the one irreversible option,
  framed as an adoption question ("does this tool trap me") rather than as
  operational steps -- the lifecycle document assumes the reader already
  decided to install and is now doing the work.
- The explicit statement that it deliberately does not restate the
  procedure, and why: two documents describing the same steps drift apart.
  This is itself the reasoning Decision 4/5 relied on to justify one
  installed copy, now written down for the pre-install reader too.
- A short, self-contained explanation of what survives `remove` by default
  and why (`BATON-LOG.txt`, `runs/`, `GUIDANCE.md`, `LESSON-LEARNED.md`,
  `lesson-learned/` are the project's own record, not Baton's output, and
  `runs/` is often the only document explaining why the tree looks the way
  it does) -- present without the full refusal table, `--purge` mechanics,
  or dry-run/apply command sequence the lifecycle document carries.
- A pointer to `PAUSE.md` for a reader who only wants to stop temporarily,
  so pause is explained in one root document, not restated in both.
- A pointer both to the installed file (`.baton/HOW-TO-UPDATE.md`) and to
  `<baton> guide remove` (and its `pause`/`lifecycle` aliases) as two ways
  to reach the same procedure once installed.

**`PAUSE.md`:**

- The same "reader may not have installed yet, cannot see `.baton/`" framing
  as `UNINSTALL.md`, so the choice between stopping temporarily and removing
  is visible from the root before either document sends the reader inward.
- That pausing changes only the rules block content (to a "Baton is paused"
  text) and nothing else -- an adoption-level reassurance, not the
  installed mechanics of which files that touches.
- That it is fully reversible and `resume` restores the block exactly --
  the fact `UNINSTALL.md` now only points at, rather than restating.
- What using the CLI while paused feels like in the reader's own terms:
  write/round-advancing commands refuse and change nothing; read-only
  status commands still work and say the project is paused. The lifecycle
  document states the same fact operationally (which commands are in
  `refusesWhilePaused`); this states it as what the reader should expect.
- The two things Director asked to flag as surprises before someone hits
  them: an open task refuses `pause` unless forced, and a paused project
  must `resume` before it can update, because update rewrites the rules
  block and would otherwise silently switch Baton back on.
- A pointer to `.baton/HOW-TO-UPDATE.md` and `<baton> guide pause` for the
  actual steps, not repeated here.

One item is intentionally in both `UNINSTALL.md` and `HOW-TO-UPDATE.md`, not
unique to either: "a user asking to stop using Baton usually means pause,
not removal." Item 2 put it in the lifecycle document for the agent about to
act; `UNINSTALL.md` states the same choice for the person deciding whether to
adopt the tool at all, then sends them to `PAUSE.md` rather than explaining
pause itself. Each of the three documents serves its own reader; none
restates another's procedure.

## Report To Director

- Suggested summary: Restored `UNINSTALL.md` and added `PAUSE.md` for the
  pre-install reader, carried the "offer pause first" reasoning into the
  lifecycle document, and moved the binary/platform paragraph back above
  the new `PROTOCOL.md` heading.
- Artifact path: `.baton/runs/20260920-1821-uninstall-reachable-RUN-02.md`

## Success Criteria Status

- `UNINSTALL.md` restored at the root, in Korean, shorter than the original,
  not restating the installed procedure: met.
- `PAUSE.md` added at the root, in Korean, shorter than `UNINSTALL.md`,
  covering what pausing does, that it is reversible, what the CLI does
  while paused, and the two surprises (open-task refusal, resume-before-
  update), without restating the procedure: met.
- `README.md` points to `UNINSTALL.md`, `PAUSE.md`, and the lifecycle
  document: met.
- N3 reasoning ("usually means pause, offer pause first") present as the
  first thing read in the English lifecycle document: met.
- N1: binary/platform paragraph back above `## Pause, Remove, And Update`,
  text unchanged: met.
- Both copies of `PROTOCOL.md` and `HOW-TO-UPDATE.md` byte-identical: met.
- No Go code, `VERSION`, or `.baton/bin/` changed; `pause`/`remove` behavior
  unchanged: met.

## Unresolved Risks

- Same as RUN-01's R3: this repo's installed `.baton/bin/baton` reports
  drift against the edited managed documents until Director rebuilds and
  bumps `VERSION` at close.
- RUN-01's V3 note (PROTOCOL.md net line growth of +3, not +2, from
  word-wrap of the added pointer sentence) still applies; round 02 did not
  touch that sentence and did not change its line count.

## Out Of Scope Returned To Director

- None. Director's four items were implemented as specified with no
  ambiguity requiring escalation.
