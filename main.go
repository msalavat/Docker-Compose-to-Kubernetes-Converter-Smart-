// Package main is the entry point for the kompoze CLI tool.
package main

import (
	"os"

	"github.com/compositor/kompoze/cmd"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
