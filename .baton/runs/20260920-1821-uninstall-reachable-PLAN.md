# PLAN: Put the removal procedure where the agent asked to do it can read it

Task ID: kovh
Date: 2026-09-20
Planner: Claude Opus 5
Status: complete

## Director Brief

- Goal: An agent standing inside an installed project, told in plain language to stop using Baton, finds the pause-or-remove procedure from the files that project actually has — without adding a sixth managed document.
- Scope: `.baton/HOW-TO-UPDATE.md` and its `bootstrap/.baton/` twin become the one lifecycle document (update, pause/resume, remove), keeping that file name; the pause and remove sentences move out of `PROTOCOL.md`'s `## Git And Updates` into a section named for what the reader is looking for, in both `PROTOCOL.md` copies; `guideDocuments` in `guide.go` gains aliases so `baton guide remove|pause|uninstall|lifecycle` print that document, and `usage()` matches; the repository-root `UNINSTALL.md` is deleted after its content is carried over in English; `BOOTSTRAP.md` and `README.md` gain one pointer each; tests.
- Success Criteria: A project updating with its *own installed* binary from any released version ends with the lifecycle document present and `lint` clean, with no second update needed (V1); the `.baton/` and `bootstrap/.baton/` copies of both edited documents are byte-identical and `go run ./cmd/baton lint` reports no drift (V2); `PROTOCOL.md`'s total line count does not grow by more than the one pointer line, because the pause and remove text is moved and not duplicated (V3); `baton guide remove` prints the lifecycle document and `baton guide update` still prints the same bytes (V4); `go test ./...`, `go vet ./...`, `gofmt -l .` clean.
- Risks: R1 the file name `HOW-TO-UPDATE.md` under-describes its new contents, and Finding 2 makes renaming it a hard break for every installed binary, so the name has to be carried by the CLI and by `PROTOCOL.md` instead; R2 deleting the one-day-old root `UNINSTALL.md` throws away Korean prose, so its substance must land in the English document before it goes; R3 editing managed documents makes the installed binary report drift until Director rebuilds at close.
- Required Checks: `go run ./cmd/baton lint`, `go test ./...`, `go vet ./...`, `gofmt -l .`, plus V1-V6 below.
- Executor Prompt: Implement `.baton/runs/20260920-1821-uninstall-reachable-PLAN.md` steps 1-9. Do not create a new file under `.baton/` and do not rename `HOW-TO-UPDATE.md` — Finding 2 shows why both break existing installs. Do not change what `pause` or `remove` do, including their output; the only Go change is `guideDocuments` plus the matching `usage()` line. Keep `.baton/HOW-TO-UPDATE.md` and `bootstrap/.baton/HOW-TO-UPDATE.md` byte-identical, and likewise both `PROTOCOL.md` copies. In `PROTOCOL.md`, move the pause and remove sentences rather than adding new ones. Run every check with `go run ./cmd/baton`, never `.baton/bin/baton`. Do not bump `VERSION` or rebuild binaries; Director does that at close.

## Goal

- An agent that has read `.baton/PROTOCOL.md` — the one file every role is told
  to read — can find, from a heading named after the thing the user said, where
  the pause and removal procedure lives.
- That procedure is a file the installed project has, in a language every agent
  installing Baton can read.
- Adding it costs the project zero new managed files, so nothing new can fall
  out of step, and a project that has not yet taken the update is not left
  failing `lint` or missing a document its binary requires.
- The repository stops holding two removal procedures with nothing keeping them
  in step.

## Scope

In scope:

- `.baton/HOW-TO-UPDATE.md` and `bootstrap/.baton/HOW-TO-UPDATE.md`: rewritten
  as the lifecycle document, same file name.
- `.baton/PROTOCOL.md` and `bootstrap/.baton/PROTOCOL.md`: the `## Git And
  Updates` section is split; pause and remove move into a section named for
  them.
- `internal/baton/guide.go`: alias keys in `guideDocuments`.
- `internal/baton/app.go`: the one `usage()` line listing guide topics.
- `UNINSTALL.md`: deleted.
- `BOOTSTRAP.md`, `README.md`: one pointer each.
- Tests.

Out of scope:

- What `pause` and `remove` do, including their stdout (constraint). No new
  output line, no new flag, no new refusal.
- The log format, event names, transition table, artifact contracts
  (constraint).
- `managedBatonFiles`, `docs.go`'s embed list, `lint`'s `requireFile` list,
  `requiredTrackedPaths`, `remove.go`'s deletion list, `BOOTSTRAP.md`'s copy
  step. Decision 1 is that none of these change; the reasoning is there, not
  here.
- `PROTOCOL-GUIDE.md`. Its only match for pause or removal is line 429, about
  `.gitignore`. It carries no lifecycle section to keep in step, as in the
  previous two rounds.
- `.baton/VERSION`, `VERSION`, and rebuilding `.baton/bin/baton`: Director's at
  close, as in the previous three rounds.

## Findings: what the code does today

Read and, where stated, executed. Not assumed.

1. **`update --apply` does create a managed file the project does not have.**
   `runUpdate` loops `managedBatonFiles` and calls `copyFile`
   (`update.go:160-164`), which is `os.ReadFile` on the source then
   `atomicWrite` on the target (`files.go:57-67`); `atomicWrite` does
   `os.MkdirAll` and `os.CreateTemp`+rename (`files.go:16-54`) with no
   existence check on the target. `preflightUpdate` (`update.go:36-48`) stats
   the *upstream* copies only. Verified by running it: a fixture project with
   `HOW-TO-UPDATE.md` deleted, updated with `go run ./cmd/baton update
   --upstream . --apply`, had the file back afterwards. So the naive worry —
   "update only replaces, it never adds" — is false.

2. **But the binary that performs the update is the project's old one, and it
   copies only the names *it* knows.** `HOW-TO-UPDATE.md` step 4 says "Use the
   installed binary as `<baton>` when available", and reserves running the
   upstream binary for legacy script-only installs and Windows. So on the
   normal macOS/Linux path, `managedBatonFiles` is the *pre-update* list.
   Two consequences, both decisive:

   - **A new managed file arrives one update late.** The old binary's
     preflight does not require it upstream and its copy loop does not copy
     it; the loop does copy the new binary, so the project ends on the new
     binary with the file absent. If `lint` requires it by name
     (`lint.go:384`), that successful update is immediately followed by
     `ERROR: missing .baton/UNINSTALL.md`. If `lint` is made tolerant instead,
     the project is silently missing the removal document — which is exactly
     the failure this round exists to fix, inflicted on every project that
     already has Baton.
   - **Renaming a managed file is a hard break.** `preflightUpdate`
     (`update.go:39-48`) requires every name in the old binary's
     `managedBatonFiles` to exist upstream and returns `missing upstream file`
     otherwise. Rename `HOW-TO-UPDATE.md` to anything and every installed
     binary refuses to update at all. A compatibility stub upstream would let
     the update through but would be copied into installed projects forever,
     with nothing in the codebase that ever deletes a managed file.

   Editing a document already on the list is the only change that reaches an
   existing project on its first update.

3. **`lint` is the mechanism that keeps the paired copies honest.**
   `checkManagedDocuments` (`lint.go:171-186`) compares each installed
   `.baton/<name>` against `docs.Managed(name)`, which reads the embedded
   `bootstrap/.baton/<name>` (`docs.go:20-26`). `.baton/HOW-TO-UPDATE.md` and
   `.baton/PROTOCOL.md` are byte-identical to their `bootstrap/` twins today
   (checked with `diff`). This is also why there can only ever be *one* text
   per managed document: a second language would be a second file, or drift.

4. **The CLI already carries documents.** `baton guide <protocol|director|
   planner|executor|update>` (`guide.go:10-31`) prints the *embedded* copy,
   and `DIRECTOR.md` already tells Director "the binary is the source of truth
   for its own commands". `guideDocuments` is a plain `map[string]string`, so
   several keys can point at one file at no cost. This is the lever Director's
   standing instruction asks for: a rule a tool can carry.

5. **`PROTOCOL.md`'s `## Git And Updates`** (lines 138-154) holds three
   paragraphs: commit `.baton/` plus the `HOW-TO-UPDATE.md` pointer; pause and
   remove; the binary and platform note. Only the middle one is misfiled.

6. **The root `UNINSTALL.md` has no code dependency.** Its only inbound link is
   `README.md`'s 더 읽기 list. Nothing embeds it, copies it, or checks it.

## Decisions

### Decision 1 — `UNINSTALL.md` does not become a sixth managed document

No. Reason, from Finding 2: a new managed file cannot reach a project on the
update that introduces it, because the old installed binary drives that update
and copies only the names it was compiled with. Every way of absorbing that is
worse than not having the problem — a strict `lint` turns a successful update
into an error, a tolerant `lint` silently withholds the removal document from
exactly the installed projects this round is about, and a "run update twice"
instruction would have to be read from the copy of `HOW-TO-UPDATE.md` the
project does not have yet.

So none of the six things Director listed changes: `managedBatonFiles`
(`update.go:12`), the embed in `docs.go:20`, `checkManagedDocuments`
(`lint.go:171`), `Lint`'s required-file list (`lint.go:384`) and
`requiredTrackedPaths` (`lint.go:129`), `remove.go`'s deletion list (it reads
`managedBatonFiles`, `remove.go:138`, so it follows automatically), and
`BOOTSTRAP.md`'s copy step (step 4.1 copies all of `bootstrap/.baton/` except
`bin/`, so it would have needed nothing anyway; the per-file update table at
line 149 already has a `HOW-TO-UPDATE.md` row).

The answer to "what happens to a project that updates from a version without
it" is therefore: nothing happens, because there is no new file. The project
gets a longer `HOW-TO-UPDATE.md`, by the same copy that has always replaced
it.

### Decision 2 — the installed text is English, and there is one copy of it

The installed lifecycle document is written in English, like the five managed
documents it joins. Two reasons, not one: managed documents are installed into
other people's projects, whose agents and readers are not this repository's
Korean-reading maintainers; and Finding 3 means `lint` enforces exactly one
text per managed document, so "an English one and a Korean one" is not a shape
this machinery has.

`UNINSTALL.md` was Korean because it was modelled on `BOOTSTRAP.md`, and that
model does not transfer. `BOOTSTRAP.md` describes work done *from this
repository* by an agent that came here — it has no installed counterpart and
cannot have one, since a project without `.baton/` has nowhere to read it.
Removal is the mirror image: it happens *inside* an installed project, where
the installed copy is the only one the agent can reach. So this is not "two
copies in two languages"; it is one document, in the repository twice
(`.baton/` and `bootstrap/.baton/`) and byte-identical, the way every managed
document already is.

### Decision 3 — the pause and remove sentences move, and `PROTOCOL.md` does not grow

They move. `## Git And Updates` becomes two sections:

- `## Git And Updates` keeps its first paragraph (commit `.baton/`, do not
  `.gitignore` it, read `HOW-TO-UPDATE.md` when updating, bootstrap and update
  are recording targets) and its last (the binary and platform note).
- A new heading — `## Pause, Remove, And Update` — takes the two existing
  sentences about `pause`/`resume` and `remove --apply` verbatim, and adds one
  line only: that all three procedures are in `.baton/HOW-TO-UPDATE.md`.

The heading leads with the words a user says ("stop using this", "take it
out"), not with "Git", because a heading is the index an agent scans. Net
change to `PROTOCOL.md`: one added line plus a heading; the rest is moved
text, which Director's standing instruction explicitly permits.

Nothing else moves into the CLI here, because there is no rule to move — this
is a pointer to a document, and a pointer is what a protocol file is for. The
CLI does carry the same pointer, in Decision 5's aliases.

### Decision 4 — one document, not two

One. The argument for two is real and should be stated: `HOW-TO-UPDATE.md` is
read at one moment ("update Baton") and removal at a different one ("stop
using Baton"), and a reader arriving for removal would page past nine update
steps to reach it. A file named for one of its two jobs is a wart.

It loses on three counts.

1. **Finding 2 decides it.** A second file cannot reach an existing project on
   the update that adds it; an edit to this one reaches every project on its
   first update. Reachability is this round's entire subject, so a solution
   that is unreachable for one update cycle on every project that has Baton
   today is not a solution.
2. **Director's standing instruction.** One fewer managed file is one fewer
   thing to keep in step — literally, in this codebase: one fewer entry in
   `managedBatonFiles`, the embed, the drift check, two `lint` lists, and the
   `BOOTSTRAP.md` table.
3. **They are the same kind of thing.** Update, pause, resume and remove are
   all "an agent was told to change Baton's presence in this project, by a
   user who did not say which one". The document's first section should be the
   triage between them — which is the most valuable page in `UNINSTALL.md`
   today, and it only works if all the destinations are in the same file.

The wart is paid for in Decision 5: the CLI answers to the names the file
does not carry.

### Decision 5 — what `BOOTSTRAP.md` and `README.md` say, and the root copy goes

The root `UNINSTALL.md` is deleted, after its substance is carried into the
English lifecycle document. Once an installed copy exists, the root copy has
no distinct audience: the agent it was written for is inside an installed
project and never sees this repository, and an agent that *is* here to install
does not need a removal procedure. What it would have is a second,
unsynchronised statement of the same procedure — the drift Baton spends
`checkManagedDocuments` preventing for every other document. Finding 6 says
nothing depends on it.

- `README.md`: the 중지 and 제거 sections stay as they are — they are the
  repository's overview of what the commands do, not a procedure, and they are
  the Korean surface a human browsing the repository reads. The 더 읽기 entry
  changes from `UNINSTALL.md` to
  `[HOW-TO-UPDATE.md](bootstrap/.baton/HOW-TO-UPDATE.md) — 업데이트·중지·제거
  절차`, and the 설치 후 구조 tree needs no change.
- `BOOTSTRAP.md`: one line at the end of the 사전 확인 or 부트스트랩 절차
  section stating that update, pause and removal are procedures the *installed*
  project reads from `.baton/HOW-TO-UPDATE.md`, so an installing agent knows
  it is not this document's job. Its existing 업데이트 절차 stays; it is what an
  agent reads when it came to this repository for an update.

The CLI carries the names the file name does not: `guideDocuments` gains
`remove`, `uninstall`, `pause` and `lifecycle` as additional keys for
`HOW-TO-UPDATE.md`, so an agent that guesses at the CLI instead of the file
tree gets the document. `usage()`'s guide line is updated to match.

### Decision 6 — what the lifecycle document contains

Rewritten `HOW-TO-UPDATE.md`, English, target 110-130 lines, in this order:

1. **Which of these is it?** — the triage table from `UNINSTALL.md`
   (temporarily off → `pause`; this one task without the protocol → nothing,
   short Q&A is not a recording target; out of this repository → remove;
   refresh in place → update), plus the sentence that `pause` is reversible
   and removal is not, so ask when unsure and offer `pause` first.
2. **Update** — the existing nine steps and the Preserve and Report sections,
   unchanged in substance.
3. **Pause and resume** — `pause`/`resume`, what they touch (the rules block
   only), the open-task refusal and `--force`, and that a paused project must
   resume before it can update.
4. **Remove** — dry run first and show the output to the user, `--apply` after
   approval, how to confirm the result, the whole-file deletion case, the
   refusal table (open tasks, drifted block with no `--force` override,
   `--purge` not committed, `--purge` not a Git repository), `--purge` and why
   it is not the default, and what is true after removal.

Cross-references stay accurate to the code as read this round: `update` and
`merge-agent-block` refuse while paused (`pause.go:246-252`), `update --apply`
refuses in a removed project (`pause.go:260-267`), `remove` is not in
`refusesWhilePaused` (`app.go:88-96`), and `lint` runs in a removed project
checking only the kept record (`lint.go:376-400`).

## Plan

1. Rewrite `bootstrap/.baton/HOW-TO-UPDATE.md` per Decision 6, in English,
   with a new H1 naming all four procedures. → verify: the four sections exist
   and the update steps are unchanged in substance.
2. Copy it byte-for-byte to `.baton/HOW-TO-UPDATE.md`. → verify: `diff` reports
   no difference.
3. Edit `bootstrap/.baton/PROTOCOL.md` per Decision 3: split the section, move
   the two sentences verbatim, add the one pointer line. → verify: `git diff`
   shows the moved text unchanged and exactly one added sentence.
4. Copy the `PROTOCOL.md` change to `.baton/PROTOCOL.md`. → verify: `diff`
   reports no difference.
5. Add the alias keys to `guideDocuments` in `internal/baton/guide.go`, and
   update the error string and the `usage()` guide line in
   `internal/baton/app.go` to list them. → verify: `baton guide remove` and
   `baton guide update` print identical bytes.
6. Delete `UNINSTALL.md`. → verify: no remaining reference to it anywhere
   (`grep -rn UNINSTALL .`, excluding `.baton/runs/`).
7. Update `README.md`'s 더 읽기 entry and add the `BOOTSTRAP.md` pointer line
   per Decision 5. → verify: both links resolve to files that exist.
8. Tests: extend `internal/baton/` tests for V4 and V5 below. → verify:
   `go test ./...`.
9. Run the full check set. → verify: all clean.

## Success Criteria

- An existing project, updating with its own installed binary, ends with the
  lifecycle document present and `lint` clean on the first update.
- The two copies of each edited managed document are byte-identical, and the
  working-tree binary reports no managed-document drift.
- `PROTOCOL.md` gains at most one sentence and one heading; the pause and
  remove text in it is the same text, relocated.
- `baton guide remove`, `baton guide uninstall`, `baton guide pause`,
  `baton guide lifecycle` and `baton guide update` all print the same document.
- No file under `.baton/` is created or renamed, and `pause` and `remove`
  behave and print exactly as before.
- The repository holds exactly one removal procedure.

## Validation

- V1 — the migration that matters: build a fixture project from
  `bootstrap/.baton/` at the *current* commit (pre-change `HOW-TO-UPDATE.md`),
  then run `go run ./cmd/baton update --upstream . --apply` against it with
  `BATON_DIR` pointed at the fixture, and confirm `HOW-TO-UPDATE.md` is the new
  text and `go run ./cmd/baton lint` passes there. The old binary's
  `managedBatonFiles` already contains this name, so this is a replace, not an
  add — V1 is the check that the claim in Decision 1 holds in practice. Never
  run against this repository's own `.baton/`.
- V2 — `diff .baton/HOW-TO-UPDATE.md bootstrap/.baton/HOW-TO-UPDATE.md` and
  `diff .baton/PROTOCOL.md bootstrap/.baton/PROTOCOL.md` are both empty, and
  `go run ./cmd/baton lint` passes with no drift error.
- V3 — `git diff --stat .baton/PROTOCOL.md` shows a net addition of at most two
  lines, and `git diff .baton/PROTOCOL.md` shows the pause and remove sentences
  removed from one place and added unchanged in another.
- V4 — a test asserting every key in `guideDocuments` resolves through
  `docs.Managed` without error, and that the alias keys and `update` resolve to
  the same file name. This is load-bearing: delete the aliases and it fails;
  point one at a non-existent file and it fails.
- V5 — a test asserting `.baton/HOW-TO-UPDATE.md` covers all four procedures,
  by requiring the strings the protocol names (`pause`, `resume`,
  `remove --apply`, `--purge`, `update --upstream`) to be present. Cheap, and
  it is the only automated guard that the document did not silently lose a
  procedure in a later edit.
- V6 — `grep -rn "UNINSTALL" . --exclude-dir=.git --exclude-dir=runs` returns
  nothing outside `.baton/runs/`.
- `go test ./...`, `go vet ./...`, `gofmt -l .` all clean.

## Report To Director

- Suggested summary: Fold the removal procedure into the one installed
  lifecycle document
- Artifact path: `.baton/runs/20260920-1821-uninstall-reachable-PLAN.md`

## Risks Or Questions

- R1 — the file keeps a name that describes one of its four jobs. Finding 2
  makes a rename a hard break for every installed binary, and a compatibility
  stub would litter installed projects permanently. The name is paid for by
  `PROTOCOL.md`'s new heading and by the `guide` aliases. A rename becomes
  possible only in a future version where the preflight tolerates a managed
  file that has gone away upstream; that is not this round.
- R2 — deleting the root `UNINSTALL.md` loses Korean prose one day after it was
  written. Mitigated by carrying its substance into the English document
  (Decision 6) before deleting, and by `README.md`'s 제거 section, which is
  already the Korean overview. The alternative — keep it as the Korean
  companion to `BOOTSTRAP.md` — loses because nothing would keep the two texts
  in step, and the agent it was written for cannot see this repository.
- R3 — editing managed documents makes the installed `.baton/bin/baton` report
  drift for the rest of the round. Every check in this plan uses
  `go run ./cmd/baton`; Director rebuilds and bumps `VERSION` at close, as in
  the previous three rounds.
- R4 — this round does not change `pause` or `remove` output, so an agent that
  runs `baton remove` without having read anything still sees only the dry-run
  plan. That is deliberate under the constraint. If Director later wants the
  dry run to name the document, that is a one-line change and its own round.
