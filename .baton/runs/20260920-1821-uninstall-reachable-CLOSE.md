# CLOSE: Put the lifecycle procedure where the agent can read it

Task ID: kovh
Date: 2026-09-20
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260920-1821-uninstall-reachable-REVIEW-02.md

## Acceptance

- An agent inside an installed project, told in plain language to stop using
  Baton, now finds the procedure. It is in the one document that reaches such a
  project, and the protocol points at it from a section named for what the
  reader is looking for rather than from the Git section.
- A new managed document could not have done this. The planner established from
  the code, and the reviewer confirmed against a released binary, that the old
  installed binary drives an update and copies only the names it was compiled
  with, so a file introduced by an update arrives one update late. Editing a
  document already on the list is the only change that reaches an existing
  project on its first update.
- Renaming was ruled out the same way: the old binary's preflight requires every
  name it knows to exist upstream and refuses the whole update otherwise. The
  file keeps a name describing one of its four jobs, and the CLI carries the
  rest through guide aliases.
- Two root pages serve a different reader: a person deciding whether to adopt
  Baton, who has installed nothing and cannot see the installed directory.
  Procedure lives in one place; the root pages frame the decision.
- Two rounds. The first review found that a reason had been lost in the merge,
  and the second confirmed nothing further went missing in the trim.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass
- A fixture installed from the released pre-round binary, updated once by the
  documented procedure, ends holding the lifecycle content with no second
  update
- Both paired documents byte-identical; all four guide aliases print the same
  bytes

## Plan Deviations

- The plan deleted the root removal page on the reasoning that it had no
  distinct audience. That reasoning was wrong and Director reversed it: the
  audience is someone deciding whether to adopt Baton, who is reading the
  repository and has installed nothing, which is the same reason the bootstrap
  page lives at the root. The page was restored, cut down, and a companion page
  for pausing was added beside it. The plan's other half stands and shaped the
  restoration: procedure is written once, in the lifecycle document, and the
  root pages point at it rather than repeating it.
- The pause page was added after the run artifact was written, because the
  instruction arrived late, so the artifact under-reported the round until it
  was corrected in place.

## Remaining Nits

- Both root pages point at the installed lifecycle document, which the reader
  they are written for cannot open. The README uses the bootstrap path for
  exactly this reason.
- The removal page says what survives a removal but never what it takes out,
  including that an instruction file Baton wrote in full is deleted rather than
  emptied.
- Three of the pause page's bullets restate the lifecycle document nearly
  verbatim, and the run artifact describes that overlap incorrectly, saying the
  lifecycle document states one fact operationally when it states it in the
  same words.
- The intermediate version of the removal page was never committed, so the
  reviewer could verify the second trim only against the run artifact's own
  account.

## Residual Risks

- Nothing guards the root pages. One was deleted in the first round with every
  check green, and nothing would catch it happening again.
- The lifecycle document's name describes one of its four jobs. Renaming it
  needs a preflight that tolerates a name vanishing upstream, which no released
  binary has.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Fold the lifecycle into the document that reaches a project
- Artifact path: .baton/runs/20260920-1821-uninstall-reachable-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the broken
pointer in both root pages and the missing account of what a removal takes out,
which Director takes up immediately as separate work.
