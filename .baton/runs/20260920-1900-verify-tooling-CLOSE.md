# CLOSE: Give delegates the checks the protocol asks of them

Task ID: cqso
Date: 2026-09-20
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260920-1900-verify-tooling-REVIEW-02.md

## Acceptance

- `baton revert-check` does what `PLANNER.md` tells every reviewer to do by
  hand: copy the tree, revert named paths to their committed state, run the
  check, and name which tests stopped passing. Of the 32 run and review
  artifacts recorded here, 28 describe building that setup by hand.
- `cmd/make-fixture` builds a drivable installed project and does not ship,
  because it needs Go, this module and bootstrap, none of which a project using
  Baton has.
- The working tree is never touched. The reviewer ran the command five times in
  this repository, which holds real records, and found 368 files byte-identical
  each time. Paths outside the repository, absolute paths and traversal are
  refused before anything is copied.
- The check string comes from argv and nowhere else. A shipped binary reading a
  command out of an artifact would undo what the injection test exists to prove.
- Two rounds. The first review blocked on three faults in the answer the command
  gives; the second closed them.

## Validation Summary

- `go build ./...`, `go vet ./...`, `gofmt -l .`: clean
- `go test ./...`: pass
- `baton lint` after the release build: pass
- Both guards fail their tests when removed, as does the older injection guard
- Real runs shown for a named pin, for nothing matching, and for a revert that
  breaks the build

## Plan Deviations

- The plan had the Executor bump `VERSION`. Director withheld that: every task
  here bumps at close alongside the release build, and two bumps in one round
  would be worse than none.
- The plan's answer to whether these commands ship was written as one answer.
  Director corrected it mid-plan: the fixture maker serves someone developing
  Baton and must not ship; the revert check serves every reviewer in every
  installed project, because the instruction to do it by hand ships to all of
  them.

## Lesson Candidates

- A procedure the protocol mandates and no tool performs gets reimplemented by
  hand every time, differently.

## Remaining Nits

- The Windows branch and the usage example are both untested: removing either
  leaves the suite green.
- A check string written in POSIX syntax still will not parse on Windows, and
  nothing says so. The binary runs where it ships; the string a caller passes
  may not.
- The guard that forbids mutating Git subcommands reads the source literally, so
  a split string or a file read would evade it.
- A directory passed as a path fails with a raw rename error, and `copy=` prints
  only after the baseline runs.

## Residual Risks

- This is the first shipped command that executes a caller-supplied string. The
  boundary holding it safe is that the string comes from argv only, which is a
  convention enforced by one assignment and a test rather than by the type
  system.
- `make-fixture` cannot remove what it exists to leave behind, so fixtures
  accumulate in the temporary directory until something else clears them.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Add revert-check and a repository-only fixture maker
- Artifact path: .baton/runs/20260920-1900-verify-tooling-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the untested
Windows path and the three blockers the first review found.
