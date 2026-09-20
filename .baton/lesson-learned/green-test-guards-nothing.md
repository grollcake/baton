# Lesson Learned: A test can be green for a reason that no longer exists

Date: 2026-09-20

## Applies When

- Reviewing a round that added or changed tests; any change that loosens or replaces a check.

## Trigger / Symptom

- A test passes, but passes just as well without the change it supposedly pins. A guard is deleted and the suite stays green.

## Context

- Three separate rounds in one day shipped a test that guarded nothing. One pinned the only guard stopping update from silently un-pausing a project. One proved a failure shape that never occurs. One asserted the absence of a string that a rename had already made impossible to produce.

## Mistake Or Risk

- Reading a test and agreeing with it says nothing about whether it discriminates. Passing tests are evidence only against the code that was there when they were written.

## Resolution

- Revert the source on a copy of the tree, run the checks, and record which tests fail and which still pass. Only the ones that fail are pinning that change. `baton revert-check` does this.

## Validation

- `baton revert-check --check '<checks>' --name-pattern '\-\-\- FAIL: (\S+)' <paths>`: names the pinning tests

## Future Check

- Before accepting that a test covers a change, delete the change and watch that test fail. If it does not, it is guarding something else.
