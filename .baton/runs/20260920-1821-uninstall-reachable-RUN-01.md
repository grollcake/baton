# RUN-01: Put the removal procedure where the agent asked to do it can read it

Task ID: kovh
Date: 2026-09-20
Executor: Claude Sonnet 5
Plan: .baton/runs/20260920-1821-uninstall-reachable-PLAN.md
Status: complete

## Changes

- Rewrote `bootstrap/.baton/HOW-TO-UPDATE.md` (and copied byte-identical to
  `.baton/HOW-TO-UPDATE.md`) as the one lifecycle document: a "Which of these
  is it?" triage, then Update (unchanged in substance), Pause And Resume, and
  Remove (dry run first, `--apply`, confirming the result, refusal table,
  `--purge` and why it is not the default, after-removal state) -- carrying
  over every item from `UNINSTALL.md`'s substance, in English.
- Edited `bootstrap/.baton/PROTOCOL.md` (and `.baton/PROTOCOL.md`): split
  `## Git And Updates` -- moved the existing pause/remove sentences verbatim
  into a new `## Pause, Remove, And Update` heading and added one sentence
  pointing at `.baton/HOW-TO-UPDATE.md`. No other rule text changed.
- `internal/baton/guide.go`: added `remove`, `uninstall`, `pause`,
  `lifecycle` as additional keys in `guideDocuments`, all resolving to
  `HOW-TO-UPDATE.md`; updated the usage/error strings to list them.
- `internal/baton/app.go`: updated the `usage()` guide line to match.
- Deleted `UNINSTALL.md` (root); no code or docs referenced it except
  `README.md`, updated below, and a historical `BATON-LOG.txt` line left as
  an append-only record.
- `README.md`: replaced the `UNINSTALL.md` link in "더 읽기" with a link to
  `bootstrap/.baton/HOW-TO-UPDATE.md`.
- `BOOTSTRAP.md`: added one line under 사전 확인 pointing update/pause/remove
  procedures at the installed `.baton/HOW-TO-UPDATE.md`; its own 업데이트
  절차 section is unchanged.
- Tests in `internal/baton/lint_test.go`: `TestGuideDocumentsAllResolve` (V4,
  every `guideDocuments` key resolves via `docs.Managed`, aliases match
  `update`), `TestGuideLifecycleAliasesPrintSameBytes` (`guide remove|
  uninstall|pause|lifecycle` print the same bytes as `guide update`),
  `TestHowToUpdateCoversAllProcedures` (V5, the installed document contains
  `pause`, `resume`, `remove --apply`, `--purge`, `update --upstream`).

Not created: no new file under `.baton/`; `HOW-TO-UPDATE.md` was not renamed;
`pause`/`remove` behavior and output are unchanged; `VERSION` was not bumped;
no binaries were rebuilt.

## Validation

- Evidence collected before fixing a user-reported defect: n/a (not a defect
  fix)
- `go run ./cmd/baton lint` (this repo): `baton-lint passed`, no drift.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton`.
- `go vet ./...`: clean, no output.
- `gofmt -l .`: clean, no output.
- Self smoke test after fixing a user-reported defect: n/a

### V1 -- the migration that matters

Built a fixture project (in the scratchpad dir, never in this repo's own
`.baton/`) by copying the current `.baton/` and reverting only
`HOW-TO-UPDATE.md` and `PROTOCOL.md` to the pre-round committed (`git show
HEAD:...`) text -- simulating a project on the last released shape, before
this round's edits. Ran, with `BATON_DIR` pointed at the fixture:

```
go run ./cmd/baton update --upstream /Users/rollcake/lab/baton --apply
```

Output: `Baton update: 0.33.0 -> 0.33.0` / `Baton updated: 0.33.0 -> 0.33.0`,
exit 0. After this single apply: `diff <fixture>/.baton/HOW-TO-UPDATE.md
bootstrap/.baton/HOW-TO-UPDATE.md` and the same for `PROTOCOL.md` were both
empty -- the fixture now carries the new lifecycle document and the split
protocol text. `BATON_DIR=<fixture>/.baton go run ./cmd/baton lint` then
reported `baton-lint passed` with no drift and no missing-file error, with no
second update run. This holds because `managedBatonFiles` (the list the old
binary would have copied) already names `HOW-TO-UPDATE.md` and `PROTOCOL.md`
-- editing a listed file, not adding a new one, is what reaches an existing
project on its first update (Plan Decision 1/2). No released shape failed to
reach this in one pass, so no claim needed softening.

### V2 -- copies stay identical

`diff .baton/HOW-TO-UPDATE.md bootstrap/.baton/HOW-TO-UPDATE.md` and `diff
.baton/PROTOCOL.md bootstrap/.baton/PROTOCOL.md`: both empty. `go run
./cmd/baton lint` on this repo reports no managed-document drift (see above).

### V3 -- PROTOCOL.md line count

Before (this round's start, both copies): 154 lines. After: 157 lines.
`git diff --stat .baton/PROTOCOL.md`: `1 file changed, 4 insertions(+), 1
deletion(-)` (same for the `bootstrap/` copy). The diff shows exactly: a new
`## Pause, Remove, And Update` heading and blank line (2 new lines, no
deletion), and the existing last sentence of the moved paragraph rewrapped
to add one pointer sentence, which itself wraps onto a second line at this
file's ~78-column style (1 line replaced by 2, net +1). Net addition is 3
lines, not 2 -- 1 line over the plan's "at most two lines" target -- entirely
from the pointer sentence's word-wrap under the file's existing line width,
not from any duplicated or restated pause/remove text. The moved text itself
is unchanged, verified by `git diff .baton/PROTOCOL.md`. Flagging this as a
minor deviation from V3's literal wording rather than silently rounding it
down; the substantive criterion in Success Criteria ("gains at most one
sentence and one heading") is met exactly.

### V4 -- guide aliases resolve

`TestGuideDocumentsAllResolve` and `TestGuideLifecycleAliasesPrintSameBytes`
(both in `internal/baton/lint_test.go`) pass. Manually confirmed too:
`go run ./cmd/baton guide remove`, `guide uninstall`, `guide pause`, `guide
lifecycle`, and `guide update` all print identical bytes (the lifecycle
document).

### V5 -- document keeps all four procedures

`TestHowToUpdateCoversAllProcedures` passes: the installed document contains
`pause`, `resume`, `remove --apply`, `--purge`, and `update --upstream`.

### V6 -- no leftover UNINSTALL reference

`grep -rn "UNINSTALL" . --exclude-dir=.git --exclude-dir=runs`: one hit, in
`.baton/BATON-LOG.txt` line 69 -- a historical, append-only log line from a
prior round's `RUN_DONE` summary, not a live reference to the deleted file.
No other reference exists.

## Report To Director

- Suggested summary: Folded the removal procedure into the one installed
  lifecycle document, reachable by name and by `guide` alias, without adding
  a managed file or growing `PROTOCOL.md` beyond one pointer sentence and
  heading.
- Artifact path: `.baton/runs/20260920-1821-uninstall-reachable-RUN-01.md`

## Success Criteria Status

- Existing project updates with its own installed binary and ends with the
  lifecycle document present, lint clean, on the first update: met (V1).
- The two copies of each edited managed document are byte-identical, no
  drift reported: met (V2).
- `PROTOCOL.md` gains at most one sentence and one heading, pause/remove text
  relocated not restated: met in substance; net raw line count is +3 not the
  literal "at most two lines" in V3's check, due to word-wrap (see V3 note) --
  partial on the literal line-count wording, met on the Success Criteria's own
  phrasing.
- `guide remove`, `uninstall`, `pause`, `lifecycle`, `update` all print the
  same document: met (V4).
- No file created or renamed under `.baton/`; `pause`/`remove` behavior and
  output unchanged: met.
- Exactly one removal procedure in the repository: met (V6; `UNINSTALL.md`
  deleted, substance carried into `HOW-TO-UPDATE.md`).

## Unresolved Risks

- R3 from the plan: this repo's installed `.baton/bin/baton` now reports
  drift against the edited managed documents until Director rebuilds and
  bumps `VERSION` at close, as in prior rounds. Not fixed here per
  instruction not to bump `VERSION` or rebuild binaries.
- V3's literal "at most two lines" was exceeded by one line, from word-wrap
  of the added pointer sentence at the file's existing column width, not
  from restated content. Returning this to Director/Planner judgment rather
  than rewording the file further to force the count down.

## Out Of Scope Returned To Director

- None encountered; the plan's scope was implemented as specified with no
  ambiguity requiring escalation.
