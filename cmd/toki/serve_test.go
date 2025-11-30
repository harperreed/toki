// ABOUTME: Tests for MCP server command
// ABOUTME: Verifies command registration and basic functionality

package main

import (
	"testing"
)

func TestServeCommand(t *testing.T) {
	t.Run("command is registered", func(t *testing.T) {
		// Find serve command in root command
		cmd, _, err := rootCmd.Find([]string{"serve"})
		if err != nil {
			t.Fatalf("serve command not found: %v", err)
		}

		if cmd.Name() != "serve" {
			t.Errorf("expected command name 'serve', got '%s'", cmd.Name())
		}
	})

	t.Run("has correct short description", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"serve"})
		if err != nil {
			t.Fatalf("serve command not found: %v", err)
		}

		if cmd.Short == "" {
			t.Error("serve command should have a short description")
		}

		// Should mention MCP and stdio
		expectedShort := "Start MCP server (stdio mode)"
		if cmd.Short != expectedShort {
			t.Errorf("expected short description '%s', got '%s'", expectedShort, cmd.Short)
		}
	})

	t.Run("has long description", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"serve"})
		if err != nil {
			t.Fatalf("serve command not found: %v", err)
		}

		if cmd.Long == "" {
			t.Error("serve command should have a long description")
		}
	})

	t.Run("appears in help output", func(t *testing.T) {
		// Reset root command output
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)

		// Get available commands
		commands := rootCmd.Commands()
		found := false
		for _, cmd := range commands {
			if cmd.Name() == "serve" {
				found = true
				break
			}
		}

		if !found {
			t.Error("serve command should appear in root command's available commands")
		}
	})
}
