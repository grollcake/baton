# REVIEW-02: Let a project remove Baton and keep its records

Task ID: wbqt
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1653-remove-baton-RUN-02.md
Status: complete

## Decision

Result: ready-for-user-decision

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

Director narrowed this review to three questions: is the fix correct, is the
new test load-bearing, and did round 02 stay in scope. `remove` was never run
against this repository in any form; every run below is a fresh fixture under
the session scratchpad, built by copying `bootstrap/` into it and driven by a
binary built from the working tree into the scratchpad, never `.baton/bin/baton`.

## What round 02 actually changed

Director's `git diff --stat` lists twelve files, which is round 01's work still
sitting uncommitted in the working tree, not round 02's. Round 02 is a strict
subset of it. Evidence, from file modification times against the artifacts
that bracket the round (RUN-01 written 17:20:39, REVIEW-01 17:26:28):

```
$ find . -type f -newermt "2026-09-20 17:26:29"   # everything after REVIEW-01
./internal/baton/remove_test.go      17:55:40
./internal/baton/remove.go           17:56:03
./.baton/BATON-LOG.txt               17:57:06   # Director-owned
./.baton/runs/...-RUN-02.md          17:57:06   # the round's own artifact
```

Every other changed file — `update.go` 17:07, `pause.go`/`pause_test.go` 17:12,
`artifact.go` 17:17, `lint.go` 17:18, `README.md` 17:18, both `PROTOCOL.md`
copies 17:18, `app.go` 17:11, `merge.go` 17:05, `docs.go` 17:04 — predates
RUN-01. So round 02 touched two source files and nothing else. RUN-01 and
REVIEW-01 are byte-untouched (17:20:39 and 17:26:28, unchanged).

## 1. The fix is correct

Fixture: `bootstrap/` copied out, committed, then three offending paths made —
two tracked files modified in the worktree (status ` M`, so the first one is
the shifted line) and one untracked file, deliberately more than the two the
Executor's test uses.

```
$ git status --porcelain --untracked-files=all -- .baton | od -c
0000000       M   .baton/GUIDANCE.md \n   M   .baton/PROTOCOL.md \n
               ? ? space .baton/ZZZ-extra.md \n
```

(The raw first line does begin with the separator space; it is what `runGit`'s
whole-output `TrimSpace` eats.)

```
$ baton remove --apply --purge
not committed: .baton/GUIDANCE.md
not committed: .baton/PROTOCOL.md
not committed: .baton/ZZZ-extra.md
cannot --purge: 3 path(s) under .baton/ are not committed and clean
exit=1

after: .baton/ still present; git status still shows the same 3 entries
```

All three paths are now whole, including the second ` M` line, which the old
code already printed correctly and the fix must not break. The guard still
refuses and still deletes nothing.

The fix is at the call site (`remove.go:238-256`), and `runGit` is genuinely
untouched: `git diff internal/baton/app.go` is round 01's two hunks only (a
`remove` case in the dispatch switch and a `remove` line in the help text).
The callers the Executor named — `purgePrecondition`'s own `ls-files` and
`rev-parse HEAD`, and `lint.go`'s Git-tracking checks — are therefore not at
risk this round; no audit of them was needed, because none of them changed and
none of them sees different bytes than before.

The fix's per-line rule is the right shape for the defect. `TrimSpace` over the
whole output can only shorten the *start* of the whole string, so only line 0
can be shifted, and only by the one leading space. `line[2] != ' '` on line 0
detects exactly that shift (a `git status --porcelain` line always has its
separator space at index 2 when intact), and every other line keeps `line[3:]`.
Reproduced above with an intact `??` line and a shifted ` M` line in the same
output.

## 2. The new test is load-bearing

On a copy of the tree, `purgePrecondition`'s loop was reverted to the round-01
unconditional `line[3:]`, the new test left in place:

```
$ go test ./internal/baton/ -run TestPurgePrecondition -count=1 -v
    remove_test.go:557: first blocked path is missing its leading byte:
        [baton/GUIDANCE.md .baton/ZZZ-extra.md]
--- FAIL: TestPurgePreconditionReportsFullPathForFirstLine
```

It fails, and it fails for the reason the change was made — the truncated path,
not some incidental assertion. The full suite on that same reverted copy
(`go test ./...`) fails on that one test and nothing else, which also shows no
RUN-01 test was already covering this. With the fix restored, the full suite
passes; `gofmt -l .` and `go vet ./...` are clean on the working tree.

The test also earns the shape RUN-02 claims for it: it asserts the fixture
really produced a shifted first line before calling anything, and it asserts a
second, unshifted offending path, so a fix that simply dropped the first
reported line, or special-cased "line 0" without the shift test, would not pass
it. Small weaknesses are recorded as N8.

## 3. Round 02 stayed in scope

- The six nits REVIEW-01 left for Director (N2-N7) are untouched. Their
  subjects are `remove.go`'s ignored-content behaviour (N2), the empty-file
  round trip (N3), `artifact.go`'s `removed` bypass (N4), `pause.go`'s
  `blockAbsent` (N5), `artifact.go`'s plan-scope deviation (N6) and the two
  carried risks (N7) — and every file involved other than `remove.go` has a
  pre-RUN-01 mtime. RUN-02 returns all six to Director rather than acting on
  them, which is what Director asked for.
- Nothing outside the fix and its test changed, per the mtime evidence above.
- Behaviour REVIEW-01 established is unchanged, re-driven against the round-02
  binary on fresh fixtures: a fully committed `.baton/` still purges
  (`Baton 0.32.0 purged; .baton/ deleted in full (kept at commit 1194952...)`,
  exit 0, tree left holding only `.git`); a hand-edited `<baton-rules>` block
  still refuses under `--apply --force` with the same message, exit 1; and the
  dry run still prints the version, both instruction-file lines and the
  `Delete:` list.

## Findings

### Blockers

- none

### Nits

- **N8 — the test asserts the parsed list, not the printed message, and by
  substring.** `TestPurgePreconditionReportsFullPathForFirstLine` calls
  `purgePrecondition` directly and checks `strings.Contains(joined,
  ".baton/GUIDANCE.md")`, so it would also accept a path with extra leading
  bytes, and it does not pin the order of `blocked`. The user-visible string is
  the `not committed: %s` line in `runRemove`, which no test covers. Reproduced
  by hand above instead, and the defect this round fixed was a *missing* byte,
  which the assertion does catch — hence a nit, not a blocker.
- **N9 — the same whole-output trim still eats a trailing byte, unfixed and
  unnamed.** `runGit`'s `TrimSpace` also trims the *end* of the output, so a
  path ending in a space on the last status line would be reported short, and
  `strings.TrimSpace(line[3:])` would eat it anyway. Git does not quote such
  paths, so the case is real but vanishingly rare, and it is out of this
  round's stated scope. Recorded so the fix is read as "the leading-byte shift
  is closed", not "the trim is now harmless".
- **N2-N7 from REVIEW-01 stand unchanged**, deliberately: Director scoped this
  round to N1 alone and RUN-02 returns the other six.

## Suggested User Checks

1. In a throwaway copy of any Baton project, commit everything, then edit one
   tracked file under `.baton/` without staging it and create one untracked
   file under `.baton/`. Run `baton remove --apply --purge` and confirm every
   path it prints is one you can actually open — this is the defect the round
   fixed.
2. In that same copy, confirm the refusal deleted nothing: `.baton/` is still
   there, `AGENTS.md` and `CLAUDE.md` still carry their blocks, and
   `git status` shows the same entries as before the run.
3. Decide whether the "not committed:" lines plus the count line are the message
   you want, or whether it should say what to do next (commit, stash, or drop
   `--purge`). The round fixed the path; it did not revisit the wording.
4. On a fully committed copy, run `baton remove --apply --purge` and confirm
   the success path is untouched by this round: it exits 0, names the commit
   that holds `.baton/`, and leaves the tree without `.baton/`.
5. Confirm you are reading round 02 and not round 01: `git diff --stat` in this
   repository still lists twelve files, because round 01's work is uncommitted.
   Round 02 changed only `internal/baton/remove.go` and
   `internal/baton/remove_test.go`.

## Evidence Reviewed

- `.baton/runs/20260920-1653-remove-baton-REVIEW-01.md` (N1's description),
  `.baton/runs/20260920-1653-remove-baton-RUN-02.md`
- `internal/baton/remove.go:217-264`, `internal/baton/remove_test.go:519-564`
- `git diff internal/baton/app.go` (`runGit` unchanged),
  `git diff --stat`, `git diff .baton/BATON-LOG.txt`
- `find -newermt` mtime census of the working tree against RUN-01/REVIEW-01
- Scratchpad copy of the source tree with the fix reverted:
  `go test ./internal/baton/ -run TestPurgePrecondition -count=1 -v` (FAIL) and
  `go test ./...` (that test only)
- Working tree: `go test ./...` (pass, via the copy), `go vet ./...`,
  `gofmt -l .` (both clean)
- Four scratchpad fixtures built from `bootstrap/`: three-offender dirty purge,
  clean purge, hand-edited block under `--force`, pristine dry run

## Report To Director

- Suggested summary: REVIEW-02 confirmed the purge refusal now names whole paths, the new test fails without the fix, and round 02 touched only `remove.go` and its test; no blockers, two new nits
- Artifact path: `.baton/runs/20260920-1653-remove-baton-REVIEW-02.md`

## Required Next Step

- ask Director to request user approval
