package memento

import (
	"errors"
	"fmt"
)

func (a *App) runPrompt(args []string) error {
	parsed, err := parseArguments(args, map[string]bool{"--task-id": true, "--key": true, "--run-number": true}, nil)
	if err != nil {
		return err
	}
	if len(parsed.pos) != 1 {
		return errors.New("prompt requires one role (plan|review|exec)")
	}
	taskID, err := requireValue(parsed, "--task-id")
	if err != nil {
		return err
	}
	key, err := requireValue(parsed, "--key")
	if err != nil {
		return err
	}
	runNumber := valueOr(parsed.values["--run-number"], "01")
	var prompt string
	switch parsed.pos[0] {
	case "plan":
		prompt = fmt.Sprintf("Planner for round `%s`. Read `.memento/PROTOCOL.md`, `.memento/PLANNER.md`, `.memento/GUIDANCE.md`, and the `.memento/LESSON-LEARNED.md` index (open matching records only).\n\nArtifact: `.memento/runs/%s-PLAN.md` (template: `.memento/templates/plan.md`). Include §Director Brief at top with goal, scope, success criteria, risks, required checks, and a minimal Executor Prompt block.\n\nSet exactly `Status: complete` as the last write before reporting completion. Return ambiguity to Director rather than guessing. Report ≤200 words with artifact path and suggested summary.\n", taskID, key)
	case "review":
		prompt = fmt.Sprintf("Planner REVIEW phase for round `%s`. Read `.memento/PROTOCOL.md`, `.memento/PLANNER.md`, `.memento/GUIDANCE.md`, and matching lesson records.\n\nInputs:\n- PLAN: `.memento/runs/%s-PLAN.md`\n- RUN-%s: `.memento/runs/%s-RUN-%s.md`\n\nArtifact: `.memento/runs/%s-REVIEW-%s.md` (template: `.memento/templates/review.md`).\n\nREVIEW gathers evidence; you do NOT approve completion. Add 3-5 user manual checks to the artifact. Set exactly `Status: complete` as the last write before reporting completion. Report ≤200 words with blockers, nits, checks, risks, artifact path, and suggested summary.\n", taskID, key, runNumber, key, runNumber, key, runNumber)
	case "exec":
		prompt = fmt.Sprintf("Executor for assigned round. Read `.memento/PROTOCOL.md`, `.memento/EXECUTOR.md`, `.memento/GUIDANCE.md`, and matching lesson records.\n\nPLAN: `.memento/runs/%s-PLAN.md` — implement §Plan steps verbatim; follow §Director Brief Executor Prompt constraints.\n\nArtifact: `.memento/runs/%s-RUN-%s.md` (template: `.memento/templates/run.md`).\n\nYou may save the RUN as a checkpoint before long validation. TODO fields or unresolved `<...>` placeholders mean incomplete; set exactly `Status: complete` before reporting completion. Return ambiguity to Director rather than guessing. Report ≤200 words with validation results, blockers or risks, artifact path, and suggested summary.\n", key, key, runNumber)
	default:
		return fmt.Errorf("unknown role: %s (plan|review|exec)", parsed.pos[0])
	}
	fmt.Fprint(a.Stdout, prompt)
	return nil
}
