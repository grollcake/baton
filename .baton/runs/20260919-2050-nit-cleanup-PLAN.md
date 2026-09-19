# PLAN: Clear six accumulated nits, decline three

Task ID: kdfa
Date: 2026-09-19
Planner: Claude Opus 5
Status: complete

## Director Brief

- Goal: Clear the six recorded nits worth clearing from `vxym`, `wqkb`, `jiaq` and `obbv`, and record the reason three others are declined.
- Scope: `internal/baton/artifact.go`, `internal/baton/events.go`, `internal/baton/commands_test.go`, `internal/baton/lint_test.go`, `internal/baton/status_test.go`. Three production edits, three test-truthfulness edits, no other behaviour change.
- Success Criteria: The six items each have a named, verifiable effect listed under Success Criteria below, and every behaviour not named there is byte-identical or output-identical to `main`.
- Risks: A cleanup round touching several files is where an unrelated behaviour change hides. Each item below states what must not change; the Executor records per-item evidence in the RUN.
- Required Checks: `go test ./...`, `go vet ./...`, `gofmt -l .`, `go run ./cmd/baton lint`
- Executor Prompt: Implement items 1, 2, 3, 4, 7 and 9 of `.baton/runs/20260919-2050-nit-cleanup-PLAN.md`, in that order, one commit-sized edit each. Do not implement items 5, 6 or 8; they are declined with reasons in the PLAN and are out of scope. Do not touch the transition table, artifact contracts beyond item 1's error text, templates, `VERSION`, or `.baton/bin/`. The installed binary lags the working tree, so use `go run ./cmd/baton` for anything that must reflect your changes. Record in the RUN, per item, the evidence that the intended thing changed and the evidence that the named must-not-change did not.

## Goal

- Nine nits were recorded across four closed tasks. Six are fixed in this round. Three are declined on the record, with reasons, so they stop being re-raised.
- The three production edits (items 1, 7, 9) are each small and locally verifiable. The three test edits (items 2, 3, 4) change no production code at all, so they carry no behaviour risk and are separable from the production edits if review wants them split.

## Decisions

### Fix: item 1, wrap the REQUEST path error

`validateRequestPath` in `internal/baton/artifact.go:96` calls
`validateArtifactPath(eventPlanned, path)`, throws the returned error away, and
substitutes one generic message. The inner validator distinguishes four
failures (prefix, dot-dot or character set, name suffix, unsupported event), so
a mistyped REQUEST path currently loses all of that.

Fix: keep the REQUEST framing and wrap the inner error with `%w`, for example
`artifact-check: REQUEST path must be a PLAN artifact path: PLUS the wrapped
inner error`. The inner text names the event as PLANNED, which is correct: a
REQUEST path is checked as the PLAN path it forward-references.

Must not change: `validateRequestPath` still returns non-nil for exactly the
same set of inputs as today, and nil for exactly the same set. The shape-only
contract stands, no existence check is added. Both callers
(`events.go:271`, `lint.go:159`) keep reporting it the same way. No test in the
tree asserts the current literal string, and no artifact contract depends on it.

### Fix: item 2, pin the other three events on the shared lint arm

`internal/baton/lint.go:148` computes `requiresArtifact` as PLANNED, EXECUTED,
REVIEW or CLOSE and routes all four through one `checkArtifact` call. Only the
PLANNED case is pinned by a test, so exempting EXECUTED, REVIEW or CLOSE from
that arm individually would not fail anything.

Fix: in `internal/baton/lint_test.go`, add cases that drive lint over a log
whose EXECUTED, REVIEW and CLOSE lines each carry a path that `checkArtifact`
must reject, and assert lint reports each one. Reuse the existing harness and
the existing `writeRun` / `writeReview` / `writeClose` helpers; follow whatever
the PLANNED case already does rather than inventing a second style.

Must not change: production code. This item is test-only. Lint's behaviour on a
valid log is unchanged, which `go run ./cmd/baton lint` on this repository's own
`baton.log` confirms.

### Fix: item 3, correct a doc comment that claims coverage that does not exist

The doc comment on `TestPlaceholderCheckFailsClosedWithoutTemplates`
(`internal/baton/commands_test.go:82`) lists "an unreadable template file"
among its cases. The three subtests are a missing directory, an empty
directory, and templates without placeholder tokens. The unreadable-file path
does refuse, but nothing here tests it.

Fix: correct the comment to describe the three subtests that exist. Do not add
the subtest: a mode-000 file is not a reliable test input (it is readable when
the suite runs as root), and buying that coverage costs a skip guard that is
larger than the gap. The `wqkb` review verified the path by hand and recorded
it; this item only stops the repository from claiming otherwise.

Must not change: the three subtests and the code under test.

### Fix: item 4, pin per-template attribution in the single-placeholder test

`TestPlaceholderCheckRefusesSinglePlaceholder`
(`internal/baton/commands_test.go:56`) reinserts
`placeholderPattern.FindString(...)`, the first token in each template. Several
of those tokens occur in more than one template, so the test does not
demonstrate that the token came from the artifact's own matching template.

Fix: give each case an explicit token that occurs in its matching template and
in no other template. Candidates verified in the current templates:
`plan.md` "what must be true when the task is complete", `run.md`
"met, not met, partial", `review.md` "ready-for-user-decision or blockers",
`close.md` "why the run satisfies the plan and success criteria", each in the
repository's angle-bracket form. Add an assertion that the chosen token appears
in its own template and in no other file under `.baton/templates/`, so the test
fails loudly if a future template edit makes the token shared again rather than
quietly reverting to the weaker check.

Must not change: production code, and the set of four events exercised. This
item is test-only.

### Fix: item 7, close the unguarded branch in `runStatus`

`internal/baton/events.go:512` guards on finding a REVIEW record and has no
else arm. If the record were absent, `next` would stay at
`user approval -> CLOSE`, where the previous code fell through to
`EXECUTED (delegate Executor)`. It is unreachable today, because `last` comes
from `lastEvent`, so `last == eventReview` implies the record exists. It is
also the one place in that change that was not equivalent to the code it
replaced, and it leans the unsafe way.

Fix: make the fail-closed default explicit. Set `next` to
`EXECUTED (delegate Executor)` for the `last == eventReview` case unless a
readable REVIEW records `ready-for-user-decision`, so an absent record lands on
the safe side by construction rather than by the guard happening to hold.

While at these lines, add the one-line comment item 8 asks for: `runStatus` and
`nextCommand` read the REVIEW independently and must stay in agreement about
what counts as ready.

Must not change: every reachable output. All three existing status paths keep
their current text exactly, which `TestStatusAfterReviewBlockersOffersNextExecutedRound`,
`TestStatusAfterReviewReadyOffersUserApproval` and
`TestStatusAfterMissingReviewArtifactOffersNextExecutedRound` already pin.
Do not touch `reviewReadyForUser`, `nextCommand`, `reviewResult`, or the
`requireReviewReady` reader the `obbv` PLAN put out of scope.

### Fix: item 9, make the unreadable-review status line one key and one value

`events.go:525` prints `review_artifact: unreadable: PATH`, two colons in a
format where every other line is one `snake_case_key` and one value
(`task_id`, `last_event`, `next_gate`, `branch`, `next_command`, `open_tasks`).
An agent splitting on the first colon gets the value `unreadable: PATH`.

Proposed wording: `review_artifact_unreadable: PATH`. The key names the
condition, the value is the path alone, the line keeps its current position
between `branch:` and `next_command:`, and it is still emitted under exactly
the same condition.

How I know nothing depends on the current shape, given it shipped today:

- The whole repository contains six occurrences of `review_artifact`: the one
  `Fprintf` at `events.go:525` and five assertions in
  `internal/baton/status_test.go`. Nothing else in code, docs, `PROTOCOL.md`,
  the role files, the bootstrap copies, or the templates mentions it.
- No agent-facing document tells a reader to parse it. The Director, Planner and
  Executor role files never name the key, so no delegate has been instructed to
  look for it.
- It has never been printed in this repository's own use. The line appears only
  when a task's last event is REVIEW and that REVIEW artifact is unreadable on
  disk; every REVIEW artifact in `.baton/runs/` is present, and the one open
  task is this one.
- It shipped inside `obbv`, closed at 20:49 today, and `.baton/bin/baton` still
  lags the working tree, so no installed binary in any project emits it yet.

The Executor re-runs those greps and records the counts in the RUN, because the
decision rests on them.

Must not change: the condition under which the line is emitted, its position in
the output, and the path value itself. Only the key changes. The three
`status_test.go` assertions that currently match `review_artifact:` must be
updated to the new key, including the two negative assertions, which would
otherwise stop testing anything because `review_artifact_unreadable:` does not
contain `review_artifact:` as a substring.

### Decline: item 5, the role-file binary caveat

The sentences in `PLANNER.md` and `EXECUTOR.md` tell a delegate to run
`baton check-artifact`. The `wqkb` review noted that in this repository the
shipped `.baton/bin/baton` lags the working tree, so a delegate following the
sentence literally here checks with a stale binary.

Declined, because the caveat is true only for Baton's own repository and the
sentences are shipped to every installed project. `PROTOCOL.md` states that
installed projects need neither Go nor a specific shell; telling their
delegates to prefer a freshly built binary instructs them to do something they
have no toolchain for, and makes the common case read like the exception. The
place for this warning is Baton's own contributor instructions, and that is a
separate decision for Director rather than an edit smuggled into a nit round.
Editing the role files would also make them managed documents that differ from
the installed binary's embedded copies until Director rebuilds at close, which
is a real cost to pay for a caveat aimed at four files' worth of readers.

Meanwhile the gap is already covered where it matters: Director's delegation
prompts for this repository carry the `go run ./cmd/baton` instruction, as this
task's own prompt does.

### Decline: item 6, the `validClose` and `writeClose` duplication

`validClose` (`coverage_test.go:162`) and `writeClose` (`testutil_test.go:225`)
both build a CLOSE body. They differ only in the title and in two list items.

Declined, because this is not a CLOSE-only accident. The package has a
four-way parallel: `validPlan`, `validRun`, `validReview` and `validClose`
return strings that coverage tests mutate with `strings.Replace` to build
near-miss artifacts, while `writePlan`, `writeRun`, `writeReview` and
`writeClose` write files and take the round and result parameters their callers
need. The two families have different jobs and different signatures.
Collapsing only the CLOSE pair makes the set inconsistent for no gain;
collapsing all four is a test refactor that was not asked for and would touch
every test file in the package. The PLAN that produced them asked for both.

### Decline: item 8, the double read of the review artifact

One `status` run reads the REVIEW artifact twice: `runStatus` reads it for
`next_gate` and the unreadable line, and `nextCommand` reads it again through
`reviewReadyForUser`.

Declined. The two callers reading independently is the safer arrangement. Each
is a self-contained fail-closed reader: `runStatus` decides which gate to
print, `nextCommand` decides whether to offer a delegation command, and neither
can be left holding a stale answer another function computed. Removing the
second read means threading an outcome into `nextCommand`, widening its
signature for one of its callers and coupling it to `runStatus` having already
run. The cost being avoided is one `os.ReadFile` of one small file per `status`
invocation, which is not a cost worth that coupling.

The real hazard the `obbv` review named is that the two readers could drift
apart about what counts as ready. That is addressed by the one-line comment
folded into item 7, at the site where the second reader is easy to miss.

## Scope

In scope:

- `internal/baton/artifact.go`: `validateRequestPath` error wrapping only.
- `internal/baton/events.go`: the `last == eventReview` block in `runStatus`
  and the `review_artifact` output key only.
- `internal/baton/commands_test.go`: items 3 and 4.
- `internal/baton/lint_test.go`: item 2.
- `internal/baton/status_test.go`: the five `review_artifact` assertions.

Out of scope:

- Items 5, 6 and 8, declined above.
- The transition table, artifact contracts beyond item 1's error text,
  `.baton/templates/`, `VERSION`, `.baton/bin/`.
- `reviewReadyForUser`, `nextCommand`, `reviewResult`, `requireReviewReady`,
  and the `validateArtifactPath` rules themselves.
- The role files and their bootstrap copies. Nothing in this round edits a
  managed document, so no rebuild is required at close beyond the routine one.

## Plan

1. Item 1: wrap the inner error in `validateRequestPath`. Verify: a mistyped
   REQUEST path reported through `go run ./cmd/baton lint` on a scratch log now
   names which rule it broke.
2. Item 7: make the fail-closed default explicit in `runStatus` and add the
   two-readers comment. Verify: the three existing status tests pass unchanged
   before the item 9 edit.
3. Item 9: rename the key to `review_artifact_unreadable` and update the five
   assertions in `status_test.go`. Verify: the greps recorded in the RUN, and
   `go test ./...`.
4. Item 2: add the EXECUTED, REVIEW and CLOSE lint cases. Verify: each new case
   fails if the corresponding event is removed from `requiresArtifact`.
5. Item 3: correct the doc comment. Verify: the comment names exactly the three
   subtests present.
6. Item 4: pin per-template tokens and assert their uniqueness. Verify: the new
   uniqueness assertion fails if a token is duplicated into a second template.
7. Run every required check, then write the RUN with per-item evidence.

## Success Criteria

- Item 1: `validateRequestPath` returns an error that contains the inner
  validator's text, and it still returns non-nil for exactly the same inputs as
  before.
- Item 2: removing `eventExecuted`, `eventReview` or `eventClose` from
  `requiresArtifact` in `lint.go` each makes at least one named test fail. The
  Executor demonstrates all three and records which test caught each.
- Item 3: the doc comment on `TestPlaceholderCheckFailsClosedWithoutTemplates`
  names only cases that exist.
- Item 4: each of the four cases uses a token that appears in its own template
  and in no other file under `.baton/templates/`, asserted in the test, and all
  four artifacts are still refused.
- Item 7: reverting the item 7 edit alone changes no test result, and the
  post-edit code reaches `EXECUTED (delegate Executor)` for a REVIEW-last task
  without a readable ready REVIEW regardless of whether the record is found.
- Item 9: `go run ./cmd/baton status --task-id` on a scratch task whose REVIEW
  artifact has been deleted prints exactly one line
  `review_artifact_unreadable:` followed by the path, between `branch:` and
  `next_command:`, and `grep -rn "review_artifact: unreadable" .` finds nothing
  outside `.baton/runs/`.
- Whole round: `git diff main --stat` touches only the five files listed In
  Scope. No template, no role file, no `VERSION`, no `.baton/bin/`, no
  transition table entry.
- Whole round: every status output text other than the item 9 key, and every
  lint and check-artifact accept or reject decision other than item 1's error
  text, is unchanged. The RUN states this explicitly, per item.

## Validation

- `go test ./...`: passes.
- `go vet ./...`: no output.
- `gofmt -l .`: no output.
- `go run ./cmd/baton lint`: passes, ending in `baton-lint passed`.
- `git diff main --stat`: exactly the five in-scope files.
- Per-item revert checks for items 1, 2, 4, 7 and 9 as described under Success
  Criteria, run on scratch copies outside the working tree.

## Report To Director

- Suggested summary: Clear six recorded nits, decline three with reasons
- Artifact path: .baton/runs/20260919-2050-nit-cleanup-PLAN.md

## Risks Or Questions

- Item 9 changes an output key that shipped today. The evidence that nothing
  consumes it is recorded above and is re-verified by the Executor. If Director
  knows of a consumer outside this repository, item 9 should be dropped rather
  than reworded.
- Item 7 is a no-op on every reachable path by design. That makes it
  unverifiable by a failing test, which is why its success criterion is a
  revert check plus the unchanged status tests, not a new test. A new test
  would have to fake an impossible state.
- Item 2 asks for three revert demonstrations rather than one. If any of the
  three cannot be made to fail on removal, that event is not actually pinned
  and the Executor returns it to Director rather than weakening the criterion.
