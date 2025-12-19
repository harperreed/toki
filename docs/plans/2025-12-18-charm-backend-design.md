# Charm Backend Migration Design

Replace the current vault sync backend (harperreed/sweet) with Charm (harperreed/charm) for data storage and synchronization.

## Motivation

1. **Self-hosting** - Run our own backend at charm.2389.dev instead of relying on external services
2. **Simpler auth** - SSH key-based "invisible auth" replaces token/refresh flows

## Architecture

### Data Model

Store entities as JSON in Charm KV with type-prefixed keys:

```
project:{uuid}  → { id, name, directory_path, created_at }
todo:{uuid}     → { id, project_id, project_name, description, done,
                    priority, notes, tags[], created_at, updated_at,
                    completed_at, due_date }
tag:{name}      → { name, created_at }
```

Key decisions:
- **Denormalized tags** - Tags stored as `[]string` inside todos. No join table.
- **project_name in todos** - Enables display without extra lookup
- **Client-side filtering** - Prefix scan + filter in Go (fast enough for todo-scale data)

### Authentication

Charm uses SSH key-based auth:
- First run auto-creates account from `~/.ssh/id_*`
- Multi-device linking via `charm link` command
- No tokens, no recovery phrases, no refresh callbacks

### Encryption

Charm Crypt provides E2E encryption:
- Keys derived from SSH key automatically
- Data encrypted before leaving device
- Server only sees ciphertext

### Configuration

```json
// ~/.config/toki/sync.json
{
  "server": "charm.2389.dev",
  "auto_sync": true
}
```

Environment: `CHARM_HOST` overrides server (Charm's native env var).

## Code Changes

### Remove

```
internal/db/           # SQLite no longer needed
internal/sync/         # Vault SDK replaced by Charm
```

Dependencies to remove:
- github.com/harperreed/sweet
- github.com/mattn/go-sqlite3

### Add

```
internal/charm/
  ├── client.go       # Charm client initialization
  ├── kv.go           # KV operations (get, set, delete, prefix scan)
  ├── models.go       # JSON structs for project, todo, tag
  └── queries.go      # Higher-level queries with filtering
```

Dependencies to add:
- github.com/charmbracelet/charm

### Modify

- `cmd/toki/*.go` - Switch from db.* to charm.* calls
- `internal/mcp/` - Same updates
- `go.mod` - Dependency changes

## Command Changes

### Unchanged (new backend)

```bash
toki add/list/done/undone/remove
toki project add/list/remove
toki tag add/remove/list
```

### Sync Commands

| Old | New | Notes |
|-----|-----|-------|
| `toki sync init` | Removed | Auto-init on first use |
| `toki sync login` | Removed | SSH auth automatic |
| `toki sync logout` | `toki sync unlink` | Remove device |
| `toki sync now` | `toki sync now` | Force sync |
| `toki sync status` | `toki sync status` | Show Charm ID, server |
| `toki sync pending` | Removed | No explicit queue |
| `toki sync wipe` | `toki sync wipe` | Clear remote data |
| (new) | `toki sync link` | Link new device |

### Environment Variables

| Old | New |
|-----|-----|
| `TOKI_SERVER` | `CHARM_HOST` |
| `TOKI_TOKEN` | Removed |
| `TOKI_VAULT_DB` | Removed |
| `TOKI_DB` | Removed |

## Simplifications

1. **No dual-write** - Single write to Charm KV (was: SQLite + vault queue)
2. **No change queue** - Charm syncs automatically
3. **No vault.db** - Charm manages its own local store
4. **Implicit sync** - Writes sync in background automatically

## Default Server

`charm.2389.dev`
