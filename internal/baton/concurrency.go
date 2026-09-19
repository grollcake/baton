package baton

import (
	"fmt"
	"os"
	"regexp"
	"sort"
)

const concurrencyFile = "CONCURRENCY.md"

// inFlightTasks lists the open tasks whose delegate may still be editing the
// checkout. A task last recorded PLANNED or FEEDBACK is in the window between
// the before-execute gate passing and its EXECUTED event being appended, so it
// counts. EXECUTED itself does not count: once a RUN is recorded, nobody is
// editing until the next PLANNED, FEEDBACK, or blocked REVIEW. A task last
// recorded REVIEW counts unless that REVIEW's own artifact records
// "Result: ready-for-user-decision" — REVIEW:EXECUTED is a valid transition
// (see validTransition), so a blocked review is about to be re-executed and is
// editing, while a review awaiting the user's decision is not. This fails
// closed: a missing or unreadable REVIEW artifact reads as reviewUnknown,
// which is not reviewReady, so it still counts as in flight, matching lint's
// reading of the same outcome.
func (a *App) inFlightTasks(records []Record, except string) []string {
	closed := map[string]bool{}
	last := map[string]string{}
	for _, record := range records {
		if record.Event == eventClose || record.Event == eventRunDone {
			closed[record.TaskID] = true
		}
		last[record.TaskID] = record.Event
	}
	var tasks []string
	for taskID, event := range last {
		if closed[taskID] || taskID == except {
			continue
		}
		switch event {
		case eventPlanned, eventFeedback:
			tasks = append(tasks, taskID)
		case eventReview:
			if record, ok := lastRecord(records, taskID, eventReview); ok {
				// Error discarded deliberately: reviewUnknown, the zero value
				// on read failure, is not reviewReady, so the task still
				// counts as in flight either way.
				outcome, _ := a.reviewResult(record.Path)
				if outcome != reviewReady {
					tasks = append(tasks, taskID)
				}
			}
		}
	}
	sort.Strings(tasks)
	return tasks
}

// requireConcurrencyApproval refuses to start work while another task is in
// flight unless CONCURRENCY.md records the user's approval and names every
// task now running. Baton runs all delegates against one checkout, so parallel
// edits are a decision for the user, not a default.
func (a *App) requireConcurrencyApproval(taskID string, others []string) error {
	path := a.batonPath(concurrencyFile)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("gate failed: task-id %s would run beside %v, which share one checkout; record the plan in .baton/%s and get explicit user approval first", taskID, others, concurrencyFile)
	}
	content := string(contentBytes)
	if placeholderPattern.MatchString(content) {
		return fmt.Errorf("gate failed: .baton/%s contains an unresolved placeholder", concurrencyFile)
	}
	for _, check := range [][2]string{
		{`^# CONCURRENCY[[:space:]]*$`, "must have a CONCURRENCY title"},
		{`^Approved By:[[:space:]]*User[[:space:]]*$`, "must record user approval"},
		{`^## Tasks[[:space:]]*$`, "must include Tasks"},
		{`^## Conflict Plan[[:space:]]*$`, "must include Conflict Plan"},
	} {
		if !regexp.MustCompile("(?m)" + check[0]).MatchString(content) {
			return fmt.Errorf("gate failed: .baton/%s %s", concurrencyFile, check[1])
		}
	}
	if !datePattern.MatchString(content) {
		return fmt.Errorf("gate failed: .baton/%s must include an ISO date", concurrencyFile)
	}
	for _, id := range append([]string{taskID}, others...) {
		listed := regexp.MustCompile(`(?m)^- ` + regexp.QuoteMeta(id) + `:[[:space:]]*\S`)
		if !listed.MatchString(content) {
			return fmt.Errorf("gate failed: .baton/%s does not list task-id %s with its owned scope; approval must name every task now running", concurrencyFile, id)
		}
	}
	return nil
}
