# RUN-01: Reduce the agent rules block to a pointer

Task ID: yitg
Date: 2026-09-20
Executor: Claude Sonnet 5
Plan: .baton/runs/20260920-1605-rules-one-line-PLAN.md
Status: complete

## Changes

- `bootstrap/AGENTS.md`: replaced the 34-line `<baton-rules>` body with the
  7-line pointer text from PLAN Decision 2, keeping the `<baton-rules>` /
  `## Baton` / `</baton-rules>` markers and all surrounding content unchanged.
- `bootstrap/CLAUDE.md`, `AGENTS.md`, `CLAUDE.md`: applied the identical
  replacement (copied, not retyped) so all four blocks are byte-for-byte
  the same.

## Validation

- Evidence collected before fixing a user-reported defect: n/a (not a defect fix)
- Four-way byte identity: `for f in AGENTS.md CLAUDE.md bootstrap/AGENTS.md bootstrap/CLAUDE.md; do sed -n '/<baton-rules>/,/<\/baton-rules>/p' "$f" | shasum; done` → all four `30702eb11211c8c2550310e2864258539924ac3e  -`, identical.
- Unchanged-remainder evidence: `git diff --stat -- AGENTS.md CLAUDE.md bootstrap/AGENTS.md bootstrap/CLAUDE.md` shows exactly these 4 files, 16 insertions / 120 deletions total; `git diff` for each file contains exactly one hunk, and every changed line lies strictly between the `<baton-rules>` and `</baton-rules>` marker lines (markers themselves untouched, content above/below in `AGENTS.md` lines 1-13 and the trailing blank line in all four files unchanged). `git status --porcelain` shows only `AGENTS.md`, `CLAUDE.md`, `bootstrap/AGENTS.md`, `bootstrap/CLAUDE.md` newly modified by this Executor; `.baton/BATON-LOG.txt` (M) and `.baton/runs/20260920-1605-rules-one-line-PLAN.md` (??) predate this round (present in the session's opening git status) and were not touched here.
- `go run ./cmd/baton lint`: passed, printed `OK: AGENTS.md and CLAUDE.md Baton blocks match` and `baton-lint passed`; exit 0.
- `go test ./...`: `ok github.com/grollcake/baton/internal/baton 9.050s`, others `[no test files]`; exit 0. `git diff --name-only | grep _test.go` → none (no test file modified).
- `go vet ./...`: no output; exit 0.
- `gofmt -l .`: no output (empty); exit 0.
- `git diff --check`: no output; exit 0.
- Read-through (new block alone, sent an agent through the chain): the new block sends the reader to `.baton/PROTOCOL.md`. That file's "Know Your Role" section (heading present, PROTOCOL.md lines ~17-27) settles which role the agent holds (Director / Planner-or-Executor via delegation prompt / ask-first when resumed without one). The role file to open next is named in the document intro, lines 3-4: "Role-specific operational details live in `DIRECTOR.md`, `PLANNER.md`, and `EXECUTOR.md`; read only the file for your role." All three chain elements are nameable; nothing required stopping to report to Director.
- Self smoke test after fixing a user-reported defect: n/a (not a defect fix)

## Report To Director

- Suggested summary: Replaced the 36-line `<baton-rules>` block in `AGENTS.md`, `CLAUDE.md`, `bootstrap/AGENTS.md`, and `bootstrap/CLAUDE.md` with the 7-line pointer to `.baton/PROTOCOL.md` specified in the PLAN's Decision 2 (fail-safe sentence kept per Director's decision). All four blocks are byte-identical; no other file, code, or test changed. `lint`, `test`, `vet`, `gofmt` all pass.
- Artifact path: .baton/runs/20260920-1605-rules-one-line-RUN-01.md

## Success Criteria Status

- Four blocks byte-identical to each other and to Decision 2 text: met
- `git diff --stat` lists exactly the four files: met
- Each file's `git diff` is exactly one hunk inside the markers: met
- `go run ./cmd/baton lint` reports blocks match and passes: met
- `go test ./...`, `go vet ./...` pass, `gofmt -l .` empty, no test edited: met
- Block contains no imperative beyond reading PROTOCOL.md and the fail-safe: met

## Unresolved Risks

- R1 (Director-accepted, PLAN Decision 3): protocol adherence is now all-or-nothing on the pointer being read; a skipping agent obeys none of the eight prior inline rules. Mitigated only by the retained fail-safe sentence.
- R2 (pre-existing, PLAN Decision 5): a project with its own text hand-edited inside the `<baton-rules>` markers loses more lines (34 vs. a few) on its next `update` than before; behavior unchanged, visibility increased.
- R3 (pre-existing, PLAN Decision 4): `lint`'s `checkAgentBlocks` skips the block check entirely when `AGENTS.md` or `CLAUDE.md` is absent, a state `BOOTSTRAP.md:60` explicitly invites for Claude-only projects.

## Out Of Scope Returned To Director

- PLAN's two open questions (both already resolved by Director's decision in this delegation): keep the fail-safe sentence (done, kept); "Know Your Role" naming role files only via the doc intro, not inline, is a readability nit reported by the Planner, not actioned here per the PLAN's own constraint against touching `PROTOCOL.md`.
- none further
