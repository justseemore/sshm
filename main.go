package main

import (
	"os"

	"github.com/justseemore/sshm/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
