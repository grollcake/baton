# CLOSE: Reduce the agent rules block to a pointer

Task ID: yitg
Date: 2026-09-20
Director: Claude Opus 5
Approved By: User
Reviewed Evidence: .baton/runs/20260920-1605-rules-one-line-REVIEW-01.md

## Acceptance

- The block in the instruction files is now four sentences that send the reader
  to the protocol, in place of thirty-four lines that restated it. The protocol
  is written in one file, so changing a rule is one edit rather than three.
- The chain closes. The reviewer started from the block text alone and followed
  it: the protocol names the role files before it classifies the reader, so a
  Director reaches its own file without inference.
- No rule was lost. The reviewer re-checked every rule the block used to carry
  and found each one stated in the protocol, with line numbers recorded. One
  clause about returning a short status lives in the Director file instead,
  which is on the chain for the only role it binds.
- The four copies are byte-identical to each other and to the planned text, and
  nothing outside the markers changed in any of them.
- A project still carrying the old block receives the pointer on update: the
  merge splices in place, verified against a scratch project.
- One round, no blockers.

## Validation Summary

- `go test ./...`: pass
- `go vet ./...`: clean
- `gofmt -l .`: no output
- `baton lint` after the release build: pass, including the line reporting the
  two blocks match
- Five lint mutations driven on a copy: deleting the block from one file fails,
  from both fails, emptying it in one file fails; rewording it identically in
  both passes, which it did before this round too

## Plan Deviations

- none. The plan's one open question, whether to keep the fail-safe sentence,
  was answered by Director before execution: keep it. The reviewer judged it
  earns the one clause it duplicates, because it is the only text an agent that
  skips the protocol still reads.

## Lesson Candidates

- Prose that restates a rule the tool enforces will drift from it. Of six rules
  sampled in the instruction block, five were already stated elsewhere.

## Remaining Nits

- The plan placed the short-status clause in the protocol; it is in the
  Director file. The reviewer corrected the citation. Nothing was lost, only
  cited one file off.
- A block emptied identically in both files passes lint, because the two still
  match. That predates this round, and it is now also the shape a project would
  produce by pausing Baton by hand, so the pause work must decide how an
  intended emptying is told apart from an accidental one.
- Lint skips the block check entirely when one instruction file is absent. That
  also predates this round, but the cost changed: it used to mean missing some
  restated rules, and now means missing the only pointer to the protocol.

## Residual Risks

- Accepted and unchanged from the plan: an agent that reads the instruction file
  and never opens the protocol used to obey eight rules and now obeys none, and
  its non-compliance looks like ordinary work rather than a violation. The
  fail-safe sentence does not force the read; it makes a skipped read harmless
  by keeping such an agent out of the timeline.

## Director Log Action

- Event to append: `CLOSE`
- Suggested summary: Reduce the rules block to a pointer at the protocol
- Artifact path: .baton/runs/20260920-1605-rules-one-line-CLOSE.md

## Approval Basis

Director writes this artifact and appends the final `CLOSE` event only after
explicit user approval. The user approved after the report naming the accepted
all-or-nothing risk, the corrected citation, and the two lint gaps the pause
work inherits.
