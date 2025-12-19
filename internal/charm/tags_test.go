// ABOUTME: Tests for tag operations
// ABOUTME: Verifies create, list, and get for tags

package charm

import (
	"testing"
	"time"
)

func TestCreateTag(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	tag := &Tag{
		Name:      "urgent",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateTag(tag); err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	retrieved, err := client.GetTag("urgent")
	if err != nil {
		t.Fatalf("failed to get tag: %v", err)
	}

	if retrieved.Name != "urgent" {
		t.Errorf("Name mismatch")
	}
}

func TestListTags(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	tags := []string{"urgent", "backend", "frontend"}
	for _, name := range tags {
		tag := &Tag{Name: name, CreatedAt: time.Now().UTC()}
		if err := client.CreateTag(tag); err != nil {
			t.Fatalf("failed to create tag: %v", err)
		}
	}

	result, err := client.ListTags()
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 tags, got %d", len(result))
	}

	// Verify sorted by name
	if result[0].Name != "backend" {
		t.Errorf("expected first tag to be 'backend', got '%s'", result[0].Name)
	}
	if result[1].Name != "frontend" {
		t.Errorf("expected second tag to be 'frontend', got '%s'", result[1].Name)
	}
	if result[2].Name != "urgent" {
		t.Errorf("expected third tag to be 'urgent', got '%s'", result[2].Name)
	}
}

func TestGetOrCreateTag(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	// First call creates
	tag1, err := client.GetOrCreateTag("new-tag")
	if err != nil {
		t.Fatalf("failed to get/create tag: %v", err)
	}

	// Second call returns existing
	tag2, err := client.GetOrCreateTag("new-tag")
	if err != nil {
		t.Fatalf("failed to get/create tag: %v", err)
	}

	if tag1.Name != tag2.Name {
		t.Errorf("tags should be the same")
	}

	if !tag1.CreatedAt.Equal(tag2.CreatedAt) {
		t.Errorf("tags should have the same CreatedAt timestamp")
	}
}

func TestDeleteTag(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	tag := &Tag{
		Name:      "to-delete",
		CreatedAt: time.Now().UTC(),
	}

	if err := client.CreateTag(tag); err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	if err := client.DeleteTag("to-delete"); err != nil {
		t.Fatalf("failed to delete tag: %v", err)
	}

	_, err := client.GetTag("to-delete")
	if err == nil {
		t.Error("expected error getting deleted tag")
	}
}
