# Lesson Learned: Prose that restates an enforced rule drifts from it

Date: 2026-09-20

## Applies When

- Writing documentation that repeats what a tool checks; adding a rule to both a document and a check.

## Trigger / Symptom

- A document lists commands or rules that the tool also knows, and the two have quietly diverged.

## Context

- A role document carried a command table that had already fallen behind the binary: three commands existed that it did not list. Of six rules sampled in the instruction block, five were stated in the protocol as well.

## Mistake Or Risk

- Two statements of one rule need someone to keep them in step, and nothing fails when they drift.

## Resolution

- State the rule where it is enforced and point at it from elsewhere. Where the tool refuses something, let the refusal be the documentation.

## Validation

- Comparing the document against the dispatch table; `baton lint` after the edit

## Future Check

- Before writing a rule into a document, ask whether a command already refuses violations of it. If so, name the command instead.
