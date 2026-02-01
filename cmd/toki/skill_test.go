// ABOUTME: Tests for install-skill command
// ABOUTME: Verifies skill installation, directory creation, and user interaction

package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillCommand(t *testing.T) {
	t.Run("command is registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"install-skill"})
		if err != nil {
			t.Fatalf("install-skill command not found: %v", err)
		}

		if cmd.Name() != "install-skill" {
			t.Errorf("expected command name 'install-skill', got '%s'", cmd.Name())
		}
	})

	t.Run("has yes flag", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"install-skill"})
		if err != nil {
			t.Fatalf("install-skill command not found: %v", err)
		}

		yesFlag := cmd.Flags().Lookup("yes")
		if yesFlag == nil {
			t.Fatal("install-skill command should have a --yes flag")
		}

		// Check shorthand
		if yesFlag.Shorthand != "y" {
			t.Errorf("expected --yes shorthand 'y', got '%s'", yesFlag.Shorthand)
		}
	})
}

//nolint:gocognit,gocyclo,funlen
func TestSkillInstallation(t *testing.T) {
	t.Run("successful installation with skip confirmation", func(t *testing.T) {
		tmpDir := t.TempDir()
		var output []string

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: true,
			output: func(s string) {
				output = append(output, s)
			},
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("installation failed: %v", err)
		}

		// Verify skill file exists
		skillPath := filepath.Join(tmpDir, ".claude", "skills", "toki", "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			t.Errorf("skill file not created at %s", skillPath)
		}

		// Verify success message was output
		found := false
		for _, line := range output {
			if strings.Contains(line, "Installed toki skill successfully") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected success message in output")
		}
	})

	t.Run("creates correct directory structure", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: true,
			output:      func(s string) {}, // Discard output
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("installation failed: %v", err)
		}

		// Verify full directory path exists
		skillDir := filepath.Join(tmpDir, ".claude", "skills", "toki")
		info, err := os.Stat(skillDir)
		if os.IsNotExist(err) {
			t.Fatalf("skill directory not created at %s", skillDir)
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", skillDir)
		}

		// Verify parent directories exist
		claudeDir := filepath.Join(tmpDir, ".claude")
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			t.Errorf(".claude directory not created")
		}

		skillsDir := filepath.Join(tmpDir, ".claude", "skills")
		if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
			t.Errorf(".claude/skills directory not created")
		}
	})

	t.Run("SKILL.md content is written correctly", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: true,
			output:      func(s string) {},
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("installation failed: %v", err)
		}

		// Read installed skill file
		skillPath := filepath.Join(tmpDir, ".claude", "skills", "toki", "SKILL.md")
		content, err := os.ReadFile(skillPath) //nolint:gosec // test file path is safe
		if err != nil {
			t.Fatalf("failed to read installed skill file: %v", err)
		}

		// Read embedded skill file for comparison
		embedded, err := skillFS.ReadFile("skill/SKILL.md")
		if err != nil {
			t.Fatalf("failed to read embedded skill: %v", err)
		}

		// Content should match exactly
		if string(content) != string(embedded) {
			t.Error("installed skill content does not match embedded content")
		}

		// Verify specific expected content
		contentStr := string(content)
		expectedStrings := []string{
			"name: toki",
			"Task and todo management",
			"mcp__toki__add_todo",
			"mcp__toki__list_todos",
			"mcp__toki__mark_done",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(contentStr, expected) {
				t.Errorf("skill content missing expected string: %s", expected)
			}
		}
	})

	t.Run("handles existing file overwrite", func(t *testing.T) {
		tmpDir := t.TempDir()
		var output []string

		// Create directory and existing file
		skillDir := filepath.Join(tmpDir, ".claude", "skills", "toki")
		err := os.MkdirAll(skillDir, 0750) // #nosec G301
		if err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		skillPath := filepath.Join(skillDir, "SKILL.md")
		existingContent := []byte("# Existing content that should be overwritten")
		err = os.WriteFile(skillPath, existingContent, 0600) // #nosec G306
		if err != nil {
			t.Fatalf("failed to create existing file: %v", err)
		}

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: true,
			output: func(s string) {
				output = append(output, s)
			},
		}

		err = installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("installation failed: %v", err)
		}

		// Verify overwrite note was displayed
		foundOverwriteNote := false
		for _, line := range output {
			if strings.Contains(line, "already exists") && strings.Contains(line, "overwritten") {
				foundOverwriteNote = true
				break
			}
		}
		if !foundOverwriteNote {
			t.Error("expected overwrite note in output when file exists")
		}

		// Verify content was overwritten
		newContent, err := os.ReadFile(skillPath) //nolint:gosec // test file path is safe
		if err != nil {
			t.Fatalf("failed to read skill file: %v", err)
		}

		if string(newContent) == string(existingContent) {
			t.Error("file was not overwritten")
		}

		// Verify it now has the correct content
		if !strings.Contains(string(newContent), "name: toki") {
			t.Error("overwritten content does not contain expected skill data")
		}
	})

	t.Run("cancellation when user says n", func(t *testing.T) {
		tmpDir := t.TempDir()
		var output []string

		// Create a reader that simulates user typing "n\n"
		input := strings.NewReader("n\n")
		reader := bufio.NewReader(input)

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: false, // Require confirmation
			input:       reader,
			output: func(s string) {
				output = append(output, s)
			},
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("unexpected error on cancellation: %v", err)
		}

		// Verify cancellation message
		foundCancelMsg := false
		for _, line := range output {
			if strings.Contains(line, "Installation cancelled") {
				foundCancelMsg = true
				break
			}
		}
		if !foundCancelMsg {
			t.Error("expected cancellation message in output")
		}

		// Verify skill file was NOT created
		skillPath := filepath.Join(tmpDir, ".claude", "skills", "toki", "SKILL.md")
		if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
			t.Error("skill file should not exist after cancellation")
		}
	})

	t.Run("confirmation with y proceeds with installation", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a reader that simulates user typing "y\n"
		input := strings.NewReader("y\n")
		reader := bufio.NewReader(input)

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: false, // Require confirmation
			input:       reader,
			output:      func(s string) {},
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("installation failed after confirmation: %v", err)
		}

		// Verify skill file was created
		skillPath := filepath.Join(tmpDir, ".claude", "skills", "toki", "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			t.Error("skill file should exist after confirming with 'y'")
		}
	})

	t.Run("confirmation with yes proceeds with installation", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a reader that simulates user typing "yes\n"
		input := strings.NewReader("yes\n")
		reader := bufio.NewReader(input)

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: false,
			input:       reader,
			output:      func(s string) {},
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("installation failed after confirmation: %v", err)
		}

		skillPath := filepath.Join(tmpDir, ".claude", "skills", "toki", "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			t.Error("skill file should exist after confirming with 'yes'")
		}
	})

	t.Run("cancellation with empty input", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a reader that simulates user just pressing enter
		input := strings.NewReader("\n")
		reader := bufio.NewReader(input)

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: false,
			input:       reader,
			output:      func(s string) {},
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should not install on empty input (default is N)
		skillPath := filepath.Join(tmpDir, ".claude", "skills", "toki", "SKILL.md")
		if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
			t.Error("skill file should not exist with empty input (default N)")
		}
	})

	t.Run("output shows destination path", func(t *testing.T) {
		tmpDir := t.TempDir()
		var output []string

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: true,
			output: func(s string) {
				output = append(output, s)
			},
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("installation failed: %v", err)
		}

		// Check that the destination path is shown
		expectedPath := filepath.Join(tmpDir, ".claude", "skills", "toki", "SKILL.md")
		foundPath := false
		for _, line := range output {
			if strings.Contains(line, expectedPath) {
				foundPath = true
				break
			}
		}
		if !foundPath {
			t.Errorf("expected destination path %s in output", expectedPath)
		}
	})

	t.Run("file permissions are correct", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: true,
			output:      func(s string) {},
		}

		err := installSkillWithOptions(opts)
		if err != nil {
			t.Fatalf("installation failed: %v", err)
		}

		// Check file permissions
		skillPath := filepath.Join(tmpDir, ".claude", "skills", "toki", "SKILL.md")
		info, err := os.Stat(skillPath)
		if err != nil {
			t.Fatalf("failed to stat skill file: %v", err)
		}

		// Should be readable by owner (at minimum)
		perm := info.Mode().Perm()
		if perm&0400 == 0 {
			t.Error("skill file should be readable by owner")
		}
	})
}

func TestDefaultInstallOptions(t *testing.T) {
	// Test that defaultInstallOptions returns the expected defaults
	opts := defaultInstallOptions()

	if opts.homeDir != "" {
		t.Error("expected homeDir to be empty (uses os.UserHomeDir)")
	}

	if opts.input != nil {
		t.Error("expected input to be nil (uses os.Stdin)")
	}

	if opts.output != nil {
		t.Error("expected output to be nil (uses fmt.Println)")
	}

	// skipConfirm depends on the global flag, which defaults to false
	// We can't easily test this without modifying the global state
}

func TestSkillInstallationErrors(t *testing.T) {
	t.Run("handles read-only directory gracefully", func(t *testing.T) {
		// Skip on Windows where permissions work differently
		if os.Getenv("GOOS") == "windows" {
			t.Skip("skipping permission test on Windows")
		}

		tmpDir := t.TempDir()

		// Create .claude directory and make it read-only
		claudeDir := filepath.Join(tmpDir, ".claude")
		err := os.MkdirAll(claudeDir, 0750) // #nosec G301
		if err != nil {
			t.Fatalf("failed to create .claude dir: %v", err)
		}

		// Make it read-only to prevent skill directory creation
		err = os.Chmod(claudeDir, 0444) // #nosec G302
		if err != nil {
			t.Fatalf("failed to chmod .claude dir: %v", err)
		}
		defer func() { _ = os.Chmod(claudeDir, 0750) }() //nolint:gosec // Restore for cleanup

		opts := installSkillOptions{
			homeDir:     tmpDir,
			skipConfirm: true,
			output:      func(s string) {},
		}

		err = installSkillWithOptions(opts)
		if err == nil {
			t.Error("expected error when directory is read-only")
		}

		if !strings.Contains(err.Error(), "failed to create skill directory") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
