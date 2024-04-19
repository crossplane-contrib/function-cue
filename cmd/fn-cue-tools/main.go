package main

import (
	"os"

	"github.com/spf13/cobra"
)

const exe = "fn-cue-tools"

func main() {
	root := &cobra.Command{Use: exe}
	root.AddCommand(
		openapiCommand(),
		packageScriptCommand(),
		extractSchemaCommand(),
		cueTestCommand(),
		versionCommand(),
	)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
