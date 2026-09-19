# REVIEW-01: Clear six accumulated nits, decline three

Task ID: kdfa
Date: 2026-09-19
Planner: Claude Opus 5
Run: .baton/runs/20260919-2050-nit-cleanup-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

Every claim below was reproduced by this reviewer on binaries and copied trees
built here, not read off RUN-01.

## Findings

### Blockers

- none

### Nits

- PLAN item 1 and RUN-01 both say the inner validator distinguishes four
  failures for a REQUEST path. Only three messages are reachable through
  `validateRequestPath`: it hardcodes `eventPlanned`, so the unsupported-event
  rule can never surface, and the dot-dot and character-set rules share one
  message. Three distinct messages is still a clear improvement on the one
  generic message `main` emitted; the count in the record is simply off.
- RUN-01's item 9 paragraph contains a half-finished sentence
  ("...prefix-terminated match... the converse also holds"). The reasoning it
  is reaching for is correct and the corrected assertions are verified below,
  but the prose is not readable as recorded.
- `baton gate before-approval` on a task whose REVIEW artifact is missing
  reports a bare `open <absolute path>: no such file or directory` with no
  `gate:` framing. This is pre-existing and byte-identical on `main`; it is
  not this round's doing and is out of scope. Noted so it is not re-discovered
  as new.

## Suggested User Checks

- Drive a scratch project to REVIEW, delete the REVIEW artifact, and run
  `go run ./cmd/baton status --task-id <id>`. Confirm one line
  `review_artifact_unreadable:` followed by the path, sitting between
  `branch:` and `next_command:`, and `next_gate: EXECUTED (delegate Executor)`.
- Repeat with the REVIEW artifact left in place, once reporting ready and once
  reporting blockers. Confirm no `review_artifact_unreadable:` line appears in
  either, and that the ready case offers user approval while the blockers case
  offers another EXECUTED round.
- Run `go run ./cmd/baton append REQUEST --task-id <id> --role Director
  --summary s --path .baton/runs/x-RUN-01.md` in a scratch project and confirm
  the refusal now names which rule broke, not just that the path is wrong.
- Read the new comment above the `last == eventReview` block in
  `internal/baton/events.go` and confirm it states the two-readers hazard
  clearly enough for a future editor of `nextCommand`.
- Skim `git diff main --stat` and confirm the change touches only five files
  under `internal/baton/` plus the Director-owned `.baton/baton.log`.

## Evidence Reviewed

- Scope. `git diff main --stat` touches `internal/baton/artifact.go`,
  `commands_test.go`, `events.go`, `lint_test.go`, `status_test.go` and
  `.baton/baton.log` only. No transition table entry, template, role file,
  bootstrap copy, `VERSION` or `.baton/bin/` change. `git status --porcelain`
  agrees.
- Whole-output comparison, done here rather than taken from RUN-01. Built
  `baton-main` from a `main` worktree and `baton-branch` from this tree, then
  drove each through `new-round -> PLANNED -> EXECUTED -> REVIEW` into fresh
  scratch projects for three states: readable ready REVIEW, readable blocked
  REVIEW, and a REVIEW whose artifact was deleted after the append. Captured
  `status --task-id` plus all three `gate` subcommands per state. Ready and
  blocked: identical line for line. Deleted-artifact state: one line differs,
  `review_artifact: unreadable: PATH` against
  `review_artifact_unreadable: PATH`; the only other delta is the scratch
  directory name inside the pre-existing gate error. Nothing moved beyond
  item 9's intended change.
- `check-artifact` comparison. Ran both binaries over a 23-case matrix: valid,
  wrong Status, wrong Task ID, leftover placeholder, missing file and wrong
  name for each of PLANNED, EXECUTED, REVIEW and CLOSE, plus an invalid Result
  for REVIEW, a missing approval line for CLOSE, an unsupported event, a path
  outside `.baton/runs/`, and a dot-dot path. Output identical between the two
  binaries in every case, so no accept-or-reject decision and no message text
  moved outside the REQUEST path.
- Item 1. Drove all reachable REQUEST failure modes through `append REQUEST`
  on both binaries, one fresh project per case. `main` returns the same generic
  sentence for all four inputs. This branch returns the REQUEST framing plus a
  distinct inner reason each time: path not under `.baton/runs/`, unsupported
  characters (for both the dot-dot and the space input), and artifact name does
  not match its event. See the first nit on the count.
- Item 7. Verified by reading, not by trusting: `last` comes from `lastEvent`,
  which returns the event of the last record carrying that task ID, so
  `last == eventReview` implies `lastRecord(records, taskID, eventReview)`
  finds a record. The path the item hardens is therefore still unreachable for
  exactly the reason RUN-01 gives. The fail-closed default is explicit: `next`
  is set to `EXECUTED (delegate Executor)` before the lookup. The previously
  non-equivalent arm is now equivalent and narrower: `reviewResult` returns
  `reviewUnknown` on a read failure, so the old `outcome != reviewReady` and
  the new `err == nil && outcome == reviewReady` select the same states, with
  the new form no longer able to leave user approval standing on an error.
  The three status states compared above confirm no reachable output moved.
- Item 9, corrected negative assertions. On a copied tree, injected a bug into
  `runStatus` that sets `unreadableReviewPath` whenever a REVIEW record is
  found, so the line is emitted for readable reviews too. With the renamed
  assertions, `TestStatusAfterReviewBlockersOffersNextExecutedRound` and
  `TestStatusAfterReviewReadyOffersUserApproval` both fail. Reverting only
  those two assertions to the old `review_artifact:` substring against the same
  injected bug: both pass. The pair would have gone silent; they did not.
- Item 2, individual pinning. On a copied tree, removed each of
  `eventExecuted`, `eventReview` and `eventClose` from `requiresArtifact` in
  `lint.go` one at a time and ran the lint tests. Each removal produced exactly
  one failure, and a different one each time:
  `TestLintRejectsInvalidExecutedArtifact`,
  `TestLintRejectsInvalidReviewArtifact`, `TestLintRejectsInvalidCloseArtifact`.
  The three arms are pinned separately, not by one shared test.
- Items 2, 3 and 4 changed no production code: the diff for those items is
  confined to `lint_test.go` and `commands_test.go`.
- Item 4, token uniqueness. Grepped each of the four chosen tokens across every
  file under `bootstrap/.baton/templates/` and `.baton/templates/`. Each token
  appears in its own template and in no other, in both copies. Appended
  `plan.md`'s token to `bootstrap/.baton/templates/run.md` on a copied tree:
  the PLANNED subtest fails, naming the token, its own template and the
  template that now also carries it. The uniqueness assertion is load-bearing.
- Item 3. The corrected comment names a missing directory, an empty one, and
  templates with no placeholder tokens. Those are exactly the three subtests
  present; the unreadable-file claim is gone and no subtest was added.
- Declined items untouched. `PLANNER.md` and `EXECUTOR.md` are not in the diff
  (item 5). `coverage_test.go` and `testutil_test.go` are not in the diff, so
  `validClose` and `writeClose` still stand apart (item 6). `reviewReadyForUser`
  and `nextCommand` are unchanged, so the second read of the REVIEW artifact
  remains (item 8); only the explanatory comment the PLAN folded into item 7
  was added.
- Old key does not survive in live code or docs. `review_artifact` now occurs
  six times: once in `events.go` and five times in `status_test.go`, all under
  the new key. Remaining hits of the old spelling are confined to append-only
  `.baton/runs/` history.
- Required checks, rerun here: `go test ./...` passes, `go vet ./...` silent,
  `gofmt -l .` silent, `go run ./cmd/baton lint` ends `baton-lint passed`.

## Report To Director

- Suggested summary: Review 01 of kdfa: six nits cleared, no blockers, three
  recording nits
- Artifact path: .baton/runs/20260919-2050-nit-cleanup-REVIEW-01.md

## Required Next Step

- ask Director to request user approval
