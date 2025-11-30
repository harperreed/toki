// ABOUTME: Tests for project database operations
// ABOUTME: Covers CRUD operations and path-based lookups

package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/harper/toki/internal/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	return db
}

func TestCreateProject(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	path := "/home/user/project"
	project := models.NewProject("test-project", &path)

	if err := CreateProject(db, project); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Verify it was created
	retrieved, err := GetProjectByID(db, project.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve project: %v", err)
	}

	if retrieved.Name != project.Name {
		t.Errorf("Expected name %s, got %s", project.Name, retrieved.Name)
	}
}

func TestGetProjectByName(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	project := models.NewProject("findme", nil)
	if err := CreateProject(db, project); err != nil {
		t.Fatal(err)
	}

	found, err := GetProjectByName(db, "findme")
	if err != nil {
		t.Fatalf("Failed to get project: %v", err)
	}

	if found.ID != project.ID {
		t.Error("Retrieved wrong project")
	}
}

func TestGetProjectByPath(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	path := "/home/user/myproject"
	project := models.NewProject("myproject", &path)
	if err := CreateProject(db, project); err != nil {
		t.Fatal(err)
	}

	found, err := GetProjectByPath(db, path)
	if err != nil {
		t.Fatalf("Failed to get project by path: %v", err)
	}

	if found.ID != project.ID {
		t.Error("Retrieved wrong project")
	}
}

func TestListProjects(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	if err := CreateProject(db, models.NewProject("project1", nil)); err != nil {
		t.Fatal(err)
	}
	if err := CreateProject(db, models.NewProject("project2", nil)); err != nil {
		t.Fatal(err)
	}

	projects, err := ListProjects(db)
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
}

func TestDeleteProject(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	project := models.NewProject("todelete", nil)
	if err := CreateProject(db, project); err != nil {
		t.Fatal(err)
	}

	err := DeleteProject(db, project.ID)
	if err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	_, err = GetProjectByID(db, project.ID)
	if err == nil {
		t.Error("Project should not exist after deletion")
	}
}
