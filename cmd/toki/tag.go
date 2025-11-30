// ABOUTME: Tag management commands
// ABOUTME: Add/remove tags from todos and list all tags

package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
}

var tagAddCmd = &cobra.Command{
	Use:     "add <uuid-prefix> <tag>",
	Aliases: []string{"a"},
	Short:   "Add a tag to a todo",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		tagName := strings.ToLower(args[1])

		todo, err := db.GetTodoByPrefix(dbConn, prefix)
		if err != nil {
			return err
		}

		if err := db.AddTagToTodo(dbConn, todo.ID, tagName); err != nil {
			return fmt.Errorf("failed to add tag: %w", err)
		}

		color.Green("✓ Added tag '%s'", tagName)
		fmt.Printf("  %s\n", todo.Description)

		return nil
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:     "remove <uuid-prefix> <tag>",
	Aliases: []string{"rm", "r"},
	Short:   "Remove a tag from a todo",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		tagName := strings.ToLower(args[1])

		todo, err := db.GetTodoByPrefix(dbConn, prefix)
		if err != nil {
			return err
		}

		if err := db.RemoveTagFromTodo(dbConn, todo.ID, tagName); err != nil {
			return fmt.Errorf("failed to remove tag: %w", err)
		}

		color.Yellow("✓ Removed tag '%s'", tagName)
		fmt.Printf("  %s\n", todo.Description)

		return nil
	},
}

var tagsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		tags, err := db.ListAllTags(dbConn)
		if err != nil {
			return fmt.Errorf("failed to list tags: %w", err)
		}

		if len(tags) == 0 {
			fmt.Println("No tags yet.")
			return nil
		}

		_, _ = color.New(color.Bold).Println("TAGS")
		for _, tag := range tags {
			fmt.Printf("  • %s\n", tag.Name)
		}

		return nil
	},
}

func init() {
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRemoveCmd)
	tagCmd.AddCommand(tagsListCmd)
	rootCmd.AddCommand(tagCmd)
}
