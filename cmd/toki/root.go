// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and Charm KV client

package main

import (
	"database/sql"
	"fmt"

	"github.com/harper/toki/internal/charm"
	"github.com/spf13/cobra"
)

// Deprecated: dbConn is kept temporarily for compatibility during migration.
// Commands should use charm.GetClient() instead. Will be removed in Phase 3.
var dbConn *sql.DB

var rootCmd = &cobra.Command{
	Use:   "toki",
	Short: "A super simple git-aware todo manager",
	Long: `
████████╗ ██████╗ ██╗  ██╗██╗
╚══██╔══╝██╔═══██╗██║ ██╔╝██║
   ██║   ██║   ██║█████╔╝ ██║
   ██║   ██║   ██║██╔═██╗ ██║
   ██║   ╚██████╔╝██║  ██╗██║
   ╚═╝    ╚═════╝ ╚═╝  ╚═╝╚═╝

         ✨ Git-aware task management ⚡

Toki is a CLI todo manager that organizes tasks by project,
supports rich metadata (priority, tags, notes, due dates),
and automatically detects project context from git repositories.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Charm KV client
		if err := charm.InitClient(); err != nil {
			return fmt.Errorf("failed to initialize charm client: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close Charm client
		return charm.CloseClient()
	},
}

func init() {
	// No persistent flags needed - Charm manages its own configuration
}
