# REVIEW-02: Put the removal procedure where the agent asked to do it can read it

Task ID: kovh
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1821-uninstall-reachable-RUN-02.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

Director confirmed the four items landed, so this review does not re-check
existence. It judges the two things a document round can get wrong: what the
second cut lost, and whether the three-way split holds as written.

## 1. Nothing was lost in the second cut

Recovered `git show HEAD:UNINSTALL.md` (123 lines, the version round 01
deleted) and compared it item by item against the current three documents
(`UNINSTALL.md` 34 lines, `PAUSE.md` 19, `.baton/HOW-TO-UPDATE.md`).

The intermediate version — `UNINSTALL.md` as round 02 first restored it,
before item 4 trimmed its pause paragraph to a pointer — is **not recoverable
from Git**. No commit was made during either round; `git stash list` is empty
and no dangling blob in `git fsck` holds it. That version is reconstructable
only from RUN-02's own account of it, which says the trimmed paragraph
explained `pause` in full. Everything that account names is now in `PAUSE.md`
in more detail than the original carried, so the trim moved content rather
than dropping it — but this is testimony, not evidence, and it is the one
claim in this review that could not be verified against an artifact.

Every reason Director named survives in at least one of the three documents:

| Reason | Where it now lives |
| --- | --- |
| Why `--purge` is not the default (`runs/` is the project's own record, often the only account of why the tree looks the way it does) | `HOW-TO-UPDATE.md` "Removing The Record Too"; also, shortened, in `UNINSTALL.md` |
| Why a Git precondition replaces a confirmation prompt (in an agent pipeline the prompt is answered by the agent that wants to proceed) | `HOW-TO-UPDATE.md`; `UNINSTALL.md` states the refusal without the reason |
| Why a hand-edited block has no `--force` override (those bytes cannot be reconstructed from what Baton ships) | `HOW-TO-UPDATE.md` "Refusals" |
| That an instruction file Baton wrote in full is deleted, not emptied, with the dry-run marker and the instruction to warn first | `HOW-TO-UPDATE.md` "Confirm the result" |
| A user saying "stop using Baton" usually means pause | `HOW-TO-UPDATE.md` "Which Of These Is It?" (N3 from REVIEW-01, now closed) and `UNINSTALL.md` |
| What pausing does, reversibility, CLI behaviour while paused, the two surprises | `PAUSE.md` and `HOW-TO-UPDATE.md` "Pause And Resume" |

Also checked present somewhere: the four-row triage including "just this one
task" (short Q&A is not a recording target), the dry-run-then-approve
sequence, what the dry run lists, the four-row refusal table, "tell the user
concretely how many timeline lines and run records disappear", `git show`
recovery after `--purge`, all four `After Removal` bullets, and `lint` still
working in a removed project.

The only thing still absent from all three is REVIEW-01's N4: the header line
saying who the document is for (the agent, though a human may follow it).
Unchanged from round 01, still cosmetic. `PAUSE.md`'s trimming dropped
nothing — it is strictly larger than the original's `## 제거 대신 중지`
section, adding the open-task refusal the original never mentioned.

**Finding: no reason was lost.** N5 stands unchanged (`BATON-LOG.txt:69` is
append-only history).

## 2. The three-way split as written

RUN-02's list holds for `UNINSTALL.md`, and is overstated for `PAUSE.md`.
Neither root document carries a *procedure* the lifecycle document carries —
no command sequence, no dry-run/apply steps, no refusal table appears twice —
so the drift this split was meant to avoid is not present. But two of the six
bullets RUN-02 lists as `PAUSE.md`'s own content are near-verbatim restatements
of lifecycle prose, not new framing. See N1 and N2.

`UNINSTALL.md` earns its place: the pre-install framing, the adoption-level
question ("does this tool trap me"), the explicit statement that it does not
restate the procedure and why, and the pointer to `PAUSE.md` exist nowhere
else. Its one duplicated passage — why `--purge` is not the default — is a
reason rather than a step, and Director already accepted the same shape of
duplication for the pause-first sentence.

## 3. Read as the person they are written for

Someone deciding whether to adopt Baton, who has installed nothing.

`PAUSE.md` answers its question completely. "Can I stop without losing
anything, and can I come back?" — yes, reversible, nothing deleted, `resume`
restores the block exactly, and here are the two places it will refuse.
Nothing that reader needs is missing.

`UNINSTALL.md` half-answers its own. The question is "if I adopt this, can I
get out?" It says what *survives* removal and never says what removal
*takes out*: the rules block, the files Baton installed, the binary, and —
the one outcome an adopter could be surprised by — that a `CLAUDE.md` or
`AGENTS.md` Baton wrote in full is deleted outright rather than emptied. That
fact is in `HOW-TO-UPDATE.md`, which by this document's own premise the reader
cannot open yet. A reader finishes `UNINSTALL.md` knowing their records are
safe and not knowing what happens to their instruction files. See N3.

## Findings

### Blockers

- None.

### Nits

- **N1 — RUN-02 mischaracterises the overlap it reports.** For `PAUSE.md`'s
  "what the CLI does while paused", RUN-02 says the lifecycle document states
  the same fact "operationally (which commands are in `refusesWhilePaused`)".
  It does not: `HOW-TO-UPDATE.md` says "commands that would write a record or
  advance a round refuse, and status commands still work and report the paused
  state", which is `PAUSE.md`'s sentence in English. Never names
  `refusesWhilePaused`. The overlap is fine; the record of it is wrong.
- **N2 — two more `PAUSE.md` bullets are restatements, not unique content.**
  "changes only the rules block" and both "surprises" (open-task refusal with
  `--force`, resume-before-update so update cannot silently lift the pause)
  appear in `HOW-TO-UPDATE.md` "Pause And Resume" in nearly the same words.
  `PAUSE.md`'s genuinely unique content is the pre-install framing and the
  no-procedure promise. Non-blocking: facts, not steps, so nothing can drift
  into contradiction on its own; but the document is thinner than RUN-02's
  six-bullet list suggests.
- **N3 — `UNINSTALL.md` does not say what removal removes.** One sentence
  ("`remove` deletes the rules block and the files Baton installed; an
  instruction file Baton wrote in full is deleted outright, so say so before
  approving") closes it. `PROTOCOL.md`'s `## Pause, Remove, And Update`
  paragraph already carries the first half in one line.
- **N4 — both root documents point at a path their stated reader cannot
  open.** `UNINSTALL.md` and `PAUSE.md` send the reader to
  `.baton/HOW-TO-UPDATE.md`, which exists only after installing — while
  `README.md`'s 더 읽기 line points at `bootstrap/.baton/HOW-TO-UPDATE.md`
  for exactly that reason. Adding the bootstrap path beside the installed one
  would let the pre-install reader actually read the procedure before deciding.
- **N5 — nothing guards the root documents.** `TestHowToUpdateCoversAllProcedures`
  checks five keywords in the lifecycle document only. Round 01 deleted
  `UNINSTALL.md` and every check stayed green. Neither root document's
  existence, nor the pause-first sentence this round restored, is covered by
  any test. Out of scope for a documents-only round; worth deciding on before
  close.
- **N6 — punctuation drift between the two root documents.** `UNINSTALL.md`
  uses `—` in prose and list items; `PAUSE.md` uses ASCII `--` in the same
  position. Cosmetic.

## Evidence Reviewed

- `git show HEAD:UNINSTALL.md` (123 lines) compared line by line against
  `UNINSTALL.md` (34), `PAUSE.md` (19), and `.baton/HOW-TO-UPDATE.md`.
- `git stash list` (empty), `git reflog` (no round commits), and a scan of
  every dangling blob from `git fsck --dangling` for a `# Uninstall` /
  `# Pause` header: the intermediate round-02 version is not in Git.
- RUN-02's "What the two root documents say that `HOW-TO-UPDATE.md` does not"
  list, checked bullet by bullet against the three files.
- `.baton/PROTOCOL.md` lines 138-157: the binary/platform paragraph is the
  last paragraph of `## Git And Updates`, immediately above
  `## Pause, Remove, And Update` (REVIEW-01 N1 closed).
- `README.md:200-202`: 더 읽기 links to `UNINSTALL.md`, `PAUSE.md`, and
  `bootstrap/.baton/HOW-TO-UPDATE.md`; all three resolve.
- Validation reproduced on a copy of the tree, never the working tree:
  `go test ./...` -> `ok .../internal/baton`; `go vet ./...` clean;
  `gofmt -l .` empty; `go run ./cmd/baton lint` -> `baton-lint passed`;
  both paired-copy `diff`s empty. Matches RUN-02.
- `internal/baton/lint_test.go:340-355`: the only document test this round
  could break; it passes and does not cover the root documents (N5). Round 02
  added no test, so there is no load-bearing test check to perform and no
  loosened check to construct a case against.

## Suggested User Checks

- Read `UNINSTALL.md` alone, as someone who has not installed Baton, and
  decide whether "what removal takes out" belongs there (N3) or is correctly
  left to the installed document.
- Open `PAUSE.md` and `.baton/HOW-TO-UPDATE.md`'s "Pause And Resume" side by
  side and judge whether `PAUSE.md`'s remaining overlap is acceptable framing
  or a second copy to maintain (N1, N2).
- Follow the links at the bottom of `UNINSTALL.md` and `PAUSE.md` as if you
  had never installed Baton, and see whether `.baton/HOW-TO-UPDATE.md` is
  reachable for you (N4).
- Read `.baton/HOW-TO-UPDATE.md` top to bottom after being told "stop using
  Baton" and confirm the new pause-first paragraph now leans far enough
  (REVIEW-01 N3).
- Decide whether a test should pin the root documents' existence, given round
  01 deleted one with every check green (N5).

## Report To Director

- Suggested summary: Round 02's second cut lost no reasoning; the split holds,
  with `PAUSE.md` thinner than RUN-02 claims; no blockers, six nits
- Artifact path: `.baton/runs/20260920-1821-uninstall-reachable-REVIEW-02.md`

## Required Next Step

- Ask Director to request user approval.
