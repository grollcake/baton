// Package docs embeds the Baton documents the binary manages, so a project's
// installed copies can be compared against the version that shipped with this
// binary. It lives at the module root because go:embed cannot reach a parent
// directory from internal/baton.
package docs

import (
	"embed"
	"io/fs"
)

//go:embed bootstrap/.baton/PROTOCOL.md bootstrap/.baton/DIRECTOR.md bootstrap/.baton/PLANNER.md bootstrap/.baton/EXECUTOR.md bootstrap/.baton/HOW-TO-UPDATE.md
var managed embed.FS

// Managed returns the shipped content of a managed document by file name, such
// as "PROTOCOL.md".
func Managed(name string) ([]byte, error) {
	return fs.ReadFile(managed, "bootstrap/.baton/"+name)
}
