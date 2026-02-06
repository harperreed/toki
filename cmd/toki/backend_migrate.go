// ABOUTME: Migration command for converting toki data between storage backends
// ABOUTME: Supports sqlite-to-markdown and markdown-to-sqlite with safety checks

package main

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/harper/toki/internal/config"
	"github.com/harper/toki/internal/storage"
)

var migrateBackendCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate data between storage backends",
	Long: `Migrate all toki data from the currently configured backend to a different backend.

Reads projects, todos, and tags from the current backend
and writes them to the target backend. Does NOT update the config file;
verify the migration was successful then update config.json manually.

Examples:
  toki migrate --to markdown
  toki migrate --to sqlite --data-dir ~/toki-sqlite
  toki migrate --to markdown --force`,
	RunE: runBackendMigrate,
}

var (
	backendMigrateTo      string
	backendMigrateDataDir string
	backendMigrateForce   bool
)

func init() {
	rootCmd.AddCommand(migrateBackendCmd)
	migrateBackendCmd.Flags().StringVar(&backendMigrateTo, "to", "", "target backend (sqlite or markdown)")
	migrateBackendCmd.Flags().StringVar(&backendMigrateDataDir, "data-dir", "", "target data directory (defaults to current config data_dir)")
	migrateBackendCmd.Flags().BoolVar(&backendMigrateForce, "force", false, "allow writing into a non-empty target directory")
	_ = migrateBackendCmd.MarkFlagRequired("to")
}

func runBackendMigrate(cmd *cobra.Command, args []string) error {
	// Load config and determine source backend
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sourceBackend := cfg.GetBackend()
	targetBackend := backendMigrateTo

	// Validate target backend
	if targetBackend != "sqlite" && targetBackend != "markdown" {
		return fmt.Errorf("invalid target backend %q: must be \"sqlite\" or \"markdown\"", targetBackend)
	}
	if targetBackend == sourceBackend {
		return fmt.Errorf("target backend %q is the same as the current backend", targetBackend)
	}

	// Determine target data directory
	targetDataDir := cfg.GetDataDir()
	if backendMigrateDataDir != "" {
		targetDataDir = config.ExpandPath(backendMigrateDataDir)
	}

	// Check if target directory is non-empty
	nonEmpty, err := storage.IsDirNonEmpty(targetDataDir)
	if err != nil {
		return fmt.Errorf("check target directory: %w", err)
	}
	if nonEmpty && !backendMigrateForce {
		return fmt.Errorf("target directory %q is not empty; use --force to overwrite", targetDataDir)
	}

	// Open source storage
	src, err := cfg.OpenStorage()
	if err != nil {
		return fmt.Errorf("open source storage (%s): %w", sourceBackend, err)
	}
	defer func() { _ = src.Close() }()

	// Open target storage
	dst, err := openTargetStorage(targetBackend, targetDataDir)
	if err != nil {
		return fmt.Errorf("open target storage (%s): %w", targetBackend, err)
	}
	defer func() { _ = dst.Close() }()

	return executeMigration(src, dst, sourceBackend, targetBackend, cfg.GetDataDir(), targetDataDir)
}

// executeMigration runs the migration and prints the results.
func executeMigration(src, dst storage.Storage, sourceBackend, targetBackend, sourceDataDir, targetDataDir string) error {
	color.Yellow("Migrating toki data:")
	fmt.Printf("  Source:  %s (%s)\n", sourceBackend, sourceDataDir)
	fmt.Printf("  Target:  %s (%s)\n", targetBackend, targetDataDir)
	fmt.Println()

	summary, err := storage.MigrateData(src, dst)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	color.Green("Migration complete!")
	fmt.Printf("  Projects: %d\n", summary.Projects)
	fmt.Printf("  Todos:    %d\n", summary.Todos)
	fmt.Printf("  Tags:     %d\n", summary.Tags)
	fmt.Println()
	color.Yellow("Note: config.json was NOT updated. To switch to the new backend, edit:")
	fmt.Printf("  %s\n", config.GetConfigPath())
	fmt.Printf("  Set \"backend\": %q", targetBackend)
	if backendMigrateDataDir != "" {
		fmt.Printf(" and \"data_dir\": %q", backendMigrateDataDir)
	}
	fmt.Println()

	return nil
}

// openTargetStorage creates a Storage implementation for the given backend and data directory.
func openTargetStorage(backend, dataDir string) (storage.Storage, error) {
	switch backend {
	case "sqlite":
		dbPath := filepath.Join(dataDir, "toki.db")
		return storage.NewSQLiteStorage(dbPath)
	case "markdown":
		return storage.NewMarkdownStore(dataDir)
	default:
		return nil, fmt.Errorf("unknown backend: %q", backend)
	}
}
