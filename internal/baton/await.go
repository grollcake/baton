package baton

import (
	"errors"
	"fmt"
	"time"
)

const (
	awaitDefaultTimeout = 30 * time.Minute
	awaitInterval       = 2 * time.Second
)

// runAwait blocks until a delegated artifact passes CheckArtifact. Director runs
// it in the background after delegating so the host wakes Director when the
// artifact is complete, instead of relying on a delegate's completion notice.
// Artifacts are never overwritten, so the awaited path cannot already hold a
// complete artifact from an earlier round.
func (a *App) runAwait(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--task-id": true, "--timeout": true}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 2 {
		return errors.New("usage: baton await <PLANNED|EXECUTED|REVIEW> <path> [--task-id <id>] [--timeout <duration>]")
	}
	event, path := parsed.pos[0], parsed.pos[1]
	if event == eventClose {
		return errors.New("await: CLOSE is written by Director and cannot be awaited")
	}
	if err := validateArtifactPath(event, path); err != nil {
		return err
	}
	timeout := awaitDefaultTimeout
	if value := parsed.values["--timeout"]; value != "" {
		timeout, err = time.ParseDuration(value)
		if err != nil || timeout <= 0 {
			return fmt.Errorf("await: invalid --timeout: %s", value)
		}
	}

	deadline := a.Now().Add(timeout)
	for {
		checkErr := a.CheckArtifact(event, path, parsed.values["--task-id"])
		if checkErr == nil {
			fmt.Fprintf(a.Stdout, "await: %s artifact complete: %s\n", event, path)
			return nil
		}
		if !a.Now().Before(deadline) {
			return fmt.Errorf("await: timed out after %s: %w", timeout, checkErr)
		}
		a.Sleep(awaitInterval)
	}
}
