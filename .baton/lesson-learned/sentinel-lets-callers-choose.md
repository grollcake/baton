# Lesson Learned: An ambiguous return value lets each caller pick which way unknown falls

Date: 2026-09-20

## Applies When

- Any helper that reports both an answer and a failure through one value; adding a caller to such a helper.

## Trigger / Symptom

- A function returns an empty string, a zero, or a nil for 'could not determine', and callers compare it against the answers instead of handling it.

## Context

- A helper read a review result and returned the empty string when the file could not be read. Three call sites each decided by hand which side that fell on. One shipped comparing against the wrong constant and opened a safety gate whenever the artifact was unreadable.

## Mistake Or Risk

- Nothing in the type forced a caller to handle the unknown, so correctness depended on every author noticing. They agreed only because someone checked.

## Resolution

- Return the unknown as its own value with the zero value on the safe side, plus an error. Then a caller that ignores the error still lands safely, and the unsafe comparison cannot be written.

## Validation

- `go test ./...` with one pinning test per call site for the unreadable case

## Future Check

- When a helper can fail, ask what a caller that ignores the failure does. If that is the unsafe direction, change the signature rather than the callers.
