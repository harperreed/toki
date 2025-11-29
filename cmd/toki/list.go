// ABOUTME: Todo list command with filtering and formatting
// ABOUTME: Supports project, tag, status, and priority filters

package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
	"github.com/harper/toki/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List todos",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse filters
		var projectID *uuid.UUID
		var done *bool
		var priority *string

		projectFlag, _ := cmd.Flags().GetString("project")
		if projectFlag != "" {
			id, err := getProjectID(projectFlag)
			if err != nil {
				return err
			}
			projectID = id
		} else {
			// Try context detection
			id, _ := detectProjectContext()
			if id != nil {
				projectID = id
			}
		}

		if cmd.Flags().Changed("done") {
			doneVal := true
			done = &doneVal
		} else if cmd.Flags().Changed("pending") {
			pendingVal := false
			done = &pendingVal
		}

		if priorityFlag, _ := cmd.Flags().GetString("priority"); priorityFlag != "" {
			priority = &priorityFlag
		}

		var tag *string
		if tagFlag, _ := cmd.Flags().GetString("tag"); tagFlag != "" {
			tag = &tagFlag
		}

		// Get todos
		todos, err := db.ListTodos(dbConn, projectID, done, priority, tag)
		if err != nil {
			return fmt.Errorf("failed to list todos: %w", err)
		}

		if len(todos) == 0 {
			fmt.Println("No todos found. Add one with 'toki add <description>'")
			return nil
		}

		// Group by project
		projectTodos := make(map[uuid.UUID][]*struct {
			todo *models.Todo
			tags []*models.Tag
		})

		for _, todo := range todos {
			tags, _ := db.GetTodoTags(dbConn, todo.ID)
			if projectTodos[todo.ProjectID] == nil {
				projectTodos[todo.ProjectID] = []*struct {
					todo *models.Todo
					tags []*models.Tag
				}{}
			}
			projectTodos[todo.ProjectID] = append(projectTodos[todo.ProjectID], &struct {
				todo *models.Todo
				tags []*models.Tag
			}{todo, tags})
		}

		// Display grouped by project
		totalCount := 0
		for projID, items := range projectTodos {
			project, err := db.GetProjectByID(dbConn, projID)
			if err != nil {
				continue
			}

			fmt.Println(ui.FormatProjectHeader(project))
			fmt.Println(ui.FormatSeparator())

			for _, item := range items {
				fmt.Print(ui.FormatTodo(item.todo, item.tags))
				totalCount++
			}

			fmt.Println()
		}

		fmt.Println(ui.FormatSeparator())
		statusText := "pending"
		if done != nil && *done {
			statusText = "completed"
		}
		fmt.Printf("%d %s todo(s) across %d project(s)\n", totalCount, statusText, len(projectTodos))

		return nil
	},
}

func init() {
	listCmd.Flags().StringP("project", "p", "", "filter by project")
	listCmd.Flags().StringP("tag", "t", "", "filter by tag")
	listCmd.Flags().Bool("done", false, "show completed todos")
	listCmd.Flags().Bool("pending", false, "show pending todos only")
	listCmd.Flags().String("priority", "", "filter by priority")

	rootCmd.AddCommand(listCmd)
}
