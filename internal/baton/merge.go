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
	return replaceBlock(target, sourceBlock)
}

// replaceBlock splices block over target's existing <baton-rules> block, or
// appends it when target carries none, preserving target's file mode (or
// 0o644 for a target that does not yet exist). It is the write half
// MergeAgentBlock has always performed; pause and resume (Decision 2) reuse
// the same splice logic, through computeReplacedContent, to write the paused
// and active block text without going through a second source file.
func replaceBlock(target string, block []byte) error {
	merged, mode, err := computeReplacedContent(target, block)
	if err != nil {
		return err
	}
	return atomicWrite(target, merged, mode)
}

// computeReplacedContent is replaceBlock's splice-or-append computation
// without the write, so a caller that must write several files can compute
// every new file's bytes first (Decision 7) before writing any of them.
func computeReplacedContent(target string, block []byte) ([]byte, os.FileMode, error) {
	targetContent, err := os.ReadFile(target)
	if err != nil && !os.IsNotExist(err) {
		return nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(target); statErr == nil {
		mode = info.Mode().Perm()
	}
	location := batonBlockPattern.FindIndex(targetContent)
	var merged []byte
	if location != nil {
		merged = append(merged, targetContent[:location[0]]...)
		merged = append(merged, block...)
		merged = append(merged, targetContent[location[1]:]...)
	} else {
		merged = append(merged, targetContent...)
		if len(merged) > 0 {
			if merged[len(merged)-1] != '\n' {
				merged = append(merged, '\n')
			}
			merged = append(merged, '\n')
		}
		merged = append(merged, block...)
		merged = append(merged, '\n')
	}
	return merged, mode, nil
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
	if err := a.refusePausedForUpdate(); err != nil {
		return err
	}
	if err := MergeAgentBlock(target, source); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "merged Baton block: %s\n", args[0])
	return nil
}
