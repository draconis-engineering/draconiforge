package main

import (
	"os"

	"github.com/inconshreveable/mousetrap"
	"github.com/spf13/cobra"
	cli "github.com/draconis-engineering/draconiforge/internal/cli"
)

func main() {
	// Cobra mousetrap for Windows double-click safety
	if mousetrap.StartedByExplorer() {
		cobra.MousetrapHelpText = ""
	}

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
