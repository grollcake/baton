// Package docs embeds the Baton documents the binary manages, so a project's
// installed copies can be compared against the version that shipped with this
// binary. It lives at the module root because go:embed cannot reach a parent
// directory from internal/baton.
package docs

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
)

// batonBlockPattern mirrors internal/baton's pattern of the same name; it
// cannot be imported across the package boundary (go:embed needs this file at
// the module root), so both copies must be kept identical by hand.
var batonBlockPattern = regexp.MustCompile(`(?s)<baton-rules>.*?</baton-rules>`)

//go:embed bootstrap/.baton/PROTOCOL.md bootstrap/.baton/DIRECTOR.md bootstrap/.baton/PLANNER.md bootstrap/.baton/EXECUTOR.md bootstrap/.baton/HOW-TO-UPDATE.md
var managed embed.FS

// Managed returns the shipped content of a managed document by file name, such
// as "PROTOCOL.md".
func Managed(name string) ([]byte, error) {
	return fs.ReadFile(managed, "bootstrap/.baton/"+name)
}

//go:embed bootstrap/AGENTS.md bootstrap/CLAUDE.md
var bootstrapInstructions embed.FS

// ActiveBlock returns the <baton-rules> block this binary ships in
// bootstrap/AGENTS.md, the active-state text pause/resume and lint compare
// installed blocks against.
func ActiveBlock() ([]byte, error) {
	content, err := fs.ReadFile(bootstrapInstructions, "bootstrap/AGENTS.md")
	if err != nil {
		return nil, err
	}
	block := batonBlockPattern.Find(content)
	if len(block) == 0 {
		return nil, errors.New("bootstrap/AGENTS.md carries no <baton-rules> block")
	}
	return block, nil
}

// ShippedInstructionFile returns the shipped bytes of an instruction file
// this binary bootstraps whole -- "AGENTS.md" or "CLAUDE.md" -- so remove can
// recognise a file it wrote in full (Decision 3) by whole-file comparison.
func ShippedInstructionFile(name string) ([]byte, error) {
	if name != "AGENTS.md" && name != "CLAUDE.md" {
		return nil, fmt.Errorf("not a shipped instruction file: %s", name)
	}
	return fs.ReadFile(bootstrapInstructions, "bootstrap/"+name)
}
