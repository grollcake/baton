package main

import (
	"fmt"
	"os"

	"github.com/grollcake/baton/internal/baton"
)

func main() {
	app, err := baton.Discover()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
