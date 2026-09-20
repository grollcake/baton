package baton

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	artifactPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	placeholderPattern  = regexp.MustCompile(`<[^<>]+>`)
	datePattern         = regexp.MustCompile(`(?m)^Date:[[:space:]]*[0-9]{4}-[0-9]{2}-[0-9]{2}[[:space:]]*$`)
	statusLinePattern   = regexp.MustCompile(`(?m)^Status:`)
)

// templatePlaceholderTokens returns the set of angle-bracket tokens found
// across the project's .baton/templates/*.md files. checkArtifact refuses an
// artifact that still contains one of these tokens verbatim, rather than
// refusing any angle-bracket text: an artifact is free to document CLI syntax
// such as `--task-id <id>` as long as that exact phrase never occurs in a
// template. If the templates cannot be read or yield no token, this returns
// an error instead of an empty set: an empty set would make the check accept
// every artifact silently, turning it off without anything failing.
// removed is true only when lint's checkLog is checking a removed project's
// history (Decision 6): remove deletes .baton/templates/ along with the rest
// of the installed machinery, and erroring here would fail lint on every
// recorded artifact in any removed project that ever ran a Relay task,
// which is not the "record is intact" question lint is supposed to keep
// asking. Every other caller passes removed=false, so a missing templates
// directory in an installed (or being-installed) project still fails closed;
// a templates/ that exists but is empty or token-free is a different, real
// misconfiguration and still errors below regardless of removed.
func (a *App) templatePlaceholderTokens(removed bool) (map[string]bool, error) {
	if removed {
		if _, err := os.Stat(a.batonPath("templates")); os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
	}
	matches, err := filepath.Glob(a.batonPath("templates", "*.md"))
	if err != nil {
		return nil, fmt.Errorf("artifact-check: cannot read templates: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("artifact-check: no template files found under %s", a.batonPath("templates"))
	}
	tokens := map[string]bool{}
	for _, match := range matches {
		content, err := os.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("artifact-check: cannot read template %s: %w", match, err)
		}
		for _, token := range placeholderPattern.FindAllString(string(content), -1) {
			tokens[token] = true
		}
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("artifact-check: templates under %s contain no placeholder tokens", a.batonPath("templates"))
	}
	return tokens, nil
}

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

// validateRequestPath validates the forward reference a REQUEST may carry:
// the PLAN artifact the task will produce. It checks shape only, because the
// file does not exist when new-round records it.
func validateRequestPath(path string) error {
	if err := validateArtifactPath(eventPlanned, path); err != nil {
		return fmt.Errorf("artifact-check: REQUEST path must be a PLAN artifact path: %w", err)
	}
	return nil
}

func (a *App) CheckArtifact(event, path, expectedTaskID string) error {
	return a.checkArtifact(event, path, expectedTaskID, false)
}

// checkArtifact validates an artifact against the current contract, with no
// relaxation for older ones: every artifact this project holds must meet it.
// removed is templatePlaceholderTokens' removed-project bypass, threaded
// through from lint's checkLog only.
func (a *App) checkArtifact(event, path, expectedTaskID string, removed bool) error {
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
	tokens, err := a.templatePlaceholderTokens(removed)
	if err != nil {
		return err
	}
	for token := range tokens {
		if strings.Contains(content, token) {
			return fmt.Errorf("artifact-check: %s artifact contains an unresolved placeholder %s: %s", event, token, path)
		}
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

	switch event {
	case eventPlanned:
		checks := [][2]string{
			{`^# PLAN: .+`, "must have a PLAN title"},
			{`^Planner:[[:space:]]*[^[:space:]].*`, "must identify the Planner"},
			{`^## Director Brief[[:space:]]*$`, "must include Director Brief"},
			{`^## Success Criteria[[:space:]]*$`, "must include Success Criteria"},
			{`^## Validation[[:space:]]*$`, "must include Validation"},
		}
		{
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
		{
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
		{
			checks = append(checks, [2]string{statusCompletePattern, "must set Status: complete"})
		}
		if err := checkArtifactLines(content, checks, event, path); err != nil {
			return err
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
		checks = append(checks,
			[2]string{`^## Plan Deviations[[:space:]]*$`, "must include Plan Deviations"},
			[2]string{`^## Lesson Candidates[[:space:]]*$`, "must include Lesson Candidates"})
		if err := checkArtifactLines(content, checks, event, path); err != nil {
			return err
		}
		for _, section := range []string{"## Plan Deviations", "## Lesson Candidates"} {
			if countSectionItems(content, section) < 1 {
				return fmt.Errorf("artifact-check: %s artifact must list at least one item under %s: %s", event, section, path)
			}
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
