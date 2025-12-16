// ABOUTME: Todo remove command
// ABOUTME: Deletes todos by UUID prefix

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

var removeCmd = &cobra.Command{
	Use:     "remove <uuid-prefix>",
	Aliases: []string{"rm"},
	Short:   "Remove a todo",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		todo, err := db.GetTodoByPrefix(dbConn, prefix)
		if err != nil {
			return err
		}

		desc := todo.Description

		// Get project name for sync (before delete)
		project, err := db.GetProjectByID(dbConn, todo.ProjectID)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		// Queue sync BEFORE delete to preserve data
		if err := queueTodoSyncRemove(cmd.Context(), todo, project.Name); err != nil {
			color.Yellow("⚠ Sync: %v", err)
		}

		if err := db.DeleteTodo(dbConn, todo.ID); err != nil {
			return fmt.Errorf("failed to delete todo: %w", err)
		}

		color.Yellow("✓ Removed todo")
		fmt.Printf("  %s\n", desc)

		return nil
	},
}

func queueTodoSyncRemove(ctx context.Context, todo *models.Todo, projectName string) error {
	cfg, err := sync.LoadConfig()
	if err != nil || !cfg.IsConfigured() {
		return nil // Sync not configured, skip silently
	}
	syncer, err := sync.NewSyncer(cfg, dbConn)
	if err != nil {
		return err
	}
	defer func() { _ = syncer.Close() }()
	return syncer.QueueTodoChange(ctx, todo, projectName, vault.OpDelete)
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
