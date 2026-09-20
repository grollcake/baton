# CLOSE: Rename the timeline so a gitignore cannot swallow it

Task ID: nnah
Date: 2026-09-20
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260920-1531-timeline-rename-REVIEW-01.md

## Acceptance

- The handoff record is now `.baton/BATON-LOG.txt`. The `*.log` rule that most
  projects carry no longer matches it, and lint no longer reports that the
  directory is fine while the record inside it is ignored.
- Migration happens on the first append, not only in update. The planner found
  that an update-only migration would never run: the update procedure tells a
  project to use its installed binary, which during an upgrade is the old one
  and carries no migration code.
- Reading the old name still works and will keep working. No removal date was
  set, because none could be justified without knowing what is installed
  anywhere.
- Every command refuses while both names exist, and the refusal happens before
  update copies anything. The reviewer proved the ordering with an upstream
  deliberately made to differ, so a passing refusal could not be mistaken for a
  copy that changed nothing.
- No timeline was lost, truncated, merged or silently replaced in any scenario
  the reviewer drove, and an old binary meeting only the new name fails loudly
  rather than starting empty.
- One round, no blockers.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass, with this repository's ignored
  binary directory carved out exactly and nothing wider
- Git records the move as a rename, and the old content is a strict prefix of
  the new file
- Four migration scenarios and the binary-by-file matrix driven on scratch
  installations and throwaway worktrees, never on the working tree

## Plan Deviations

- The target name changed after the plan was requested and after the REQUEST
  line was recorded. The request says TIMELINE.txt; the work delivered
  BATON-LOG.txt. Director changed it for recognition during migration: every
  existing project holds a baton.log, and a name that still reads as the same
  artifact needs no explanation at the moment it is renamed. The cost accepted
  is that the name keeps the word log, so it keeps part of the misclassification
  the rename was meant to correct, and its prefix restates the directory it sits
  in. The recorded line was left alone as append-only history; the correct name
  is on the record from the PLANNED event onward.
- The executor made one commit on the task branch, which every earlier round
  left to Director. It did so to verify that Git tracks the move as a rename,
  which needs a commit, and reported it rather than leaving it to be found. The
  protocol assigns the timeline and task state to Director but says nothing
  about commits, so this is a gap rather than a violation. Recorded here; not
  addressed in this round, because stating a new rule was out of scope.

## Lesson Candidates

- A migration that only runs in update never runs, because the update is
  driven by the binary being replaced. Migrate where the new code first writes.

## Remaining Nits

- The update preflight re-implements the two-name check inline instead of using
  the single resolver the plan called for. It is the one place the rule slips
  and the only pattern a future caller could copy.
- The both-present branch inside the migration helper is unreachable, and the
  test covering the update refusal passes for the wrong reason, because its
  harness upstream is identical to the target.
- The preserve lists in the update and bootstrap documents dropped the legacy
  name that the update command's own dry run still prints.
- In a project that is not a Git repository, the ignore check says nothing at
  all.

## Residual Risks

- The fallback is permanent by decision. The reviewer judged it cheap to keep
  and not spreading, since it is two constants and one resolver, with the
  preflight inline pair as the only exception.
- Projects that ignore their runs directory or a managed document will see the
  new per-path check fail. That is the defect surfacing, but it will arrive as
  a new failure in a working installation.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Rename the timeline to BATON-LOG.txt and migrate on first append
- Artifact path: .baton/runs/20260920-1531-timeline-rename-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the five
nits, the executor's commit, and the name change recorded against the request.
