// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and database connection

package main

import (
	"database/sql"
	"fmt"

	"github.com/harper/toki/internal/db"
	"github.com/spf13/cobra"
)

var (
	dbPath string
	dbConn *sql.DB
)

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
		// Initialize database connection
		var err error
		dbConn, err = db.InitDB(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close database connection
		if dbConn != nil {
			return dbConn.Close()
		}
		return nil
	},
}

func init() {
	defaultPath := db.GetDefaultDBPath()
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultPath, "database file path")
}
