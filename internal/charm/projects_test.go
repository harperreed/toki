// ABOUTME: Tests for project CRUD operations
// ABOUTME: Verifies create, read, update, delete, and list for projects

package charm

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupTestClient(t *testing.T) (*Client, func()) {
	t.Helper()

	tmpDir := t.TempDir() // automatically cleaned up

	// Use t.Setenv for test-scoped env var (avoids race conditions in parallel tests)
	t.Setenv("CHARM_DATA_DIR", tmpDir)

	client, err := NewClient("toki-test-" + uuid.New().String()[:8])
	if err != nil {
		t.Skipf("skipping test - charm KV not available: %v", err)
	}

	cleanup := func() {
		_ = client.Close()
	}

	return client, cleanup
}

func TestCreateProject(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:            uuid.New(),
		Name:          "test-project",
		DirectoryPath: "/path/to/project",
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
	}

	err := client.CreateProject(project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Verify it was stored
	retrieved, err := client.GetProject(project.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if retrieved.Name != project.Name {
		t.Errorf("Name mismatch: got %v, want %v", retrieved.Name, project.Name)
	}
}

func TestGetProjectByName(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "unique-name",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	retrieved, err := client.GetProjectByName("unique-name")
	if err != nil {
		t.Fatalf("failed to get project by name: %v", err)
	}

	if retrieved.ID != project.ID {
		t.Errorf("ID mismatch")
	}
}

func TestGetProjectByPath(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:            uuid.New(),
		Name:          "path-project",
		DirectoryPath: "/unique/path",
		CreatedAt:     time.Now().UTC(),
	}

	if err := client.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	retrieved, err := client.GetProjectByPath("/unique/path")
	if err != nil {
		t.Fatalf("failed to get project by path: %v", err)
	}

	if retrieved.ID != project.ID {
		t.Errorf("ID mismatch")
	}
}

func TestListProjects(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	// Create multiple projects
	for i := 0; i < 3; i++ {
		project := &Project{
			ID:        uuid.New(),
			Name:      fmt.Sprintf("project-%d", i),
			CreatedAt: time.Now().UTC(),
		}
		if err := client.CreateProject(project); err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
	}

	projects, err := client.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("expected 3 projects, got %d", len(projects))
	}
}

func TestUpdateProject(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "original-name",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.DirectoryPath = "/new/path"
	if err := client.UpdateProject(project); err != nil {
		t.Fatalf("failed to update project: %v", err)
	}

	retrieved, err := client.GetProject(project.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if retrieved.DirectoryPath != "/new/path" {
		t.Errorf("DirectoryPath not updated")
	}
}

func TestDeleteProject(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "to-delete",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if err := client.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	_, err := client.GetProject(project.ID)
	if err == nil {
		t.Error("expected error getting deleted project")
	}
}
