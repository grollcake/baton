# Lesson Learned: Merging documents keeps the steps and loses the reasons

Date: 2026-09-20

## Applies When

- Consolidating two documents; trimming a document; moving guidance between files.

## Trigger / Symptom

- The merged document can still be followed, but no longer says why any of it is that way.

## Context

- A removal page was folded into a lifecycle document. Every step survived. The sentence saying that a request to stop using the tool usually means pausing, not removing, did not. That sentence was the one turning an irreversible action into a reversible one.

## Mistake Or Risk

- Steps are easy to check for and reasons are not, so a merge reviewed for completeness passes while the judgement is gone.

## Resolution

- Recover the previous version from history and diff it for reasoning specifically, naming each reason and where it landed. A reason that exists nowhere afterwards is a loss even when every step survived.

## Validation

- `git show <rev>:<path>` compared item by item against the merged document

## Future Check

- After a merge, list what the old document explained rather than what it instructed, and find each one in the new text.
