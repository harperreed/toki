// ABOUTME: Tests for MarkdownStore file-based storage backend
// ABOUTME: Covers CRUD for projects, todos, tags, filtering, and edge cases

package storage

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newTestMarkdownStore creates a MarkdownStore in a temporary directory for testing.
func newTestMarkdownStore(t *testing.T) *MarkdownStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := NewMarkdownStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test markdown store: %v", err)
	}
	return store
}

func TestNewMarkdownStore(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "toki-data")

	store, err := NewMarkdownStore(dataDir)
	if err != nil {
		t.Fatalf("NewMarkdownStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	if store == nil {
		t.Fatal("NewMarkdownStore returned nil")
	}

	// Verify data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Fatal("Data directory was not created")
	}
}

func TestMarkdownClose(t *testing.T) {
	store := newTestMarkdownStore(t)
	err := store.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// --- Project CRUD ---

func TestMarkdownProjectCreateAndRead(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:            uuid.New(),
		Name:          "test-project",
		DirectoryPath: "/path/to/project",
		CreatedAt:     time.Now().UTC().Truncate(time.Millisecond),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	got, err := store.GetProject(project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if got.Name != project.Name {
		t.Errorf("expected name %q, got %q", project.Name, got.Name)
	}
	if got.DirectoryPath != project.DirectoryPath {
		t.Errorf("expected path %q, got %q", project.DirectoryPath, got.DirectoryPath)
	}

	got, err = store.GetProjectByName("test-project")
	if err != nil {
		t.Fatalf("GetProjectByName failed: %v", err)
	}
	if got.ID != project.ID {
		t.Errorf("expected ID %s, got %s", project.ID, got.ID)
	}

	got, err = store.GetProjectByPath("/path/to/project")
	if err != nil {
		t.Fatalf("GetProjectByPath failed: %v", err)
	}
	if got.ID != project.ID {
		t.Errorf("expected ID %s, got %s", project.ID, got.ID)
	}
}

func TestMarkdownProjectUpdateAndDelete(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:            uuid.New(),
		Name:          "test-project",
		DirectoryPath: "/path/to/project",
		CreatedAt:     time.Now().UTC().Truncate(time.Millisecond),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	project.Name = "updated-project"
	project.DirectoryPath = "/new/path"
	if err := store.UpdateProject(project); err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}
	got, _ := store.GetProject(project.ID)
	if got.Name != "updated-project" {
		t.Errorf("expected updated name, got %q", got.Name)
	}
	if got.DirectoryPath != "/new/path" {
		t.Errorf("expected updated path, got %q", got.DirectoryPath)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}

	if err := store.DeleteProject(project.ID); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}
	_, err = store.GetProject(project.ID)
	if err == nil {
		t.Error("expected error getting deleted project")
	}
}

func TestMarkdownProjectDuplicateName(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project1 := &Project{
		ID:        uuid.New(),
		Name:      "duplicate",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project1); err != nil {
		t.Fatalf("CreateProject (first) failed: %v", err)
	}

	project2 := &Project{
		ID:        uuid.New(),
		Name:      "duplicate",
		CreatedAt: time.Now().UTC(),
	}
	err := store.CreateProject(project2)
	if err == nil {
		t.Error("expected error creating project with duplicate name")
	}
}

func TestMarkdownListProjectsSorted(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	names := []string{"gamma", "alpha", "beta"}
	for _, name := range names {
		project := &Project{
			ID:        uuid.New(),
			Name:      name,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.CreateProject(project); err != nil {
			t.Fatalf("CreateProject(%q) failed: %v", name, err)
		}
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}
	if projects[0].Name != "alpha" || projects[1].Name != "beta" || projects[2].Name != "gamma" {
		t.Error("projects not sorted by name")
	}
}

func TestMarkdownProjectDirectoryCreated(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "my-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	projDir := filepath.Join(store.dataDir, "my-project")
	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		t.Error("project directory should be created")
	}
}

func TestMarkdownProjectDirectoryRemovedOnDelete(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "to-delete",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	projDir := filepath.Join(store.dataDir, "to-delete")
	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		t.Fatal("project directory should exist before delete")
	}

	if err := store.DeleteProject(project.ID); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	if _, err := os.Stat(projDir); !os.IsNotExist(err) {
		t.Error("project directory should be removed on delete")
	}
}

func TestMarkdownGetProjectNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	_, err := store.GetProject(uuid.New())
	if err == nil {
		t.Error("expected error getting non-existent project")
	}
}

func TestMarkdownGetProjectByNameNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	_, err := store.GetProjectByName("non-existent")
	if err == nil {
		t.Error("expected error getting non-existent project by name")
	}
}

func TestMarkdownGetProjectByPathNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	_, err := store.GetProjectByPath("/non/existent/path")
	if err == nil {
		t.Error("expected error getting non-existent project by path")
	}
}

func TestMarkdownUpdateProjectNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "non-existent",
		CreatedAt: time.Now().UTC(),
	}
	err := store.UpdateProject(project)
	if err == nil {
		t.Error("expected error updating non-existent project")
	}
}

func TestMarkdownDeleteProjectNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	err := store.DeleteProject(uuid.New())
	if err == nil {
		t.Error("expected error deleting non-existent project")
	}
}

// --- Todo CRUD ---

// createTestProjectAndTodo sets up a project and a todo for testing.
func createTestProjectAndTodo(t *testing.T, store *MarkdownStore) (*Project, *Todo) {
	t.Helper()
	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	dueDate := now.Add(24 * time.Hour)
	todo := &Todo{
		ID: uuid.New(), ProjectID: project.ID, ProjectName: project.Name,
		Description: "Test todo item", Done: false, Priority: "high",
		Notes: "Some notes", Tags: []string{"tag1", "tag2"},
		CreatedAt: now, UpdatedAt: now, DueDate: &dueDate,
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}
	return project, todo
}

func TestMarkdownTodoCreateAndRead(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project, todo := createTestProjectAndTodo(t, store)

	got, err := store.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("GetTodo failed: %v", err)
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
	if got.Notes != todo.Notes {
		t.Errorf("Notes mismatch: got %v, want %v", got.Notes, todo.Notes)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags count mismatch: got %d, want 2", len(got.Tags))
	}
	if got.ProjectName != project.Name {
		t.Errorf("ProjectName mismatch: got %v, want %v", got.ProjectName, project.Name)
	}
	if got.DueDate == nil {
		t.Error("DueDate should be set")
	}
}

func TestMarkdownTodoUpdateAndDelete(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	_, todo := createTestProjectAndTodo(t, store)

	todo.Description = "Updated"
	todo.Priority = "low"
	todo.Notes = "Updated notes"
	if err := store.UpdateTodo(todo); err != nil {
		t.Fatalf("UpdateTodo failed: %v", err)
	}
	got, _ := store.GetTodo(todo.ID)
	if got.Description != "Updated" {
		t.Errorf("Description not updated: got %v", got.Description)
	}
	if got.Priority != "low" {
		t.Errorf("Priority not updated: got %v", got.Priority)
	}
	if got.Notes != "Updated notes" {
		t.Errorf("Notes not updated: got %v", got.Notes)
	}

	if err := store.DeleteTodo(todo.ID); err != nil {
		t.Fatalf("DeleteTodo failed: %v", err)
	}
	_, err := store.GetTodo(todo.ID)
	if err == nil {
		t.Error("expected error getting deleted todo")
	}
}

func TestMarkdownGetTodoByPrefix(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
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
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	prefix := todoID.String()[:6]
	got, err := store.GetTodoByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetTodoByPrefix failed: %v", err)
	}
	if got.ID != todoID {
		t.Errorf("ID mismatch: got %v, want %v", got.ID, todoID)
	}
}

func TestMarkdownGetTodoByPrefixNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	_, err := store.GetTodoByPrefix("xxxxxx")
	if err == nil {
		t.Error("expected error getting non-existent todo by prefix")
	}
}

func TestMarkdownGetTodoNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	_, err := store.GetTodo(uuid.New())
	if err == nil {
		t.Error("expected error getting non-existent todo")
	}
}

func TestMarkdownDeleteTodoNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	err := store.DeleteTodo(uuid.New())
	if err == nil {
		t.Error("expected error deleting non-existent todo")
	}
}

func TestMarkdownUpdateTodoNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		Description: "Non-existent",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	err := store.UpdateTodo(todo)
	if err == nil {
		t.Error("expected error updating non-existent todo")
	}
}

func TestMarkdownMarkTodoDone(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
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
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	// Mark done
	if err := store.MarkTodoDone(todo.ID, true); err != nil {
		t.Fatalf("MarkTodoDone(true) failed: %v", err)
	}
	got, _ := store.GetTodo(todo.ID)
	if !got.Done {
		t.Error("todo should be marked done")
	}
	if got.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}

	// Mark undone
	if err := store.MarkTodoDone(todo.ID, false); err != nil {
		t.Fatalf("MarkTodoDone(false) failed: %v", err)
	}
	got, _ = store.GetTodo(todo.ID)
	if got.Done {
		t.Error("todo should be marked undone")
	}
	if got.CompletedAt != nil {
		t.Error("CompletedAt should be nil")
	}
}

func TestMarkdownMarkTodoDoneNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	err := store.MarkTodoDone(uuid.New(), true)
	if err == nil {
		t.Error("expected error marking non-existent todo")
	}
}

// --- Filters ---

func TestMarkdownListTodosWithFilters(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	now := time.Now().UTC()
	todos := []*Todo{
		{ID: uuid.New(), ProjectID: project.ID, ProjectName: project.Name, Description: "High priority", Priority: "high", Done: false, CreatedAt: now.Add(-3 * time.Minute), UpdatedAt: now},
		{ID: uuid.New(), ProjectID: project.ID, ProjectName: project.Name, Description: "Low priority", Priority: "low", Done: false, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now},
		{ID: uuid.New(), ProjectID: project.ID, ProjectName: project.Name, Description: "Done item", Priority: "medium", Done: true, CreatedAt: now.Add(-1 * time.Minute), UpdatedAt: now},
	}
	for _, td := range todos {
		if err := store.CreateTodo(td); err != nil {
			t.Fatalf("CreateTodo failed: %v", err)
		}
	}

	// Filter by done
	done := false
	list, err := store.ListTodos(&TodoFilter{Done: &done})
	if err != nil {
		t.Fatalf("ListTodos (done=false) failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 pending todos, got %d", len(list))
	}

	// Filter by priority
	priority := "high"
	list, err = store.ListTodos(&TodoFilter{Priority: &priority})
	if err != nil {
		t.Fatalf("ListTodos (priority=high) failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 high priority todo, got %d", len(list))
	}

	// Filter by project
	list, err = store.ListTodos(&TodoFilter{ProjectID: &project.ID})
	if err != nil {
		t.Fatalf("ListTodos (projectID) failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 todos in project, got %d", len(list))
	}

	// Nil filter = all
	list, err = store.ListTodos(nil)
	if err != nil {
		t.Fatalf("ListTodos (nil) failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 todos, got %d", len(list))
	}
}

func TestMarkdownFilterByTag(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
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
	if err := store.CreateTodo(todo1); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
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
	if err := store.CreateTodo(todo2); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	tagFilter := "urgent"
	list, err := store.ListTodos(&TodoFilter{Tag: &tagFilter})
	if err != nil {
		t.Fatalf("ListTodos with tag filter failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 todo with urgent tag, got %d", len(list))
	}
}

func TestMarkdownOverdueFilter(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
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
	if err := store.CreateTodo(todo1); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
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
	if err := store.CreateTodo(todo2); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	overdue := true
	list, err := store.ListTodos(&TodoFilter{Overdue: &overdue})
	if err != nil {
		t.Fatalf("ListTodos (overdue) failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 overdue todo, got %d", len(list))
	}
	if list[0].Description != "Overdue todo" {
		t.Errorf("expected overdue todo, got %v", list[0].Description)
	}
}

// --- Tag operations ---

// createTodoForTagTest sets up a project and todo for tag operation tests.
func createTodoForTagTest(t *testing.T, store *MarkdownStore) *Todo {
	t.Helper()
	project := &Project{
		ID: uuid.New(), Name: "test-project", CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	todo := &Todo{
		ID: uuid.New(), ProjectID: project.ID, ProjectName: project.Name,
		Description: "Todo with tags", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}
	return todo
}

func TestMarkdownTagAddAndGet(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	todo := createTodoForTagTest(t, store)

	if err := store.AddTagToTodo(todo.ID, "tag1"); err != nil {
		t.Fatalf("AddTagToTodo (tag1) failed: %v", err)
	}
	if err := store.AddTagToTodo(todo.ID, "tag2"); err != nil {
		t.Fatalf("AddTagToTodo (tag2) failed: %v", err)
	}

	tags, err := store.GetTagsForTodo(todo.ID)
	if err != nil {
		t.Fatalf("GetTagsForTodo failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	allTags, err := store.ListTags()
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(allTags) != 2 {
		t.Errorf("expected 2 tags in total, got %d", len(allTags))
	}

	tag, err := store.GetOrCreateTag("tag1")
	if err != nil {
		t.Fatalf("GetOrCreateTag failed: %v", err)
	}
	if tag.Name != "tag1" {
		t.Errorf("expected tag name 'tag1', got %q", tag.Name)
	}
}

func TestMarkdownTagRemoveAndDelete(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	todo := createTodoForTagTest(t, store)
	if err := store.AddTagToTodo(todo.ID, "tag1"); err != nil {
		t.Fatalf("AddTagToTodo (tag1) failed: %v", err)
	}
	if err := store.AddTagToTodo(todo.ID, "tag2"); err != nil {
		t.Fatalf("AddTagToTodo (tag2) failed: %v", err)
	}

	if err := store.RemoveTagFromTodo(todo.ID, "tag1"); err != nil {
		t.Fatalf("RemoveTagFromTodo failed: %v", err)
	}
	tags, _ := store.GetTagsForTodo(todo.ID)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag after removal, got %d", len(tags))
	}
	if tags[0] != "tag2" {
		t.Errorf("expected tag2, got %v", tags[0])
	}

	if err := store.AddTagToTodo(todo.ID, "tag1"); err != nil {
		t.Fatalf("AddTagToTodo (re-add) failed: %v", err)
	}
	if err := store.DeleteTag("tag1"); err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}
	tags, _ = store.GetTagsForTodo(todo.ID)
	for _, tg := range tags {
		if tg == "tag1" {
			t.Error("tag1 should have been removed by DeleteTag")
		}
	}
}

func TestMarkdownAddTagIdempotent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	// Add same tag twice - should be idempotent
	if err := store.AddTagToTodo(todo.ID, "dup"); err != nil {
		t.Fatalf("AddTagToTodo (first) failed: %v", err)
	}
	if err := store.AddTagToTodo(todo.ID, "dup"); err != nil {
		t.Fatalf("AddTagToTodo (second) failed: %v", err)
	}

	tags, _ := store.GetTagsForTodo(todo.ID)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag (idempotent add), got %d", len(tags))
	}
}

func TestMarkdownDeleteTagNotFound(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	err := store.DeleteTag("non-existent")
	if err == nil {
		t.Error("expected error deleting non-existent tag")
	}
}

func TestMarkdownGetTagsForTodoNoTags(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo without tags",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	tags, err := store.GetTagsForTodo(todo.ID)
	if err != nil {
		t.Fatalf("GetTagsForTodo failed: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

// --- Cascade delete ---

func TestMarkdownCascadeDeleteProject(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "cascade-test",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Will be cascade deleted",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	// Delete project - should cascade to todos
	if err := store.DeleteProject(project.ID); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	// Todo should also be gone
	_, err := store.GetTodo(todo.ID)
	if err == nil {
		t.Error("expected error getting todo after project cascade delete")
	}
}

// --- Maintenance ---

func TestMarkdownVacuum(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	// Vacuum is a no-op for markdown, should not error
	if err := store.Vacuum(); err != nil {
		t.Errorf("Vacuum failed: %v", err)
	}
}

func TestMarkdownIntegrityCheck(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	ok, err := store.IntegrityCheck()
	if err != nil {
		t.Fatalf("IntegrityCheck failed: %v", err)
	}
	if !ok {
		t.Error("integrity check should return true for empty store")
	}

	// Create some data and check again
	project := &Project{
		ID:        uuid.New(),
		Name:      "check-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Check todo",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	ok, err = store.IntegrityCheck()
	if err != nil {
		t.Fatalf("IntegrityCheck (with data) failed: %v", err)
	}
	if !ok {
		t.Error("integrity check should return true for valid store")
	}
}

// --- Todo with multiline notes ---

func TestMarkdownTodoWithMultilineNotes(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	notes := "Line 1\nLine 2\n\n```go\nfunc main() {}\n```\n\n- item 1\n- item 2"
	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo with complex notes",
		Notes:       notes,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	got, err := store.GetTodo(todo.ID)
	if err != nil {
		t.Fatalf("GetTodo failed: %v", err)
	}
	if got.Notes != notes {
		t.Errorf("Notes mismatch:\nwant: %q\ngot:  %q", notes, got.Notes)
	}
}

// --- Todo file naming ---

func TestMarkdownTodoFileLayout(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "my-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	todoID := uuid.New()
	todo := &Todo{
		ID:          todoID,
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Check file layout",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	expectedFile := filepath.Join(store.dataDir, "my-project", "todo-"+todoID.String()[:8]+".md")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected todo file at %s", expectedFile)
	}

	// Verify the content is a valid markdown file with frontmatter
	data, err := os.ReadFile(expectedFile) // #nosec G304 - test file in temp dir
	if err != nil {
		t.Fatalf("failed to read todo file: %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		t.Error("expected file to start with frontmatter delimiter")
	}
	if !strings.Contains(content, "description: Check file layout") {
		t.Error("expected description in frontmatter")
	}
}

// --- CreateTodo with tags ---

func TestMarkdownCreateTodoWithTags(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	project := &Project{
		ID:        uuid.New(),
		Name:      "test-project",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	todo := &Todo{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Description: "Todo with initial tags",
		Tags:        []string{"alpha", "beta", "gamma"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.CreateTodo(todo); err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	tags, err := store.GetTagsForTodo(todo.ID)
	if err != nil {
		t.Fatalf("GetTagsForTodo failed: %v", err)
	}
	sort.Strings(tags)
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}
	if tags[0] != "alpha" || tags[1] != "beta" || tags[2] != "gamma" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

// --- Empty project listing ---

func TestMarkdownListProjectsEmpty(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer func() { _ = store.Close() }()

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}
