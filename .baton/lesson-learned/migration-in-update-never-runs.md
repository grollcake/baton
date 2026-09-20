# Lesson Learned: A migration that only runs in update never runs

Date: 2026-09-20

## Applies When

- Renaming or moving a file that installed projects hold; anything that must happen once per installation.

## Trigger / Symptom

- A migration is written into the update path and appears correct, but no installed project ever reaches it.

## Context

- The event timeline was renamed. The migration was placed in update, which the procedure runs with the installed binary. During an upgrade that is the old binary, which has no migration code, and it replaces itself, so the project lands on the new binary with the migration already skipped.

## Mistake Or Risk

- The update is driven by the version being replaced, not the version being installed. Any code added to the update path is code the upgrading project does not have yet.

## Resolution

- Migrate where the new code first writes, not where the upgrade happens. The first append under the new binary is what carries an active project forward.

## Validation

- Fixture installed from the released binary, updated once by the documented procedure, ends correct with no second update

## Future Check

- Ask which binary executes the new code during the upgrade that introduces it. If the answer is the old one, the code is unreachable.
