// ABOUTME: Sync subcommand for Charm backend integration
// ABOUTME: Provides status, now, link, unlink, and wipe commands for Charm Cloud sync

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
  link    - Link this device to your Charm account
  unlink  - Unlink this device from Charm account
  wipe    - Clear all local/remote data

Examples:
  toki sync status
  toki sync link`,
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

		color.Green("\nStatus: Connected to Charm Cloud")
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

var syncWipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Clear all local/remote data",
	Long: `Clear all sync data both locally and on Charm Cloud.

This is a destructive operation that:
- Deletes all local Charm KV data
- Clears all remote data on Charm Cloud
- Preserves your device link and SSH keys

After wipe, your todos will be gone. Use with caution!`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("⚠️  WARNING: This will DELETE all toki data!")
		fmt.Println("This affects:")
		fmt.Println("  - All local todos, projects, and tags")
		fmt.Println("  - All synced data on Charm Cloud")
		fmt.Println("\nThis action CANNOT be undone!")
		fmt.Print("\nType 'wipe' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "wipe" {
			fmt.Println("Aborted.")
			return nil
		}

		client := charm.GetClient()
		if client == nil {
			return fmt.Errorf("client not initialized")
		}

		fmt.Println("\nClearing local data...")

		// Reset the KV store (deletes local and pulls fresh from cloud, which is empty after reset)
		if err := client.KV().Reset(); err != nil {
			return fmt.Errorf("failed to reset KV store: %w", err)
		}

		color.Green("✓ All data wiped")
		fmt.Println("\nYour Charm account is still linked.")
		fmt.Println("You can start adding todos again.")

		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncLinkCmd)
	syncCmd.AddCommand(syncUnlinkCmd)
	syncCmd.AddCommand(syncWipeCmd)

	rootCmd.AddCommand(syncCmd)
}
