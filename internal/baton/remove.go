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

// shippedTemplateNames lists the template file names this binary ships under
// bootstrap/.baton/templates/, so remove can recognise a project's own
// template (Decision 1) by name rather than by content: a project's own
// customised copy of a shipped template is still named after it, and content
// comparison would treat any customisation as "not Baton's" and leave it
// behind, which is not what "remove" promises for its own installed files.
var shippedTemplateNames = []string{
	"close.md", "concurrency.md", "guidance.md", "lesson-learned.md",
	"plan.md", "review.md", "run.md",
}

// instructionPlan is what remove does to a single present instruction file.
type instructionPlan struct {
	path        string
	unchanged   bool   // carries no Baton block; remove does not touch it
	deleteWhole bool   // Baton wrote the whole file (Decision 3); delete it
	newContent  []byte // the file's content with the block spliced out
}

// removePlan is the deletion plan buildRemovePlan computes, shared by the dry
// run report and the apply path so both describe exactly the same operation.
type removePlan struct {
	instructions []instructionPlan
	blockPresent bool

	version string // .baton/VERSION content, trimmed; "" if absent

	deleteFiles []string // non-bin .baton/ files to delete
	deleteDirs  []string // non-bin .baton/ dirs to rmdir if empty (templates/)

	binDeleteFiles []string // bin/baton[.exe], bin/SHA256SUMS
	binDeleteDirs  []string // bin/, if it will be empty

	keptPaths         []string // paths under .baton/ that survive, for the report
	unrecognizedPaths []string // subset of keptPaths remove declined to recognise

	timelineLines int
	runsCount     int
}

// nothingToDelete reports whether this plan's physical machinery (the
// instruction blocks and everything under .baton/ remove would otherwise
// delete) is already gone, i.e. no baton command run here would change
// anything except append a no-op record.
func (plan removePlan) nothingToDelete() bool {
	return !plan.blockPresent &&
		len(plan.deleteFiles) == 0 && len(plan.deleteDirs) == 0 &&
		len(plan.binDeleteFiles) == 0 && len(plan.binDeleteDirs) == 0
}

// buildInstructionPlan classifies each present instruction file (Decision 3).
// A file with no block is left unchanged. A file whose block is neither the
// active nor the paused text this binary ships stops the whole removal: it is
// collected into a combined error rather than returned on the first miss, so
// a project with two drifted files is told about both at once, and so no
// instruction file is touched -- partially removing one while refusing the
// other is exactly the disagreement removal must never produce.
func (a *App) buildInstructionPlan() ([]instructionPlan, error) {
	active, err := docs.ActiveBlock()
	if err != nil {
		return nil, err
	}
	var plans []instructionPlan
	var unrecognized []string
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		path := filepath.Join(a.ProjectDir, name)
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		location := batonBlockPattern.FindIndex(content)
		if location == nil {
			plans = append(plans, instructionPlan{path: path, unchanged: true})
			continue
		}
		block := content[location[0]:location[1]]
		status, classifyErr := classifyBlock(block)
		if classifyErr != nil {
			return nil, classifyErr
		}
		if status != blockActive && status != blockPaused {
			unrecognized = append(unrecognized, name)
			continue
		}

		normalized := make([]byte, 0, len(content)-len(block)+len(active))
		normalized = append(normalized, content[:location[0]]...)
		normalized = append(normalized, active...)
		normalized = append(normalized, content[location[1]:]...)
		if shipped, shipErr := docs.ShippedInstructionFile(name); shipErr == nil && bytes.Equal(normalized, shipped) {
			plans = append(plans, instructionPlan{path: path, deleteWhole: true})
			continue
		}

		newContent, _, wholeEmpty, removeErr := computeRemovedContent(path)
		if removeErr != nil {
			return nil, removeErr
		}
		if wholeEmpty {
			plans = append(plans, instructionPlan{path: path, deleteWhole: true})
			continue
		}
		plans = append(plans, instructionPlan{path: path, newContent: newContent})
	}
	if len(unrecognized) > 0 {
		return nil, fmt.Errorf("%s: Baton block matches neither the active nor the paused block this binary ships; move your text outside the <baton-rules> markers and re-run", strings.Join(unrecognized, ", "))
	}
	return plans, nil
}

// buildRemovePlan computes the full deletion plan (Decision 1's table),
// without changing anything on disk.
func (a *App) buildRemovePlan() (removePlan, error) {
	instructions, err := a.buildInstructionPlan()
	if err != nil {
		return removePlan{}, err
	}
	plan := removePlan{instructions: instructions}
	for _, ip := range instructions {
		if !ip.unchanged {
			plan.blockPresent = true
		}
	}

	for _, name := range managedBatonFiles {
		path := a.batonPath(name)
		installed, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		shipped, shipErr := docs.Managed(name)
		if shipErr == nil && bytes.Equal(bytes.TrimRight(installed, "\n"), bytes.TrimRight(shipped, "\n")) {
			plan.deleteFiles = append(plan.deleteFiles, path)
			continue
		}
		plan.keptPaths = append(plan.keptPaths, path)
		plan.unrecognizedPaths = append(plan.unrecognizedPaths, path)
	}

	versionPath := a.batonPath("VERSION")
	if versionBytes, readErr := os.ReadFile(versionPath); readErr == nil {
		plan.version = strings.TrimSpace(string(versionBytes))
		plan.deleteFiles = append(plan.deleteFiles, versionPath)
	}

	templatesDir := a.batonPath("templates")
	if entries, readErr := os.ReadDir(templatesDir); readErr == nil {
		shipped := map[string]bool{}
		for _, name := range shippedTemplateNames {
			shipped[name] = true
		}
		empty := true
		for _, entry := range entries {
			full := filepath.Join(templatesDir, entry.Name())
			if !entry.IsDir() && shipped[entry.Name()] {
				plan.deleteFiles = append(plan.deleteFiles, full)
				continue
			}
			empty = false
			plan.keptPaths = append(plan.keptPaths, full)
			plan.unrecognizedPaths = append(plan.unrecognizedPaths, full)
		}
		if empty {
			plan.deleteDirs = append(plan.deleteDirs, templatesDir)
		}
	}

	binDir := a.batonPath("bin")
	if entries, readErr := os.ReadDir(binDir); readErr == nil {
		known := map[string]bool{binaryName(a.GOOS): true, "SHA256SUMS": true}
		empty := true
		for _, entry := range entries {
			full := filepath.Join(binDir, entry.Name())
			if known[entry.Name()] {
				plan.binDeleteFiles = append(plan.binDeleteFiles, full)
				continue
			}
			empty = false
			plan.keptPaths = append(plan.keptPaths, full)
			plan.unrecognizedPaths = append(plan.unrecognizedPaths, full)
		}
		if empty {
			plan.binDeleteDirs = append(plan.binDeleteDirs, binDir)
		}
	}

	for _, name := range []string{timelineFile, legacyTimelineFile, "GUIDANCE.md", "LESSON-LEARNED.md", "lesson-learned", "runs", "CONCURRENCY.md"} {
		path := a.batonPath(name)
		if _, statErr := os.Stat(path); statErr == nil {
			plan.keptPaths = append(plan.keptPaths, path)
		}
	}

	if records, readErr := a.readRecords(); readErr == nil {
		plan.timelineLines = len(records)
	}
	if entries, readErr := os.ReadDir(a.batonPath("runs")); readErr == nil {
		plan.runsCount = len(entries)
	}

	return plan, nil
}

// purgePrecondition checks Decision 2's guard: every path under .baton/ that
// Git does not ignore must be tracked and clean. It returns the commit that
// holds .baton/ when the guard passes, or the paths that block it when it
// does not. In a project that is not a Git repository, or where .baton/ has
// no tracked path at all, it always refuses.
func (a *App) purgePrecondition() (commit string, blocked []string, err error) {
	if _, gitErr := a.runGit("rev-parse", "--git-dir"); gitErr != nil {
		return "", nil, errors.New("cannot --purge: this project is not a Git repository")
	}
	tracked, err := a.runGit("ls-files", "--", a.BatonDir)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(tracked) == "" {
		return "", nil, errors.New("cannot --purge: no path under .baton/ is tracked by Git")
	}
	status, err := a.runGit("status", "--porcelain", "--untracked-files=all", "--", a.BatonDir)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(status) != "" {
		for i, line := range strings.Split(strings.TrimRight(status, "\n"), "\n") {
			// Every `git status --porcelain` line is exactly two status
			// characters, a separator space, then the path -- so line[2] is
			// always ' ' in the untouched output. runGit trims the whole
			// command output (not line by line), which strips a leading
			// space from the very first line only when its first status
			// character is itself a space (e.g. " M path" -> "M path"),
			// shifting that one line left by one byte. Detecting the missing
			// separator on line 0 and cutting from index 2 instead of 3
			// recovers the real path there; every other line is never
			// affected, so it keeps using index 3.
			if i == 0 && len(line) > 2 && line[2] != ' ' {
				blocked = append(blocked, strings.TrimSpace(line[2:]))
				continue
			}
			if len(line) > 3 {
				blocked = append(blocked, strings.TrimSpace(line[3:]))
			}
		}
		return "", blocked, nil
	}
	commit, err = a.runGit("rev-parse", "HEAD")
	if err != nil {
		return "", nil, err
	}
	return commit, nil, nil
}

func (a *App) printRemoveDryRun(plan removePlan, open []openTask, purge bool, purgeCommit string, purgeBlocked []string) {
	version := plan.version
	if version == "" {
		version = "unknown"
	}
	fmt.Fprintf(a.Stdout, "Baton version: %s\n", version)

	for _, ip := range plan.instructions {
		switch {
		case ip.unchanged:
			fmt.Fprintf(a.Stdout, "%s: carries no Baton block; left unchanged\n", ip.path)
		case ip.deleteWhole:
			fmt.Fprintf(a.Stdout, "%s: deleted (Baton wrote the whole file)\n", ip.path)
		default:
			fmt.Fprintf(a.Stdout, "%s: Baton block spliced out\n", ip.path)
		}
	}

	if purge {
		if len(purgeBlocked) > 0 {
			fmt.Fprintln(a.Stdout, "--purge refused: the following paths under .baton/ are not committed and clean:")
			for _, path := range purgeBlocked {
				fmt.Fprintf(a.Stdout, "  %s\n", path)
			}
		} else {
			fmt.Fprintf(a.Stdout, "--purge would delete .baton/ in full; it is held at commit %s\n", purgeCommit)
		}
	} else {
		fmt.Fprintln(a.Stdout, "Delete:")
		for _, path := range plan.deleteFiles {
			fmt.Fprintf(a.Stdout, "  %s\n", path)
		}
		for _, path := range plan.binDeleteFiles {
			fmt.Fprintf(a.Stdout, "  %s\n", path)
		}
		fmt.Fprintln(a.Stdout, "Keep:")
		for _, path := range plan.keptPaths {
			fmt.Fprintf(a.Stdout, "  %s\n", path)
		}
		fmt.Fprintf(a.Stdout, "timeline: %d line(s); runs/: %d file(s)\n", plan.timelineLines, plan.runsCount)
		if len(plan.unrecognizedPaths) > 0 {
			fmt.Fprintln(a.Stdout, "Declined to recognise as Baton's own (kept):")
			for _, path := range plan.unrecognizedPaths {
				fmt.Fprintf(a.Stdout, "  %s\n", path)
			}
		}
	}

	if len(open) == 0 {
		fmt.Fprintln(a.Stdout, "open tasks: none")
	} else {
		for _, task := range open {
			fmt.Fprintf(a.Stdout, "open_task: %s | last_event: %s | %s\n", task.id, task.lastEvent, task.summary)
		}
	}

	fmt.Fprintln(a.Stdout, "Dry run. Re-run with --apply to remove Baton: after --apply, no baton command can be run from this project, and re-installing means bootstrapping again.")
}

func (a *App) runRemove(args []string) error {
	parsed, err := parseArguments(args, nil, map[string]bool{"--apply": true, "--force": true, "--purge": true})
	if err != nil {
		return err
	}
	if len(parsed.pos) != 0 {
		return errors.New("remove accepts flags only")
	}
	apply := parsed.flags["--apply"]
	force := parsed.flags["--force"]
	purge := parsed.flags["--purge"]

	plan, err := a.buildRemovePlan()
	if err != nil {
		return err
	}
	if plan.nothingToDelete() && (!purge || len(plan.keptPaths) == 0) {
		fmt.Fprintln(a.Stdout, "Baton is not installed in this project")
		return nil
	}

	var records []Record
	if _, statErr := os.Stat(a.batonPath(timelineFile)); statErr == nil {
		if records, err = a.readRecords(); err != nil {
			return err
		}
	} else if _, statErr := os.Stat(a.batonPath(legacyTimelineFile)); statErr == nil {
		if records, err = a.readRecords(); err != nil {
			return err
		}
	}
	open := openTaskList(records)
	if len(open) > 0 && !force {
		for _, task := range open {
			fmt.Fprintf(a.Stderr, "open_task: %s | last_event: %s | %s\n", task.id, task.lastEvent, task.summary)
		}
		return fmt.Errorf("cannot remove: %d open task(s); close them or re-run with --force", len(open))
	}

	var purgeCommit string
	var purgeBlocked []string
	if purge {
		purgeCommit, purgeBlocked, err = a.purgePrecondition()
		if err != nil {
			return err
		}
	}

	if !apply {
		a.printRemoveDryRun(plan, open, purge, purgeCommit, purgeBlocked)
		return nil
	}

	if purge && len(purgeBlocked) > 0 {
		for _, path := range purgeBlocked {
			fmt.Fprintf(a.Stderr, "not committed: %s\n", path)
		}
		return fmt.Errorf("cannot --purge: %d path(s) under .baton/ are not committed and clean", len(purgeBlocked))
	}

	// Order of operations (Decision 4): instruction files first, then the
	// .baton/ deletions, then the timeline append, then bin/ last.
	var files []string
	var content [][]byte
	for _, ip := range plan.instructions {
		if ip.unchanged {
			continue
		}
		files = append(files, ip.path)
		if ip.deleteWhole {
			content = append(content, nil)
		} else {
			content = append(content, ip.newContent)
		}
	}
	if len(files) > 0 {
		if err := writeInstructionBlocks(files, content); err != nil {
			return err
		}
	}

	if purge {
		if err := os.RemoveAll(a.BatonDir); err != nil {
			return err
		}
		version := plan.version
		if version == "" {
			version = "unknown"
		}
		fmt.Fprintf(a.Stdout, "Baton %s purged; .baton/ deleted in full (kept at commit %s)\n", version, purgeCommit)
		return nil
	}

	for _, path := range plan.deleteFiles {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	for _, dir := range plan.deleteDirs {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	version := plan.version
	if version == "" {
		version = "unknown"
	}
	summary := fmt.Sprintf("Baton %s removed; records kept under .baton/", version)
	if len(open) > 0 {
		ids := make([]string, len(open))
		for i, task := range open {
			ids[i] = task.id
		}
		summary = fmt.Sprintf("Baton %s removed with %d open task: %s; records kept under .baton/", version, len(open), strings.Join(ids, ", "))
	}
	taskID, err := a.generateTaskID(records)
	if err != nil {
		return err
	}
	now := a.Now().Format("2006-01-02T15:04:05")
	if _, err := a.appendRecord(Record{Timestamp: now, TaskID: taskID, Event: eventRequest, Role: "Director", Summary: "Remove Baton"}); err != nil {
		return err
	}
	if _, err := a.appendRecord(Record{Timestamp: now, TaskID: taskID, Event: eventRunDone, Role: "Director", Summary: summary}); err != nil {
		return err
	}

	for _, path := range plan.binDeleteFiles {
		if err := os.Remove(path); err != nil {
			if a.GOOS == "windows" && filepath.Base(path) == binaryName(a.GOOS) {
				fmt.Fprintf(a.Stdout, "left %s in place; delete it yourself (Windows cannot delete a running program)\n", path)
				continue
			}
			return err
		}
	}
	for _, dir := range plan.binDeleteDirs {
		_ = os.Remove(dir)
	}

	if len(plan.unrecognizedPaths) > 0 {
		fmt.Fprintln(a.Stdout, "Kept (not recognised as Baton's own):")
		for _, path := range plan.unrecognizedPaths {
			fmt.Fprintf(a.Stdout, "  %s\n", path)
		}
	}
	fmt.Fprintln(a.Stdout, summary)
	return nil
}
