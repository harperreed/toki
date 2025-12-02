// ABOUTME: CLI entry point for toki
// ABOUTME: Initializes and executes root command

package main

import (
	"fmt"
	"os"
)

var (
	// These variables are set via ldflags during build by GoReleaser.
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
