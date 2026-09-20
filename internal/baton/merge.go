package baton

import (
	"bytes"
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

// computeRemovedContent computes target's content with its <baton-rules>
// block cut out, following Decision 3's seam rules:
//
//   - block last in the file (what MergeAgentBlock produces when it appends,
//     and so the shape every bootstrapped or updated project has): cut the
//     block and everything after it, then trim the remainder to exactly one
//     trailing newline;
//   - block in the middle (only reachable by hand placement): cut the block
//     and collapse the run of newlines spanning the seam to exactly one
//     blank line;
//   - nothing but the block, i.e. the remainder on both sides is empty or
//     whitespace only: report wholeFileEmpty so the caller deletes the file
//     instead of writing an empty one.
//
// It returns an error if target carries no <baton-rules> block; the caller is
// expected to have already classified the block as removable.
func computeRemovedContent(target string) (content []byte, mode os.FileMode, wholeFileEmpty bool, err error) {
	original, err := os.ReadFile(target)
	if err != nil {
		return nil, 0, false, err
	}
	fileMode := os.FileMode(0o644)
	if info, statErr := os.Stat(target); statErr == nil {
		fileMode = info.Mode().Perm()
	}
	location := batonBlockPattern.FindIndex(original)
	if location == nil {
		return nil, 0, false, fmt.Errorf("%s carries no <baton-rules> block", target)
	}
	before := original[:location[0]]
	after := original[location[1]:]

	if len(bytes.TrimSpace(before)) == 0 && len(bytes.TrimSpace(after)) == 0 {
		return nil, fileMode, true, nil
	}

	if len(bytes.TrimSpace(after)) == 0 {
		trimmed := bytes.TrimRight(before, "\n")
		result := append(append([]byte{}, trimmed...), '\n')
		return result, fileMode, false, nil
	}

	trimmedBefore := bytes.TrimRight(before, "\n")
	trimmedAfter := bytes.TrimLeft(after, "\n")
	var result []byte
	result = append(result, trimmedBefore...)
	if len(trimmedBefore) > 0 {
		result = append(result, '\n', '\n')
	}
	result = append(result, trimmedAfter...)
	return result, fileMode, false, nil
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
