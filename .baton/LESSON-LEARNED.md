# Lesson Learned Index

This file is the index of accepted records under `.baton/lesson-learned/`.
At the start of each recordable phase, scan this index and open only records
whose `Applies When` or `Trigger / Symptom` matches the current scope. Do not
rely only on selections made by an earlier role.

| Applies When | Trigger / Symptom | Lesson File | Resolution Summary |
|---|---|---|---|
| Reviewing a round that added or changed tests | A guard is deleted and the suite stays green | [green-test-guards-nothing.md](lesson-learned/green-test-guards-nothing.md) | Revert the source on a copy and record which tests fail; only those pin the change |
| Adding a caller to a helper that can fail | A helper returns an empty value for 'could not determine' | [sentinel-lets-callers-choose.md](lesson-learned/sentinel-lets-callers-choose.md) | Return the unknown as its own value with the safe zero, plus an error |
| Renaming or moving a file installed projects hold | A migration looks right but no project reaches it | [migration-in-update-never-runs.md](lesson-learned/migration-in-update-never-runs.md) | Migrate where the new code first writes, not in the update path |
| Designing a destructive command an agent may invoke | The design relies on someone answering yes | [prompt-is-not-a-safeguard.md](lesson-learned/prompt-is-not-a-safeguard.md) | Guard with a precondition that makes the action recoverable |
| Consolidating or trimming documents | The merged text can be followed but no longer says why | [merging-docs-loses-reasons.md](lesson-learned/merging-docs-loses-reasons.md) | Diff the previous version for reasoning, not for steps |
| Writing documentation that repeats what a tool checks | A document and the binary have quietly diverged | [prose-restating-a-rule-drifts.md](lesson-learned/prose-restating-a-rule-drifts.md) | State the rule where it is enforced and point at it |
