# REVIEW-01: Placeholder check accepts documented command syntax

Task ID: wqkb
Date: 2026-09-19
Planner: Planner (Claude Opus 5)
Run: .baton/runs/20260919-1902-placeholder-check-RUN-01.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

## Findings

### Blockers

- none.

### Nits

- The RUN's Unresolved Risks says EXECUTED artifacts keep "the separate
  unresolved-task-marker check as a second line of defense" against a
  hand-invented marker. That is overstated. The EXECUTED-only check matches the
  literal substring `TODO` and nothing else, so an invented bracketed marker is
  now accepted for EXECUTED exactly as it is for PLANNED, REVIEW, and CLOSE. I
  confirmed this with a freshly built binary on a complete, otherwise-valid RUN
  fixture, and confirmed the pre-change binary refused the same fixture. The
  PLAN's narrower phrasing ("EXECUTED artifacts keep the separate TODO check")
  is literally true but reads as coverage it does not provide. Worth correcting
  in the CLOSE so the residual gap is recorded accurately.
- The doc comment on `TestPlaceholderCheckFailsClosedWithoutTemplates` says its
  cases include "an unreadable template file", but the three subtests are a
  missing directory, an empty directory, and templates without placeholder
  tokens. The unreadable-file path exists in the code and does refuse (verified
  by hand, see Evidence), but no test covers it. Either add the subtest or fix
  the comment.
- `TestPlaceholderCheckRefusesSinglePlaceholder` reinserts the *first* token
  found in each matching template, which for several templates is a token that
  also occurs in other templates. The test therefore does not strictly
  demonstrate per-template attribution. Behaviour is correct regardless: I
  reproduced the case by hand with a token unique to each template and all four
  kinds were refused.
- When the templates directory itself is unreadable (directory mode 000), the
  glob returns no matches and the error says "no template files found", which
  is misleading but still fails closed. Cosmetic only.
- The new role-file sentences say to run `baton check-artifact`. In this
  repository the shipped `.baton/bin/baton` lags the working tree, so a
  delegate following the sentence literally here would check with a stale
  binary. Harmless for installed projects; a reader of this repo may want the
  `go run ./cmd/baton` caveat that the PLAN's Executor Prompt carried.

## Evidence Reviewed

- Unmodified templates still refused. On a scratch fixture project holding only
  a copy of `.baton/templates/` and a `.baton/runs/` directory, a freshly built
  binary refused an unmodified copy of `plan.md`, `run.md`, `review.md`, and
  `close.md` for PLANNED, EXECUTED, REVIEW, and CLOSE respectively, each naming
  a distinct offending token. Checked directly, not by reading the tests.
- Partial filling still caught. Four complete, valid artifacts (the four
  `20260919-1837-request-plan-path` artifacts) were confirmed to be accepted
  as a baseline, then each had exactly one placeholder drawn from its matching
  template appended. All four were refused, naming that token. This is the
  PLAN's main claim and it holds.
- The two task `vxym` lines are accepted. The same four baseline artifacts,
  with `baton prompt plan --task-id <id> --key <key>` and
  `--path <valid PLAN path with no file>` appended, were all accepted for their
  respective events, and the artifacts were otherwise valid (they pass with the
  lines removed). The pre-change binary refused the same files. The RUN tested
  only EXECUTED and PLANNED; REVIEW and CLOSE also pass.
- Failure direction when the vocabulary cannot be read. Constructed four cases
  by hand against a valid artifact: templates directory absent, templates
  directory present but empty, templates directory holding a `.md` file with no
  bracketed token, and a template file with mode 000. All four refused with an
  `artifact-check:` error; none accepted. A fifth case, the templates directory
  itself at mode 000, also refused. The implementation matches what the RUN
  claims. It cannot reach an empty vocabulary that silently accepts, because
  `templatePlaceholderTokens` returns an error on an empty glob, on any read
  failure, and on a zero-size token set, and `checkArtifact` returns that error
  before any other rule runs. There is no code path that substitutes an empty
  map. This is a genuine fail-closed design, not a claim taken on trust.
- Residual vocabulary risk, tested. A *partial* template set does yield a
  non-empty vocabulary and so does not fail closed. I deleted `plan.md` from a
  fixture and then reduced the directory to `concurrency.md` alone; in both
  cases an unmodified `plan.md` copy was still refused, because tokens overlap
  across templates. The hole is real in principle but hard to trigger with a
  template set resembling Baton's own. Not a blocker.
- What the narrowing gave up, concretely. An artifact of any of the four kinds
  that is complete except for an author-invented bracketed marker such as a
  bracketed TBD or "fill this in later" was refused by the pre-change binary
  for all four events and is accepted by the new binary for all four events,
  EXECUTED included. That is the whole of the regression in strictness: the
  check no longer sees markers that did not come from a template. Text quoted
  verbatim from a template is also now refused wherever it appears, which is
  why this review paraphrases template wording.
- Role-file sentences. `cmp` reports `.baton/PLANNER.md` byte-identical to
  `bootstrap/.baton/PLANNER.md`, and the same for `EXECUTOR.md`. Both sentences
  are present in the Completion Marker sections, both name the concrete
  `check-artifact` invocation for that role, and both state that lint only
  inspects artifacts already in `baton.log`. Judged against the `vxym`
  incident, where the Executor reported complete on the strength of lint: the
  wording addresses it directly, because it names the specific command to run,
  places it last before reporting, and removes the exact mistaken inference by
  explaining why lint cannot see an unappended artifact. It is instruction
  rather than enforcement, so it depends on the delegate reading the role file,
  but it is the right instruction.
- Load-bearing. On a copied tree with only `internal/baton/artifact.go`
  reverted, `TestPlaceholderCheckAcceptsDocumentedSyntax` fails (the blanket
  scan refuses the `vxym` lines) and all three subtests of
  `TestPlaceholderCheckFailsClosedWithoutTemplates` fail (the old path never
  derives a vocabulary, so it never fails closed).
  `TestPlaceholderCheckRefusesSinglePlaceholder` still passes on the reverted
  tree, because the old blanket scan already caught a leftover token; it guards
  preserved behaviour rather than new behaviour. This matches the RUN's own
  account exactly, including its admission about the third test. The working
  tree was not modified.
- Scope. `git status --porcelain` shows modifications only to the two role
  files in both locations, `baton.log`, `internal/baton/artifact.go`, and
  `internal/baton/commands_test.go`, plus the two untracked artifacts for this
  task. No existing file under `.baton/runs/` was edited; `.baton/VERSION` and
  `.baton/bin/` are untouched.
- Required checks, all clean: `go test ./...` passes, `go vet ./...` reports
  nothing, `gofmt -l .` prints nothing, `go run ./cmd/baton lint` prints
  `baton-lint passed` including the managed-document comparison.

## Suggested User Checks

- Copy `.baton/templates/close.md` to a throwaway path under `.baton/runs/`
  ending in `-CLOSE.md` and run `go run ./cmd/baton check-artifact CLOSE` on
  it. It should refuse and name a placeholder. Delete the throwaway file after.
- Take any completed artifact under `.baton/runs/`, copy it to a throwaway
  name, paste one bracketed phrase from the matching template into it, and
  check it. It should refuse and name that phrase. Then remove the phrase and
  check again: it should pass.
- Paste a line such as a command example using angle brackets for its argument
  names into that same throwaway copy and check it. It should pass. This is the
  behaviour task `vxym` was blocked on.
- Decide whether the accepted tradeoff is acceptable to you: an artifact whose
  author writes their own bracketed marker, for example a bracketed TBD, is now
  accepted for every artifact kind including RUN. Only the literal word TODO is
  still caught, and only in a RUN.
- Read the new paragraph in `.baton/PLANNER.md` and `.baton/EXECUTOR.md` and
  judge whether it is clear enough that a delegate will actually run the
  self-check rather than rely on lint.

## Report To Director

- Suggested summary: Placeholder check verified: templates and single leftover placeholders still refused for all four kinds, vxym command syntax accepted, vocabulary failures fail closed; no blockers, one documentation nit about EXECUTED coverage
- Artifact path: .baton/runs/20260919-1902-placeholder-check-REVIEW-01.md

## Required Next Step

- Ask Director to request user approval.
