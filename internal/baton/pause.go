package baton

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	docs "github.com/grollcake/baton"
)

// pausedBlock is the paused-state <baton-rules> block (Decision 2), the only
// other text lint and blockState ever classify a project's instruction
// blocks against. It is a Go constant, not a bootstrap/ file, because the
// binary is the only thing that ever writes it.
const pausedBlock = "<baton-rules>\n\n## Baton\n\n" +
	"Baton is installed in this project and is paused. Do not follow\n" +
	"`.baton/PROTOCOL.md`, do not run a `baton` command, and do not write under\n" +
	"`.baton/`: work as if Baton were not installed. `.baton/` holds this project's\n" +
	"records; leave every file in it unchanged. Resume only when the user asks for\n" +
	"it, with `.baton/bin/baton resume`.\n\n" +
	"</baton-rules>"

// blockStatus classifies the <baton-rules> block(s) a project's instruction
// files carry.
type blockStatus string

const (
	blockActive  blockStatus = "active"
	blockPaused  blockStatus = "paused"
	blockAbsent  blockStatus = "absent"
	blockUnknown blockStatus = "unknown"
)

// classifyBlock reports whether block matches the active block this binary
// ships (bootstrap/AGENTS.md) or the paused block (Decision 2), or neither.
func classifyBlock(block []byte) (blockStatus, error) {
	active, err := docs.ActiveBlock()
	if err != nil {
		return blockUnknown, err
	}
	switch {
	case bytes.Equal(block, active):
		return blockActive, nil
	case bytes.Equal(block, []byte(pausedBlock)):
		return blockPaused, nil
	default:
		return blockUnknown, nil
	}
}

// instructionFiles lists the instruction files present in the project, in a
// stable order (AGENTS.md, then CLAUDE.md).
func (a *App) instructionFiles() []string {
	var files []string
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		path := filepath.Join(a.ProjectDir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			files = append(files, path)
		}
	}
	return files
}

// classifyInstructionFile reports whether path is present and, if so, how its
// <baton-rules> block classifies: blockActive, blockPaused, blockUnknown (a
// present block matching neither shipped text), or blockAbsent (the file
// exists but carries no block at all -- Decision 6). present is false only
// when the file does not exist; a present file never returns an error here,
// so the caller decides what to do with an absent or unknown block instead of
// this function erroring on their behalf.
func classifyInstructionFile(path string) (status blockStatus, present bool, err error) {
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return blockAbsent, false, nil
	}
	block := batonBlockPattern.Find(content)
	if len(block) == 0 {
		return blockAbsent, true, nil
	}
	status, err = classifyBlock(block)
	return status, true, err
}

// projectBlockState reads whichever of AGENTS.md and CLAUDE.md are present
// and requires them to agree (Decision 6, gap A). blockAbsent now covers two
// cases under one name: no instruction file exists, or every present
// instruction file exists but carries no block at all (the state removal
// produces). It errors, rather than picking a side, when one present file
// carries a block and another does not, or when two present blocks differ --
// both are states pause, resume, and remove must refuse over, not silently
// resolve.
func (a *App) projectBlockState() (blockStatus, error) {
	agentsPath := filepath.Join(a.ProjectDir, "AGENTS.md")
	claudePath := filepath.Join(a.ProjectDir, "CLAUDE.md")
	agentsStatus, haveAgents, err := classifyInstructionFile(agentsPath)
	if err != nil {
		return blockUnknown, err
	}
	claudeStatus, haveClaude, err := classifyInstructionFile(claudePath)
	if err != nil {
		return blockUnknown, err
	}

	switch {
	case !haveAgents && !haveClaude:
		return blockAbsent, nil
	case haveAgents && haveClaude:
		if agentsStatus != claudeStatus {
			switch {
			case agentsStatus == blockAbsent:
				return blockUnknown, errors.New("CLAUDE.md carries a Baton block and AGENTS.md does not")
			case claudeStatus == blockAbsent:
				return blockUnknown, errors.New("AGENTS.md carries a Baton block and CLAUDE.md does not")
			default:
				return blockUnknown, errors.New("AGENTS.md and CLAUDE.md Baton blocks differ")
			}
		}
		return agentsStatus, nil
	case haveAgents:
		return agentsStatus, nil
	default:
		return claudeStatus, nil
	}
}

// writeInstructionBlocks writes plannedContent[i] to files[i]; a nil entry
// deletes that file instead. Every file's original bytes are read first; the
// writes then happen in sequence, and if one after the first fails, every
// already-written file is restored to its original state (Decision 7) --
// recreated from its original bytes if it had one, removed if this call
// created it. This is what keeps "the two instruction files disagree"
// unreachable through a partial pause, resume, or remove: either every file
// ends up in its planned state, or every file is back to what it held before
// this call ran.
func writeInstructionBlocks(files []string, plannedContent [][]byte) error {
	if len(files) != len(plannedContent) {
		return fmt.Errorf("writeInstructionBlocks: %d files but %d planned contents", len(files), len(plannedContent))
	}
	type planned struct {
		path     string
		original []byte
		hadFile  bool
		mode     os.FileMode
		content  []byte
		delete   bool
	}
	plan := make([]planned, 0, len(files))
	for i, path := range files {
		original, err := os.ReadFile(path)
		hadFile := err == nil
		if err != nil && !hadFile && !os.IsNotExist(err) {
			return err
		}
		mode := os.FileMode(0o644)
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
		content := plannedContent[i]
		if content == nil {
			plan = append(plan, planned{path: path, original: original, hadFile: hadFile, mode: mode, delete: true})
			continue
		}
		plan = append(plan, planned{path: path, original: original, hadFile: hadFile, mode: mode, content: content})
	}
	written := make([]planned, 0, len(plan))
	for _, item := range plan {
		var writeErr error
		if item.delete {
			writeErr = os.Remove(item.path)
		} else {
			writeErr = atomicWrite(item.path, item.content, item.mode)
		}
		if writeErr != nil {
			for _, done := range written {
				if done.hadFile {
					_ = atomicWrite(done.path, done.original, done.mode)
				} else {
					_ = os.Remove(done.path)
				}
			}
			return writeErr
		}
		written = append(written, item)
	}
	return nil
}

// spliceBlockIntoFiles computes, for each path in files, that file's content
// with block spliced over its existing <baton-rules> block (or appended, if
// it has none) via computeReplacedContent -- the shared computation
// writeInstructionBlocks needs one planned content per file, and pause and
// resume both splice the same block into every file they touch.
func spliceBlockIntoFiles(files []string, block []byte) ([][]byte, error) {
	content := make([][]byte, len(files))
	for i, path := range files {
		merged, _, err := computeReplacedContent(path, block)
		if err != nil {
			return nil, err
		}
		content[i] = merged
	}
	return content, nil
}

// openTask is the information reportOpenTasks (status --open) and pause and
// resume each display about a task the timeline leaves open.
type openTask struct {
	id        string
	lastEvent string
	summary   string
}

// openTaskList reports every task whose timeline has not reached CLOSE or
// RUN_DONE, in the order their REQUEST first appeared. reportOpenTasks and
// runPause/runResume share this so open-task detection has one
// implementation (Decision 5).
func openTaskList(records []Record) []openTask {
	closed := map[string]bool{}
	var order []string
	for _, record := range records {
		if record.Event == eventRequest {
			order = append(order, record.TaskID)
		}
		if record.Event == eventClose || record.Event == eventRunDone {
			closed[record.TaskID] = true
		}
	}
	var open []openTask
	for _, taskID := range order {
		if closed[taskID] {
			continue
		}
		request, _ := lastRecord(records, taskID, eventRequest)
		open = append(open, openTask{id: taskID, lastEvent: lastEvent(records, taskID), summary: request.Summary})
	}
	return open
}

// pausedUpdateMessage reports the update-trap refusal (Decision 4a) when the
// project is paused, and "" otherwise. runUpdate and runMergeAgentBlock share
// it so update --apply and merge-agent-block refuse with the same message,
// and update's dry run can print it without treating it as a failure.
func (a *App) pausedUpdateMessage() string {
	status, _ := a.projectBlockState()
	if status != blockPaused {
		return ""
	}
	return "baton is paused in this project; update would restore the active rules block.\n" +
		`Run "baton resume", update, then "baton pause" again.`
}

// removedUpdateMessage reports the refusal update --apply must give when no
// instruction file carries a block, which is a removed project or one that
// never had Baton; the tool cannot tell them apart, so the message does not
// claim to (Decision 6): MergeAgentBlock appends a block when it finds
// none, so an unguarded update --apply would silently reinstall Baton.
// merge-agent-block is deliberately not gated by this message -- it is the
// install primitive bootstrap and HOW-TO-UPDATE.md step 7 depend on, and it
// cannot tell "not installed yet" from "removed".
func (a *App) removedUpdateMessage() string {
	status, err := a.projectBlockState()
	if err != nil || status != blockAbsent {
		return ""
	}
	return "no <baton-rules> block is present in this project; update would add one.\n" +
		"If Baton was removed, re-install it with bootstrap. If it was never installed here, bootstrap is the way in."
}

// refusePausedForUpdate refuses runMergeAgentBlock while paused, with the
// same update-trap message runUpdate uses, so a manual update following
// HOW-TO-UPDATE.md step 7 cannot un-pause the project either.
func (a *App) refusePausedForUpdate() error {
	if message := a.pausedUpdateMessage(); message != "" {
		return errors.New(message)
	}
	return nil
}

// refuseIfPaused is the gate App.Run's dispatch applies, once, to every
// command in the plan's refuse list (Decision 3) other than update and
// merge-agent-block, which carry their own more specific refusal (Decision
// 4a). It only refuses a confirmed pause; a drifted or absent block is left
// to lint and to pause/resume's own refusals to report.
func (a *App) refuseIfPaused() error {
	status, _ := a.projectBlockState()
	if status != blockPaused {
		return nil
	}
	return errors.New(`baton is paused in this project; run "baton resume" first (records under .baton/ are unchanged)`)
}

func (a *App) runPause(args []string) error {
	parsed, err := parseArguments(args, nil, map[string]bool{"--force": true})
	if err != nil {
		return err
	}
	if len(parsed.pos) != 0 {
		return errors.New("pause accepts flags only")
	}
	status, err := a.projectBlockState()
	if err != nil {
		return fmt.Errorf("cannot pause: %w", err)
	}
	switch status {
	case blockPaused:
		fmt.Fprintln(a.Stdout, "Baton is already paused in this project")
		return nil
	case blockAbsent:
		return errors.New("cannot pause: no instruction file carries a <baton-rules> block; Baton is not installed here")
	case blockUnknown:
		return errors.New("cannot pause: Baton block matches neither the active nor the paused block this binary ships; run an update or restore it before pausing")
	}

	records, err := a.readRecords()
	if err != nil {
		return err
	}
	open := openTaskList(records)
	if len(open) > 0 && !parsed.flags["--force"] {
		for _, task := range open {
			fmt.Fprintf(a.Stderr, "open_task: %s | last_event: %s | %s\n", task.id, task.lastEvent, task.summary)
		}
		return fmt.Errorf("cannot pause: %d open task(s); close them or re-run with --force", len(open))
	}

	files := a.instructionFiles()
	content, err := spliceBlockIntoFiles(files, []byte(pausedBlock))
	if err != nil {
		return err
	}
	if err := writeInstructionBlocks(files, content); err != nil {
		return err
	}

	summary := "Baton paused"
	if len(open) > 0 {
		ids := make([]string, len(open))
		for i, task := range open {
			ids[i] = task.id
		}
		summary = fmt.Sprintf("Baton paused with %d open task: %s", len(open), strings.Join(ids, ", "))
	}
	taskID, err := a.generateTaskID(records)
	if err != nil {
		return err
	}
	now := a.Now().Format("2006-01-02T15:04:05")
	if _, err := a.appendRecord(Record{Timestamp: now, TaskID: taskID, Event: eventRequest, Role: "Director", Summary: "Pause Baton"}); err != nil {
		return err
	}
	if _, err := a.appendRecord(Record{Timestamp: now, TaskID: taskID, Event: eventRunDone, Role: "Director", Summary: summary}); err != nil {
		return err
	}
	fmt.Fprintln(a.Stdout, summary)
	return nil
}

func (a *App) runResume(args []string) error {
	parsed, err := parseArguments(args, nil, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 0 {
		return errors.New("resume accepts no arguments")
	}
	status, err := a.projectBlockState()
	if err != nil {
		return fmt.Errorf("cannot resume: %w", err)
	}
	switch status {
	case blockActive:
		fmt.Fprintln(a.Stdout, "Baton is already active in this project")
		return nil
	case blockAbsent:
		return errors.New("cannot resume: no instruction file carries a <baton-rules> block; Baton was removed from this project -- re-install it with bootstrap to bring it back")
	case blockUnknown:
		return errors.New("cannot resume: Baton block is not the paused block; it may have been edited while paused")
	}

	active, err := docs.ActiveBlock()
	if err != nil {
		return err
	}
	files := a.instructionFiles()
	content, err := spliceBlockIntoFiles(files, active)
	if err != nil {
		return err
	}
	if err := writeInstructionBlocks(files, content); err != nil {
		return err
	}

	records, err := a.readRecords()
	if err != nil {
		return err
	}
	taskID, err := a.generateTaskID(records)
	if err != nil {
		return err
	}
	now := a.Now().Format("2006-01-02T15:04:05")
	if _, err := a.appendRecord(Record{Timestamp: now, TaskID: taskID, Event: eventRequest, Role: "Director", Summary: "Resume Baton"}); err != nil {
		return err
	}
	if _, err := a.appendRecord(Record{Timestamp: now, TaskID: taskID, Event: eventRunDone, Role: "Director", Summary: "Baton resumed"}); err != nil {
		return err
	}

	for _, task := range openTaskList(records) {
		fmt.Fprintf(a.Stdout, "open_task: %s | last_event: %s | %s\n", task.id, task.lastEvent, task.summary)
	}
	fmt.Fprintln(a.Stdout, `Baton resumed; run "baton status" for open work`)
	return nil
}
