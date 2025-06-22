package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/aokhrimenko/gpsd-simulator/internal/cmd"
)

var Version = "v1.1.0-dev"

func main() {
	runCmd := cmd.Run(Version)
	root := &cobra.Command{
		Use:   "gpsd-simulator",
		Short: "GPS simulator tool",
		RunE:  runCmd.RunE,
	}
	root.AddCommand(runCmd, cmd.Import(Version))
	runCmd.Flags().VisitAll(func(f *pflag.Flag) {
		root.Flags().AddFlag(f)
	})
	err := root.Execute()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
