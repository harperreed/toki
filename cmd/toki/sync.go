// ABOUTME: Sync subcommand for vault integration
// ABOUTME: Provides init, login, status, now, logout, pending, and wipe commands for cloud sync

package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/sync"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manage cloud sync for toki data",
	Long: `Sync your toki data securely to the cloud using E2E encryption.

Commands:
  init    - Initialize sync configuration
  login   - Login to sync server
  status  - Show sync status
  now     - Manually trigger sync
  logout  - Clear authentication
  pending - Show changes waiting to sync
  wipe    - Wipe all sync data and start fresh

Examples:
  toki sync init
  toki sync login --server https://api.storeusa.org
  toki sync status
  toki sync now`,
}

var syncInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize sync configuration",
	Long:  `Creates a new sync configuration with a unique device ID.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if sync.ConfigExists() {
			return fmt.Errorf("config already exists at %s\nUse 'toki sync status' to view or delete the file to reinitialize", sync.ConfigPath())
		}

		cfg, err := sync.InitConfig()
		if err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		color.Green("✓ Sync initialized")
		fmt.Printf("  Config: %s\n", sync.ConfigPath())
		fmt.Printf("  Device: %s\n", cfg.DeviceID)
		fmt.Println("\nNext: Run 'toki sync login' to authenticate")

		return nil
	},
}

var syncLoginCmd = &cobra.Command{
	Use:   "login [email]",
	Short: "Login to sync server",
	Long: `Authenticate with the sync server using email, password, and recovery phrase.

The recovery phrase is your BIP39 mnemonic that was given to you when you
registered. Only the derived key is stored locally, not the mnemonic.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")

		cfg, _ := sync.LoadConfig()
		if cfg == nil {
			cfg = &sync.Config{}
		}

		serverURL := server
		if serverURL == "" {
			serverURL = cfg.Server
		}
		if serverURL == "" {
			serverURL = "https://api.storeusa.org"
		}

		reader := bufio.NewReader(os.Stdin)

		// Get email
		var email string
		if len(args) > 0 {
			email = args[0]
		} else {
			fmt.Print("Email: ")
			emailInput, _ := reader.ReadString('\n')
			email = strings.TrimSpace(emailInput)
		}
		if email == "" {
			return fmt.Errorf("email required")
		}

		// Get password
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(syscall.Stdin)
		fmt.Println()
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		password := string(passwordBytes)

		// Get mnemonic
		fmt.Print("\nEnter your recovery phrase:\n> ")
		mnemonic, _ := reader.ReadString('\n')
		mnemonic = strings.TrimSpace(mnemonic)

		// Validate mnemonic
		if _, err := vault.ParseMnemonic(mnemonic); err != nil {
			return fmt.Errorf("invalid recovery phrase: %w", err)
		}

		// Ensure we have a device ID before login (v0.3.0 requires device registration)
		if cfg.DeviceID == "" {
			cfg.DeviceID = ulid.Make().String()
		}

		// Login to server with device registration (v0.3.0 API)
		fmt.Printf("\nLogging in to %s...\n", serverURL)
		client := vault.NewPBAuthClient(serverURL)
		result, err := client.Login(context.Background(), email, password, cfg.DeviceID)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// Derive key from mnemonic (we only store the derived key, not the mnemonic)
		seed, err := vault.ParseSeedPhrase(mnemonic)
		if err != nil {
			return fmt.Errorf("parse mnemonic: %w", err)
		}
		derivedKeyHex := hex.EncodeToString(seed.Raw)

		// Save config
		cfg.Server = serverURL
		cfg.UserID = result.UserID
		cfg.Token = result.Token.Token
		cfg.RefreshToken = result.RefreshToken
		cfg.TokenExpires = result.Token.Expires.Format(time.RFC3339)
		cfg.DerivedKey = derivedKeyHex
		if cfg.VaultDB == "" {
			cfg.VaultDB = sync.ConfigDir() + "/vault.db"
		}

		if err := sync.SaveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		color.Green("\n✓ Logged in to toki sync")
		fmt.Printf("Token expires: %s\n", result.Token.Expires.Format(time.RFC3339))

		return nil
	},
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status",
	Long:  `Display current sync configuration and authentication status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := sync.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		fmt.Printf("Config:    %s\n", sync.ConfigPath())
		fmt.Printf("Server:    %s\n", valueOrNone(cfg.Server))
		fmt.Printf("User ID:   %s\n", valueOrNone(cfg.UserID))
		fmt.Printf("Device ID: %s\n", valueOrNone(cfg.DeviceID))
		fmt.Printf("Vault DB:  %s\n", valueOrNone(cfg.VaultDB))
		fmt.Printf("Auto-sync: %v\n", cfg.AutoSync)

		if cfg.DerivedKey != "" {
			fmt.Println("Keys:      " + color.GreenString("✓ configured"))
		} else {
			fmt.Println("Keys:      " + color.YellowString("(not set)"))
		}

		printTokenStatus(cfg)

		// Show sync state if configured
		if cfg.IsConfigured() {
			syncer, err := sync.NewSyncer(cfg, dbConn)
			if err == nil {
				defer func() { _ = syncer.Close() }()
				ctx := context.Background()

				pending, err := syncer.PendingCount(ctx)
				if err == nil {
					fmt.Print("\nPending:   ")
					if pending == 0 {
						color.Green("0 changes (up to date)")
					} else {
						color.Yellow("%d changes waiting to push", pending)
					}
					fmt.Println()
				}

				lastSeq, err := syncer.LastSyncedSeq(ctx)
				if err == nil && lastSeq != "0" {
					fmt.Printf("Last sync: seq %s\n", lastSeq)
				}
			}
		}

		return nil
	},
}

var syncNowCmd = &cobra.Command{
	Use:   "now",
	Short: "Manually trigger sync",
	Long:  `Push local changes and pull remote changes from the sync server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")

		cfg, err := sync.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if !cfg.IsConfigured() {
			return fmt.Errorf("sync not configured - run 'toki sync login' first")
		}

		syncer, err := sync.NewSyncer(cfg, dbConn)
		if err != nil {
			return fmt.Errorf("create syncer: %w", err)
		}
		defer func() { _ = syncer.Close() }()

		ctx := context.Background()

		var events *vault.SyncEvents
		if verbose {
			events = &vault.SyncEvents{
				OnStart: func() {
					fmt.Println("Syncing...")
				},
				OnPush: func(pushed, remaining int) {
					fmt.Printf("  ↑ pushed %d changes (%d remaining)\n", pushed, remaining)
				},
				OnPull: func(pulled int) {
					if pulled > 0 {
						fmt.Printf("  ↓ pulled %d changes\n", pulled)
					}
				},
				OnComplete: func(pushed, pulled int) {
					fmt.Printf("  Total: %d pushed, %d pulled\n", pushed, pulled)
				},
			}
		} else {
			fmt.Println("Syncing...")
		}

		if err := syncer.SyncWithEvents(ctx, events); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		color.Green("✓ Sync complete")
		return nil
	},
}

var syncLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear authentication",
	Long:  `Remove auth tokens from config. The derived key is preserved for re-login.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := sync.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.Token == "" {
			fmt.Println("Not logged in")
			return nil
		}

		cfg.Token = ""
		cfg.RefreshToken = ""
		cfg.TokenExpires = ""

		if err := sync.SaveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		color.Green("✓ Logged out successfully")
		return nil
	},
}

var syncPendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "Show changes waiting to sync",
	Long:  `List all changes in the outbox that haven't been pushed to the server yet.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := sync.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if !cfg.IsConfigured() {
			fmt.Println("Sync not configured. Run 'toki sync login' first.")
			return nil
		}

		syncer, err := sync.NewSyncer(cfg, dbConn)
		if err != nil {
			return fmt.Errorf("create syncer: %w", err)
		}
		defer func() { _ = syncer.Close() }()

		items, err := syncer.PendingChanges(context.Background())
		if err != nil {
			return fmt.Errorf("get pending: %w", err)
		}

		if len(items) == 0 {
			color.Green("✓ No pending changes - everything is synced!")
			return nil
		}

		fmt.Printf("Pending changes (%d):\n\n", len(items))
		for _, item := range items {
			fmt.Printf("  %s  %-10s  %s\n",
				color.New(color.Faint).Sprint(item.ChangeID[:8]),
				item.Entity,
				item.TS.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("\nRun 'toki sync now' to push these changes.\n")

		return nil
	},
}

var syncWipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Wipe all sync data and start fresh",
	Long: `Clear all server-side sync data and local vault database.

This is useful when:
- Sync data becomes corrupted
- You changed userID/AAD strategy mid-stream
- You want to start fresh during development

After wipe, run 'toki sync now' to re-push local data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := sync.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if !cfg.IsConfigured() {
			return fmt.Errorf("sync not configured - run 'toki sync login' first")
		}

		// Confirm with user
		fmt.Println("This will DELETE all sync data on the server and locally.")
		fmt.Println("Your local toki data will NOT be affected.")
		fmt.Print("\nType 'wipe' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "wipe" {
			fmt.Println("Aborted.")
			return nil
		}

		// Wipe server-side data
		fmt.Println("\nWiping server data...")
		client := vault.NewClient(vault.SyncConfig{
			AppID:     sync.AppID,
			BaseURL:   cfg.Server,
			DeviceID:  cfg.DeviceID,
			AuthToken: cfg.Token,
		})

		ctx := context.Background()
		deleted, err := client.Wipe(ctx)
		if err != nil {
			return fmt.Errorf("wipe server data: %w", err)
		}
		color.Green("✓ Server data wiped (%d records deleted)", deleted)

		// Remove local vault.db
		fmt.Println("Removing local vault database...")
		if err := os.Remove(cfg.VaultDB); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove vault.db: %w", err)
		}
		color.Green("✓ Local vault.db removed")

		fmt.Println("\nSync data cleared. Run 'toki sync now' to re-push local data.")
		return nil
	},
}

func init() {
	syncLoginCmd.Flags().String("server", "", "sync server URL (default: https://api.storeusa.org)")
	syncNowCmd.Flags().BoolP("verbose", "v", false, "show detailed sync information")

	syncCmd.AddCommand(syncInitCmd)
	syncCmd.AddCommand(syncLoginCmd)
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncNowCmd)
	syncCmd.AddCommand(syncLogoutCmd)
	syncCmd.AddCommand(syncPendingCmd)
	syncCmd.AddCommand(syncWipeCmd)

	rootCmd.AddCommand(syncCmd)
}

// valueOrNone returns "(not set)" if the string is empty.
func valueOrNone(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

// printTokenStatus displays token validity information.
func printTokenStatus(cfg *sync.Config) {
	if cfg.Token == "" {
		fmt.Println("\nStatus:    " + color.YellowString("Not logged in"))
		return
	}

	fmt.Println()
	if cfg.TokenExpires == "" {
		fmt.Println("Token:     valid (no expiry info)")
		return
	}

	expires, err := time.Parse(time.RFC3339, cfg.TokenExpires)
	if err != nil {
		fmt.Printf("Token:     valid (invalid expiry: %v)\n", err)
		return
	}

	now := time.Now()
	if expires.Before(now) {
		fmt.Print("Token:     ")
		color.Red("EXPIRED (%s ago)", now.Sub(expires).Round(time.Second))
		fmt.Println()
		if cfg.RefreshToken != "" {
			fmt.Println("           (has refresh token - run 'toki sync now' to refresh)")
		}
	} else {
		fmt.Print("Token:     ")
		color.Green("valid")
		fmt.Printf(" (expires in %s)\n", formatDuration(expires.Sub(now)))
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
