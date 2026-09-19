package baton

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func MergeAgentBlock(target, source string) error {
	sourceContent, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("missing source: %s", source)
	}
	sourceBlock := batonBlockPattern.Find(sourceContent)
	if len(sourceBlock) == 0 {
		return errors.New("source missing <baton-rules> block")
	}

	targetContent, err := os.ReadFile(target)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(target); statErr == nil {
		mode = info.Mode().Perm()
	}
	location := batonBlockPattern.FindIndex(targetContent)
	var merged []byte
	if location != nil {
		merged = append(merged, targetContent[:location[0]]...)
		merged = append(merged, sourceBlock...)
		merged = append(merged, targetContent[location[1]:]...)
	} else {
		merged = append(merged, targetContent...)
		if len(merged) > 0 {
			if merged[len(merged)-1] != '\n' {
				merged = append(merged, '\n')
			}
			merged = append(merged, '\n')
		}
		merged = append(merged, sourceBlock...)
		merged = append(merged, '\n')
	}
	return atomicWrite(target, merged, mode)
}

func (a *App) runMergeAgentBlock(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: baton merge-agent-block <target-file> <source-file>")
	}
	target, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	source, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	if err := MergeAgentBlock(target, source); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "merged Baton block: %s\n", args[0])
	return nil
}
