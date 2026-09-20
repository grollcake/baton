# REVIEW-01: Put the removal procedure where the agent asked to do it can read it

Task ID: kovh
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1821-uninstall-reachable-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- None.

### Nits

- **N1 — `PROTOCOL.md`'s binary paragraph changed sections.** Decision 3 said
  `## Git And Updates` keeps its first paragraph *and* its last (the binary and
  platform note). The heading was inserted above the pause paragraph, so the
  binary/platform paragraph now sits under `## Pause, Remove, And Update`,
  where it is off-topic. Fix is to move that paragraph above the new heading.
  Non-blocking: no rule text changed, and both copies agree.
- **N2 — RUN-01's V1 lint was run with a different binary than the procedure
  names.** I built the *released* pre-round binary, installed it into a fixture
  project, and ran one `update --upstream <worktree> --apply` with it. The
  documents arrive correctly on that single update (both files byte-identical
  to upstream afterwards), but `.baton/bin/baton lint` — the installed binary,
  which `HOW-TO-UPDATE.md` step 8 tells the project to run — then fails with
  `HOW-TO-UPDATE.md differs from the copy this baton binary shipped with` and
  the same for `PROTOCOL.md`. RUN-01 reported "lint passed" because it ran
  `go run ./cmd/baton lint` instead. This is R3, not a defect: the shipped
  binaries still embed the pre-round documents. I re-ran the whole fixture
  against a rebuilt upstream (binary rebuilt for darwin-arm64, SHA256SUMS
  updated) and the result is clean — see Evidence. The claim holds; the RUN's
  wording overstates what was demonstrated.
- **N3 — one reason was lost with `UNINSTALL.md`.** Recovered from
  `git show main:UNINSTALL.md` and compared line by line. Everything Director
  named survives with its reasoning intact: why `--purge` is not the default
  (`runs/` is the project's record, often the only account of why the tree
  looks the way it does), why the Git precondition replaces a prompt (in an
  agent pipeline the prompt is answered by the agent that wants to proceed),
  why a hand-edited block has no `--force` override (those bytes cannot be
  reconstructed from what Baton ships), and that an instruction file Baton
  wrote in full is deleted rather than emptied, with the dry-run marker and the
  instruction to warn the user first. What did not survive is the opening
  claim that when a user says "let's stop using Baton" they usually do not mean
  removal. The new document keeps the consequence ("offer `pause` first if the
  answer is unclear") and the reversibility asymmetry, but not the reading of
  user intent that motivates them. One sentence in `## Which Of These Is It?`
  restores it.
- **N4 — cosmetic losses, no action needed.** The header line saying the
  document is for the agent (a human may follow it too) and the top-of-file
  link to `BOOTSTRAP.md` are gone; `BOOTSTRAP.md` is still linked from
  `After Removal`.
- **N5** — `BATON-LOG.txt:69` still names `UNINSTALL.md`. Correct: the log is
  append-only and that line is history, not a reference.

### Judgment Director asked for: the three-line growth

The criterion was protecting against restated or duplicated rules, and it
caught nothing of the kind. `git diff` on `PROTOCOL.md` is 4 insertions, 1
deletion: the heading and its blank line (2, no deletion), and the paragraph's
final line replaced by two because one appended pointer sentence wraps at the
file's ~78-column width. No pause or remove rule appears twice anywhere in the
file.

The moved text is genuinely unchanged — more strongly than the plan required.
It was not relocated at all: the heading was inserted above it, so every byte
of both sentences is where it was, and the only edit inside them is the
appended `All three procedures are in `.baton/HOW-TO-UPDATE.md`.`. The
Executor was right to report the number rather than reword the file to hit it;
rewording prose to satisfy a line count would have been the worse outcome.

### Judged from the diff, not reproduced

- **Guide aliases.** `guideDocuments` gains four keys pointing at
  `HOW-TO-UPDATE.md`; the usage string and the error string both list them.
  No behavior outside `guide` is touched.
- **README and BOOTSTRAP pointers.** The 더 읽기 entry now resolves to
  `bootstrap/.baton/HOW-TO-UPDATE.md`, which exists; `BOOTSTRAP.md` gains one
  line telling an installing agent that lifecycle procedures live in the
  installed copy. Both are one line each, as planned.
- **Tests are load-bearing.** Verified on a copy: reverting `guide.go` fails
  `TestGuideDocumentsAllResolve` and `TestGuideLifecycleAliasesPrintSameBytes`;
  reverting the document fails `TestHowToUpdateCoversAllProcedures` on
  `remove --apply`. Each fails for the reason the change was made.

## Suggested User Checks

- Read `bootstrap/.baton/HOW-TO-UPDATE.md` top to bottom as if you had just
  been told "stop using Baton" — does `## Which Of These Is It?` route you, and
  does it lean toward `pause` strongly enough without N3's missing sentence?
- Read `.baton/PROTOCOL.md` lines 138-157 and decide whether the binary and
  platform paragraph belongs under `## Pause, Remove, And Update` (N1).
- Compare `git show main:UNINSTALL.md` against the new `## Remove` section and
  confirm nothing you valued in the Korean prose is missing beyond N3/N4.
- Confirm you accept English as the installed lifecycle text (Decision 2),
  given `README.md`'s 중지/제거 sections remain the Korean surface.
- At close, after the binaries are rebuilt, run `baton guide remove` and
  confirm it prints the lifecycle document.

## Evidence Reviewed

- `.baton/runs/20260920-1821-uninstall-reachable-PLAN.md`, `-RUN-01.md`.
- `git diff main` over all eleven changed paths; `git show main:UNINSTALL.md`.
- Fixture A: pre-round project built from `main`'s `bootstrap/.baton` with the
  *released* `darwin-arm64` binary installed; `lint` clean before update. One
  `update --upstream <worktree> --apply` with that binary: exit 0, both
  documents byte-identical to upstream, `lint` with the installed binary fails
  on the two expected drift errors (N2).
- Fixture B: same, against an upstream copy with the binary rebuilt and
  SHA256SUMS updated. One update with the project's own old binary: `lint`
  passes, `guide remove|uninstall|pause|lifecycle` all print the same bytes as
  `guide update`. This is the round trip the design rests on, and it holds.
- Test load-bearing checks (guide.go reverted; document reverted) on a copy.
- Working tree: `go test ./...`, `go vet ./...`, `gofmt -l .` clean; both
  `diff` pairs empty; `grep -rn UNINSTALL` finds only `BATON-LOG.txt:69`.

## Report To Director

- Suggested summary: Review confirms the lifecycle document reaches an existing
  project in one update; no blockers, five nits
- Artifact path: `.baton/runs/20260920-1821-uninstall-reachable-REVIEW-01.md`

## Required Next Step

- Ask Director to request user approval.
