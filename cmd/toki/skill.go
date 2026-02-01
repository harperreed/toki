// ABOUTME: Install Claude Code skill for toki
// ABOUTME: Embeds and installs the skill definition to ~/.claude/skills/

package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

var skillSkipConfirm bool

var installSkillCmd = &cobra.Command{
	Use:   "install-skill",
	Short: "Install Claude Code skill",
	Long: `Install the toki skill for Claude Code.

This copies the skill definition to ~/.claude/skills/toki/
so Claude Code can use toki commands contextually.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return installSkill()
	},
}

func init() {
	installSkillCmd.Flags().BoolVarP(&skillSkipConfirm, "yes", "y", false, "Skip confirmation prompt")
	rootCmd.AddCommand(installSkillCmd)
}

// installSkillOptions contains configuration for skill installation.
// Used to inject dependencies for testing.
type installSkillOptions struct {
	homeDir     string        // Override home directory (empty = use os.UserHomeDir)
	input       *bufio.Reader // Override stdin reader (nil = use os.Stdin)
	skipConfirm bool          // Skip confirmation prompt
	output      func(string)  // Output function (nil = use fmt.Println)
}

// defaultInstallOptions returns installation options using real system values.
func defaultInstallOptions() installSkillOptions {
	return installSkillOptions{
		homeDir:     "",
		input:       nil,
		skipConfirm: skillSkipConfirm,
		output:      nil,
	}
}

func installSkill() error {
	return installSkillWithOptions(defaultInstallOptions())
}

//nolint:funlen
func installSkillWithOptions(opts installSkillOptions) error {
	// Helper for output
	println := func(s string) {
		if opts.output != nil {
			opts.output(s)
		} else {
			fmt.Println(s)
		}
	}
	printf := func(format string, args ...interface{}) {
		s := fmt.Sprintf(format, args...)
		if opts.output != nil {
			opts.output(s)
		} else {
			fmt.Print(s)
		}
	}

	// Determine destination
	home := opts.homeDir
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
	}

	skillDir := filepath.Join(home, ".claude", "skills", "toki")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Show explanation
	println("┌─────────────────────────────────────────────────────────────┐")
	println("│              Toki Skill for Claude Code                     │")
	println("└─────────────────────────────────────────────────────────────┘")
	println("")
	println("This will install the toki skill, enabling Claude Code to:")
	println("")
	println("  • Manage your todos and tasks via natural language")
	println("  • Create, update, and organize projects")
	println("  • Track priorities and due dates")
	println("  • Use the /toki slash command")
	println("")
	println("Destination:")
	printf("  %s\n", skillPath)
	println("")

	// Check if already installed
	if _, err := os.Stat(skillPath); err == nil {
		println("Note: A skill file already exists and will be overwritten.")
		println("")
	}

	// Ask for confirmation unless skipConfirm is set
	if !opts.skipConfirm {
		printf("Install the toki skill? [y/N] ")
		reader := opts.input
		if reader == nil {
			reader = bufio.NewReader(os.Stdin)
		}
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			println("Installation cancelled.")
			return nil
		}
		println("")
	}

	// Read embedded skill file
	content, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded skill: %w", err)
	}

	// Create directory
	if err := os.MkdirAll(skillDir, 0750); err != nil { // #nosec G301 -- skill directory needs group/other read for Claude
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill file
	if err := os.WriteFile(skillPath, content, 0600); err != nil { // #nosec G306
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	println("✓ Installed toki skill successfully!")
	println("")
	println("Claude Code will now recognize /toki commands.")
	println("Try asking Claude: \"Show my todos\" or \"Add a task to fix the login bug\"")

	return nil
}
