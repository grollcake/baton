# CLOSE: Let a project pause Baton without removing it

Task ID: srkz
Date: 2026-09-20
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260920-1618-pause-baton-REVIEW-02.md

## Acceptance

- A project can set Baton aside with `baton pause` and come back with
  `baton resume`, keeping every record. The paused state is carried by the
  rules block itself, because a marker inside the directory is invisible to an
  agent and two sources of truth can disagree silently.
- The round trip is byte-exact beyond this repository's shape. The reviewer
  drove it on projects with different surrounding content, a missing final
  newline, trailing whitespace, and only one of the two instruction files.
- Commands that would write or advance a round refuse while paused and touch
  nothing; commands that only report work and say the project is paused.
- Update cannot quietly switch Baton back on. Both the update command and the
  manual merge route refuse while paused and print the order to follow, and a
  paused project can still take an update it asks for.
- The two lint gaps this depended on are closed: a block emptied identically in
  both files is now an error, and lint checks whichever instruction files exist
  instead of skipping when one is absent.
- Two rounds. The first review found a blocker; the second closed it.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass
- Round trip verified byte-identical apart from the four timeline lines the two
  commands append
- Partial write constructed in the shape a real project has, with the first
  file restored byte-for-byte and nothing recorded
- Every refusing command exits non-zero with the tree byte-identical before and
  after

## Plan Deviations

- The plan asked Director to settle whether pausing may append to the timeline.
  It may. The constraint meant that no existing line is modified. A project
  returning after a week must be able to see when Baton stopped and whether a
  task was open then, which the stale next gate alone does not explain.

## Lesson Candidates

- The guard that matters most is the one nothing tests. Deleting the update
  refusal left the whole suite green.

## Remaining Nits

- The regression test for the update refusal detects that the guard is gone
  rather than the damage it prevents: against the minimal upstream the run
  fails earlier, before the byte-identity assertions. The reviewer confirmed
  against a complete tree that the damage is real, so the test is load-bearing
  but pinned to the weaker of its two assertions.
- The partial-write test relies on a flag that exists only on this family of
  systems and skips elsewhere, with the reason stated. The alternative needs
  root, which would test a different precondition.
- Five further nits stand in the first review: the paused text is pinned to
  itself, a file carrying two blocks pauses only the first yet lints as paused,
  one decision's prose and table disagree about a project with no instruction
  files, a regexp is duplicated, and the restore loop discards errors.

## Residual Risks

- Lint is stricter for every project now, not only paused ones. The reviewer
  found this wider than the plan stated: not only a hand-edited block but a
  block from an earlier release now fails, which is any project that has not
  yet taken the update that rewrites it.
- The removal round cannot reuse the classifier as the plan assumed. Its
  absent state means no instruction files at all, not files carrying no block,
  which is exactly what removal produces.
- The Korean guide says nothing about pausing.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Add pause and resume, carried by the rules block
- Artifact path: .baton/runs/20260920-1618-pause-baton-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the blocker
the first review found, the weaker assertion the new test lands on, and the
wider reading of the lint strictness.
