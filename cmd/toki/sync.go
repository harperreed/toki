// ABOUTME: Sync subcommand for Charm backend integration
// ABOUTME: Provides status, now, link, unlink, repair, reset, and wipe commands for Charm Cloud sync

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/charm/kv"
	"github.com/fatih/color"
	"github.com/harper/toki/internal/charm"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manage Charm Cloud sync for toki data",
	Long: `Sync your toki data securely to Charm Cloud using SSH key authentication.

Charm automatically handles encryption and syncing across devices linked
to your Charm account.

Commands:
  status  - Show sync status and configuration
  now     - Trigger immediate sync
  link    - Link this device to your Charm account
  unlink  - Unlink this device from Charm account
  repair  - Repair a corrupted local database
  reset   - Delete local database and re-download from cloud
  wipe    - Permanently delete ALL data (local and cloud)

Examples:
  toki sync status
  toki sync now
  toki sync link
  toki sync repair --force`,
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status and configuration",
	Long:  `Display current Charm Cloud configuration and connection status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := charm.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		fmt.Printf("Config:    %s\n", charm.ConfigPath())
		fmt.Printf("Server:    %s\n", cfg.Server)
		fmt.Printf("Auto-sync: %v\n", cfg.AutoSync)

		client := charm.GetClient()
		if client == nil {
			color.Yellow("\nStatus: Client not initialized")
			fmt.Println("\nRun 'toki sync link' to connect to Charm Cloud")
			return nil
		}

		// Get and display Charm ID
		id, err := client.ID()
		if err != nil {
			color.Yellow("\nStatus: Connected (ID unavailable)")
		} else {
			color.Green("\nStatus: Connected to Charm Cloud")
			fmt.Printf("ID:        %s\n", id)
		}

		fmt.Println("\nCharm uses SSH keys for authentication - no login required!")
		fmt.Println("Sync happens automatically in the background.")

		return nil
	},
}

var syncLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link this device to your Charm account",
	Long: `Link this device to your Charm account for cross-device sync.

This will generate SSH keys if needed and register this device with
your Charm account. You'll be able to sync data across all linked devices.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := charm.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		fmt.Printf("Linking to Charm Cloud (%s)...\n\n", cfg.Server)
		fmt.Println("Charm uses SSH key authentication.")
		fmt.Println("If this is your first time, you may need to register your account.")
		fmt.Println("Visit the URL that appears to complete registration.")

		// For now, we rely on the Charm library's automatic linking
		// When a client is created, it will handle authentication
		client := charm.GetClient()
		if client == nil {
			return fmt.Errorf("failed to initialize client - check your Charm installation")
		}

		// Test the connection by attempting a sync
		if err := client.Sync(); err != nil {
			return fmt.Errorf("link failed: %w", err)
		}

		color.Green("\n✓ Device linked successfully")
		fmt.Println("\nYour device is now syncing with Charm Cloud!")

		return nil
	},
}

var syncUnlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Unlink this device from Charm account",
	Long: `Remove this device from your Charm account.

This will stop syncing but won't delete your local data.
You can re-link later with 'toki sync link'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("This will unlink your device from Charm Cloud.")
		fmt.Println("Local data will be preserved, but won't sync anymore.")
		fmt.Print("\nType 'unlink' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "unlink" {
			fmt.Println("Aborted.")
			return nil
		}

		// Close the client to disconnect
		if err := charm.CloseClient(); err != nil {
			return fmt.Errorf("failed to close client: %w", err)
		}

		// Note: Charm doesn't provide a direct "unlink" API
		// To fully unlink, users should remove SSH keys from their Charm account
		color.Yellow("\n⚠ Device connection closed")
		fmt.Println("\nTo fully unlink, remove this device's SSH key from your Charm account.")
		fmt.Println("Local data has been preserved.")

		return nil
	},
}

var syncNowCmd = &cobra.Command{
	Use:   "now",
	Short: "Trigger immediate sync with Charm Cloud",
	Long: `Force an immediate sync with Charm Cloud.

This pushes local changes and pulls remote changes from other devices.
Normally sync happens automatically, but use this to force a refresh.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := charm.GetClient()
		if client == nil {
			return fmt.Errorf("client not initialized - run 'toki sync link' first")
		}

		fmt.Println("Syncing with Charm Cloud...")
		if err := client.Sync(); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		color.Green("✓ Sync complete")
		return nil
	},
}

var syncRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair a corrupted local database",
	Long: `Repair a corrupted local database using SQLite recovery tools.

This command will:
- Checkpoint the WAL (Write-Ahead Log)
- Remove shared memory files
- Run integrity checks
- Vacuum the database
- Attempt REINDEX recovery if --force is specified

Use this if you encounter database corruption errors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		fmt.Println("Repairing database...")
		result, err := kv.Repair("toki", force)

		if result.WalCheckpointed {
			fmt.Println("  ✓ WAL checkpointed")
		}
		if result.ShmRemoved {
			fmt.Println("  ✓ SHM file removed")
		}
		if result.IntegrityOK {
			fmt.Println("  ✓ Integrity check passed")
		} else {
			fmt.Println("  ✗ Integrity check failed")
		}
		if result.Vacuumed {
			fmt.Println("  ✓ Database vacuumed")
		}

		if err != nil {
			if !force {
				fmt.Println("\nRun with --force to attempt recovery.")
			}
			return err
		}
		fmt.Println("\nRepair complete.")
		return nil
	},
}

var syncResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete local database and re-download from cloud",
	Long: `Delete your local database and re-download from Charm Cloud.

This is useful if your local database is corrupted or out of sync.
Your cloud data will be preserved and re-synced to a fresh local database.

WARNING: Any unsynced local changes will be lost!`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("This will delete your local database and re-download from Charm Cloud.")
		fmt.Println("Any unsynced local data will be lost.")
		fmt.Print("\nContinue? [y/N] ")

		var confirm string
		if _, err := fmt.Scanln(&confirm); err != nil {
			fmt.Println("Cancelled.")
			return nil
		}
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}

		fmt.Println("\nResetting database...")
		if err := kv.Reset("toki"); err != nil {
			return fmt.Errorf("reset failed: %w", err)
		}

		fmt.Println("  ✓ Local database deleted")
		fmt.Println("  ✓ Synced from cloud")
		fmt.Println("\nReset complete.")
		return nil
	},
}

var syncWipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Permanently delete ALL data (local and cloud)",
	Long: `Permanently delete ALL toki data from both local storage and Charm Cloud.

WARNING: This is a DESTRUCTIVE operation that:
- Deletes all local database files
- Deletes all cloud backups
- Cannot be undone

Your device link and SSH keys will be preserved, but all todos,
projects, and tags will be permanently deleted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("WARNING: This will permanently delete ALL data!")
		fmt.Println("This includes local AND cloud data. This cannot be undone.")
		fmt.Print("\nType 'wipe' to confirm: ")

		var confirm string
		if _, err := fmt.Scanln(&confirm); err != nil {
			fmt.Println("Cancelled.")
			return nil
		}
		if confirm != "wipe" {
			fmt.Println("Cancelled.")
			return nil
		}

		fmt.Println("\nWiping database...")
		result, err := kv.Wipe("toki")
		if err != nil {
			return fmt.Errorf("wipe failed: %w", err)
		}

		if result.CloudBackupsDeleted > 0 {
			fmt.Printf("  ✓ %d cloud backups deleted\n", result.CloudBackupsDeleted)
		}
		if result.LocalFilesDeleted > 0 {
			fmt.Printf("  ✓ %d local files deleted\n", result.LocalFilesDeleted)
		}

		fmt.Println("\nWipe complete.")
		return nil
	},
}

func init() {
	syncRepairCmd.Flags().Bool("force", false, "Attempt REINDEX recovery if corruption detected")

	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncNowCmd)
	syncCmd.AddCommand(syncLinkCmd)
	syncCmd.AddCommand(syncUnlinkCmd)
	syncCmd.AddCommand(syncRepairCmd)
	syncCmd.AddCommand(syncResetCmd)
	syncCmd.AddCommand(syncWipeCmd)

	rootCmd.AddCommand(syncCmd)
}
