package baton

import (
	"errors"
	"fmt"

	docs "github.com/grollcake/baton"
)

var guideDocuments = map[string]string{
	"protocol":  "PROTOCOL.md",
	"director":  "DIRECTOR.md",
	"planner":   "PLANNER.md",
	"executor":  "EXECUTOR.md",
	"update":    "HOW-TO-UPDATE.md",
	"remove":    "HOW-TO-UPDATE.md",
	"uninstall": "HOW-TO-UPDATE.md",
	"pause":     "HOW-TO-UPDATE.md",
	"lifecycle": "HOW-TO-UPDATE.md",
}

func (a *App) runGuide(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: baton guide <protocol|director|planner|executor|update|remove|uninstall|pause|lifecycle>")
	}
	name, found := guideDocuments[args[0]]
	if !found {
		return fmt.Errorf("unknown guide: %s (protocol|director|planner|executor|update|remove|uninstall|pause|lifecycle)", args[0])
	}
	content, err := docs.Managed(name)
	if err != nil {
		return err
	}
	_, err = a.Stdout.Write(content)
	return err
}
