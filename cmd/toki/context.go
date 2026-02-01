// ABOUTME: Context detection for git-aware project lookup
// ABOUTME: Determines current project from git repo or prompts user

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/git"
	"github.com/harper/toki/internal/storage"
)

// detectProjectContext attempts to find project from current directory.
func detectProjectContext() (*uuid.UUID, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Try to find git root
	gitRoot, err := git.FindGitRoot(cwd)
	if err != nil {
		// Not in a git repo
		return nil, nil //nolint:nilerr,nilnil // Intentional: not being in a git repo is not an error
	}

	// Look up project by path
	project, err := GetStorage().GetProjectByPath(gitRoot)
	if err == nil {
		return &project.ID, nil
	}

	// Project not found - offer to create
	projectName := filepath.Base(gitRoot)
	fmt.Printf("Git repository detected: %s\n", gitRoot)
	fmt.Printf("Create project '%s'? [Y/n]: ", projectName)

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" || response == "y" || response == "yes" {
		project := &storage.Project{
			ID:            uuid.New(),
			Name:          projectName,
			DirectoryPath: gitRoot,
			CreatedAt:     time.Now().UTC(),
		}
		if err := GetStorage().CreateProject(project); err != nil {
			return nil, fmt.Errorf("failed to create project: %w", err)
		}
		fmt.Printf("✓ Created project '%s'\n", projectName)
		return &project.ID, nil
	}

	return nil, nil //nolint:nilnil // Intentional: user declined project creation, not an error
}

// getProjectID gets project ID from flag or context.
func getProjectID(projectFlag string) (*uuid.UUID, error) {
	if projectFlag != "" {
		project, err := GetStorage().GetProjectByName(projectFlag)
		if err != nil {
			return nil, fmt.Errorf("project '%s' not found", projectFlag)
		}
		return &project.ID, nil
	}

	// Try context detection
	projectID, err := detectProjectContext()
	if err != nil {
		return nil, err
	}

	if projectID != nil {
		return projectID, nil
	}

	// No project context - use or create "default" project
	project, err := getOrCreateDefaultProject()
	if err != nil {
		return nil, fmt.Errorf("failed to create default project: %w", err)
	}

	return &project.ID, nil
}

// getOrCreateDefaultProject creates or returns the default project.
func getOrCreateDefaultProject() (*storage.Project, error) {
	project, err := GetStorage().GetProjectByName("default")
	if err == nil {
		return project, nil
	}

	project = &storage.Project{
		ID:        uuid.New(),
		Name:      "default",
		CreatedAt: time.Now().UTC(),
	}

	if err := GetStorage().CreateProject(project); err != nil {
		return nil, err
	}

	return project, nil
}
