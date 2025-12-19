// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and Charm KV client

package main

import (
	"database/sql"
	"fmt"

	"github.com/harper/toki/internal/charm"
	"github.com/harper/toki/internal/db"
	"github.com/spf13/cobra"
)

// Deprecated: dbConn is kept temporarily for sync/mcp commands only.
// All other commands use charm.GetClient(). Will be removed when sync layer is deprecated.
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

		// Initialize SQLite DB for sync/mcp commands only
		var err error
		dbConn, err = db.InitDB(db.GetDefaultDBPath())
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close database
		if dbConn != nil {
			_ = dbConn.Close()
		}

		// Close Charm client
		return charm.CloseClient()
	},
}

func init() {
	// No persistent flags needed - Charm manages its own configuration
}
