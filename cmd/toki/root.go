// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and storage client

package main

import (
	"fmt"
	"sync"

	"github.com/harper/toki/internal/storage"
	"github.com/spf13/cobra"
)

// Global storage for command usage (thread-safe initialization).
var (
	globalStorage *storage.SQLiteStorage
	storageOnce   sync.Once
	storageErr    error
)

// InitStorage initializes the global SQLite storage (thread-safe).
func InitStorage() error {
	storageOnce.Do(func() {
		dbPath := storage.DefaultDBPath()
		globalStorage, storageErr = storage.NewSQLiteStorage(dbPath)
	})
	return storageErr
}

// CloseStorage closes the global storage.
func CloseStorage() error {
	if globalStorage != nil {
		return globalStorage.Close()
	}
	return nil
}

// GetStorage returns the global storage.
func GetStorage() *storage.SQLiteStorage {
	return globalStorage
}

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
		// Initialize SQLite storage
		if err := InitStorage(); err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close storage
		return CloseStorage()
	},
}

func init() {
	// No persistent flags needed - storage manages its own configuration
}
