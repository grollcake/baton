# Lesson Learned: A confirmation prompt is not a safeguard in an agent pipeline

Date: 2026-09-20

## Applies When

- Designing any destructive or irreversible command that an agent may invoke.

## Trigger / Symptom

- A design relies on someone answering yes before something cannot be undone.

## Context

- Removal can delete a project's records. The guard chosen was not a prompt but a precondition: everything under the directory that Git does not ignore must be tracked and clean, so whatever is deleted remains in history.

## Mistake Or Risk

- In a pipeline the prompt is answered by the agent that wanted to proceed. It records consent without obtaining it.

## Resolution

- Guard irreversible actions with a machine-checkable precondition that makes the action recoverable, and refuse when it does not hold.

## Validation

- Fixture with uncommitted content refuses and deletes nothing; a clean one is allowed and the content is still readable from history

## Future Check

- If a safeguard can be satisfied by the actor it is protecting against, it is not a safeguard.
