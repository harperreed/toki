// ABOUTME: Tests for SQLite storage implementation
// ABOUTME: Covers CRUD operations for projects, todos, and tags

package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupTestDB(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "toki-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	storage, err := NewSQLiteStorage(dbPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("failed to create storage: %v", err)
	}

	cleanup := func() {
		_ = storage.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return storage, cleanup
}

func TestNewSQLiteStorage(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	if storage == nil {
		t.Fatal("expected non-nil storage")
	}
}

func TestCreateAndGetProject(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:            uuid.New(),
		Name:          "test-project",
		DirectoryPath: "/path/to/project",
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
	}

	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	got, err := storage.GetProject(project.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if got.ID != project.ID {
		t.Errorf("ID mismatch: got %v, want %v", got.ID, project.ID)
	}
	if got.Name != project.Name {
		t.Errorf("Name mismatch: got %v, want %v", got.Name, project.Name)
	}
	if got.DirectoryPath != project.DirectoryPath {
		t.Errorf("DirectoryPath mismatch: got %v, want %v", got.DirectoryPath, project.DirectoryPath)
	}
}

func TestGetProjectByName(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "unique-project",
		CreatedAt: time.Now().UTC(),
	}

	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	got, err := storage.GetProjectByName("unique-project")
	if err != nil {
		t.Fatalf("failed to get project by name: %v", err)
	}

	if got.ID != project.ID {
		t.Errorf("ID mismatch: got %v, want %v", got.ID, project.ID)
	}
}

func TestGetProjectByPath(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:            uuid.New(),
		Name:          "path-project",
		DirectoryPath: "/specific/path",
		CreatedAt:     time.Now().UTC(),
	}

	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	got, err := storage.GetProjectByPath("/specific/path")
	if err != nil {
		t.Fatalf("failed to get project by path: %v", err)
	}

	if got.ID != project.ID {
		t.Errorf("ID mismatch: got %v, want %v", got.ID, project.ID)
	}
}

func TestListProjects(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	projects := []*Project{
		{ID: uuid.New(), Name: "alpha", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), Name: "beta", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), Name: "gamma", CreatedAt: time.Now().UTC()},
	}

	for _, p := range projects {
		if err := storage.CreateProject(p); err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
	}

	list, err := storage.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 projects, got %d", len(list))
	}

	// Should be sorted by name
	if list[0].Name != "alpha" || list[1].Name != "beta" || list[2].Name != "gamma" {
		t.Error("projects not sorted by name")
	}
}

func TestUpdateProject(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "original-name",
		CreatedAt: time.Now().UTC(),
	}

	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Name = "updated-name"
	project.DirectoryPath = "/new/path"

	if err := storage.UpdateProject(project); err != nil {
		t.Fatalf("failed to update project: %v", err)
	}

	got, err := storage.GetProject(project.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if got.Name != "updated-name" {
		t.Errorf("Name not updated: got %v, want updated-name", got.Name)
	}
	if got.DirectoryPath != "/new/path" {
		t.Errorf("DirectoryPath not updated: got %v, want /new/path", got.DirectoryPath)
	}
}

func TestDeleteProject(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "to-delete",
		CreatedAt: time.Now().UTC(),
	}

	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if err := storage.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	_, err := storage.GetProject(project.ID)
	if err == nil {
		t.Error("expected error getting deleted project")
	}
}

func TestCreateAndGetTodo(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	dueDate := now.Add(24 * time.Hour)
	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Test todo item",
		Done:        false,
		Priority:    "high",
		Notes:       "Some notes",
		Tags:        []string{"tag1", "tag2"},
		CreatedAt:   now,
		UpdatedAt:   now,
		DueDate:     &dueDate,
	}

	if err := storage.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	got, err := storage.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if got.ID != todo.ID {
		t.Errorf("ID mismatch: got %v, want %v", got.ID, todo.ID)
	}
	if got.Description != todo.Description {
		t.Errorf("Description mismatch: got %v, want %v", got.Description, todo.Description)
	}
	if got.Priority != todo.Priority {
		t.Errorf("Priority mismatch: got %v, want %v", got.Priority, todo.Priority)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags count mismatch: got %d, want 2", len(got.Tags))
	}
}

func TestGetTodoByPrefix(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todoID := uuid.New()
	todo := &Todo{
		ID:          todoID,
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Find me by prefix",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := storage.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Use first 6 characters as prefix
	prefix := todoID.String()[:6]
	got, err := storage.GetTodoByPrefix(prefix)
	if err != nil {
		t.Fatalf("failed to get todo by prefix: %v", err)
	}

	if got.ID != todoID {
		t.Errorf("ID mismatch: got %v, want %v", got.ID, todoID)
	}
}

func TestListTodosWithFilters(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todos := []*Todo{
		{ID: uuid.New(), ProjectID: project.ID, ProjectName: project.Name, Description: "High priority", Priority: "high", Done: false, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), ProjectID: project.ID, ProjectName: project.Name, Description: "Low priority", Priority: "low", Done: false, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), ProjectID: project.ID, ProjectName: project.Name, Description: "Done item", Priority: "medium", Done: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}

	for _, todo := range todos {
		if err := storage.CreateTodo(todo); err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}
	}

	// Test filter by done
	done := false
	list, err := storage.ListTodos(&TodoFilter{Done: &done})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 pending todos, got %d", len(list))
	}

	// Test filter by priority
	priority := "high"
	list, err = storage.ListTodos(&TodoFilter{Priority: &priority})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 high priority todo, got %d", len(list))
	}

	// Test filter by project
	list, err = storage.ListTodos(&TodoFilter{ProjectID: &project.ID})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 todos in project, got %d", len(list))
	}
}

func TestMarkTodoDone(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Mark me done",
		Done:        false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := storage.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := storage.MarkTodoDone(todo.ID, true); err != nil {
		t.Fatalf("failed to mark todo done: %v", err)
	}

	got, err := storage.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if !got.Done {
		t.Error("todo should be marked done")
	}
	if got.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}

	// Now mark undone
	if err := storage.MarkTodoDone(todo.ID, false); err != nil {
		t.Fatalf("failed to mark todo undone: %v", err)
	}

	got, err = storage.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if got.Done {
		t.Error("todo should be marked undone")
	}
	if got.CompletedAt != nil {
		t.Error("CompletedAt should be nil")
	}
}

func TestDeleteTodo(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Delete me",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := storage.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := storage.DeleteTodo(todo.ID); err != nil {
		t.Fatalf("failed to delete todo: %v", err)
	}

	_, err := storage.GetTodo(todo.ID)
	if err == nil {
		t.Error("expected error getting deleted todo")
	}
}

func TestTagOperations(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	// Create tag
	tag, err := storage.GetOrCreateTag("urgent")
	if err != nil {
		t.Fatalf("failed to get or create tag: %v", err)
	}
	if tag.Name != "urgent" {
		t.Errorf("tag name mismatch: got %v, want urgent", tag.Name)
	}

	// Get same tag again (should not error)
	tag2, err := storage.GetOrCreateTag("urgent")
	if err != nil {
		t.Fatalf("failed to get existing tag: %v", err)
	}
	if tag2.ID != tag.ID {
		t.Error("should return same tag")
	}

	// List tags
	tags, err := storage.ListTags()
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}
	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(tags))
	}

	// Delete tag
	if err := storage.DeleteTag("urgent"); err != nil {
		t.Fatalf("failed to delete tag: %v", err)
	}

	tags, err = storage.ListTags()
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

func TestTodoTagAssociation(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo with tags",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := storage.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Add tags
	if err := storage.AddTagToTodo(todo.ID, "tag1"); err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}
	if err := storage.AddTagToTodo(todo.ID, "tag2"); err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}

	// Get tags for todo
	tags, err := storage.GetTagsForTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	// Remove tag
	if err := storage.RemoveTagFromTodo(todo.ID, "tag1"); err != nil {
		t.Fatalf("failed to remove tag: %v", err)
	}

	tags, err = storage.GetTagsForTodo(todo.ID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}
	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(tags))
	}
	if tags[0] != "tag2" {
		t.Errorf("expected tag2, got %v", tags[0])
	}
}

func TestFilterByTag(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo1 := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo with urgent tag",
		Tags:        []string{"urgent"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := storage.CreateTodo(todo1); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	todo2 := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo without urgent tag",
		Tags:        []string{"normal"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := storage.CreateTodo(todo2); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Filter by tag
	tagFilter := "urgent"
	list, err := storage.ListTodos(&TodoFilter{Tag: &tagFilter})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 todo with urgent tag, got %d", len(list))
	}
}

func TestOverdueFilter(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	now := time.Now().UTC()
	past := now.Add(-48 * time.Hour)
	future := now.Add(48 * time.Hour)

	todo1 := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Overdue todo",
		Done:        false,
		DueDate:     &past,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := storage.CreateTodo(todo1); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	todo2 := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Future todo",
		Done:        false,
		DueDate:     &future,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := storage.CreateTodo(todo2); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Filter overdue
	overdue := true
	list, err := storage.ListTodos(&TodoFilter{Overdue: &overdue})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 overdue todo, got %d", len(list))
	}
	if list[0].Description != "Overdue todo" {
		t.Errorf("expected overdue todo, got %v", list[0].Description)
	}
}

func TestCascadeDeleteProject(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	project := &Project{
		ID:        uuid.New(),
		Name:      "to-delete",
		CreatedAt: time.Now().UTC(),
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Will be cascade deleted",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := storage.CreateTodo(todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Delete project - should cascade to todos
	if err := storage.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	// Todo should also be deleted
	_, err := storage.GetTodo(todo.ID)
	if err == nil {
		t.Error("expected error getting todo after project cascade delete")
	}
}

func TestIntegrityCheck(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	ok, err := storage.IntegrityCheck()
	if err != nil {
		t.Fatalf("integrity check failed: %v", err)
	}
	if !ok {
		t.Error("integrity check should return true for new database")
	}
}

func TestVacuum(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	// Just test that vacuum doesn't error
	if err := storage.Vacuum(); err != nil {
		t.Errorf("vacuum failed: %v", err)
	}
}
