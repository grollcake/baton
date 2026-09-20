package baton

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var managedBatonFiles = []string{
	"HOW-TO-UPDATE.md",
	"PROTOCOL.md",
	"DIRECTOR.md",
	"PLANNER.md",
	"EXECUTOR.md",
}

func (a *App) upstreamBinary(upstream string) string {
	return filepath.Join(upstream, "bootstrap", ".baton", "bin", a.GOOS+"-"+a.GOARCH, binaryName(a.GOOS))
}

func (a *App) upstreamBinaryRelative() string {
	return filepath.Join(a.GOOS+"-"+a.GOARCH, binaryName(a.GOOS))
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(leftAbs), filepath.Clean(rightAbs))
}

func (a *App) preflightUpdate(upstream string) error {
	upstreamBaton := filepath.Join(upstream, "bootstrap", ".baton")
	checksumFile := filepath.Join(upstreamBaton, "bin", "SHA256SUMS")
	required := []string{filepath.Join(upstream, "VERSION"), filepath.Join(upstream, "bootstrap", "AGENTS.md"), a.upstreamBinary(upstream), checksumFile}
	for _, name := range managedBatonFiles {
		required = append(required, filepath.Join(upstreamBaton, name))
	}
	for _, path := range required {
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return fmt.Errorf("missing upstream file: %s", path)
		}
	}
	if err := verifyChecksum(a.upstreamBinary(upstream), checksumFile, a.upstreamBinaryRelative()); err != nil {
		return fmt.Errorf("invalid upstream binary: %w", err)
	}
	if runtime.GOOS == "windows" {
		if executable, err := os.Executable(); err == nil && samePath(executable, a.installedBinaryPath()) {
			return errors.New("Windows update must be run with the new upstream baton.exe from a temporary path")
		}
	}
	return nil
}

func (a *App) runUpdate(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--upstream": true}, map[string]bool{"--apply": true})
	if err != nil {
		return err
	}
	if len(parsed.pos) != 0 {
		return errors.New("update accepts flags only")
	}
	upstream, err := requireValue(parsed, "--upstream")
	if err != nil {
		return err
	}
	upstream, err = filepath.Abs(upstream)
	if err != nil {
		return err
	}
	if info, statErr := os.Stat(filepath.Join(upstream, "bootstrap", ".baton")); statErr != nil || !info.IsDir() {
		return fmt.Errorf("invalid upstream: %s", upstream)
	}
	currentVersionBytes, err := os.ReadFile(a.batonPath("VERSION"))
	if err != nil {
		return fmt.Errorf("missing %s", a.batonPath("VERSION"))
	}
	nextVersionBytes, err := os.ReadFile(filepath.Join(upstream, "VERSION"))
	if err != nil {
		return errors.New("missing upstream VERSION")
	}
	currentVersion := strings.TrimSpace(string(currentVersionBytes))
	nextVersion := strings.TrimSpace(string(nextVersionBytes))
	fmt.Fprintf(a.Stdout, "Baton update: %s -> %s\n", currentVersion, nextVersion)

	pausedMessage := a.pausedUpdateMessage()
	removedMessage := a.removedUpdateMessage()
	if !parsed.flags["--apply"] {
		if pausedMessage != "" {
			fmt.Fprintln(a.Stdout, pausedMessage)
		}
		if removedMessage != "" {
			fmt.Fprintln(a.Stdout, removedMessage)
		}
		fmt.Fprint(a.Stdout, `Dry run. Re-run with --apply to update:
- WARNING: --apply replaces the managed files below; review local customizations first.
- AGENTS.md Baton block
- CLAUDE.md Baton block when present
- .baton/HOW-TO-UPDATE.md
- .baton/PROTOCOL.md
- .baton/DIRECTOR.md
- .baton/PLANNER.md
- .baton/EXECUTOR.md
- .baton/bin/baton
- .baton/templates/
- .baton/VERSION

Preserved:
- .baton/GUIDANCE.md
- .baton/LESSON-LEARNED.md
- .baton/lesson-learned/
- .baton/BATON-LOG.txt (or the legacy .baton/baton.log, migrated in place)
- .baton/runs/
`)
		return nil
	}
	if pausedMessage != "" {
		return errors.New(pausedMessage)
	}
	if removedMessage != "" {
		return errors.New(removedMessage)
	}
	if err := a.preflightUpdate(upstream); err != nil {
		return err
	}

	upstreamBaton := filepath.Join(upstream, "bootstrap", ".baton")
	agentsPath := filepath.Join(a.ProjectDir, "AGENTS.md")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		if err := copyFile(filepath.Join(upstream, "bootstrap", "AGENTS.md"), agentsPath, 0o644); err != nil {
			return err
		}
	} else if err := MergeAgentBlock(agentsPath, filepath.Join(upstream, "bootstrap", "AGENTS.md")); err != nil {
		return err
	}
	claudePath := filepath.Join(a.ProjectDir, "CLAUDE.md")
	if _, err := os.Stat(claudePath); err == nil {
		if err := MergeAgentBlock(claudePath, filepath.Join(upstream, "bootstrap", "CLAUDE.md")); err != nil {
			return err
		}
	}
	for _, name := range managedBatonFiles {
		if err := copyFile(filepath.Join(upstreamBaton, name), a.batonPath(name), 0o644); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(a.batonPath("templates"), 0o755); err != nil {
		return err
	}
	templates, err := filepath.Glob(filepath.Join(upstreamBaton, "templates", "*.md"))
	if err != nil {
		return err
	}
	for _, source := range templates {
		if err := copyFile(source, a.batonPath("templates", filepath.Base(source)), 0o644); err != nil {
			return err
		}
	}
	if err := copyFile(a.upstreamBinary(upstream), a.installedBinaryPath(), 0o755); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(upstreamBaton, "bin", "SHA256SUMS"), a.batonPath("bin", "SHA256SUMS"), 0o644); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(upstream, "VERSION"), a.batonPath("VERSION"), 0o644); err != nil {
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
	if _, err := a.appendRecord(Record{Timestamp: now, TaskID: taskID, Event: eventRequest, Role: "Director", Summary: fmt.Sprintf("Update Baton %s -> %s", currentVersion, nextVersion)}); err != nil {
		return err
	}
	if _, err := a.appendRecord(Record{Timestamp: now, TaskID: taskID, Event: eventRunDone, Role: "Director", Summary: fmt.Sprintf("Baton updated to %s", nextVersion)}); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "Baton updated: %s -> %s\n", currentVersion, nextVersion)
	return nil
}

func (a *App) runVersion(args []string) error {
	if len(args) != 0 {
		return errors.New("version accepts no arguments")
	}
	version, err := os.ReadFile(a.batonPath("VERSION"))
	if err != nil {
		return err
	}
	fmt.Fprintln(a.Stdout, strings.TrimSpace(string(version)))
	return nil
}
