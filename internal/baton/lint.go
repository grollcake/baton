package baton

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	docs "github.com/grollcake/baton"
)

type lintState struct {
	app    *App
	errors int
}

func (state *lintState) ok(format string, args ...any) {
	fmt.Fprintf(state.app.Stdout, "OK: "+format+"\n", args...)
}

func (state *lintState) err(format string, args ...any) {
	state.errors++
	fmt.Fprintf(state.app.Stderr, "ERROR: "+format+"\n", args...)
}

func (state *lintState) requireFile(path string) {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		state.ok("%s exists", path)
	} else {
		state.err("missing %s", path)
	}
}

func (state *lintState) requireDir(path string) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		state.ok("%s exists", path)
	} else {
		state.err("missing %s", path)
	}
}

// requireTimeline reports the resolved timeline file (Decision 3), erroring
// when neither name is present or when both are present, and noting the
// legacy name so a project reading this output knows the next append will
// rename it.
func (state *lintState) requireTimeline() {
	path, err := state.app.timelinePath()
	if err != nil {
		state.err("%s", err)
		return
	}
	if filepath.Base(path) == legacyTimelineFile {
		state.ok("%s exists (legacy name; the next append renames it to %s)", path, timelineFile)
		return
	}
	state.ok("%s exists", path)
}

var batonBlockPattern = regexp.MustCompile(`(?s)<baton-rules>.*?</baton-rules>`)

func (state *lintState) checkAgentBlocks() {
	agentsPath := filepath.Join(state.app.ProjectDir, "AGENTS.md")
	claudePath := filepath.Join(state.app.ProjectDir, "CLAUDE.md")
	agents, agentsErr := os.ReadFile(agentsPath)
	claude, claudeErr := os.ReadFile(claudePath)
	if agentsErr != nil || claudeErr != nil {
		return
	}
	agentsBlock := batonBlockPattern.Find(agents)
	claudeBlock := batonBlockPattern.Find(claude)
	if len(agentsBlock) == 0 {
		state.err("AGENTS.md missing <baton-rules> block")
	} else if len(claudeBlock) == 0 {
		state.err("CLAUDE.md missing <baton-rules> block")
	} else if string(agentsBlock) != string(claudeBlock) {
		state.err("AGENTS.md and CLAUDE.md Baton blocks differ")
	} else {
		state.ok("AGENTS.md and CLAUDE.md Baton blocks match")
	}
}

// checkGitTracking refuses a .baton that Git ignores, because Baton state is
// project handoff data and must travel with the repository.
func (state *lintState) checkGitTracking() {
	if _, err := state.app.runGit("rev-parse", "--git-dir"); err != nil {
		return
	}
	if _, err := state.app.runGit("check-ignore", "--quiet", state.app.BatonDir); err == nil {
		state.err(".baton is ignored by Git; remove the .gitignore rule that covers it")
		return
	}
	state.ok(".baton is not ignored by Git")
}

// requiredTrackedPaths lists the protocol-required paths under .baton/ that
// checkRequiredPathsTracked checks individually. .baton/bin/ (the installed
// binary and bin/SHA256SUMS) is deliberately excluded: this repository's own
// .gitignore ignores it on purpose as a build artifact, unlike the handoff
// data every path here carries (Decision 5).
var requiredTrackedPaths = []string{
	"PROTOCOL.md", "DIRECTOR.md", "PLANNER.md", "EXECUTOR.md",
	"HOW-TO-UPDATE.md", "VERSION", "GUIDANCE.md", "LESSON-LEARNED.md",
	"templates", "runs", "lesson-learned",
}

// checkRequiredPathsTracked checks each protocol-required path individually,
// because a project's .gitignore can match a single file (e.g. "*.log") or a
// single directory without matching the whole .baton directory, which
// checkGitTracking's whole-directory check cannot see.
func (state *lintState) checkRequiredPathsTracked() {
	if _, err := state.app.runGit("rev-parse", "--git-dir"); err != nil {
		return
	}
	var ignored []string
	for _, name := range requiredTrackedPaths {
		full := state.app.batonPath(name)
		if _, err := state.app.runGit("check-ignore", "--quiet", full); err == nil {
			ignored = append(ignored, full)
		}
	}
	if timelinePath, err := state.app.timelinePath(); err == nil {
		if _, err := state.app.runGit("check-ignore", "--quiet", timelinePath); err == nil {
			ignored = append(ignored, timelinePath)
		}
	}
	if len(ignored) > 0 {
		state.err("Git ignores required Baton path(s): %s", strings.Join(ignored, ", "))
		return
	}
	state.ok("no individually required Baton path is ignored by Git")
}

// checkManagedDocuments compares the installed managed documents against the
// copies this binary shipped with, so a project cannot run a protocol its
// binary does not implement.
func (state *lintState) checkManagedDocuments() {
	for _, name := range managedBatonFiles {
		installed, err := os.ReadFile(state.app.batonPath(name))
		if err != nil {
			continue
		}
		shipped, err := docs.Managed(name)
		if err != nil {
			continue
		}
		if !bytes.Equal(bytes.TrimRight(installed, "\n"), bytes.TrimRight(shipped, "\n")) {
			state.err("%s differs from the copy this baton binary shipped with; run an update", name)
		}
	}
}

func (state *lintState) checkLog() {
	logPath, err := state.app.timelinePath()
	if err != nil {
		return
	}
	file, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer file.Close()

	lastEvents := map[string]string{}
	taskKeys := map[string]string{}
	pendingRounds := map[string]string{}
	pendingKeys := map[string]string{}
	reviewResults := map[string]reviewOutcome{}
	legacyLines := 0

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		record, parseErr := parseRecord(line, lineNumber)
		if parseErr != nil {
			if isLegacyRecordLine(line) {
				legacyLines++
				continue
			}
			state.err("%s", parseErr)
			continue
		}
		if record.TaskID == "" {
			state.err("missing task-id on line %d", lineNumber)
		}
		expectedRole, validEvent := expectedRoles[record.Event]
		if !validEvent {
			state.err("invalid event on line %d: %s", lineNumber, record.Event)
		} else if record.Role != expectedRole {
			state.err("invalid role on line %d: %s requires %s, got %s", lineNumber, record.Event, expectedRole, record.Role)
		}
		if record.TaskID != "" && validEvent {
			prior := lastEvents[record.TaskID]
			if !validTransition(prior, record.Event) {
				shownPrior := valueOr(prior, "START")
				state.err("invalid transition on line %d for task-id %s: %s -> %s", lineNumber, record.TaskID, shownPrior, record.Event)
			}
			lastEvents[record.TaskID] = record.Event
		}

		requiresArtifact := record.Event == eventPlanned || record.Event == eventExecuted || record.Event == eventReview || record.Event == eventClose
		if requiresArtifact && record.Path == "" {
			state.err("missing path on line %d: %s", lineNumber, record.Event)
		}
		if record.Path != "" {
			switch {
			case requiresArtifact:
				if artifactErr := state.app.checkArtifact(record.Event, record.Path, record.TaskID, true); artifactErr != nil {
					state.err("Baton log line %d: %s", lineNumber, artifactErr)
				}
			case record.Event == eventRequest:
				if pathErr := validateRequestPath(record.Path); pathErr != nil {
					state.err("Baton log line %d: %s", lineNumber, pathErr)
				}
			default:
				if info, statErr := os.Stat(state.app.projectPath(record.Path)); statErr != nil || info.IsDir() {
					state.err("artifact path not found on line %d: %s", lineNumber, record.Path)
				}
			}
		}

		switch record.Event {
		case eventPlanned:
			if record.Path != "" {
				taskKeys[record.TaskID] = artifactKey(record.Path, record.Event)
			}
		case eventExecuted:
			if record.Path == "" {
				continue
			}
			round := artifactRound(record.Path, record.Event)
			key := artifactKey(record.Path, record.Event)
			if key != taskKeys[record.TaskID] {
				state.err("RUN artifact key does not match PLAN on line %d for task-id %s", lineNumber, record.TaskID)
			}
			if round == "" {
				state.err("invalid RUN artifact name on line %d: %s", lineNumber, record.Path)
			} else {
				pendingRounds[record.TaskID] = round
				pendingKeys[record.TaskID] = key
			}
		case eventReview:
			if record.Path == "" {
				continue
			}
			round := artifactRound(record.Path, record.Event)
			key := artifactKey(record.Path, record.Event)
			pendingRound, hasRun := pendingRounds[record.TaskID]
			if round == "" {
				state.err("invalid REVIEW artifact name on line %d: %s", lineNumber, record.Path)
			} else if !hasRun {
				state.err("REVIEW has no corresponding EXECUTED on line %d for task-id %s", lineNumber, record.TaskID)
			} else {
				if round != pendingRound {
					state.err("round mismatch on line %d for task-id %s: RUN-%s -> REVIEW-%s", lineNumber, record.TaskID, pendingRound, round)
				}
				if key != pendingKeys[record.TaskID] {
					state.err("artifact key mismatch on line %d for task-id %s", lineNumber, record.TaskID)
				}
				if key != taskKeys[record.TaskID] {
					state.err("REVIEW artifact key does not match PLAN on line %d for task-id %s", lineNumber, record.TaskID)
				}
				delete(pendingRounds, record.TaskID)
				delete(pendingKeys, record.TaskID)
			}
			// The error is discarded here, not ignored: an unreadable artifact
			// already produced a lint error above through checkArtifact on
			// this same log line, and reviewUnknown (the zero value) still
			// compares unequal to reviewReady below, so the CLOSE guard fails
			// closed regardless.
			outcome, _ := state.app.reviewResult(record.Path)
			reviewResults[record.TaskID] = outcome
		case eventClose:
			if record.Path != "" && artifactKey(record.Path, record.Event) != taskKeys[record.TaskID] {
				state.err("CLOSE artifact key does not match PLAN on line %d for task-id %s", lineNumber, record.TaskID)
			}
			if result, found := reviewResults[record.TaskID]; found && result != reviewReady {
				state.err("CLOSE on line %d follows a REVIEW reporting blockers for task-id %s", lineNumber, record.TaskID)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		state.err("cannot read %s: %s", logPath, err)
	}
	if legacyLines > 0 {
		state.ok("legacy Baton log lines preserved: %d", legacyLines)
	}

	closedCount := 0
	for _, event := range lastEvents {
		if event == eventClose || event == eventRunDone {
			closedCount++
		}
	}
	state.ok("Baton tasks: %d total, %d closed, %d open", len(lastEvents), closedCount, len(lastEvents)-closedCount)
}

// reviewOutcome is the safety-relevant reading of a REVIEW artifact's
// Result line. reviewUnknown is the zero value on purpose: it equals neither
// reviewReady nor reviewBlockers, so a call site that discards the error
// (result, _ := a.reviewResult(path)) still lands on the same value an
// unreadable artifact produces, and a comparison against either named
// outcome still fails closed instead of silently reading as ready.
type reviewOutcome int

const (
	reviewUnknown reviewOutcome = iota
	reviewReady
	reviewBlockers
)

// reviewResult reports the Result a REVIEW artifact records, so callers can
// detect a review with blockers. It returns (reviewUnknown, err) when the
// artifact cannot be read, so the read failure is carried by both a distinct
// zero-value outcome and a returned error, not by a value that could be
// mistaken for a normal answer.
func (a *App) reviewResult(path string) (reviewOutcome, error) {
	content, err := os.ReadFile(a.projectPath(path))
	if err != nil {
		return reviewUnknown, err
	}
	if hasExactLine(string(content), "Result: ready-for-user-decision") {
		return reviewReady, nil
	}
	return reviewBlockers, nil
}

func (a *App) Lint() error {
	state := &lintState{app: a}
	for _, name := range []string{"PROTOCOL.md", "DIRECTOR.md", "PLANNER.md", "EXECUTOR.md", "HOW-TO-UPDATE.md", "VERSION", "GUIDANCE.md", "LESSON-LEARNED.md"} {
		state.requireFile(a.batonPath(name))
	}
	state.requireTimeline()
	state.requireFile(a.installedBinaryPath())
	state.requireFile(a.batonPath("bin", "SHA256SUMS"))
	for _, name := range []string{"templates", "runs", "lesson-learned", "bin"} {
		state.requireDir(a.batonPath(name))
	}
	if a.GOOS != "windows" {
		if info, err := os.Stat(a.installedBinaryPath()); err == nil && info.Mode()&0o111 != 0 {
			state.ok("baton binary is executable")
		} else {
			state.err("baton binary is not executable")
		}
	}
	if err := verifyChecksum(a.installedBinaryPath(), a.batonPath("bin", "SHA256SUMS"), filepath.Join(a.GOOS+"-"+a.GOARCH, binaryName(a.GOOS))); err != nil {
		state.err("baton binary checksum failed: %s", err)
	} else {
		state.ok("baton binary checksum matches")
	}
	if _, err := os.Stat(a.batonPath("scripts")); os.IsNotExist(err) {
		state.ok("legacy scripts directory absent")
	} else {
		state.err("legacy scripts directory must be removed")
	}
	if _, err := os.Stat(a.batonPath("protocol-guard")); os.IsNotExist(err) {
		state.ok("legacy protocol-guard absent")
	} else {
		state.err("legacy protocol-guard must be removed")
	}
	state.checkAgentBlocks()
	state.checkManagedDocuments()
	state.checkGitTracking()
	state.checkRequiredPathsTracked()
	state.checkLog()
	if state.errors > 0 {
		return fmt.Errorf("baton-lint failed: %d error(s)", state.errors)
	}
	fmt.Fprintln(a.Stdout, "baton-lint passed")
	return nil
}
