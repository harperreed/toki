// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and storage client

package main

import (
	"fmt"
	"sync"

	"github.com/harper/toki/internal/config"
	"github.com/harper/toki/internal/storage"
	"github.com/spf13/cobra"
)

// Global storage for command usage (thread-safe initialization).
var (
	globalStorage storage.Storage
	storageOnce   sync.Once
	storageErr    error
)

// InitStorage initializes the global storage using the configured backend (thread-safe).
func InitStorage() error {
	storageOnce.Do(func() {
		cfg, err := config.Load()
		if err != nil {
			storageErr = fmt.Errorf("load config: %w", err)
			return
		}
		globalStorage, storageErr = cfg.OpenStorage()
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
func GetStorage() storage.Storage {
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
		// Initialize storage from config
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
