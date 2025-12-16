// ABOUTME: Todo done and undone commands
// ABOUTME: Marks todos as complete or incomplete using UUID prefixes

package main

import (
	"context"
	"fmt"

	"github.com/harperreed/sweet/vault"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
	"github.com/harper/toki/internal/sync"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:     "done <uuid-prefix> [uuid-prefix...]",
	Aliases: []string{"d"},
	Short:   "Mark todos as done",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, prefix := range args {
			todo, err := db.GetTodoByPrefix(dbConn, prefix)
			if err != nil {
				return err
			}

			todo.MarkDone()

			if err := db.UpdateTodo(dbConn, todo); err != nil {
				return fmt.Errorf("failed to update todo: %w", err)
			}

			// Get project name for sync
			project, err := db.GetProjectByID(dbConn, todo.ProjectID)
			if err != nil {
				return fmt.Errorf("failed to get project: %w", err)
			}

			// Queue sync after successful update
			if err := queueTodoSyncDone(cmd.Context(), todo, project.Name); err != nil {
				color.Yellow("⚠ Sync: %v", err)
			}

			color.Green("✓ Marked todo as done")
			fmt.Printf("  %s %s\n", prefix, todo.Description)
		}

		return nil
	},
}

var undoneCmd = &cobra.Command{
	Use:     "undone <uuid-prefix> [uuid-prefix...]",
	Aliases: []string{"ud"},
	Short:   "Mark todos as not done",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, prefix := range args {
			todo, err := db.GetTodoByPrefix(dbConn, prefix)
			if err != nil {
				return err
			}

			todo.MarkUndone()

			if err := db.UpdateTodo(dbConn, todo); err != nil {
				return fmt.Errorf("failed to update todo: %w", err)
			}

			// Get project name for sync
			project, err := db.GetProjectByID(dbConn, todo.ProjectID)
			if err != nil {
				return fmt.Errorf("failed to get project: %w", err)
			}

			// Queue sync after successful update
			if err := queueTodoSyncDone(cmd.Context(), todo, project.Name); err != nil {
				color.Yellow("⚠ Sync: %v", err)
			}

			color.Yellow("✓ Marked todo as not done")
			fmt.Printf("  %s %s\n", prefix, todo.Description)
		}

		return nil
	},
}

func queueTodoSyncDone(ctx context.Context, todo *models.Todo, projectName string) error {
	cfg, err := sync.LoadConfig()
	if err != nil || !cfg.IsConfigured() {
		return nil // Sync not configured, skip silently
	}
	syncer, err := sync.NewSyncer(cfg, dbConn)
	if err != nil {
		return err
	}
	defer func() { _ = syncer.Close() }()
	return syncer.QueueTodoChange(ctx, todo, projectName, vault.OpUpsert)
}

func init() {
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(undoneCmd)
}
