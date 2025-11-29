// ABOUTME: Project management commands
// ABOUTME: Handles add, list, set-path, and remove operations

package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/git"
	"github.com/harper/toki/internal/models"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"p"},
	Short:   "Manage projects",
}

var projectAddCmd = &cobra.Command{
	Use:     "add <name>",
	Aliases: []string{"a"},
	Short:   "Add a new project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		pathFlag, _ := cmd.Flags().GetString("path")
		var dirPath *string

		if pathFlag != "" {
			normalized, err := git.NormalizePath(pathFlag)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}
			dirPath = &normalized
		}

		project := models.NewProject(name, dirPath)

		if err := db.CreateProject(dbConn, project); err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		color.Green("✓ Created project '%s'", name)
		if dirPath != nil {
			fmt.Printf("  Path: %s\n", *dirPath)
		}

		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := db.ListProjects(dbConn)
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("No projects yet. Create one with 'toki project add <name>'")
			return nil
		}

		color.New(color.Bold).Println("PROJECTS")
		fmt.Println(color.New(color.Faint).Sprint("────────────────────────────────────────"))

		for _, p := range projects {
			fmt.Printf("%s\n", color.New(color.Bold, color.FgCyan).Sprint(p.Name))
			if p.DirectoryPath != nil {
				fmt.Printf("  %s\n", color.New(color.Faint).Sprint(*p.DirectoryPath))
			}
		}

		return nil
	},
}

func init() {
	projectAddCmd.Flags().String("path", "", "directory path to associate with project")

	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
	rootCmd.AddCommand(projectCmd)
}
