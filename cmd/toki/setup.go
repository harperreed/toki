// ABOUTME: Cobra command for interactive toki storage configuration.
// ABOUTME: Launches a bubbletea TUI wizard to select backend and data directory.
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/harper/toki/internal/config"
	"github.com/harper/toki/internal/tui"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure toki storage backend",
	Long:  "Interactive wizard to configure storage backend and data directory.",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	model := tui.NewSetupModel(cfg.Backend, cfg.DataDir)

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	final, ok := result.(tui.SetupModel)
	if !ok {
		return fmt.Errorf("unexpected model type from TUI")
	}
	if !final.ShouldSave() {
		fmt.Println("Setup cancelled.")
		return nil
	}

	backend, dataDir := final.Result()
	cfg.Backend = backend
	cfg.DataDir = dataDir

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Config saved to %s\n", config.GetConfigPath())
	return nil
}
