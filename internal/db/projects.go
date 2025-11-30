// ABOUTME: Project database operations (CRUD)
// ABOUTME: Handles project creation, retrieval, updates, and deletion

package db

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

// CreateProject inserts a new project into the database.
func CreateProject(db *sql.DB, project *models.Project) error {
	query := `INSERT INTO projects (id, name, directory_path, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, project.ID.String(), project.Name, project.DirectoryPath, project.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

// GetProjectByID retrieves a project by its UUID.
func GetProjectByID(db *sql.DB, id uuid.UUID) (*models.Project, error) {
	query := `SELECT id, name, directory_path, created_at FROM projects WHERE id = ?`

	var project models.Project
	var idStr string

	err := db.QueryRow(query, id.String()).Scan(&idStr, &project.Name, &project.DirectoryPath, &project.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	return &project, nil
}

// GetProjectByName retrieves a project by its name.
func GetProjectByName(db *sql.DB, name string) (*models.Project, error) {
	query := `SELECT id, name, directory_path, created_at FROM projects WHERE name = ?`

	var project models.Project
	var idStr string

	err := db.QueryRow(query, name).Scan(&idStr, &project.Name, &project.DirectoryPath, &project.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	return &project, nil
}

// GetProjectByPath retrieves a project by its directory path.
func GetProjectByPath(db *sql.DB, path string) (*models.Project, error) {
	query := `SELECT id, name, directory_path, created_at FROM projects WHERE directory_path = ?`

	var project models.Project
	var idStr string

	err := db.QueryRow(query, path).Scan(&idStr, &project.Name, &project.DirectoryPath, &project.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID, _ = uuid.Parse(idStr)
	return &project, nil
}

// ListProjects returns all projects.
func ListProjects(db *sql.DB) ([]*models.Project, error) {
	query := `SELECT id, name, directory_path, created_at FROM projects ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var projects []*models.Project
	for rows.Next() {
		var project models.Project
		var idStr string

		if err := rows.Scan(&idStr, &project.Name, &project.DirectoryPath, &project.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		project.ID, _ = uuid.Parse(idStr)
		projects = append(projects, &project)
	}

	return projects, nil
}

// UpdateProjectPath updates the directory path for a project.
func UpdateProjectPath(db *sql.DB, id uuid.UUID, path *string) error {
	query := `UPDATE projects SET directory_path = ? WHERE id = ?`
	_, err := db.Exec(query, path, id.String())
	if err != nil {
		return fmt.Errorf("failed to update project path: %w", err)
	}
	return nil
}

// DeleteProject deletes a project (cascades to todos).
func DeleteProject(db *sql.DB, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = ?`
	_, err := db.Exec(query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}
