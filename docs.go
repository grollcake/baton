// Package docs embeds the Baton documents the binary manages, so a project's
// installed copies can be compared against the version that shipped with this
// binary. It lives at the module root because go:embed cannot reach a parent
// directory from internal/baton.
package docs

import (
	"embed"
	"errors"
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

//go:embed bootstrap/AGENTS.md
var bootstrapAgents embed.FS

// ActiveBlock returns the <baton-rules> block this binary ships in
// bootstrap/AGENTS.md, the active-state text pause/resume and lint compare
// installed blocks against.
func ActiveBlock() ([]byte, error) {
	content, err := fs.ReadFile(bootstrapAgents, "bootstrap/AGENTS.md")
	if err != nil {
		return nil, err
	}
	block := batonBlockPattern.Find(content)
	if len(block) == 0 {
		return nil, errors.New("bootstrap/AGENTS.md carries no <baton-rules> block")
	}
	return block, nil
}
