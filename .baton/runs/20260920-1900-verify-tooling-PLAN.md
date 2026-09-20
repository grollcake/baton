# PLAN: Make the two checks the protocol mandates into commands

Task ID: cqso
Date: 2026-09-20
Planner: Claude Opus 5
Status: complete

## Director Brief

- Goal: turn the two verification procedures `PLANNER.md` already mandates into
  commands, one shipped (`baton revert-check`) and one repository-only
  (`go run ./cmd/make-fixture`), without changing any existing command.
- Scope: new `internal/baton/revertcheck.go` plus tests and one dispatch/usage
  line; new `cmd/make-fixture/main.go`; the load-bearing paragraph in both
  copies of `PLANNER.md`; `VERSION`; one `GUIDANCE.md` convention line.
- Success Criteria: `baton revert-check --check 'go test ./internal/baton/...'
  internal/baton/revertcheck.go` run in this repository reports the round's own
  new tests as pinned, writes nothing outside its temporary directory, and
  leaves nothing behind; `go run ./cmd/make-fixture` prints a fixture an agent
  can drive with no further setup.
- Risks: a shipped command that runs a caller-supplied command string is new
  behaviour for this binary; the boundary is that the string comes from argv
  only, never from a record. A `revert-check` that could write outside its copy
  would be worse than no command.
- Required Checks: `go build ./...`, `go vet ./...`, `gofmt -l .`,
  `go test ./...`, `go run ./cmd/baton lint`
- Executor Prompt: Implement the six decisions in
  `.baton/runs/20260920-1900-verify-tooling-PLAN.md`. Add `baton revert-check`
  (`internal/baton/revertcheck.go`, dispatch case, usage line, tests) and a
  repository-only `cmd/make-fixture`. Edit the load-bearing paragraph
  identically in `.baton/PLANNER.md` and `bootstrap/.baton/PLANNER.md`, bump
  `VERSION` to `0.37.0`, and add one `GUIDANCE.md` convention line for
  `make-fixture`. Change no existing command's behaviour, no log format, no
  event name, no transition table, no artifact contract. Use `go run
  ./cmd/baton` for every check so it reflects the working tree; do not run the
  release build.

## Goal

- The two procedures `PLANNER.md` mandates -- drive a Baton install that is not
  this repository, and find out which tests a change actually pins -- are each
  one command, and the reviewer no longer composes them from five to ten shell
  steps that vary between agents.
- Neither command can write into a working tree, by construction rather than by
  care.

## Scope

In scope:

- `internal/baton/revertcheck.go` (new) and `internal/baton/revertcheck_test.go`
  (new)
- `internal/baton/app.go`: one `case` in `Run` and one line in `usage`
- `cmd/make-fixture/main.go` (new)
- `.baton/PLANNER.md` and `bootstrap/.baton/PLANNER.md`: replace the
  load-bearing paragraph's prose with the command name, same line count
- `VERSION` and `.baton/VERSION` -> `0.37.0`
- `.baton/GUIDANCE.md`: one line under `Conventions`

Out of scope:

- Any change to what `append`, `gate`, `new-round`, `feedback`, `status`,
  `prompt`, `await`, `check-artifact`, `guide`, `lint`, `merge-agent-block`,
  `update`, `remove`, `pause`, `resume`, `models`, or `version` does.
- The log format, event names, role pairs, the transition table, and the
  artifact contracts. This round adds a command that records nothing.
- `EXECUTOR.md` (Decision 6), `README.md`, `PROTOCOL-GUIDE.md`, the Korean
  guide, and `docs.go`'s embed list.
- The release build (`go run ./cmd/build-release`). Editing a managed document
  makes the installed binary report drift; Director rebuilds at close.
- Replacing the `newHarness` test fixture in `testutil_test.go`. It is a Go test
  helper and already works; `make-fixture` serves shell-driven checks.

## Decisions

### 1. Names, arguments, and output

**`go run ./cmd/make-fixture [--binary working|released] [--no-git]`**

Creates a throwaway installed project and prints where it is. `--binary`
selects what lands at `.baton/bin/baton`: `working` (default) builds
`./cmd/baton` from the working tree, `released` copies
`bootstrap/.baton/bin/<goos>-<goarch>/baton`, which is the previous release and
is what an update-from-a-released-binary check needs. `--no-git` skips the Git
repository; the default creates one, because a real Baton install lives in a Git
repository and the protocol assumes it, and because `remove --purge`'s
precondition cannot be exercised without one.

There is deliberately **no destination argument**; see Decision 3.

Output is `key=value`, one per line, on stdout:

```text
dir=/tmp/baton-fixture-3f2a91
baton_dir=/tmp/baton-fixture-3f2a91/.baton
binary=/tmp/baton-fixture-3f2a91/.baton/bin/baton
binary_source=working-tree
version=0.37.0
git=/tmp/baton-fixture-3f2a91 (commit 8c1d4e2)
drive=BATON_DIR=/tmp/baton-fixture-3f2a91/.baton /tmp/baton-fixture-3f2a91/.baton/bin/baton
cleanup=rm -rf /tmp/baton-fixture-3f2a91
```

Reason for the shape: `new-round` already prints `key=value` and agents already
parse it that way, so this adds no new output convention. Every line is a value
the next step needs. `drive=` exists because an agent's shell working directory
does not survive between calls, so the one thing it will otherwise get wrong is
composing `BATON_DIR` -- which is also the only way to drive the fixture without
`cd`. `cleanup=` exists because Decision 2 makes the agent the owner.

**`baton revert-check --check '<command>' [--rev <ref>] [--name-pattern <re>]
[--keep] PATH...`**

Copies the project to a throwaway directory, runs `--check` there for a
baseline, restores the named paths to their committed state **in the copy**,
runs `--check` again, and reports the difference. `--rev` defaults to `HEAD`.
Paths are repository-relative; absolute paths and paths escaping the root are
refused.

Output, `key=value`, stdout:

```text
copy=/tmp/baton-revert-24b7c0
rev=HEAD
reverted=internal/baton/artifact.go
reverted=internal/baton/lint.go
baseline=pass
after_revert=fail
pins=TestArtifactRequiresProtocolSections
pins=TestLintToleratesLegacyCloseSections
new_output=--- FAIL: TestArtifactRequiresProtocolSections (0.00s)
new_output=    artifact_test.go:88: check-artifact accepted a CLOSE without ## Plan Deviations
```

`pins=` is the answer Director asked for: the names of the tests that pass
before the revert and fail after it, i.e. the tests this change is what makes
pass. Names are extracted with `--name-pattern`, a regexp with one capture
group applied to each new output line. **It has no default.** A Go default
(`--- FAIL: (\S+)`) would be right here and silently produce an empty `pins=`
list in a project whose runner writes something else, which is the failure mode
this whole round exists to remove; refusing to guess matches how the protocol
treats ambiguity everywhere else. The usage text carries `--- FAIL: (\S+)` as
the Go example, so the cost is one flag, not one guess. When `--name-pattern`
is absent, the tool prints `pins=unavailable (pass --name-pattern)` and the
reviewer reads the names out of `new_output=`.

`new_output=` is always printed: every line present in the reverted run's
combined output and absent from the baseline's, verbatim, in order, capped at
200 lines with a trailing `new_output_truncated=<n>`. It is what makes the
answer checkable rather than trusted, and it carries the reason a test failed,
which `PLANNER.md` also requires ("fail for the reason the change was made").

Exit status: `0` whenever both runs completed, whatever they found -- "nothing
newly failed" is an answer, not a tool failure, and a reviewer must be able to
record it. Non-zero only when the tool could not answer: not a Git repository,
an unknown path, `--check` missing, or **`baseline=fail`**, which stops before
the revert because a comparison against a red baseline means nothing.

`revert-check` is not added to `refusesWhilePaused`: it writes no record and
advances no round, and a paused project's developer has the same question about
their tests as anyone else.

### 2. Where the fixture is created, and who removes it

Both commands create their directory with `os.MkdirTemp("", ...)`, which honours
`TMPDIR` and therefore lands in the session-scoped scratch area an agent already
has, and nowhere else. Neither takes a destination.

- `revert-check` owns its copy for exactly one run and removes it on every exit
  path, including failure, with `defer`. `--keep` suppresses removal and is for
  the reviewer who needs to open a failing copy; even then the directory is
  under `TMPDIR` and the session reaps it. The `copy=` line is printed in both
  cases so a `--keep` run is never a directory nobody knows about.
- `make-fixture` **cannot** remove its own directory: the agent drives it after
  the command exits, which is the entire point. That is a weaker guarantee and
  this plan does not pretend otherwise. What contains it is: the directory is
  under `TMPDIR`, the command prints `cleanup=`, and the `GUIDANCE.md` line from
  Decision 6 says the delegate runs it. A sweep of older `baton-fixture-*`
  directories was considered and rejected: a tool that deletes directories it
  did not create in this run, based on their names, is a new hazard added to
  remove a tidiness problem the session already solves.

### 3. What stops either command from touching a working tree

Guarantees, in order of strength:

1. **Neither command accepts a destination path.** There is no argument that can
   name a working tree. `make-fixture` takes no positional arguments at all;
   `revert-check`'s positionals are paths *to revert inside its own copy*, and
   are rejected unless `filepath.Rel(root, joined)` stays under the copy root.
2. **Every write goes through one helper** that joins a relative path onto the
   command's own `MkdirTemp` root and refuses anything that does not resolve
   under it. No write call in either file takes a caller-supplied absolute path.
   This is one choke point and a test asserts the refusal, so it is checkable
   rather than a convention.
3. **`revert-check` never checks anything out, anywhere.** It runs exactly four
   Git subcommands, all read-only, all with `Dir` set to the project root:
   `rev-parse` (resolve `--rev`, confirm a repository), `ls-files -c -o
   --exclude-standard` (the file list to copy: tracked plus untracked,
   ignored files excluded, which is also why a copy does not drag in build
   output), `cat-file -e` and `show REV:PATH` (the committed bytes). The
   reverted content is written with an ordinary file write under the copy root
   via (2). `checkout`, `restore`, `stash`, `clean` and `worktree` appear
   nowhere in the round; a test greps the new source for them, mirroring how
   `TestLintDoesNotExecutePaths` pins lint's no-execution property. A path that
   did not exist at `--rev` is reverted by deleting it from the copy, not by
   any Git operation.
4. **`.git` is not copied.** The copy is not a repository, so no Git command run
   inside it, by the check command or otherwise, can resolve to this project's
   repository.
5. **The `--check` command runs with its working directory set to the copy**,
   with no argument naming the real tree. The tool cannot constrain an arbitrary
   command beyond that, and this plan states it plainly instead of overclaiming:
   `revert-check` guarantees *its own* writes are confined; a `--check` string
   that reaches outside the copy is the caller's doing. An ordinary `go test
   ./...` cannot.
6. The `--check` string comes from **argv only**. `revert-check` never reads it
   from a `PLAN`, from `BATON-LOG.txt`, from `GUIDANCE.md`, or from any other
   file. Reading a command out of a record is precisely what lint is tested
   never to do, and a shipped binary that executed strings from artifacts would
   undo that. `Required Checks` in the `PLAN` is where a reviewer *reads* the
   command; typing it is the step that keeps a record from becoming an
   instruction.

`make-fixture` only reads from the repository (`bootstrap/`, and `go build -o
<copy>/...`, whose sole write is the `-o` path). Both locate the repository root
by walking up for a `go.mod` naming this module and refuse if there is none.

Both are therefore safe to run in this repository: they write nothing under
`.baton/` here, and `revert-check` run here reverts inside its copy only.

### 4a. Does `make-fixture` belong in the shipped binary? No.

It serves someone developing Baton and nobody else. It needs a Go toolchain,
this module's source, and `bootstrap/`, none of which exist in an installed
project -- a Python or TypeScript project that uses Baton would carry a command
that cannot run. The shipped binary is committed for six platforms under
`bootstrap/.baton/bin/`, so every byte is stored six times and copied into every
install. `remove.go` and `lint` are repository-shaped but *user*-facing: they
act on the reader's own project. `make-fixture` acts on Baton's source.

It therefore lives at `cmd/make-fixture`, a repository-only main package beside
the existing `cmd/build-release`, which is the same kind of thing and sets the
precedent for one directory per repository tool with no subcommand router.

### 4b. Does `revert-check` belong in the shipped binary? Yes.

Director's correction is right and I am adopting it. "Does this test actually
guard this change" is not Baton-specific, and `PLANNER.md` -- which ships to
every installed project and is embedded in the binary -- already orders every
reviewer there to do it by hand. Shipping the instruction without the tool is
what makes every delegate re-derive the procedure, which is the problem this
round was opened for.

The objection I expected to be decisive, that the binary cannot know how to run
an arbitrary project's tests, dissolves once the command refuses to guess: it
takes `--check` and requires it. The protocol already puts that command in front
of the reviewer, in the `PLAN`'s `Required Checks`. Reading it from there
automatically is the one thing this plan forbids (Decision 3, item 6).

So `revert-check` lives in `internal/baton/revertcheck.go` with a `case` in
`App.Run` and one usage line, like every other command. Costs accepted, stated
rather than waved away: the shipped surface grows by one command; the binary can
now run a string a caller typed; and a large project's copy is proportional to
its tracked-plus-unignored file count. Mitigations are Decision 3's confinement,
argv-only sourcing, and `--exclude-standard` keeping ignored build output out of
the copy.

The two commands share almost nothing -- `make-fixture` copies one small known
tree, `revert-check` copies a file list from Git -- so no shared package is
created. A new `internal/` package for one copy helper would be more structure
than either needs, and `cmd/make-fixture` can import `internal/baton` directly
if it wants `binaryName` or the checksum helper.

### 5. What the fixture must contain, and what it cannot serve

`make-fixture` produces: `.baton/` copied from `bootstrap/.baton` (all managed
documents, all seven templates, `LESSON-LEARNED.md`, empty `lesson-learned/` and
`runs/`), both `AGENTS.md` and `CLAUDE.md` from `bootstrap/`, `.baton/bin/baton`
plus a matching `.baton/bin/SHA256SUMS` for the current platform, a
`BATON-LOG.txt` seeded with the bootstrap `REQUEST` and `RUN_DONE` pair, and by
default a Git repository with all of it committed.

Against the four checks agents ran today:

- **Pause and remove** -- served fully. Both instruction files carry the shipped
  block, so `pause`, `resume` and lint's two-file comparison all have something
  to act on; `remove` has the installed binary, the checksums and the
  by-name-recognisable templates it needs to classify. The Claude-only and
  Agents-only variants get **no flag**: deleting one file from the fixture is a
  single command, and a flag for it would be surface that saves nothing.
- **`--purge` preconditions needing a Git repository** -- served by the default;
  `--no-git` produces the not-a-repository refusal case. Everything including
  `.baton/bin/baton` is committed, because a real install commits `.baton/`
  (this repository ignores `.baton/bin/` only because it is Baton's own source),
  and purge's precondition is about committed-ness.
- **Update from a released binary** -- served by `--binary released`, which
  installs the previous release from `bootstrap/.baton/bin/`. The upstream for
  `update --upstream` is then this repository itself, whose working-tree
  `bootstrap/` is what `preflightUpdate` reads. This is the check that caught
  the "a new managed document arrives one update late" finding, and it is the
  most expensive one to build by hand.
- **Artifact validation** -- **not served, and deliberately not**. The fixture
  gives the templates and an empty `runs/`; every artifact check needs one
  specific malformation, and a seeded sample artifact would be a shape agents
  start editing instead of a shape they chose. These stay hand-written, which is
  two lines of heredoc.

Also not served, and by intent: a fixture in any particular round state. The
fixture is a *drivable* install, not a set of states; an agent reaches
`REQUEST`/`PLANNED`/`REVIEW` by running the installed binary, which is more
faithful than a hand-written timeline and is how three of today's false greens
would have been caught earlier.

### 6. Should `PLANNER.md` or `EXECUTOR.md` name them?

- **`PLANNER.md` names `revert-check`. Yes** -- and it does not grow. The
  document already spends a paragraph describing the procedure by hand; that
  paragraph is replaced with one naming the command, at the same line count.
  Director's standing instruction is against growth, not against substitution,
  and this is the case where prose becomes a name.
- **`PLANNER.md` must not name `make-fixture`.** `PLANNER.md` ships. A shipped
  document that names a command the reader's project does not have is an
  instruction that cannot be followed, which is worse than the prose it
  replaced. This is the direct consequence of 4a and 4b differing: only the
  shipped command can be named in a shipped role file.
- **`EXECUTOR.md`: no.** It does not currently mandate the load-bearing check at
  all, so naming the command there would be *adding* a requirement to a
  document under a no-growth instruction. Out of scope for this round; if
  Director wants Executors held to the check, that is its own decision.
- **`make-fixture` is recorded in `.baton/GUIDANCE.md`**, one line under
  `Conventions`. `GUIDANCE.md` is this project's own file, preserved by
  `update` and never shipped, and every role rereads it at the start of each
  recordable phase -- which is exactly the reach a repository-only tool should
  have. Because guidance updates require user acceptance, Executor drafts the
  line in the `RUN`; Director appends it at close if the user accepts.

The edit, both copies, replacing the existing second Review paragraph verbatim:

```text
Check that each test the round added is load-bearing: `baton revert-check
--check '<test command>' <changed source path>...` reverts those paths on a copy
and reports which tests fail only after the revert. A test that passes either
way is guarding something else.
```

## Plan

1. `internal/baton/revertcheck.go`: `runRevertCheck(args []string) error` using
   `parseArguments` (value flags `--check`, `--rev`, `--name-pattern`; bool
   flags `--keep`; positionals are the paths). Root/`rev` resolution, the
   Git-read helpers, the confined-write helper, copy, baseline run, revert,
   second run, output. -> verify: `go build ./...`, `go vet ./...`.
2. `internal/baton/app.go`: `case "revert-check"` in `Run` and one usage line
   under `Delegate a stage`. Do not add it to `refusesWhilePaused`. -> verify:
   `go run ./cmd/baton help` lists it; every other usage line is unchanged
   (`git diff` on `app.go` is two added lines).
3. `internal/baton/revertcheck_test.go`: the six tests in Validation. -> verify:
   `go test ./internal/baton/...`.
4. `cmd/make-fixture/main.go`. -> verify: `go run ./cmd/make-fixture` prints the
   keys, and the printed `drive=` line runs `status` and `lint` clean in the
   fixture.
5. Edit the paragraph identically in `.baton/PLANNER.md` and
   `bootstrap/.baton/PLANNER.md`; bump `VERSION` and `.baton/VERSION` to
   `0.37.0`. -> verify: `diff .baton/PLANNER.md bootstrap/.baton/PLANNER.md` is
   empty; `go run ./cmd/baton lint` passes.
6. Draft the `GUIDANCE.md` convention line in the `RUN` for Director to append
   at close; do not edit `GUIDANCE.md`. -> verify: `git diff --stat` lists no
   `GUIDANCE.md`.
7. Self-check the artifact with `go run ./cmd/baton check-artifact EXECUTED
   .baton/runs/20260920-1900-verify-tooling-RUN-01.md cqso`.

## Success Criteria

- `go run ./cmd/baton revert-check --check 'go test ./internal/baton/...'
  --name-pattern '\-\-\- FAIL: (\S+)' internal/baton/revertcheck.go` run in this
  repository prints `baseline=pass` and a `pins=` line for each test this round
  added that genuinely depends on the new file, with the reason visible in
  `new_output=`. Any added test that does not appear is reported in the `RUN` as
  not load-bearing, with why, rather than quietly dropped.
- `go run ./cmd/make-fixture` prints a directory in which
  `BATON_DIR=<dir>/.baton <dir>/.baton/bin/baton lint` passes and `new-round`
  through `gate` can be driven with no further setup, and
  `--binary released` produces a fixture the documented update procedure
  upgrades in one `update --apply`.
- Neither command writes anything outside its own temporary directory: `git
  status --short` in this repository is unchanged by running either, and the
  `revert-check` copy is gone afterwards unless `--keep` was given.
- No existing command changes behaviour: the diff touches no existing function
  body outside `App.Run`'s switch and `usage`.
- `.baton/PLANNER.md` and `bootstrap/.baton/PLANNER.md` are byte-identical and
  the Review section's line count is unchanged.

## Validation

- `go build ./...`, `go vet ./...`, `gofmt -l .`: clean, no output.
- `go test ./...`: pass. New tests, all against `t.TempDir()` projects, never
  this repository:
  - `TestRevertCheckReportsNewlyFailingTests`: a scratch Git repository with a
    committed source file and a test, then a change making a new assertion pass;
    `pins=` names that test and only it.
  - `TestRevertCheckStopsOnRedBaseline`: a copy whose check already fails exits
    non-zero, prints `baseline=fail`, and performs no revert.
  - `TestRevertCheckRefusesEscapingPath`: `../outside.go` and an absolute path
    are refused, and nothing outside the copy is written or read.
  - `TestRevertCheckRemovesItsCopy`: the `copy=` directory is gone after a
    successful run and after a failing one; present with `--keep`.
  - `TestRevertCheckNeverCheckoutsOrRunsFromRecords`: the source of
    `revertcheck.go` contains no `checkout`/`restore`/`stash`/`clean`/`worktree`
    Git subcommand and reads the check command only from the parsed arguments.
  - `TestRevertCheckDeletesPathAbsentAtRev`: a file added by the round and named
    for revert is deleted in the copy, not fetched.
- `go run ./cmd/baton lint`: pass, including the managed-document comparison,
  which is why every check uses `go run ./cmd/baton` and not `.baton/bin/baton`.
- `go run ./cmd/make-fixture`, then in the printed fixture: `lint`, `new-round`,
  `status`, `remove` dry run, and `remove --purge` dry run on the Git default and
  on `--no-git`; then `rm -rf` the fixture and record that it is gone.
- `go run ./cmd/make-fixture --binary released`, then `update --upstream <this
  repo>` dry run and `--apply`, then `lint` in the fixture.
- The round's own load-bearing check, using the new command on itself, recorded
  verbatim in the `RUN`.
- `git status --short` before and after all of the above: only the intended
  files.

## Report To Director

- Suggested summary: Planner defined two verification commands -- shipped
  `baton revert-check` and repository-only `cmd/make-fixture` -- with argv-only
  check strings, temp-dir-confined writes, no destination argument, and a
  same-length `PLANNER.md` substitution naming only the shipped one
- Artifact path: `.baton/runs/20260920-1900-verify-tooling-PLAN.md`

## Risks Or Questions

- Shipping `revert-check` means the binary runs a caller-supplied command for
  the first time. The line held here is argv-only (Decision 3, item 6). If a
  later round is tempted to read `Required Checks` out of the `PLAN`
  automatically, that would make a record an instruction and should be refused;
  it is worth a lesson record at close.
- `revert-check` copies every tracked and unignored file. In this repository
  that is 142 files including seven committed platform binaries, roughly 40 MB
  and about a second. In a large project with big tracked binaries it will be
  slower. No exclusion flag is planned, because a wrong exclusion silently
  changes what the check measures; if it bites, the fix is a measurement, not a
  guess.
- The `--name-pattern` decision is the one most open to reversal. If Director
  prefers a Go default, the change is one line and the cost is a silently empty
  `pins=` list in non-Go projects. Recorded here so the choice is visible rather
  than discovered later.
- `VERSION` is bumped in the round but the release build is not run, so the
  installed `.baton/bin/baton` reports drift until Director rebuilds at close.
  This is the stated norm; the `RUN` should say so explicitly so the drift is
  not read as a defect.
- `make-fixture` cannot delete the directory it exists to leave behind
  (Decision 2). The `cleanup=` line and the `GUIDANCE.md` convention are the
  whole mitigation, and the `GUIDANCE.md` half depends on user acceptance at
  close.
