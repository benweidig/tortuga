package main

import (
	"fmt"
	"os"

	"github.com/benweidig/tortuga/cmd"
	"github.com/benweidig/tortuga/git"
)

func main() {
	err := git.IsAvailable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "git not found.")
		os.Exit(1)
	}

	cmd.RootCmd.Execute()
}
