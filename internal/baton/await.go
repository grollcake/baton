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

// runAwait blocks until one of the delegated artifacts passes CheckArtifact.
// Director runs it in the background after delegating so the host wakes
// Director when an artifact is complete, instead of relying on a delegate's
// completion notice. Several EVENT/path pairs may be watched at once, and the
// first one to complete ends the wait, so parallel delegations need one wait
// rather than one per artifact. Artifacts are never overwritten, so an awaited
// path cannot already hold a complete artifact from an earlier round.
func (a *App) runAwait(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--task-id": true, "--timeout": true}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) < 2 || len(parsed.pos)%2 != 0 {
		return errors.New("usage: baton await <PLANNED|EXECUTED|REVIEW> <path> [<EVENT> <path>...] [--task-id <id>] [--timeout <duration>]")
	}
	type watch struct{ event, path string }
	var watches []watch
	for index := 0; index < len(parsed.pos); index += 2 {
		event, path := parsed.pos[index], parsed.pos[index+1]
		if event == eventClose {
			return errors.New("await: CLOSE is written by Director and cannot be awaited")
		}
		if err := validateArtifactPath(event, path); err != nil {
			return err
		}
		watches = append(watches, watch{event, path})
	}
	taskID := parsed.values["--task-id"]
	if taskID != "" && len(watches) > 1 {
		return errors.New("await: --task-id applies to a single artifact; omit it when awaiting several")
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
		var lastErr error
		for _, item := range watches {
			checkErr := a.CheckArtifact(item.event, item.path, taskID)
			if checkErr == nil {
				fmt.Fprintf(a.Stdout, "await: %s artifact complete: %s\n", item.event, item.path)
				return nil
			}
			lastErr = checkErr
		}
		if !a.Now().Before(deadline) {
			return fmt.Errorf("await: timed out after %s: %w", timeout, lastErr)
		}
		a.Sleep(awaitInterval)
	}
}
