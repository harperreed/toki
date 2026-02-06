// ABOUTME: Data migration between toki storage backends
// ABOUTME: Copies projects, todos, and tags from source to destination

package storage

import (
	"fmt"
	"os"
)

// MigrateSummary holds counts of migrated entities.
type MigrateSummary struct {
	Projects int
	Todos    int
	Tags     int
}

// MigrateData copies all data from src to dst storage.
// It iterates through projects and their todos in order, creating each entity
// in the destination. The destination should be empty before calling this function.
func MigrateData(src, dst Storage) (*MigrateSummary, error) {
	summary := &MigrateSummary{}

	// List all projects
	projects, err := src.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("list source projects: %w", err)
	}

	for _, project := range projects {
		if err := dst.CreateProject(project); err != nil {
			return nil, fmt.Errorf("create project %q: %w", project.Name, err)
		}
		summary.Projects++

		if err := migrateProjectTodos(src, dst, project, summary); err != nil {
			return nil, err
		}
	}

	return summary, nil
}

// migrateProjectTodos copies all todos for a single project.
func migrateProjectTodos(src, dst Storage, project *Project, summary *MigrateSummary) error {
	todos, err := src.ListTodos(&TodoFilter{ProjectID: &project.ID})
	if err != nil {
		return fmt.Errorf("list todos for project %q: %w", project.Name, err)
	}

	for _, todo := range todos {
		if err := dst.CreateTodo(todo); err != nil {
			return fmt.Errorf("create todo %q in project %q: %w", todo.Description, project.Name, err)
		}
		summary.Todos++

		// Tags are migrated as part of the todo (they are in todo.Tags)
		summary.Tags += len(todo.Tags)
	}

	return nil
}

// IsDirNonEmpty checks whether a directory exists and contains any files or subdirectories.
// Returns false if the directory does not exist or is empty.
func IsDirNonEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read directory %q: %w", path, err)
	}
	return len(entries) > 0, nil
}
