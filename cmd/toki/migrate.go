// ABOUTME: Import command for YAML-based data import
// ABOUTME: Imports data from toki export yaml files

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/harper/toki/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Import command for importing from YAML export.
var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import data from YAML export",
	Long: `Import toki data from a YAML export file.

This command reads a YAML file created by 'toki export yaml' and imports
all projects and todos into the database.

The import is idempotent - items with matching IDs will be skipped.

Examples:
  toki import backup.yaml
  toki import --dry-run backup.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	importCmd.Flags().Bool("dry-run", false, "show what would be imported without making changes")
	rootCmd.AddCommand(importCmd)
}

//nolint:funlen // Import logic is inherently sequential and splitting would reduce clarity
func runImport(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	filename := args[0]

	// Read the YAML file
	data, err := os.ReadFile(filename) // #nosec G304 - user-provided import path is intentional
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var exportData ExportData
	if err := yaml.Unmarshal(data, &exportData); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	store := GetStorage()
	if store == nil {
		return fmt.Errorf("storage not initialized")
	}

	if dryRun {
		fmt.Println("=== DRY RUN MODE ===")
		fmt.Printf("Importing from: %s\n", filename)
		fmt.Printf("Export date: %s\n\n", exportData.ExportedAt)
	} else {
		fmt.Printf("Importing from: %s\n", filename)
		fmt.Printf("Export date: %s\n\n", exportData.ExportedAt)
	}

	// Import projects and todos
	projectCount := 0
	todoCount := 0
	tagCount := 0

	for _, proj := range exportData.Projects {
		imported, err := importProject(store, proj, dryRun)
		if err != nil {
			fmt.Printf("  [error] Failed to import project '%s': %v\n", proj.Name, err)
			continue
		}
		if imported {
			projectCount++
		}

		// Import todos for this project
		for _, todo := range proj.Todos {
			imported, tags, err := importTodo(store, proj.ID, todo, dryRun)
			if err != nil {
				fmt.Printf("  [error] Failed to import todo '%s': %v\n", truncateStr(todo.Description, 30), err)
				continue
			}
			if imported {
				todoCount++
				tagCount += tags
			}
		}
	}

	fmt.Println()
	if dryRun {
		color.Cyan("=== DRY RUN SUMMARY ===")
		fmt.Printf("Would import %d projects\n", projectCount)
		fmt.Printf("Would import %d todos\n", todoCount)
		fmt.Printf("Would create %d tag associations\n", tagCount)
	} else {
		color.Green("=== IMPORT COMPLETE ===")
		fmt.Printf("Imported %d projects\n", projectCount)
		fmt.Printf("Imported %d todos\n", todoCount)
		fmt.Printf("Created %d tag associations\n", tagCount)
	}

	return nil
}

func importProject(store storage.Storage, proj ExportProject, dryRun bool) (bool, error) {
	// Parse UUID
	id, err := uuid.Parse(proj.ID)
	if err != nil {
		return false, fmt.Errorf("invalid project ID: %w", err)
	}

	// Check if already exists
	existing, err := store.GetProject(id)
	if err == nil {
		fmt.Printf("  [skip] Project '%s' already exists\n", existing.Name)
		return false, nil
	}

	project := &storage.Project{
		ID:            id,
		Name:          proj.Name,
		DirectoryPath: proj.Path,
		CreatedAt:     time.Now(), // Export doesn't include project creation time
	}

	if dryRun {
		fmt.Printf("  [would import] Project '%s'\n", proj.Name)
	} else {
		if err := store.CreateProject(project); err != nil {
			return false, err
		}
		fmt.Printf("  [imported] Project '%s'\n", proj.Name)
	}

	return true, nil
}

//nolint:funlen // Import logic is inherently sequential and splitting would reduce clarity
func importTodo(store storage.Storage, projectID string, todo ExportTodo, dryRun bool) (bool, int, error) {
	// Parse UUIDs
	todoID, err := uuid.Parse(todo.ID)
	if err != nil {
		return false, 0, fmt.Errorf("invalid todo ID: %w", err)
	}

	projID, err := uuid.Parse(projectID)
	if err != nil {
		return false, 0, fmt.Errorf("invalid project ID: %w", err)
	}

	// Check if already exists
	existing, err := store.GetTodo(todoID)
	if err == nil {
		fmt.Printf("  [skip] Todo '%s' already exists\n", truncateStr(existing.Description, 30))
		return false, 0, nil
	}

	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, todo.CreatedAt)
	if err != nil {
		return false, 0, fmt.Errorf("invalid created_at: %w", err)
	}

	updatedAt := createdAt
	if todo.UpdatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, todo.UpdatedAt); err == nil {
			updatedAt = parsed
		}
	}

	var completedAt *time.Time
	if todo.CompletedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, todo.CompletedAt); err == nil {
			completedAt = &parsed
		}
	}

	var dueDate *time.Time
	if todo.DueDate != "" {
		if parsed, err := time.Parse(time.RFC3339, todo.DueDate); err == nil {
			dueDate = &parsed
		}
	}

	todoData := &storage.Todo{
		ID:          todoID,
		ProjectID:   projID,
		Description: todo.Description,
		Done:        todo.Done,
		Priority:    todo.Priority,
		Notes:       todo.Notes,
		Tags:        todo.Tags,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		CompletedAt: completedAt,
		DueDate:     dueDate,
	}

	if dryRun {
		fmt.Printf("  [would import] Todo '%s'\n", truncateStr(todo.Description, 40))
	} else {
		if err := store.CreateTodo(todoData); err != nil {
			return false, 0, err
		}
		fmt.Printf("  [imported] Todo '%s'\n", truncateStr(todo.Description, 40))
	}
	tagCount := len(todo.Tags)

	return true, tagCount, nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
