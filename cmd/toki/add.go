// ABOUTME: Todo add command with git-aware context detection
// ABOUTME: Creates todos with metadata (priority, tags, notes, due date)

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:     "add <description>",
	Aliases: []string{"a"},
	Short:   "Add a new todo",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := strings.Join(args, " ")

		if len(description) < 3 {
			return fmt.Errorf("description must be at least 3 characters")
		}

		projectFlag, _ := cmd.Flags().GetString("project")
		projectID, err := getProjectID(projectFlag)
		if err != nil {
			return err
		}

		todo := models.NewTodo(*projectID, description)

		// Handle optional flags
		if priority, _ := cmd.Flags().GetString("priority"); priority != "" {
			priority = strings.ToLower(priority)
			if priority != "low" && priority != "medium" && priority != "high" {
				return fmt.Errorf("priority must be low, medium, or high")
			}
			todo.Priority = &priority
		}

		if notes, _ := cmd.Flags().GetString("notes"); notes != "" {
			todo.Notes = &notes
		}

		if dueStr, _ := cmd.Flags().GetString("due"); dueStr != "" {
			dueDate, err := time.Parse("2006-01-02", dueStr)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
			todo.DueDate = &dueDate
		}

		// Create todo
		if err := db.CreateTodo(dbConn, todo); err != nil {
			return fmt.Errorf("failed to create todo: %w", err)
		}

		// Handle tags
		if tagsStr, _ := cmd.Flags().GetString("tags"); tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					if err := db.AddTagToTodo(dbConn, todo.ID, tag); err != nil {
						return fmt.Errorf("failed to add tag: %w", err)
					}
				}
			}
		}

		color.Green("✓ Added todo")
		fmt.Printf("  %s %s\n", color.New(color.Faint).Sprint(todo.ID.String()[:6]), description)

		return nil
	},
}

func init() {
	addCmd.Flags().StringP("project", "p", "", "project name")
	addCmd.Flags().String("priority", "", "priority (low, medium, high)")
	addCmd.Flags().String("tags", "", "comma-separated tags")
	addCmd.Flags().String("notes", "", "additional notes")
	addCmd.Flags().String("due", "", "due date (YYYY-MM-DD)")

	rootCmd.AddCommand(addCmd)
}
