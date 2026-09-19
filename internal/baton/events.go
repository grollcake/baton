package baton

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	eventRequest  = "REQUEST"
	eventPlanned  = "PLANNED"
	eventExecuted = "EXECUTED"
	eventReview   = "REVIEW"
	eventFeedback = "FEEDBACK"
	eventClose    = "CLOSE"
	eventRunDone  = "RUN_DONE"
)

var expectedRoles = map[string]string{
	eventRequest:  "Director",
	eventPlanned:  "Planner",
	eventExecuted: "Executor",
	eventReview:   "Planner",
	eventFeedback: "Director",
	eventClose:    "Director",
	eventRunDone:  "Director",
}

type Record struct {
	Timestamp string
	TaskID    string
	Event     string
	Role      string
	Summary   string
	Path      string
	Raw       string
	Line      int
}

func parseRecord(line string, lineNumber int) (Record, error) {
	fields := strings.Split(line, "|")
	if len(fields) < 4 || len(fields) > 6 {
		return Record{}, fmt.Errorf("malformed Baton log line %d", lineNumber)
	}
	record := Record{
		Timestamp: strings.TrimSpace(fields[0]),
		TaskID:    strings.TrimSpace(fields[1]),
		Event:     strings.TrimSpace(fields[2]),
		Role:      strings.TrimSpace(fields[3]),
		Raw:       line,
		Line:      lineNumber,
	}
	if len(fields) >= 5 {
		record.Summary = strings.TrimSpace(fields[4])
	}
	if len(fields) == 6 {
		record.Path = strings.TrimSpace(fields[5])
	}
	return record, nil
}

func isLegacyRecordLine(line string) bool {
	return strings.Contains(line, "agent=") || strings.Contains(line, "task=") ||
		strings.Contains(line, "TASK_BEGIN") || strings.Contains(line, "TASK_END")
}

func (a *App) readRecords() ([]Record, error) {
	file, err := os.Open(a.batonPath("baton.log"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []Record
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		record, parseErr := parseRecord(line, lineNumber)
		if parseErr != nil {
			if isLegacyRecordLine(line) {
				continue
			}
			return nil, parseErr
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func lastEvent(records []Record, taskID string) string {
	last := ""
	for _, record := range records {
		if record.TaskID == taskID {
			last = record.Event
		}
	}
	return last
}

func lastRecord(records []Record, taskID, event string) (Record, bool) {
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].TaskID == taskID && records[i].Event == event {
			return records[i], true
		}
	}
	return Record{}, false
}

func validTransition(prior, next string) bool {
	switch prior + ":" + next {
	case ":REQUEST",
		"REQUEST:PLANNED", "REQUEST:RUN_DONE",
		"PLANNED:EXECUTED",
		"EXECUTED:REVIEW",
		"REVIEW:EXECUTED", "REVIEW:FEEDBACK", "REVIEW:CLOSE",
		"FEEDBACK:EXECUTED":
		return true
	default:
		return false
	}
}

func validateTransition(taskID, prior, next string) error {
	if validTransition(prior, next) {
		return nil
	}
	if prior == "" {
		return fmt.Errorf("invalid transition for task-id %s: task must start with REQUEST, got %s", taskID, next)
	}
	return fmt.Errorf("invalid transition for task-id %s: %s -> %s", taskID, prior, next)
}

func formatRecord(record Record) string {
	if record.Path != "" {
		return fmt.Sprintf("%s | %-4s | %-8s | %-8s | %s | %s", record.Timestamp, record.TaskID, record.Event, record.Role, record.Summary, record.Path)
	}
	return fmt.Sprintf("%s | %-4s | %-8s | %-8s | %s", record.Timestamp, record.TaskID, record.Event, record.Role, record.Summary)
}

func (a *App) appendRecord(record Record) (string, error) {
	line := formatRecord(record)
	file, err := os.OpenFile(a.batonPath("baton.log"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.WriteString(file, line+"\n"); err != nil {
		return "", err
	}
	return line, nil
}

func validateLogField(name, value string) error {
	if strings.ContainsRune(value, '|') {
		return fmt.Errorf("%s must not contain | or control characters", name)
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf("%s must not contain | or control characters", name)
		}
	}
	return nil
}

func (a *App) validateArtifactForEvent(records []Record, event, path, taskID string) error {
	if err := a.CheckArtifact(event, path, taskID); err != nil {
		return err
	}
	if event == eventPlanned {
		return nil
	}

	planned, found := lastRecord(records, taskID, eventPlanned)
	if !found {
		return fmt.Errorf("%s has no PLAN for task-id %s", event, taskID)
	}
	key := strings.TrimSuffix(planned.Path, "-PLAN.md")
	matched := false
	switch event {
	case eventExecuted:
		matched, _ = regexp.MatchString("^"+regexp.QuoteMeta(key)+`-RUN-[0-9]{2}\.md$`, path)
	case eventReview:
		matched, _ = regexp.MatchString("^"+regexp.QuoteMeta(key)+`-REVIEW-[0-9]{2}\.md$`, path)
	case eventClose:
		matched = path == key+"-CLOSE.md"
	}
	if !matched {
		return fmt.Errorf("%s artifact key does not match PLAN: %s", event, path)
	}
	if event == eventReview {
		executed, found := lastRecord(records, taskID, eventExecuted)
		if !found {
			return fmt.Errorf("REVIEW has no RUN for task-id %s", taskID)
		}
		round := strings.TrimSuffix(executed.Path[strings.LastIndex(executed.Path, "-RUN-")+5:], ".md")
		if path != key+"-REVIEW-"+round+".md" {
			return fmt.Errorf("REVIEW round does not match the latest RUN: %s", path)
		}
	}
	return nil
}

func (a *App) runAppend(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--task-id": true, "--role": true, "--summary": true, "--path": true}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 1 {
		return errors.New("append requires one EVENT")
	}
	event := parsed.pos[0]
	expectedRole, valid := expectedRoles[event]
	if !valid {
		return fmt.Errorf("invalid event: %s", event)
	}
	taskID, err := requireValue(parsed, "--task-id")
	if err != nil {
		return err
	}
	role, err := requireValue(parsed, "--role")
	if err != nil {
		return err
	}
	summary, err := requireValue(parsed, "--summary")
	if err != nil {
		return err
	}
	path := parsed.values["--path"]
	if err := validateLogField("--summary", summary); err != nil {
		return err
	}
	if err := validateLogField("--path", path); err != nil {
		return err
	}
	if role != expectedRole {
		return fmt.Errorf("%s requires --role %s", event, expectedRole)
	}
	records, err := a.readRecords()
	if err != nil {
		return err
	}
	if err := validateTransition(taskID, lastEvent(records, taskID), event); err != nil {
		return err
	}
	if event == eventPlanned || event == eventExecuted || event == eventReview || event == eventClose {
		if path == "" {
			return fmt.Errorf("%s requires --path", event)
		}
		if err := a.validateArtifactForEvent(records, event, path, taskID); err != nil {
			return err
		}
	}
	record := Record{
		Timestamp: a.Now().Format("2006-01-02T15:04:05"),
		TaskID:    taskID, Event: event, Role: role, Summary: summary, Path: path,
	}
	line, err := a.appendRecord(record)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.Stdout, line)
	return nil
}

func (a *App) runGate(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--task-id": true}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 1 {
		return errors.New("gate requires one gate name")
	}
	taskID, err := requireValue(parsed, "--task-id")
	if err != nil {
		return err
	}
	records, err := a.readRecords()
	if err != nil {
		return err
	}
	event := lastEvent(records, taskID)
	if event == "" {
		return fmt.Errorf("gate failed: no events for task-id %s", taskID)
	}
	gate := parsed.pos[0]
	allowed := (gate == "before-execute" && (event == eventPlanned || event == eventReview || event == eventFeedback)) ||
		(gate == "before-review" && event == eventExecuted) ||
		(gate == "before-approval" && event == eventReview)
	if !allowed {
		if gate != "before-execute" && gate != "before-review" && gate != "before-approval" {
			return fmt.Errorf("unknown gate: %s", gate)
		}
		return fmt.Errorf("gate failed: %s is not allowed after %s for task-id %s", gate, event, taskID)
	}
	record, _ := lastRecord(records, taskID, event)
	if gate == "before-approval" {
		if err := a.requireReviewReady(record, taskID); err != nil {
			return err
		}
	}
	fmt.Fprintln(a.Stdout, record.Raw)
	return nil
}

// requireReviewReady blocks the approval gate while the latest REVIEW reports
// blockers, so a blocked review cannot reach the user as if it were ready.
func (a *App) requireReviewReady(review Record, taskID string) error {
	content, err := os.ReadFile(a.projectPath(review.Path))
	if err != nil {
		return err
	}
	if !hasExactLine(string(content), "Result: ready-for-user-decision") {
		return fmt.Errorf("gate failed: REVIEW reports blockers for task-id %s; run another EXECUTED -> REVIEW round: %s", taskID, review.Path)
	}
	return nil
}

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func (a *App) generateTaskID(records []Record) (string, error) {
	used := make(map[string]bool, len(records))
	for _, record := range records {
		used[record.TaskID] = true
	}
	buffer := make([]byte, 4)
	for attempt := 0; attempt < 100; attempt++ {
		if _, err := io.ReadFull(a.Random, buffer); err != nil {
			return "", err
		}
		for i := range buffer {
			buffer[i] = 'a' + buffer[i]%26
		}
		candidate := string(buffer)
		if !used[candidate] {
			return candidate, nil
		}
	}
	return "", errors.New("task-id generation exhausted retries")
}

func (a *App) runNewRound(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--summary": true, "--branch": true}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 1 || !slugPattern.MatchString(parsed.pos[0]) {
		return errors.New("slug must be lowercase kebab-case (a-z, 0-9, -)")
	}
	summary, err := requireValue(parsed, "--summary")
	if err != nil {
		return err
	}
	if err := validateLogField("--summary", summary); err != nil {
		return err
	}
	records, err := a.readRecords()
	if err != nil {
		return err
	}
	if branch := parsed.values["--branch"]; branch != "" {
		if _, err := a.runGit("rev-parse", "--git-dir"); err != nil {
			return errors.New("not in a git repo")
		}
		listed, err := a.runGit("branch", "--list", branch)
		if err != nil {
			return err
		}
		if listed != "" {
			return fmt.Errorf("branch already exists: %s", branch)
		}
		if _, err := a.runGit("checkout", "-b", branch); err != nil {
			return fmt.Errorf("branch create failed: %w", err)
		}
	}
	taskID, err := a.generateTaskID(records)
	if err != nil {
		return err
	}
	now := a.Now()
	key := now.Format("20060102-1504") + "-" + parsed.pos[0]
	matches, err := filepath.Glob(a.batonPath("runs", key+"-*"))
	if err != nil {
		return err
	}
	if len(matches) > 0 {
		return fmt.Errorf("artifact key already exists: %s", key)
	}
	line, err := a.appendRecord(Record{
		Timestamp: now.Format("2006-01-02T15:04:05"), TaskID: taskID,
		Event: eventRequest, Role: "Director", Summary: summary,
	})
	if err != nil {
		return err
	}
	_ = line
	fmt.Fprintf(a.Stdout, "task_id='%s'\nkey='%s'\nplan_path='.baton/runs/%s-PLAN.md'\nrun_path_template='.baton/runs/%s-RUN-NN.md'\nreview_path_template='.baton/runs/%s-REVIEW-NN.md'\nclose_path='.baton/runs/%s-CLOSE.md'\n", taskID, key, key, key, key, key)
	return nil
}

func (a *App) runFeedback(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--task-id": true, "--summary": true}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 0 {
		return errors.New("feedback accepts flags only")
	}
	taskID, err := requireValue(parsed, "--task-id")
	if err != nil {
		return err
	}
	summary, err := requireValue(parsed, "--summary")
	if err != nil {
		return err
	}
	return a.runAppend([]string{eventFeedback, "--task-id", taskID, "--role", "Director", "--summary", summary})
}

func (a *App) runStatus(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--task-id": true}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 0 {
		return errors.New("status accepts flags only")
	}
	records, err := a.readRecords()
	if err != nil {
		return err
	}
	taskID := parsed.values["--task-id"]
	if taskID == "" {
		closed := map[string]bool{}
		for _, record := range records {
			if record.Event == eventClose || record.Event == eventRunDone {
				closed[record.TaskID] = true
			}
			if record.Event == eventRequest && !closed[record.TaskID] {
				taskID = record.TaskID
			}
		}
		if taskID != "" && closed[taskID] {
			taskID = ""
			for i := len(records) - 1; i >= 0; i-- {
				if records[i].Event == eventRequest && !closed[records[i].TaskID] {
					taskID = records[i].TaskID
					break
				}
			}
		}
	}
	branch := "not-a-git-repo"
	if current, gitErr := a.runGit("rev-parse", "--abbrev-ref", "HEAD"); gitErr == nil {
		branch = current
	}
	if taskID == "" {
		fmt.Fprintln(a.Stdout, "no open task")
		fmt.Fprintf(a.Stdout, "branch: %s\n", branch)
		return nil
	}
	fmt.Fprintf(a.Stdout, "task_id: %s\nevents:\n", taskID)
	for _, record := range records {
		if record.TaskID == taskID {
			fmt.Fprintf(a.Stdout, "  %s\n", record.Raw)
		}
	}
	last := lastEvent(records, taskID)
	fmt.Fprintf(a.Stdout, "last_event: %s\n", valueOr(last, "none"))
	next := map[string]string{
		eventRequest: "PLANNED (delegate Planner)", eventPlanned: "EXECUTED (delegate Executor)",
		eventExecuted: "REVIEW (delegate Planner review)", eventReview: "user approval -> CLOSE",
		eventFeedback: "EXECUTED (resume Executor work)", eventClose: "closed",
		eventRunDone: "closed (direct work)",
	}[last]
	fmt.Fprintf(a.Stdout, "next_gate: %s\nbranch: %s\n", valueOr(next, "unknown"), branch)
	for _, pending := range a.pendingArtifacts(records, taskID, last) {
		fmt.Fprintf(a.Stdout, "pending_artifact: %s\n", pending)
	}
	return nil
}

// pendingArtifacts lists complete artifacts that the log does not record yet,
// so Director recovers when a delegate's completion was missed.
func (a *App) pendingArtifacts(records []Record, taskID, last string) []string {
	event, pattern := "", ""
	planned, hasPlan := lastRecord(records, taskID, eventPlanned)
	key := filepath.Base(artifactKey(planned.Path, eventPlanned))
	switch {
	case last == eventRequest:
		event, pattern = eventPlanned, "*-PLAN.md"
	case !hasPlan:
		return nil
	case last == eventPlanned || last == eventReview || last == eventFeedback:
		event, pattern = eventExecuted, key+"-RUN-*.md"
	case last == eventExecuted:
		executed, _ := lastRecord(records, taskID, eventExecuted)
		event, pattern = eventReview, key+"-REVIEW-"+artifactRound(executed.Path, eventExecuted)+".md"
	default:
		return nil
	}

	matches, _ := filepath.Glob(filepath.Join(a.projectPath(".baton/runs"), pattern))
	var pending []string
	for _, match := range matches {
		path := ".baton/runs/" + filepath.Base(match)
		logged := false
		for _, record := range records {
			if record.TaskID == taskID && record.Event == event && record.Path == path {
				logged = true
			}
		}
		if logged {
			continue
		}
		if a.CheckArtifact(event, path, taskID) == nil {
			pending = append(pending, fmt.Sprintf("%s (complete, not appended as %s)", path, event))
		}
	}
	return pending
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
