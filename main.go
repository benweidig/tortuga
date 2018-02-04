package main

import (
	"log"
	"os/exec"

	"github.com/benweidig/tortuga/cmd"
)

func main() {
	// Without git nothing works so check and exit and necessary
	_, err := exec.LookPath("git")
	if err != nil {
		log.Fatalln("Command 'git' not found!")
	}

	cmd.RootCmd.Execute()
}
