package baton

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// handoffFile is the session-state record a stopping session can seal to
// carry what dies with it. It is not a round artifact: it gets no
// BATON-LOG.txt event and no runs/ key, the same footing as CONCURRENCY.md.
const handoffFile = "HANDOFF.md"

var (
	handoffDateLine      = regexp.MustCompile(`(?m)^Date:.*$`)
	handoffLogLineLine   = regexp.MustCompile(`(?m)^Log Line:.*$`)
	handoffDateValue     = regexp.MustCompile(`(?m)^Date:[[:space:]]*(.+)$`)
	handoffLogLineValue  = regexp.MustCompile(`(?m)^Log Line:[[:space:]]*(.+)$`)
	handoffDelegatesLine = regexp.MustCompile(`(?m)^## Delegates[ \t]*\n`)
	handoffNextHeading   = regexp.MustCompile(`(?m)^## `)
)

// handoffDelegatesMarker opens the stamped body of the ## Delegates section.
// sealHandoff refuses to overwrite anything under that heading unless the
// existing body is either the template's unfilled placeholder or a body this
// marker already opens -- i.e. one the binary itself stamped, which a
// re-seal is free to replace. Anything else is a hand-written value sitting
// where the stamp goes, and gets refused rather than silently clobbered. It
// deliberately avoids angle brackets so placeholderPattern (which this
// section's own placeholder text uses) never matches the marker itself.
const handoffDelegatesMarker = "[baton-generated: do not edit by hand; refreshed by 'baton handoff']"

// handoffOutcome is the explicit state handoffState reports. Each state is
// its own value rather than an empty string or zero standing in for
// "unknown" (see lesson-learned/sentinel-lets-callers-choose.md): a caller
// that ignores the accompanying error still lands on an outcome that is
// either accurate or, for handoffUnreadable, safely not mistaken for live.
type handoffOutcome int

const (
	handoffAbsent handoffOutcome = iota
	handoffLive
	handoffSuperseded
	handoffUnreadable
)

// handoffStatus is what handoffState reports about .baton/HANDOFF.md: whether
// one exists, whether the log has moved past it, and, for an unreadable
// record, why it could not be classified.
type handoffStatus struct {
	outcome     handoffOutcome
	path        string
	sealedDate  string
	sealedLine  int
	eventsSince int
	reason      string
}

// handoffState classifies the current .baton/HANDOFF.md against records, the
// already-read BATON-LOG.txt: absent when no file exists, unreadable when it
// exists but its Date or Log Line cannot be parsed (never reported as live),
// superseded once any event with a higher line number than the one it was
// sealed against has been appended, and live otherwise. Staleness has this
// one axis only, the append-only log; the plan considered and rejected a date
// or commit-based heuristic, so a handoff sealed for work that never appends
// another event stays live until cleared with 'baton handoff --clear'.
func (a *App) handoffState(records []Record) (handoffStatus, error) {
	path := a.batonPath(handoffFile)
	reportPath := ".baton/" + handoffFile
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return handoffStatus{outcome: handoffAbsent}, nil
		}
		return handoffStatus{outcome: handoffUnreadable, path: reportPath, reason: err.Error()}, err
	}
	content := string(contentBytes)
	dateMatch := handoffDateValue.FindStringSubmatch(content)
	lineMatch := handoffLogLineValue.FindStringSubmatch(content)
	if dateMatch == nil || lineMatch == nil {
		return handoffStatus{outcome: handoffUnreadable, path: reportPath, reason: "missing Date or Log Line"}, nil
	}
	sealedLine, convErr := strconv.Atoi(strings.TrimSpace(lineMatch[1]))
	if convErr != nil {
		return handoffStatus{outcome: handoffUnreadable, path: reportPath, reason: "Log Line is not a number"}, nil
	}
	currentLine := 0
	if len(records) > 0 {
		currentLine = records[len(records)-1].Line
	}
	eventsSince := currentLine - sealedLine
	if eventsSince < 0 {
		// The binary owns Log Line:, so a negative value here means the
		// stamp was hand-edited to a line past the log's current end, or the
		// log itself was truncated since sealing. Either way this is not a
		// record the log has moved past, so it reads as live (0 events
		// since) rather than as somehow "more than live".
		eventsSince = 0
	}
	status := handoffStatus{
		path:       reportPath,
		sealedDate: strings.TrimSpace(dateMatch[1]),
		sealedLine: sealedLine,
	}
	if eventsSince > 0 {
		status.outcome = handoffSuperseded
		status.eventsSince = eventsSince
		return status, nil
	}
	status.outcome = handoffLive
	return status, nil
}

// printHandoffLine prints the one status line for the current
// .baton/HANDOFF.md, or nothing when none exists. status and reportOpenTasks
// both call this so a sealed record surfaces without anyone knowing to look
// for it.
func (a *App) printHandoffLine(records []Record) {
	status, _ := a.handoffState(records)
	switch status.outcome {
	case handoffLive:
		fmt.Fprintf(a.Stdout, "handoff: %s | sealed: %s | read it before acting\n", status.path, status.sealedDate)
	case handoffSuperseded:
		fmt.Fprintf(a.Stdout, "handoff_superseded: %s | sealed: %s | %d event(s) appended since; clear it with 'baton handoff --clear'\n", status.path, status.sealedDate, status.eventsSince)
	case handoffUnreadable:
		fmt.Fprintf(a.Stdout, "handoff_unreadable: %s | %s\n", status.path, status.reason)
	}
}

func (a *App) runHandoff(args []string) error {
	parsed, err := parseArguments(args, nil, map[string]bool{"--clear": true})
	if err != nil {
		return err
	}
	if len(parsed.pos) != 0 {
		return errors.New("handoff accepts flags only")
	}
	if parsed.flags["--clear"] {
		return a.clearHandoff()
	}
	return a.sealHandoff()
}

// clearHandoff removes .baton/HANDOFF.md. Removing a file that is not there
// is an error, not a silent success, so a stray --clear cannot be mistaken
// for having cleared something.
func (a *App) clearHandoff() error {
	path := a.batonPath(handoffFile)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no handoff record to clear: .baton/%s", handoffFile)
		}
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "cleared: .baton/%s\n", handoffFile)
	return nil
}

// sealHandoff stamps and validates .baton/HANDOFF.md, writing it back only on
// success. It refuses, writing nothing, when the file is absent, when a
// placeholder-token remains anywhere in it after stamping, or when a required
// title or section line is missing.
func (a *App) sealHandoff() error {
	path := a.batonPath(handoffFile)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no .baton/%s to seal; fill in .baton/templates/handoff.md as .baton/%s first", handoffFile, handoffFile)
		}
		return err
	}
	content := string(contentBytes)

	records, err := a.readRecords()
	if err != nil {
		return err
	}
	lastLine := 0
	if len(records) > 0 {
		lastLine = records[len(records)-1].Line
	}

	content = handoffDateLine.ReplaceAllString(content, "Date: "+a.Now().Format("2006-01-02"))
	content = handoffLogLineLine.ReplaceAllString(content, fmt.Sprintf("Log Line: %d", lastLine))

	// Re-sealing an already-sealed record, superseded or not, is an explicit
	// act by a session that is stopping again: it re-stamps Date:, Log
	// Line:, and Delegates from what is on disk right now, and a previously
	// superseded record reads as live again afterward. There is no separate
	// "still superseded" state to preserve; sealing always means "this is
	// current as of now".
	content, err = a.stampDelegatesSection(content, records)
	if err != nil {
		return err
	}

	if placeholderPattern.MatchString(content) {
		return fmt.Errorf("handoff record still has an unresolved placeholder; fill it in before sealing: .baton/%s", handoffFile)
	}
	checks := [][2]string{
		{`^# HANDOFF[[:space:]]*$`, "must have a HANDOFF title"},
		{`^Date:[[:space:]]*[0-9]{4}-[0-9]{2}-[0-9]{2}[[:space:]]*$`, "must include a stamped ISO Date"},
		{`^Log Line:[[:space:]]*[0-9]+[[:space:]]*$`, "must include a stamped Log Line"},
		{`^## Branch Strategy[[:space:]]*$`, "must include Branch Strategy"},
		{`^## Unrecorded Decisions[[:space:]]*$`, "must include Unrecorded Decisions"},
		{`^## Delegates[[:space:]]*$`, "must include Delegates"},
		{`^## Next Intent[[:space:]]*$`, "must include Next Intent"},
	}
	for _, check := range checks {
		if !regexp.MustCompile("(?m)" + check[0]).MatchString(content) {
			return fmt.Errorf("handoff record %s: .baton/%s", check[1], handoffFile)
		}
	}
	if err := atomicWrite(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "sealed: .baton/%s\n", handoffFile)
	return nil
}

// stampDelegatesSection replaces the body of the ## Delegates section with a
// freshly generated one. It refuses, leaving content unchanged, when that
// body is neither the template's unfilled placeholder nor a body this
// binary already stamped (handoffDelegatesMarker) -- a hand-written value
// sitting where the stamp goes. A missing ## Delegates heading is left for
// the required-section checks in sealHandoff to report.
func (a *App) stampDelegatesSection(content string, records []Record) (string, error) {
	loc := handoffDelegatesLine.FindStringIndex(content)
	if loc == nil {
		return content, nil
	}
	bodyStart := loc[1]
	rest := content[bodyStart:]
	bodyEnd := len(content)
	if end := handoffNextHeading.FindStringIndex(rest); end != nil {
		bodyEnd = bodyStart + end[0]
	}
	body := content[bodyStart:bodyEnd]
	if strings.TrimSpace(body) != "" && !placeholderPattern.MatchString(body) && !strings.Contains(body, handoffDelegatesMarker) {
		return content, fmt.Errorf("the ## Delegates section is written by 'baton handoff', not by hand; run 'baton handoff' again instead of editing it: .baton/%s", handoffFile)
	}
	stamped := "\n" + a.delegatesStampedBody(records) + "\n\n"
	return content[:bodyStart] + stamped + content[bodyEnd:], nil
}

// delegatesStampedBody builds the ## Delegates block: one line per open
// task, naming its key, last event, next gate, and the artifact that stage
// is expected to produce with its on-disk state. It reuses openTaskList
// (pause.go), nextGateDescription, and expectedArtifact rather than
// re-deriving any of that from the log.
func (a *App) delegatesStampedBody(records []Record) string {
	open := openTaskList(records)
	lines := []string{handoffDelegatesMarker}
	if len(open) == 0 {
		lines = append(lines, "- no open tasks")
	}
	for _, task := range open {
		key := "unknown"
		if request, found := lastRecord(records, task.id, eventRequest); found && request.Path != "" {
			key = filepath.Base(artifactKey(request.Path, eventPlanned))
		}
		nextGate, _ := a.nextGateDescription(records, task.id, task.lastEvent)
		artifact := "none"
		if event, path, ok := a.expectedArtifact(records, task.id, task.lastEvent); ok {
			artifact = fmt.Sprintf("%s: %s", path, a.delegateArtifactState(event, path, task.id))
		}
		lines = append(lines, fmt.Sprintf("- %s (key %s): last %s, next %s, artifact %s",
			task.id, key, valueOr(task.lastEvent, "none"), valueOr(nextGate, "unknown"), artifact))
	}
	return strings.Join(lines, "\n")
}

// delegateArtifactState reports the on-disk state of an expected artifact:
// missing (no file there yet), checkpoint (Status: checkpoint), or complete
// but not yet appended to the log (passes CheckArtifact) -- or incomplete for
// a file that exists but is neither, e.g. one still being written.
func (a *App) delegateArtifactState(event, path, taskID string) string {
	content, err := os.ReadFile(a.projectPath(path))
	if err != nil {
		return "missing"
	}
	if hasExactLine(string(content), "Status: checkpoint") {
		return "checkpoint"
	}
	if err := a.CheckArtifact(event, path, taskID); err != nil {
		return "incomplete"
	}
	return "complete, not appended"
}
