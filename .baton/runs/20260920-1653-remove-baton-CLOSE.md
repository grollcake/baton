# CLOSE: Let a project remove Baton and keep its records

Task ID: wbqt
Date: 2026-09-20
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260920-1653-remove-baton-REVIEW-02.md

## Acceptance

- `baton remove` takes Baton out of a project and leaves what the project did
  under it. It deletes only bytes Baton wrote and can still recognise as its
  own, and keeps anything it cannot, naming each kept path.
- Deleting records needs `--purge`, and its confirmation is a Git precondition
  rather than a prompt. Everything under the directory that Git does not ignore
  must be tracked and clean, so what is deleted is recoverable from history. A
  prompt would be answered by the agent that wanted to proceed.
- Text a person wrote inside the markers refuses the whole removal, with no
  override, because those bytes cannot be recovered from upstream.
- Removal is a dry run unless told otherwise, and prints what it would delete,
  what it would keep, and that no command will run afterwards.
- Two rounds. The review found one real bug among its nits and the second round
  fixed it.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass
- Purge guard reproduced: refuses on uncommitted or untracked content, outside
  Git, and where the directory is ignored, deleting nothing; allows a clean
  committed project, after which the timeline is still readable from history
- Hand-edited block refuses under apply, force and dry run, with the whole tree
  byte-identical
- Install then remove returns a file with project text above the block to its
  original bytes; a file holding only the block is deleted; a file with no
  final newline gains exactly one byte, as the plan predicted

## Plan Deviations

- The executor changed `artifact.go`, which the plan's scope did not name.
  Without it, lint in a removed project that still holds recorded artifacts
  failed, because the placeholder vocabulary is read from templates the removal
  deletes. The reviewer established that the relaxed path needs both the
  removed flag and missing templates, that the flag reaches it only from lint's
  own call, and that the public check still fails closed. Accepted, and
  recorded here because it was reported rather than found.

## Lesson Candidates

- A confirmation prompt is not a safeguard in an agent pipeline: the agent that
  wanted to proceed answers it. A machine-checkable precondition is.

## Remaining Nits

- `--purge` deletes ignored content under a tracked directory without naming it
  in the report.
- An instruction file that was empty before Baton was installed is deleted
  rather than restored to empty.
- Lint in a removed project no longer reports unfilled placeholders in kept
  artifacts.
- The message about a removed project is reachable in one that never had Baton.
- The round-02 test asserts the parsed paths rather than the printed refusal
  line, and does not pin their order.
- `runGit` still trims trailing whitespace, so a path ending in a space would
  be reported short. Rare, and out of this round's scope.

## Residual Risks

- Removal cannot be undone, and bootstrap refuses a project that still holds
  the directory, so reinstalling later needs an explicit re-initialisation the
  protocol does not yet describe.
- `--purge` is the first flag in Baton that can delete a record. A project that
  keeps its Baton directory untracked cannot use it at all, which is the
  correct refusal but will read as an obstacle.
- The Korean guide now lags by two features, pause and removal.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Add remove, keeping the records by default
- Artifact path: .baton/runs/20260920-1653-remove-baton-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the purge
guard, the unplanned change to the artifact check, and the nine nits.
