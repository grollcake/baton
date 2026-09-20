# REVIEW-01: Make the two checks the protocol mandates into commands

Task ID: cqso
Date: 2026-09-20
Planner: Claude Opus 5
Run: .baton/runs/20260920-1900-verify-tooling-RUN-01.md
Status: complete

## Decision

Result: blockers

This review is evidence for the user's decision, not approval. Only the user
can approve `CLOSE`.

Both boundaries Director narrowed the review to hold. `revert-check` did not
touch this working tree under any argument I could construct, and the `--check`
string is reachable only from argv. The blockers are all in the same place: the
command answers "which test pins what" only when the reviewer already knows how
to ask, and can answer with silence when it cannot answer at all.

## Findings

### Blockers

- **`pins=` is silently absent when `--name-pattern` matches nothing.** With a
  pattern supplied and no line matching it, the command prints *no* `pins=`
  line at all -- not `pins=none`, not a reason. An agent parsing the output
  cannot tell "no test pins this change" from "the check never got far enough
  to name a test". This is reachable by the `PLAN`'s own success-criterion
  command: `revert-check --check 'go test ./internal/baton/...'
  --name-pattern '\-\-\- FAIL: (\S+)' internal/baton/revertcheck.go` reverts a
  file that did not exist at `HEAD`, so it is deleted from the copy, the
  package no longer compiles, and the output is `after_revert=fail`, zero
  `pins=` lines, and `new_output=internal/baton/app.go:143:12:
  a.runRevertCheck undefined ... [build failed]`. Decision 1 rejected a default
  pattern precisely to avoid "a silently empty `pins=` list"; this is that
  failure mode arriving by another door. Fix is one line: always print a
  `pins=` line, with `none` or a stated reason when nothing matched.
- **The `--name-pattern` example the `PLAN` relied on was not implemented, and
  `PLANNER.md` does not mention the flag.** Decision 1 justified having no
  default with "The usage text carries `--- FAIL: (\S+)` as the Go example, so
  the cost is one flag, not one guess." The shipped usage line is
  `revert-check --check <command> [--rev <ref>] [--name-pattern <re>] [--keep]
  PATH...` -- the example is nowhere in the binary, and the new `PLANNER.md`
  paragraph shows only `--check '<test command>' <changed path>...`. So, to
  Director's question: **no, an agent following `PLANNER.md` would not know to
  pass the flag**, and would get exactly the `pins=unavailable (pass
  --name-pattern)` Director got. The decision to require the flag is defensible
  only with the example the `PLAN` promised; without it the command's most
  useful output is off by default and undiscoverable from either place a
  reviewer looks.
- **`runCheckCommand` hardcodes `sh -c`** (`revertcheck.go:259`), the only
  non-`git` `exec.Command` in the shipped binary. Baton ships
  `windows-amd64` and `windows-arm64` binaries, and `PROTOCOL.md` states it
  "does not require ... a specific shell" and "can run from PowerShell, cmd, or
  Git Bash". `revert-check` is now named in the shipped `PLANNER.md`, so every
  reviewer on Windows without `sh` on `PATH` is ordered to run a command that
  fails with `run --check: exec: "sh": executable file not found`. It fails
  loudly, not silently, but it makes a shipped instruction unfollowable on two
  shipped platforms -- the same objection Decision 6 used to keep
  `make-fixture` out of `PLANNER.md`.

### Nits

- The source-grep guard is narrower than the `RUN` presents it. Verified on a
  copy: injecting `exec.Command("git", "checkout", ".")` fails
  `TestRevertCheckNeverCheckoutsOrRunsFromRecords` as claimed, but
  `exec.Command("git", "check"+"out", ".")` **passes**, and so does adding
  `os.ReadFile(".baton/runs/some-PLAN.md")` -- the record grep covers only
  `BATON-LOG`, `timelineFile`, `GUIDANCE.md`, `legacyTimelineFile`, so a future
  round that reads `Required Checks` out of a `PLAN`, the exact regression the
  `PLAN`'s risk list names, would not be caught. Today's code is correct; the
  test pins the naive case only, and the `RUN` should not be read as more.
- A directory argument is accepted and then fails with a raw internal error:
  `revert-check --check true internal/baton` prints `baseline=pass` and then
  `rename .../internal/.baton-2692727808 .../internal/baton: file exists`,
  exit 1. Nothing outside the copy is touched, but `PLANNER.md` says "`<changed
  path>...`", so an agent will pass a directory. Refuse it by name, or support
  it.
- `copy=` is printed only after the baseline run (`revertcheck.go:148`), while
  the directory is created at line 109. With `--keep`, any error during the
  file copy leaves a kept directory whose path was never printed -- contrary to
  Decision 2's "the `copy=` line is printed in both cases so a `--keep` run is
  never a directory nobody knows about".
- The replacement `PLANNER.md` paragraph drops "confirm they fail for the
  reason the change was made" and does not point the reviewer at `new_output=`,
  which is where that reason now lives. Same line count, slightly less
  instruction than it replaced.

### Judgements from the diff, not re-driven

- **`cmd/make-fixture` cannot reach an installed project.** It takes no
  positional arguments (`flag.NArg() != 0` is fatal), has no destination flag,
  and every path it writes is built from one `os.MkdirTemp` root; the only
  write outside that root is `go build -o <fixture>/.baton/bin/baton`, whose
  `-o` is inside it. `findModuleRoot` walks up for a `go.mod` containing
  `module github.com/grollcake/baton` and refuses otherwise, so it cannot even
  start in an installed project, and it reads `bootstrap/` only. `grep -rn
  make-fixture internal/ bootstrap/` is empty, so it does not ship.
- **The `PLANNER.md` edit replaces rather than adds.** `git diff` on both
  copies is `6 +++---`: three lines out, three lines in, no other line touched,
  and the two files are byte-identical.
- **Output shape:** usable for the case where the revert leaves a compiling
  tree (the `app.go` run below names four tests and the reason each fails),
  not usable without the two blockers above for the case the round's own
  success criterion names.

## Suggested User Checks

- Run `.baton/bin/baton revert-check` (or `go run ./cmd/baton revert-check`)
  with no `--name-pattern` and decide whether `pins=unavailable (pass
  --name-pattern)` is an acceptable default for a command `PLANNER.md` now
  orders every reviewer to use, or whether the example belongs in the usage
  text and the paragraph.
- Run `go run ./cmd/baton revert-check --check 'go test ./internal/baton/...'
  --name-pattern '\-\-\- FAIL: (\S+)' internal/baton/revertcheck.go` and
  confirm you agree that printing no `pins=` line at all, when the reverted
  file was new and the build broke, is a wrong answer rather than a quiet one.
- Read the three replaced lines in `.baton/PLANNER.md` (Review section) and
  judge whether a reviewer who has only that paragraph can run the command
  correctly and report the reason a test failed.
- Decide whether `revert-check` shipping to Windows with `sh -c` is acceptable
  for now, given Baton publishes `windows-amd64` and `windows-arm64` binaries.
- Run `go run ./cmd/make-fixture`, drive the printed `drive=` line once
  (`lint`, `status`), then run the printed `cleanup=` line, and judge whether
  "the agent owns the directory" is a mitigation you accept.

## Evidence Reviewed

- Boundary 1, working tree untouched: checksummed all 366 tracked and
  untracked files (excluding `.git`) before and after every run below;
  `diff` of the two `shasum -a 256` listings is empty each time, including
  `.baton/BATON-LOG.txt` and every other `.baton/` record. `git status
  --short` unchanged. No `baton-revert-*` directory survives under `TMPDIR`.
- Boundary 1, real run reproducing the `RUN`'s emphasis 3: `go run
  ./cmd/baton revert-check --check 'go test ./internal/baton/... -run
  TestRevertCheck -v' --name-pattern '--- FAIL: (\S+)' internal/baton/app.go`
  -> `baseline=pass`, `after_revert=fail`, and the same four `pins=` names the
  `RUN` reports, with 63 `new_output=` lines carrying `unknown command:
  revert-check`. Copy removed.
- Boundary 1, adversarial arguments (exit status and message each):
  `../outside.txt` -> refused, "refusing path escaping the repository";
  `/etc/passwd` -> refused, "refusing absolute path";
  `internal/../../escape.go` -> refused; `.baton/../../../tmp/x` -> refused;
  `../baton/internal/baton/app.go` -> refused even though it resolves back
  inside. No `copy=` line and no temporary directory for any refusal.
  `internal/baton/../../.baton/BATON-LOG.txt` is accepted and normalised to
  `.baton/BATON-LOG.txt`, reverted inside the copy only; the real
  `BATON-LOG.txt` is byte-identical afterwards.
- Boundary 1, hostile `--check`: `--check 'echo INTRUSION >
  /Users/rollcake/lab/baton/.baton/rc-probe.txt; ls -la >
  /Users/rollcake/lab/baton/rc-probe2.txt'` **did create both files in this
  repository**. This is Decision 3 item 5 behaving exactly as written --
  `revert-check` confines its own writes, not the caller's command -- and the
  `PLAN` states it plainly rather than overclaiming. `pwd` inside the check
  confirmed the working directory is the copy. Both probe files removed; tree
  re-verified byte-identical.
- Boundary 1, guard removal (on a copy of the tree in the scratchpad, never the
  working tree): injecting `exec.Command("git", "checkout", ".")` into
  `revertcheck.go` fails `TestRevertCheckNeverCheckoutsOrRunsFromRecords`
  with `found "checkout"`; restored, green. Evasions that pass: `"check"+"out"`
  and an `os.ReadFile` of a `PLAN` path.
- Boundary 2, argv only: `revertcheck.go` has exactly one `os.ReadFile`
  (line 133, the per-file copy loop keyed on `git ls-files` output) and one
  `gitShowRaw`; `checkCommand` is assigned once, from `requireValue(parsed,
  "--check")`, and used only at lines 143 and 184. No `PLAN`, `BATON-LOG.txt`,
  `GUIDANCE.md` or `Required Checks` path can reach it.
- Boundary 2, injection test: `TestLintDoesNotExecutePaths` passes unmodified;
  on the copy, replacing `lint.go:253`'s `validateRequestPath(record.Path)`
  with a nil error makes it fail with `lint accepted a malicious path`;
  restored, green.
- `RUN` validation claims reproduced on the copy: `gofmt -l .` empty, `go vet
  ./...` clean, `go test ./...` `ok internal/baton 12.077s`, `go run
  ./cmd/baton lint` -> `baton-lint passed`.
- `diff .baton/PLANNER.md bootstrap/.baton/PLANNER.md`: empty.
- Read in full: `internal/baton/revertcheck.go`,
  `internal/baton/revertcheck_test.go`, `cmd/make-fixture/main.go`, the
  `app.go` and `PLANNER.md` diffs, `args.go`, `files.go:atomicWrite`.

## Report To Director

- Suggested summary: REVIEW-01 confirms both boundaries -- `revert-check` left
  this repository byte-identical under every adversarial argument and its
  `--check` string is reachable only from argv -- but blocks on the answer
  side: `pins=` is silently absent when nothing matches, the usage example the
  `PLAN` promised is missing, `PLANNER.md` never names `--name-pattern`, and
  the shipped command hardcodes `sh -c` on a project that ships Windows
  binaries
- Artifact path: .baton/runs/20260920-1900-verify-tooling-REVIEW-01.md

## Required Next Step

- Ask Director for a decision: the three blockers are small, local fixes
  (always print a `pins=` line; add `--- FAIL: (\S+)` to the usage line and the
  `PLANNER.md` paragraph; choose a shell per platform in `runCheckCommand`) and
  fit one `RUN-02`. If Director judges the Windows blocker out of scope for
  this round, it needs saying rather than leaving, because `PLANNER.md` now
  ships the instruction.
