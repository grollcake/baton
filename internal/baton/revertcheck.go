package baton

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// confinedJoin joins relative onto root and refuses to return a path that
// resolves outside root. It is the one place either fixture command writes
// or deletes a file, so an escaping relative path is refused here rather than
// relied on elsewhere.
func confinedJoin(root, relative string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(relative))
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to write outside %s: %s", root, relative)
	}
	return target, nil
}

func writeUnderRoot(root, relative string, data []byte, mode os.FileMode) error {
	target, err := confinedJoin(root, relative)
	if err != nil {
		return err
	}
	return atomicWrite(target, data, mode)
}

func removeUnderRoot(root, relative string) error {
	target, err := confinedJoin(root, relative)
	if err != nil {
		return err
	}
	return os.RemoveAll(target)
}

func repositoryRelativePath(candidate string) (string, error) {
	if filepath.IsAbs(candidate) {
		return "", fmt.Errorf("refusing absolute path: %s", candidate)
	}
	cleaned := filepath.ToSlash(filepath.Clean(candidate))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("refusing path escaping the repository: %s", candidate)
	}
	return cleaned, nil
}

// runRevertCheck implements `baton revert-check`. The check command comes
// only from the parsed --check argument: this function never reads it, or
// the paths to revert, from a record, an artifact, or any project file.
func (a *App) runRevertCheck(args []string) error {
	parsed, err := parseArguments(args,
		map[string]bool{"--check": true, "--rev": true, "--name-pattern": true},
		map[string]bool{"--keep": true})
	if err != nil {
		return err
	}
	checkCommand, err := requireValue(parsed, "--check")
	if err != nil {
		return err
	}
	rev := parsed.values["--rev"]
	if rev == "" {
		rev = "HEAD"
	}
	namePattern := parsed.values["--name-pattern"]
	var nameRegexp *regexp.Regexp
	if namePattern != "" {
		nameRegexp, err = regexp.Compile(namePattern)
		if err != nil {
			return fmt.Errorf("invalid --name-pattern: %w", err)
		}
	}
	if len(parsed.pos) == 0 {
		return errors.New("revert-check requires at least one path to revert")
	}
	relativePaths := make([]string, 0, len(parsed.pos))
	for _, candidate := range parsed.pos {
		relative, err := repositoryRelativePath(candidate)
		if err != nil {
			return err
		}
		relativePaths = append(relativePaths, relative)
	}

	if _, err := a.runGit("rev-parse", "--git-dir"); err != nil {
		return fmt.Errorf("not a Git repository: %w", err)
	}
	resolvedRev, err := a.runGit("rev-parse", rev)
	if err != nil {
		return fmt.Errorf("cannot resolve %s: %w", rev, err)
	}

	listing, err := a.runGit("ls-files", "-c", "-o", "--exclude-standard")
	if err != nil {
		return fmt.Errorf("git ls-files: %w", err)
	}

	copyRoot, err := os.MkdirTemp("", "baton-revert-")
	if err != nil {
		return err
	}
	keep := parsed.flags["--keep"]
	removed := false
	cleanup := func() {
		if !keep && !removed {
			os.RemoveAll(copyRoot)
			removed = true
		}
	}
	defer cleanup()

	if listing != "" {
		for _, relative := range strings.Split(listing, "\n") {
			if relative == "" {
				continue
			}
			source := a.projectPath(relative)
			info, statErr := os.Lstat(source)
			if statErr != nil || !info.Mode().IsRegular() {
				continue
			}
			data, readErr := os.ReadFile(source)
			if readErr != nil {
				return fmt.Errorf("read %s: %w", relative, readErr)
			}
			if err := writeUnderRoot(copyRoot, relative, data, info.Mode().Perm()); err != nil {
				return err
			}
		}
	}

	baselineOutput, baselinePass, err := runCheckCommand(checkCommand, copyRoot)
	if err != nil {
		return err
	}

	fmt.Fprintf(a.Stdout, "copy=%s\n", copyRoot)
	fmt.Fprintf(a.Stdout, "rev=%s\n", resolvedRev)
	for _, relative := range relativePaths {
		fmt.Fprintf(a.Stdout, "reverted=%s\n", relative)
	}
	if baselinePass {
		fmt.Fprintln(a.Stdout, "baseline=pass")
	} else {
		fmt.Fprintln(a.Stdout, "baseline=fail")
		return errors.New("baseline check failed before any revert; refusing to compare against a red baseline")
	}

	for _, relative := range relativePaths {
		exists, err := a.pathExistsAtRev(resolvedRev, relative)
		if err != nil {
			return err
		}
		if !exists {
			if err := removeUnderRoot(copyRoot, relative); err != nil {
				return err
			}
			continue
		}
		content, err := a.gitShowRaw(resolvedRev, relative)
		if err != nil {
			return fmt.Errorf("git show %s:%s: %w", resolvedRev, relative, err)
		}
		mode := os.FileMode(0o644)
		if info, statErr := os.Lstat(a.projectPath(relative)); statErr == nil {
			mode = info.Mode().Perm()
		}
		if err := writeUnderRoot(copyRoot, relative, content, mode); err != nil {
			return err
		}
	}

	afterOutput, afterPass, err := runCheckCommand(checkCommand, copyRoot)
	if err != nil {
		return err
	}
	if afterPass {
		fmt.Fprintln(a.Stdout, "after_revert=pass")
	} else {
		fmt.Fprintln(a.Stdout, "after_revert=fail")
	}

	newLines := newOutputLines(baselineOutput, afterOutput)
	const maxLines = 200
	truncated := 0
	if len(newLines) > maxLines {
		truncated = len(newLines) - maxLines
		newLines = newLines[:maxLines]
	}

	switch {
	case nameRegexp == nil:
		fmt.Fprintln(a.Stdout, "pins=unavailable (pass --name-pattern)")
	default:
		seen := map[string]bool{}
		var names []string
		for _, line := range newLines {
			match := nameRegexp.FindStringSubmatch(line)
			if len(match) < 2 {
				continue
			}
			name := match[1]
			if seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
		switch {
		case len(names) > 0:
			for _, name := range names {
				fmt.Fprintf(a.Stdout, "pins=%s\n", name)
			}
		case afterPass:
			// The revert changed nothing the check noticed: a true
			// negative, not a silence to be misread as one.
			fmt.Fprintln(a.Stdout, "pins=none")
		default:
			// after_revert failed, but no line matched --name-pattern --
			// e.g. the reverted file no longer compiles, so the check
			// never got far enough to name a test. Say that plainly
			// instead of printing nothing.
			fmt.Fprintln(a.Stdout, "pins=none (after_revert failed but no line matched --name-pattern; see new_output)")
		}
	}
	for _, line := range newLines {
		fmt.Fprintf(a.Stdout, "new_output=%s\n", line)
	}
	if truncated > 0 {
		fmt.Fprintf(a.Stdout, "new_output_truncated=%d\n", truncated)
	}

	return nil
}

func (a *App) pathExistsAtRev(rev, relative string) (bool, error) {
	command := exec.Command("git", "cat-file", "-e", rev+":"+relative)
	command.Dir = a.ProjectDir
	if err := command.Run(); err != nil {
		return false, nil
	}
	return true, nil
}

// gitShowRaw returns the exact committed bytes for rev:relative, unlike
// runGit which trims output meant to be read as a single value.
func (a *App) gitShowRaw(rev, relative string) ([]byte, error) {
	command := exec.Command("git", "show", rev+":"+relative)
	command.Dir = a.ProjectDir
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, errors.New(message)
	}
	return stdout.Bytes(), nil
}

// runCheckCommand runs the caller-supplied check string with its working
// directory set to dir, with no argument naming any other tree.
func runCheckCommand(checkCommand, dir string) (string, bool, error) {
	// Baton ships Windows binaries and PROTOCOL.md promises no particular
	// shell, so this does not assume a POSIX shell is on PATH: it runs the
	// platform's own always-present command interpreter, cmd.exe on
	// Windows and sh elsewhere, rather than requiring sh universally.
	var command *exec.Cmd
	if runtime.GOOS == "windows" {
		command = exec.Command("cmd", "/C", checkCommand)
	} else {
		command = exec.Command("sh", "-c", checkCommand)
	}
	command.Dir = dir
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if err == nil {
		return output.String(), true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return output.String(), false, nil
	}
	return output.String(), false, fmt.Errorf("run --check: %w", err)
}

func newOutputLines(before, after string) []string {
	beforeLines := map[string]int{}
	for _, line := range strings.Split(before, "\n") {
		beforeLines[line]++
	}
	var added []string
	for _, line := range strings.Split(after, "\n") {
		if beforeLines[line] > 0 {
			beforeLines[line]--
			continue
		}
		added = append(added, line)
	}
	return added
}
