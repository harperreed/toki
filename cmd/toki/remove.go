// ABOUTME: Todo remove command
// ABOUTME: Deletes todos by UUID prefix

package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
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

		if err := db.DeleteTodo(dbConn, todo.ID); err != nil {
			return fmt.Errorf("failed to delete todo: %w", err)
		}

		color.Yellow("✓ Removed todo")
		fmt.Printf("  %s\n", desc)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
