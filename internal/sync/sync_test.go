// ABOUTME: Tests for vault sync integration
// ABOUTME: Verifies change queuing, syncing, and pending count tracking

package sync

import (
	"context"
	"path/filepath"
	"testing"

	"suitesync/vault"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyncer(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test app database
	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	// Create seed and derive key
	seed, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
		AutoSync:   false,
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	require.NotNil(t, syncer)
	defer func() { _ = syncer.Close() }()

	assert.Equal(t, cfg, syncer.config)
	assert.NotNil(t, syncer.store)
	assert.NotNil(t, syncer.client)
	assert.NotNil(t, syncer.keys)

	// Verify keys were derived correctly
	expectedKeys, err := vault.DeriveKeys(seed, "", vault.DefaultKDFParams())
	require.NoError(t, err)
	assert.Equal(t, expectedKeys.EncKey, syncer.keys.EncKey)
}

func TestNewSyncerNoDerivedKey(t *testing.T) {
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	cfg := &Config{
		Server:   "https://test.example.com",
		DeviceID: "test-device",
		VaultDB:  filepath.Join(tmpDir, "vault.db"),
	}

	_, err := NewSyncer(cfg, appDB)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "derived key not configured")
}

func TestQueueProjectChange(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	project := models.NewProject("test-project", nil)

	// Queue project create
	err := syncer.QueueProjectChange(ctx, project, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueTodoChange(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	projectID := uuid.New()
	todo := models.NewTodo(projectID, "test todo")
	todo.Priority = strPtr("high")
	todo.Notes = strPtr("test notes")

	// Queue todo create
	err := syncer.QueueTodoChange(ctx, todo, "test-project", vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueTagChange(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	tag := &models.Tag{
		Name: "urgent",
	}

	// Queue tag create
	err := syncer.QueueTagChange(ctx, tag, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueTodoTagChange(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	todoID := uuid.New()
	tagName := "urgent"

	// Queue todo-tag association
	err := syncer.QueueTodoTagChange(ctx, todoID, tagName, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueDeleteChange(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	project := models.NewProject("test-project", nil)

	// Queue project delete
	err := syncer.QueueProjectChange(ctx, project, vault.OpDelete)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPendingCount(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	// Initially zero
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Queue multiple changes
	project := models.NewProject("project-1", nil)
	err = syncer.QueueProjectChange(ctx, project, vault.OpUpsert)
	require.NoError(t, err)

	todo := models.NewTodo(project.ID, "todo-1")
	err = syncer.QueueTodoChange(ctx, todo, project.Name, vault.OpUpsert)
	require.NoError(t, err)

	tag := &models.Tag{Name: "urgent"}
	err = syncer.QueueTagChange(ctx, tag, vault.OpUpsert)
	require.NoError(t, err)

	// Verify count
	count, err = syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestMultipleChanges(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	// Create multiple projects
	for i := 0; i < 5; i++ {
		project := models.NewProject("project-"+string(rune('A'+i)), nil)
		err := syncer.QueueProjectChange(ctx, project, vault.OpUpsert)
		require.NoError(t, err)
	}

	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestAutoSyncDisabled(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	// AutoSync is disabled by default in test setup
	assert.False(t, syncer.config.AutoSync)

	project := models.NewProject("test-project", nil)
	err := syncer.QueueProjectChange(ctx, project, vault.OpUpsert)
	require.NoError(t, err)

	// Change should be queued but not synced
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSyncNotConfigured(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	// Create syncer with missing server config
	cfg := &Config{
		Server:     "", // Empty server
		UserID:     "",
		Token:      "",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	defer func() { _ = syncer.Close() }()

	// Sync should fail with helpful error
	err = syncer.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync not configured")
}

// Helper functions moved to test_helpers.go
