package baton

import (
	"errors"
	"fmt"

	docs "github.com/grollcake/baton"
)

var guideDocuments = map[string]string{
	"protocol": "PROTOCOL.md",
	"director": "DIRECTOR.md",
	"planner":  "PLANNER.md",
	"executor": "EXECUTOR.md",
	"update":   "HOW-TO-UPDATE.md",
}

func (a *App) runGuide(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: baton guide <protocol|director|planner|executor|update>")
	}
	name, found := guideDocuments[args[0]]
	if !found {
		return fmt.Errorf("unknown guide: %s (protocol|director|planner|executor|update)", args[0])
	}
	content, err := docs.Managed(name)
	if err != nil {
		return err
	}
	_, err = a.Stdout.Write(content)
	return err
}
