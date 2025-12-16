// ABOUTME: Vault sync integration for toki
// ABOUTME: Handles change queuing, syncing, and applying remote changes

package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/models"
)

const (
	// AppID is the unique namespace UUID for toki app.
	AppID = "2c8f9e3a-7b4d-4e1a-9f5c-3d6e8a2b4f1c"

	EntityProject = "project"
	EntityTodo    = "todo"
	EntityTag     = "tag"
	EntityTodoTag = "todo_tag"
)

// Syncer manages vault sync for toki data.
type Syncer struct {
	config *Config
	store  *vault.Store
	keys   vault.Keys
	client *vault.Client
	appDB  *sql.DB
}

// NewSyncer creates a new syncer from config.
func NewSyncer(cfg *Config, appDB *sql.DB) (*Syncer, error) {
	if cfg.DerivedKey == "" {
		return nil, errors.New("derived key not configured - run 'toki sync login' first")
	}

	if cfg.DeviceID == "" {
		return nil, fmt.Errorf("device ID required - run 'toki sync login' to register device")
	}

	// DerivedKey is stored as hex-encoded seed or BIP39 mnemonic
	seed, err := vault.ParseSeedPhrase(cfg.DerivedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid derived key: %w", err)
	}

	keys, err := vault.DeriveKeys(seed, "", vault.DefaultKDFParams())
	if err != nil {
		return nil, fmt.Errorf("derive keys: %w", err)
	}

	store, err := vault.OpenStore(cfg.VaultDB)
	if err != nil {
		return nil, fmt.Errorf("open vault store: %w", err)
	}

	client := vault.NewClient(vault.SyncConfig{
		AppID:     AppID,
		BaseURL:   cfg.Server,
		DeviceID:  cfg.DeviceID,
		AuthToken: cfg.Token,
	})

	return &Syncer{
		config: cfg,
		store:  store,
		keys:   keys,
		client: client,
		appDB:  appDB,
	}, nil
}

// Close releases syncer resources.
func (s *Syncer) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// QueueProjectChange queues a change for a project.
func (s *Syncer) QueueProjectChange(ctx context.Context, project *models.Project, op vault.Op) error {
	var payload map[string]any
	if op != vault.OpDelete {
		payload = map[string]any{
			"id":         project.ID.String(),
			"name":       project.Name,
			"created_at": project.CreatedAt.Unix(),
		}
		if project.DirectoryPath != nil {
			payload["directory_path"] = *project.DirectoryPath
		}
	}

	return s.queueChange(ctx, EntityProject, project.ID.String(), op, payload)
}

// QueueTodoChange queues a change for a todo.
func (s *Syncer) QueueTodoChange(ctx context.Context, todo *models.Todo, projectName string, op vault.Op) error {
	var payload map[string]any
	if op != vault.OpDelete {
		payload = map[string]any{
			"id":           todo.ID.String(),
			"project_name": projectName,
			"description":  todo.Description,
			"done":         todo.Done,
			"created_at":   todo.CreatedAt.Unix(),
			"updated_at":   todo.UpdatedAt.Unix(),
		}
		if todo.Priority != nil {
			payload["priority"] = *todo.Priority
		}
		if todo.Notes != nil {
			payload["notes"] = *todo.Notes
		}
		if todo.DueDate != nil {
			payload["due_date"] = todo.DueDate.Unix()
		}
		if todo.CompletedAt != nil {
			payload["completed_at"] = todo.CompletedAt.Unix()
		}
	}

	return s.queueChange(ctx, EntityTodo, todo.ID.String(), op, payload)
}

// QueueTagChange queues a change for a tag.
func (s *Syncer) QueueTagChange(ctx context.Context, tag *models.Tag, op vault.Op) error {
	var payload map[string]any
	if op != vault.OpDelete {
		payload = map[string]any{
			"name": tag.Name,
		}
	}

	return s.queueChange(ctx, EntityTag, tag.Name, op, payload)
}

// QueueTodoTagChange queues a change for a todo-tag association.
func (s *Syncer) QueueTodoTagChange(ctx context.Context, todoID uuid.UUID, tagName string, op vault.Op) error {
	var payload map[string]any
	if op != vault.OpDelete {
		payload = map[string]any{
			"todo_id":  todoID.String(),
			"tag_name": tagName,
		}
	}

	entityID := todoID.String() + "|" + tagName
	return s.queueChange(ctx, EntityTodoTag, entityID, op, payload)
}

func (s *Syncer) queueChange(ctx context.Context, entity, entityID string, op vault.Op, payload map[string]any) error {
	change, err := vault.NewChange(entity, entityID, op, payload)
	if err != nil {
		return fmt.Errorf("create change: %w", err)
	}
	if op == vault.OpDelete {
		change.Deleted = true
	}

	plain, err := json.Marshal(change)
	if err != nil {
		return fmt.Errorf("marshal change: %w", err)
	}

	aad := change.AAD(s.config.UserID, s.config.DeviceID)
	env, err := vault.Encrypt(s.keys.EncKey, plain, aad)
	if err != nil {
		return fmt.Errorf("encrypt change: %w", err)
	}

	if err := s.store.EnqueueEncryptedChange(ctx, change, s.config.UserID, s.config.DeviceID, env); err != nil {
		return fmt.Errorf("enqueue change: %w", err)
	}

	// Auto-sync if enabled
	if s.config.AutoSync && s.canSync() {
		return s.Sync(ctx)
	}

	return nil
}

func (s *Syncer) canSync() bool {
	return s.config.Server != "" && s.config.Token != "" && s.config.UserID != ""
}

// Sync pushes local changes and pulls remote changes.
func (s *Syncer) Sync(ctx context.Context) error {
	return s.SyncWithEvents(ctx, nil)
}

// SyncWithEvents pushes local changes and pulls remote changes with progress callbacks.
func (s *Syncer) SyncWithEvents(ctx context.Context, events *vault.SyncEvents) error {
	if !s.canSync() {
		return errors.New("sync not configured - run 'toki sync login' first")
	}

	err := vault.Sync(ctx, s.store, s.client, s.keys, s.config.UserID, s.applyChange, events)
	if err != nil {
		if strings.Contains(err.Error(), "device") || strings.Contains(err.Error(), "403") {
			return fmt.Errorf("device not registered - please run 'toki sync login' again: %w", err)
		}
		return err
	}
	return nil
}

// applyChange applies a remote change to the local database.
// Routes changes to entity-specific handlers in apply.go.
func (s *Syncer) applyChange(ctx context.Context, c vault.Change) error {
	switch c.Entity {
	case EntityProject:
		return s.applyProjectChange(ctx, c)
	case EntityTodo:
		return s.applyTodoChange(ctx, c)
	case EntityTag:
		return s.applyTagChange(ctx, c)
	case EntityTodoTag:
		return s.applyTodoTagChange(ctx, c)
	default:
		// Ignore unknown entities
		return nil
	}
}

// PendingCount returns the number of changes waiting to be synced.
func (s *Syncer) PendingCount(ctx context.Context) (int, error) {
	return s.store.PendingCount(ctx)
}

// PendingItem represents a pending change in the queue.
type PendingItem struct {
	ChangeID string
	Entity   string
	TS       time.Time
}

// PendingChanges returns details of changes waiting to be synced.
func (s *Syncer) PendingChanges(ctx context.Context) ([]PendingItem, error) {
	batch, err := s.store.DequeueBatch(ctx, 100)
	if err != nil {
		return nil, err
	}

	items := make([]PendingItem, len(batch))
	for i, env := range batch {
		items[i] = PendingItem{
			ChangeID: env.ChangeID,
			Entity:   env.Entity,
			TS:       time.Unix(env.TS, 0),
		}
	}
	return items, nil
}

// LastSyncedSeq returns the last pulled sequence number.
func (s *Syncer) LastSyncedSeq(ctx context.Context) (string, error) {
	return s.store.GetState(ctx, "last_pulled_seq", "0")
}
