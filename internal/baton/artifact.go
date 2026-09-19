package baton

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	artifactPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	placeholderPattern  = regexp.MustCompile(`<[^<>]+>`)
	datePattern         = regexp.MustCompile(`(?m)^Date:[[:space:]]*[0-9]{4}-[0-9]{2}-[0-9]{2}[[:space:]]*$`)
	statusLinePattern   = regexp.MustCompile(`(?m)^Status:`)
)

const statusCompletePattern = `^Status:[[:space:]]*complete[[:space:]]*$`

func artifactNameMatches(event, path string) bool {
	var pattern string
	switch event {
	case eventPlanned:
		pattern = `^\.baton/runs/.+-PLAN\.md$`
	case eventExecuted:
		pattern = `^\.baton/runs/.+-RUN-[0-9]{2}\.md$`
	case eventReview:
		pattern = `^\.baton/runs/.+-REVIEW-[0-9]{2}\.md$`
	case eventClose:
		pattern = `^\.baton/runs/.+-CLOSE\.md$`
	default:
		return false
	}
	return regexp.MustCompile(pattern).MatchString(path)
}

func requireArtifactLine(content, pattern, event, message, path string) error {
	if !regexp.MustCompile("(?m)" + pattern).MatchString(content) {
		return fmt.Errorf("artifact-check: %s artifact %s: %s", event, message, path)
	}
	return nil
}

func validateArtifactPath(event, path string) error {
	if event != eventPlanned && event != eventExecuted && event != eventReview && event != eventClose {
		return fmt.Errorf("artifact-check: unsupported event: %s", event)
	}
	if !strings.HasPrefix(path, ".baton/runs/") {
		return fmt.Errorf("artifact-check: %s path must be under .baton/runs/: %s", event, path)
	}
	if strings.Contains(path, "..") || !artifactPathPattern.MatchString(path) {
		return fmt.Errorf("artifact-check: %s path contains unsupported characters: %s", event, path)
	}
	if !artifactNameMatches(event, path) {
		return fmt.Errorf("artifact-check: %s artifact name does not match its event: %s", event, path)
	}
	return nil
}

func (a *App) CheckArtifact(event, path, expectedTaskID string) error {
	return a.checkArtifact(event, path, expectedTaskID, false)
}

// checkArtifact validates an artifact. Rules added after a project installed
// Baton cannot be met by artifacts already in its append-only history, so
// allowLegacy relaxes those for lint while append still enforces them on new
// artifacts. A present Status must be complete either way.
func (a *App) checkArtifact(event, path, expectedTaskID string, allowLegacy bool) error {
	if err := validateArtifactPath(event, path); err != nil {
		return err
	}

	contentBytes, err := os.ReadFile(a.projectPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("artifact-check: %s path not found: %s", event, path)
		}
		return err
	}
	content := string(contentBytes)
	if placeholderPattern.MatchString(content) {
		return fmt.Errorf("artifact-check: %s artifact contains an unresolved placeholder: %s", event, path)
	}
	if event == eventExecuted && strings.Contains(content, "TODO") {
		return fmt.Errorf("artifact-check: %s artifact contains an unresolved TODO: %s", event, path)
	}
	if err := requireArtifactLine(content, `^Task ID:[[:space:]]*[^[:space:]].*`, event, "must include Task ID", path); err != nil {
		return err
	}
	if expectedTaskID != "" && !hasExactLine(content, "Task ID: "+expectedTaskID) {
		return fmt.Errorf("artifact-check: %s artifact Task ID does not match %s: %s", event, expectedTaskID, path)
	}
	if !datePattern.MatchString(content) {
		return fmt.Errorf("artifact-check: %s artifact must include an ISO date: %s", event, path)
	}

	statusRequired := !allowLegacy || statusLinePattern.MatchString(content)
	switch event {
	case eventPlanned:
		checks := [][2]string{
			{`^# PLAN: .+`, "must have a PLAN title"},
			{`^Planner:[[:space:]]*[^[:space:]].*`, "must identify the Planner"},
			{`^## Director Brief[[:space:]]*$`, "must include Director Brief"},
			{`^## Success Criteria[[:space:]]*$`, "must include Success Criteria"},
			{`^## Validation[[:space:]]*$`, "must include Validation"},
		}
		if statusRequired {
			checks = append(checks, [2]string{statusCompletePattern, "must set Status: complete"})
		}
		return checkArtifactLines(content, checks, event, path)
	case eventExecuted:
		round := artifactRound(path, event)
		checks := [][2]string{
			{`^# RUN-` + regexp.QuoteMeta(round) + `: .+`, "title must match RUN-" + round},
			{`^Executor:[[:space:]]*[^[:space:]].*`, "must identify the Executor"},
			{statusCompletePattern, "must set Status: complete"},
			{`^## Validation[[:space:]]*$`, "must include Validation"},
			{`^## Success Criteria Status[[:space:]]*$`, "must include Success Criteria Status"},
		}
		if !allowLegacy {
			checks = append(checks,
				[2]string{`^## Changes[[:space:]]*$`, "must include Changes"},
				[2]string{`^## Unresolved Risks[[:space:]]*$`, "must include Unresolved Risks"})
		}
		return checkArtifactLines(content, checks, event, path)
	case eventReview:
		round := artifactRound(path, event)
		checks := [][2]string{
			{`^# REVIEW-` + regexp.QuoteMeta(round) + `: .+`, "title must match REVIEW-" + round},
			{`^Planner:[[:space:]]*[^[:space:]].*`, "must identify the Planner"},
			{`^Result:[[:space:]]*(ready-for-user-decision|blockers)[[:space:]]*$`, "must set a valid Result"},
			{`^## Suggested User Checks[[:space:]]*$`, "must include Suggested User Checks"},
			{`^## Evidence Reviewed[[:space:]]*$`, "must include Evidence Reviewed"},
		}
		if statusRequired {
			checks = append(checks, [2]string{statusCompletePattern, "must set Status: complete"})
		}
		if err := checkArtifactLines(content, checks, event, path); err != nil {
			return err
		}
		if allowLegacy {
			return nil
		}
		if items := countSectionItems(content, "## Suggested User Checks"); items < 3 || items > 5 {
			return fmt.Errorf("artifact-check: %s artifact must list three to five Suggested User Checks, found %d: %s", event, items, path)
		}
		return nil
	case eventClose:
		checks := [][2]string{
			{`^# CLOSE: .+`, "must have a CLOSE title"},
			{`^Director:[[:space:]]*[^[:space:]].*`, "must identify the Director"},
			{`^Approved By:[[:space:]]*User[[:space:]]*$`, "must record user approval"},
			{`^## Acceptance[[:space:]]*$`, "must include Acceptance"},
			{`^## Validation Summary[[:space:]]*$`, "must include Validation Summary"},
		}
		if !allowLegacy {
			checks = append(checks, [2]string{`^## Plan Deviations[[:space:]]*$`, "must include Plan Deviations"})
		}
		if err := checkArtifactLines(content, checks, event, path); err != nil {
			return err
		}
		if allowLegacy {
			return nil
		}
		if countSectionItems(content, "## Plan Deviations") < 1 {
			return fmt.Errorf("artifact-check: %s artifact must list at least one Plan Deviations item: %s", event, path)
		}
		return nil
	}
	return nil
}

var listItemPattern = regexp.MustCompile(`^[-*]\s+\S|^[0-9]+\.\s+\S`)

// countSectionItems counts the list items directly under a heading, so a
// REVIEW can be held to the three-to-five manual checks the protocol requires.
func countSectionItems(content, heading string) int {
	lines := strings.Split(content, "\n")
	items, inSection := 0, false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inSection {
				break
			}
			inSection = trimmed == heading
			continue
		}
		if inSection && listItemPattern.MatchString(trimmed) {
			items++
		}
	}
	return items
}

func checkArtifactLines(content string, checks [][2]string, event, path string) error {
	for _, check := range checks {
		if err := requireArtifactLine(content, check[0], event, check[1], path); err != nil {
			return err
		}
	}
	return nil
}

func hasExactLine(content, wanted string) bool {
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if line == wanted {
			return true
		}
	}
	return false
}

func artifactRound(path, event string) string {
	marker := "-RUN-"
	if event == eventReview {
		marker = "-REVIEW-"
	}
	index := strings.LastIndex(path, marker)
	if index == -1 {
		return ""
	}
	return strings.TrimSuffix(path[index+len(marker):], ".md")
}

func artifactKey(path, event string) string {
	switch event {
	case eventPlanned:
		return strings.TrimSuffix(path, "-PLAN.md")
	case eventExecuted:
		return strings.TrimSuffix(path, "-RUN-"+artifactRound(path, event)+".md")
	case eventReview:
		return strings.TrimSuffix(path, "-REVIEW-"+artifactRound(path, event)+".md")
	case eventClose:
		return strings.TrimSuffix(path, "-CLOSE.md")
	default:
		return path
	}
}

func (a *App) runCheckArtifact(args []string) error {
	if len(args) < 2 || len(args) > 3 {
		return errors.New("usage: baton check-artifact <PLANNED|EXECUTED|REVIEW|CLOSE> <path> [task-id]")
	}
	taskID := ""
	if len(args) == 3 {
		taskID = args[2]
	}
	return a.CheckArtifact(args[0], args[1], taskID)
}
