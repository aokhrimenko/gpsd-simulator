package main

import (
	"os"

	"github.com/aokhrimenko/gpsd-simulator/internal/cmd"
)

var Version = "v1.0.0-dev"

func main() {
	command := cmd.Root(Version)
	err := command.Execute()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
