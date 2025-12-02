// ABOUTME: Version command to display build information
// ABOUTME: Shows version, commit hash, and build date

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  "Display the version, commit hash, and build date of toki",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("toki version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
