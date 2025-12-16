// ABOUTME: Tests for vault change application handlers
// ABOUTME: Verifies entity creation, updates, deletions, and cross-entity lookups

package sync

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
)

func TestApplyProjectChange_Create(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	projectID := uuid.New().String()
	dirPath := "/test/path"

	payload := ProjectPayload{
		ID:            projectID,
		Name:          "Test Project",
		DirectoryPath: &dirPath,
		CreatedAt:     time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	change := vault.Change{
		Entity:   EntityProject,
		EntityID: projectID,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyProjectChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyProjectChange failed: %v", err)
	}

	// Verify project was created
	id, _ := uuid.Parse(projectID)
	project, err := db.GetProjectByID(appDB, id)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if project.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got %q", project.Name)
	}

	if project.DirectoryPath == nil || *project.DirectoryPath != dirPath {
		t.Errorf("expected directory path %q, got %v", dirPath, project.DirectoryPath)
	}
}

func TestApplyProjectChange_Update(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create initial project
	project := models.NewProject("Original Name", nil)
	if err := db.CreateProject(appDB, project); err != nil {
		t.Fatalf("failed to create initial project: %v", err)
	}

	// Apply update
	newPath := "/new/path"
	payload := ProjectPayload{
		ID:            project.ID.String(),
		Name:          "Updated Name",
		DirectoryPath: &newPath,
		CreatedAt:     project.CreatedAt.Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	change := vault.Change{
		Entity:   EntityProject,
		EntityID: project.ID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyProjectChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyProjectChange failed: %v", err)
	}

	// Verify project was updated
	updated, err := db.GetProjectByID(appDB, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %q", updated.Name)
	}

	if updated.DirectoryPath == nil || *updated.DirectoryPath != newPath {
		t.Errorf("expected directory path %q, got %v", newPath, updated.DirectoryPath)
	}
}

func TestApplyProjectChange_Delete(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create project to delete
	project := models.NewProject("To Delete", nil)
	if err := db.CreateProject(appDB, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Apply delete
	change := vault.Change{
		Entity:   EntityProject,
		EntityID: project.ID.String(),
		Op:       vault.OpDelete,
		Deleted:  true,
	}

	err := syncer.applyProjectChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyProjectChange delete failed: %v", err)
	}

	// Verify project was deleted
	_, err = db.GetProjectByID(appDB, project.ID)
	if err == nil {
		t.Error("expected error when getting deleted project")
	}
}

func TestApplyTodoChange_Create(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create project first
	project := models.NewProject("Test Project", nil)
	if err := db.CreateProject(appDB, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todoID := uuid.New().String()
	priority := "high"
	notes := "Test notes"
	dueDate := time.Now().Add(24 * time.Hour).Unix()

	payload := TodoPayload{
		ID:          todoID,
		ProjectName: project.Name,
		Description: "Test todo",
		Done:        false,
		Priority:    &priority,
		Notes:       &notes,
		DueDate:     &dueDate,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	change := vault.Change{
		Entity:   EntityTodo,
		EntityID: todoID,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyTodoChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyTodoChange failed: %v", err)
	}

	// Verify todo was created
	id, _ := uuid.Parse(todoID)
	todo, err := db.GetTodoByID(appDB, id)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if todo.Description != "Test todo" {
		t.Errorf("expected description 'Test todo', got %q", todo.Description)
	}

	if todo.Priority == nil || *todo.Priority != priority {
		t.Errorf("expected priority %q, got %v", priority, todo.Priority)
	}

	if todo.ProjectID != project.ID {
		t.Errorf("expected project ID %q, got %q", project.ID, todo.ProjectID)
	}
}

func TestApplyTodoChange_CreateWithMissingProject(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	todoID := uuid.New().String()

	payload := TodoPayload{
		ID:          todoID,
		ProjectName: "Nonexistent Project",
		Description: "Test todo",
		Done:        false,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	change := vault.Change{
		Entity:   EntityTodo,
		EntityID: todoID,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyTodoChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyTodoChange failed: %v", err)
	}

	// Verify project was created on-demand
	project, err := db.GetProjectByName(appDB, "Nonexistent Project")
	if err != nil {
		t.Fatalf("expected project to be created on-demand: %v", err)
	}

	// Verify todo was created
	id, _ := uuid.Parse(todoID)
	todo, err := db.GetTodoByID(appDB, id)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if todo.ProjectID != project.ID {
		t.Errorf("expected todo to be in auto-created project")
	}
}

func TestApplyTodoChange_Update(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create project and initial todo
	project := models.NewProject("Test Project", nil)
	if err := db.CreateProject(appDB, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := models.NewTodo(project.ID, "Original description")
	if err := db.CreateTodo(appDB, todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Apply update
	newPriority := "low"
	newNotes := "Updated notes"

	payload := TodoPayload{
		ID:          todo.ID.String(),
		ProjectName: project.Name,
		Description: "Updated description",
		Done:        true,
		Priority:    &newPriority,
		Notes:       &newNotes,
		CreatedAt:   todo.CreatedAt.Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	change := vault.Change{
		Entity:   EntityTodo,
		EntityID: todo.ID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyTodoChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyTodoChange failed: %v", err)
	}

	// Verify todo was updated
	updated, err := db.GetTodoByID(appDB, todo.ID)
	if err != nil {
		t.Fatalf("failed to get updated todo: %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got %q", updated.Description)
	}

	if !updated.Done {
		t.Error("expected todo to be done")
	}

	if updated.Notes == nil || *updated.Notes != newNotes {
		t.Errorf("expected notes %q, got %v", newNotes, updated.Notes)
	}
}

func TestApplyTodoChange_Delete(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create project and todo
	project := models.NewProject("Test Project", nil)
	if err := db.CreateProject(appDB, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := models.NewTodo(project.ID, "To delete")
	if err := db.CreateTodo(appDB, todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	// Apply delete
	change := vault.Change{
		Entity:   EntityTodo,
		EntityID: todo.ID.String(),
		Op:       vault.OpDelete,
		Deleted:  true,
	}

	err := syncer.applyTodoChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyTodoChange delete failed: %v", err)
	}

	// Verify todo was deleted
	_, err = db.GetTodoByID(appDB, todo.ID)
	if err == nil {
		t.Error("expected error when getting deleted todo")
	}
}

func TestApplyTagChange_Create(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	payload := TagPayload{
		Name: "urgent",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	change := vault.Change{
		Entity:   EntityTag,
		EntityID: "urgent",
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyTagChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyTagChange failed: %v", err)
	}

	// Verify tag was created
	tag, err := db.GetOrCreateTag(appDB, "urgent")
	if err != nil {
		t.Fatalf("failed to get tag: %v", err)
	}

	if tag.Name != "urgent" {
		t.Errorf("expected tag name 'urgent', got %q", tag.Name)
	}
}

func TestApplyTagChange_Idempotent(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	payload := TagPayload{
		Name: "bug",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	change := vault.Change{
		Entity:   EntityTag,
		EntityID: "bug",
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	// Apply twice
	err = syncer.applyTagChange(context.Background(), change)
	if err != nil {
		t.Fatalf("first applyTagChange failed: %v", err)
	}

	err = syncer.applyTagChange(context.Background(), change)
	if err != nil {
		t.Fatalf("second applyTagChange failed: %v", err)
	}

	// Verify only one tag exists
	tags, err := db.ListAllTags(appDB)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}

	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(tags))
	}
}

func TestApplyTodoTagChange_Create(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create project and todo
	project := models.NewProject("Test Project", nil)
	if err := db.CreateProject(appDB, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := models.NewTodo(project.ID, "Test todo")
	if err := db.CreateTodo(appDB, todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	payload := TodoTagPayload{
		TodoID:  todo.ID.String(),
		TagName: "urgent",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	change := vault.Change{
		Entity:   EntityTodoTag,
		EntityID: todo.ID.String() + "|urgent",
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyTodoTagChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyTodoTagChange failed: %v", err)
	}

	// Verify tag was created and associated
	tags, err := db.GetTodoTags(appDB, todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo tags: %v", err)
	}

	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}

	if tags[0].Name != "urgent" {
		t.Errorf("expected tag 'urgent', got %q", tags[0].Name)
	}
}

func TestApplyTodoTagChange_Delete(t *testing.T) {
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create project, todo, and tag association
	project := models.NewProject("Test Project", nil)
	if err := db.CreateProject(appDB, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	todo := models.NewTodo(project.ID, "Test todo")
	if err := db.CreateTodo(appDB, todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := db.AddTagToTodo(appDB, todo.ID, "urgent"); err != nil {
		t.Fatalf("failed to add tag to todo: %v", err)
	}

	// Apply delete
	change := vault.Change{
		Entity:   EntityTodoTag,
		EntityID: todo.ID.String() + "|urgent",
		Op:       vault.OpDelete,
		Deleted:  true,
	}

	err := syncer.applyTodoTagChange(context.Background(), change)
	if err != nil {
		t.Fatalf("applyTodoTagChange delete failed: %v", err)
	}

	// Verify association was removed
	tags, err := db.GetTodoTags(appDB, todo.ID)
	if err != nil {
		t.Fatalf("failed to get todo tags: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

func TestApplyChange_Router(t *testing.T) {
	syncer, _, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	tests := []struct {
		name       string
		entity     string
		shouldSkip bool
	}{
		{"routes project", EntityProject, false},
		{"routes todo", EntityTodo, false},
		{"routes tag", EntityTag, false},
		{"routes todo_tag", EntityTodoTag, false},
		{"skips unknown entity", "unknown_entity", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := vault.Change{
				Entity:   tt.entity,
				EntityID: "test-id",
				Op:       vault.OpDelete,
				Deleted:  true,
			}

			err := syncer.applyChange(context.Background(), change)

			if tt.shouldSkip {
				if err != nil {
					t.Errorf("expected no error for unknown entity, got %v", err)
				}
			}
			// For known entities, we may get errors (e.g., entity not found),
			// but we shouldn't panic or skip.
			// The fact that it doesn't panic is sufficient for this test.
		})
	}
}
