// ABOUTME: Tag operations for Charm KV
// ABOUTME: Implements create, list, and get for tags (used for autocomplete)

package charm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// CreateTag stores a new tag in the KV store.
func (c *Client) CreateTag(tag *Tag) error {
	if c.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	data, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("failed to marshal tag: %w", err)
	}

	key := TagKey(tag.Name)
	if err := c.kv.Set([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store tag: %w", err)
	}

	c.syncIfEnabled()
	return nil
}

// GetTag retrieves a tag by name.
func (c *Client) GetTag(name string) (*Tag, error) {
	if err := c.SyncIfStale(); err != nil {
		return nil, fmt.Errorf("failed to sync: %w", err)
	}

	key := TagKey(name)
	data, err := c.kv.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("tag not found: %w", err)
	}

	var tag Tag
	if err := json.Unmarshal(data, &tag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tag: %w", err)
	}

	return &tag, nil
}

// GetOrCreateTag retrieves a tag by name, creating it if it doesn't exist.
func (c *Client) GetOrCreateTag(name string) (*Tag, error) {
	tag, err := c.GetTag(name)
	if err == nil {
		return tag, nil
	}

	tag = &Tag{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	if err := c.CreateTag(tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// ListTags returns all tags, sorted by name.
func (c *Client) ListTags() ([]*Tag, error) {
	if err := c.SyncIfStale(); err != nil {
		return nil, fmt.Errorf("failed to sync: %w", err)
	}

	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	tags := make([]*Tag, 0, len(keys))
	for _, key := range keys {
		keyStr := string(key)
		if !strings.HasPrefix(keyStr, TagKeyPrefix) {
			continue
		}

		data, err := c.kv.Get(key)
		if err != nil {
			continue
		}

		var tag Tag
		if err := json.Unmarshal(data, &tag); err != nil {
			continue
		}

		tags = append(tags, &tag)
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	return tags, nil
}

// DeleteTag removes a tag by name.
func (c *Client) DeleteTag(name string) error {
	if c.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	key := TagKey(name)
	if err := c.kv.Delete([]byte(key)); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	c.syncIfEnabled()
	return nil
}
