package baton

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
		prompt = fmt.Sprintf("You are Planner for task `%s`, PLAN phase. Director-owned commands (new-round, append, gate, feedback, update) are not yours to run; report to Director instead.\n\nRead `.baton/PROTOCOL.md`, `.baton/PLANNER.md`, `.baton/GUIDANCE.md`, and the `.baton/LESSON-LEARNED.md` index (open matching records only).\n\nArtifact: `.baton/runs/%s-PLAN.md` (template: `.baton/templates/plan.md`). Include §Director Brief at top with goal, scope, success criteria, risks, required checks, and a minimal Executor Prompt block.\n\nSet exactly `Status: complete` as the last write before reporting completion. Return ambiguity to Director rather than guessing. Report ≤200 words with artifact path and suggested summary.\n", taskID, key)
	case "review":
		prompt = fmt.Sprintf("You are Planner for task `%s`, REVIEW phase of round `%s`. Director-owned commands (new-round, append, gate, feedback, update) are not yours to run; report to Director instead.\n\nRead `.baton/PROTOCOL.md`, `.baton/PLANNER.md`, `.baton/GUIDANCE.md`, and matching lesson records.\n\nInputs:\n- PLAN: `.baton/runs/%s-PLAN.md`\n- RUN-%s: `.baton/runs/%s-RUN-%s.md`\n\nArtifact: `.baton/runs/%s-REVIEW-%s.md` (template: `.baton/templates/review.md`).\n\nREVIEW gathers evidence; you do NOT approve completion. Add 3-5 user manual checks to the artifact. Set exactly `Status: complete` as the last write before reporting completion. Report ≤200 words with blockers, nits, checks, risks, artifact path, and suggested summary.\n", taskID, runNumber, key, runNumber, key, runNumber, key, runNumber)
	case "exec":
		prompt = fmt.Sprintf("You are Executor for task `%s`, round `%s`. Director-owned commands (new-round, append, gate, feedback, update) are not yours to run; report to Director instead.\n\nRead `.baton/PROTOCOL.md`, `.baton/EXECUTOR.md`, `.baton/GUIDANCE.md`, and matching lesson records.\n\nPLAN: `.baton/runs/%s-PLAN.md` — implement §Plan steps verbatim; follow §Director Brief Executor Prompt constraints.\n\nArtifact: `.baton/runs/%s-RUN-%s.md` (template: `.baton/templates/run.md`).\n\nYou may save the RUN as a checkpoint before long validation. TODO fields or unresolved `<...>` placeholders mean incomplete; set exactly `Status: complete` before reporting completion. Return ambiguity to Director rather than guessing. Report ≤200 words with validation results, blockers or risks, artifact path, and suggested summary.\n", taskID, runNumber, key, key, runNumber)
	default:
		return fmt.Errorf("unknown role: %s (plan|review|exec)", parsed.pos[0])
	}
	fmt.Fprint(a.Stdout, prompt)
	return nil
}
