// ABOUTME: Project management commands
// ABOUTME: Handles add, list, set-path, and remove operations

package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/harper/toki/internal/git"
	"github.com/harper/toki/internal/storage"
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

		// Check if project already exists
		existingProject, err := GetStorage().GetProjectByName(name)
		if err == nil {
			color.Yellow("Project '%s' already exists (ID: %s)", name, existingProject.ID.String()[:8])
			if existingProject.DirectoryPath != "" {
				fmt.Printf("  Path: %s\n", existingProject.DirectoryPath)
			}
			return nil
		}

		pathFlag, _ := cmd.Flags().GetString("path")
		dirPath := ""

		if pathFlag != "" {
			normalized, err := git.NormalizePath(pathFlag)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}
			dirPath = normalized
		}

		project := &storage.Project{
			ID:            uuid.New(),
			Name:          name,
			DirectoryPath: dirPath,
			CreatedAt:     time.Now().UTC(),
		}

		if err := GetStorage().CreateProject(project); err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		color.Green("✓ Created project '%s'", name)
		if dirPath != "" {
			fmt.Printf("  Path: %s\n", dirPath)
		}

		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := GetStorage().ListProjects()
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("No projects yet. Create one with 'toki project add <name>'")
			return nil
		}

		_, _ = color.New(color.Bold).Println("PROJECTS")
		fmt.Println(color.New(color.Faint).Sprint("────────────────────────────────────────"))

		for _, p := range projects {
			fmt.Printf("%s\n", color.New(color.Bold, color.FgCyan).Sprint(p.Name))
			if p.DirectoryPath != "" {
				fmt.Printf("  %s\n", color.New(color.Faint).Sprint(p.DirectoryPath))
			}
		}

		return nil
	},
}

var projectSetPathCmd = &cobra.Command{
	Use:     "set-path <name> <path>",
	Aliases: []string{"sp"},
	Short:   "Set directory path for a project",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		pathArg := args[1]

		project, err := GetStorage().GetProjectByName(name)
		if err != nil {
			return fmt.Errorf("project not found: %w", err)
		}

		normalized, err := git.NormalizePath(pathArg)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		project.DirectoryPath = normalized

		if err := GetStorage().UpdateProject(project); err != nil {
			return fmt.Errorf("failed to update path: %w", err)
		}

		color.Green("✓ Updated path for '%s'", name)
		fmt.Printf("  Path: %s\n", normalized)

		return nil
	},
}

var projectRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "r"},
	Short:   "Remove a project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		project, err := GetStorage().GetProjectByName(name)
		if err != nil {
			return fmt.Errorf("project not found: %w", err)
		}

		// Count todos that will be deleted
		filter := &storage.TodoFilter{ProjectID: &project.ID}
		todos, err := GetStorage().ListTodos(filter)
		if err != nil {
			return fmt.Errorf("failed to list project todos: %w", err)
		}

		// Delete project (cascades to todos due to foreign key)
		if err := GetStorage().DeleteProject(project.ID); err != nil {
			return fmt.Errorf("failed to delete project: %w", err)
		}

		color.Yellow("✓ Removed project '%s' and %d todos", name, len(todos))

		return nil
	},
}

var projectCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove duplicate projects (keeps first occurrence)",
	Long: `Removes duplicate projects that have the same name.

This is useful when sync has caused multiple projects with the same name
to exist. The first project (by creation time) is kept, duplicates are removed.
Todos attached to duplicate projects are reassigned to the kept project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := GetStorage().ListProjects()
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		// Group projects by name
		projectsByName := make(map[string][]*storage.Project)
		for _, p := range projects {
			projectsByName[p.Name] = append(projectsByName[p.Name], p)
		}

		totalRemoved := 0
		totalTodosMoved := 0

		for name, projs := range projectsByName {
			if len(projs) <= 1 {
				continue
			}

			// Keep the first one (oldest by CreatedAt)
			keep := projs[0]
			for _, p := range projs[1:] {
				if p.CreatedAt.Before(keep.CreatedAt) {
					keep = p
				}
			}

			// Remove duplicates and move their todos
			for _, p := range projs {
				if p.ID == keep.ID {
					continue
				}

				// Move todos from duplicate to kept project
				filter := &storage.TodoFilter{ProjectID: &p.ID}
				todos, err := GetStorage().ListTodos(filter)
				if err == nil {
					for _, todo := range todos {
						todo.ProjectID = keep.ID
						_ = GetStorage().UpdateTodo(todo)
						totalTodosMoved++
					}
				}

				// Delete duplicate project
				if err := GetStorage().DeleteProject(p.ID); err != nil {
					fmt.Printf("Warning: failed to delete duplicate project %s: %v\n", p.ID, err)
				} else {
					totalRemoved++
				}
			}

			if len(projs) > 1 {
				fmt.Printf("  '%s': kept 1, removed %d duplicates\n", name, len(projs)-1)
			}
		}

		if totalRemoved == 0 {
			fmt.Println("No duplicate projects found.")
		} else {
			color.Green("✓ Removed %d duplicate projects, moved %d todos", totalRemoved, totalTodosMoved)
		}

		return nil
	},
}

func init() {
	projectAddCmd.Flags().String("path", "", "directory path to associate with project")

	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectSetPathCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectCleanupCmd)
	rootCmd.AddCommand(projectCmd)
}
