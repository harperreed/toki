// ABOUTME: Sync subcommand for database maintenance
// ABOUTME: Provides repair, vacuum, and integrity check commands for local SQLite

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/storage"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Database maintenance commands",
	Long: `Database maintenance for toki.

Note: Cloud sync has been removed. Toki now uses local SQLite storage.
Use 'toki export' to backup your data.

Commands:
  status  - Show database status
  repair  - Repair database (vacuum and integrity check)
  reset   - Delete and recreate the database (DESTRUCTIVE)`,
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show database status",
	Long:  `Display current database path and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := storage.DefaultDBPath()

		fmt.Printf("Database: %s\n", dbPath)

		// Check if file exists
		info, err := os.Stat(dbPath)
		if os.IsNotExist(err) {
			color.Yellow("\nStatus: No database file found")
			fmt.Println("A new database will be created on first use.")
			return nil
		}

		fmt.Printf("Size:     %d bytes\n", info.Size())
		fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

		// Run integrity check
		store := GetStorage()
		if store != nil {
			ok, err := store.IntegrityCheck()
			if err != nil {
				color.Yellow("\nStatus: Integrity check failed: %v", err)
			} else if ok {
				color.Green("\nStatus: Database is healthy")
			} else {
				color.Red("\nStatus: Database corruption detected")
				fmt.Println("Run 'toki sync repair' to attempt recovery.")
			}
		}

		return nil
	},
}

var syncRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair and optimize the database",
	Long: `Repair and optimize the local database.

This command will:
- Run integrity checks
- Vacuum the database to reclaim space
- Optimize indexes

Use this if you encounter database errors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		store := GetStorage()
		if store == nil {
			return fmt.Errorf("storage not initialized")
		}

		fmt.Println("Repairing database...")

		// Integrity check
		ok, err := store.IntegrityCheck()
		if err != nil {
			fmt.Printf("  ✗ Integrity check error: %v\n", err)
		} else if ok {
			fmt.Println("  ✓ Integrity check passed")
		} else {
			fmt.Println("  ✗ Integrity check failed")
			color.Yellow("Warning: Database may be corrupted. Consider restoring from backup.")
		}

		// Vacuum
		if err := store.Vacuum(); err != nil {
			fmt.Printf("  ✗ Vacuum failed: %v\n", err)
		} else {
			fmt.Println("  ✓ Database vacuumed")
		}

		color.Green("\nRepair complete.")
		return nil
	},
}

var syncResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete and recreate the database (DESTRUCTIVE)",
	Long: `Delete the local database and start fresh.

WARNING: This will permanently delete ALL your data!
Make sure to export your data first with 'toki export yaml'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("WARNING: This will permanently delete ALL your data!")
		fmt.Println("Make sure you have exported your data with 'toki export yaml'.")
		fmt.Print("\nType 'reset' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "reset" {
			fmt.Println("Aborted.")
			return nil
		}

		dbPath := storage.DefaultDBPath()

		// Close current storage
		if err := CloseStorage(); err != nil {
			return fmt.Errorf("failed to close storage: %w", err)
		}

		// Delete database files
		for _, suffix := range []string{"", "-wal", "-shm"} {
			path := dbPath + suffix
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Warning: failed to delete %s: %v\n", path, err)
			}
		}

		fmt.Println("  ✓ Database deleted")
		fmt.Println("\nA new database will be created on next use.")
		color.Green("Reset complete.")

		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncRepairCmd)
	syncCmd.AddCommand(syncResetCmd)

	rootCmd.AddCommand(syncCmd)
}
