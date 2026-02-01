# Toki - Charm Removal Plan

## 1. Charmbracelet Dependencies Identified

From `go.mod`:

**Direct Dependencies:**
- `github.com/charmbracelet/charm v0.17.0` (replaced with `github.com/2389-research/charm v0.20.0`)

**Transitive Dependencies:**
- `github.com/charmbracelet/bubbles v0.20.0`
- `github.com/charmbracelet/bubbletea v1.3.3`
- `github.com/charmbracelet/keygen v0.5.1`
- `github.com/charmbracelet/lipgloss v1.0.0`
- `github.com/charmbracelet/log v0.2.2`
- `github.com/charmbracelet/x/ansi v0.8.0`
- `github.com/charmbracelet/x/term v0.2.1`

## 2. Charm Package Usage by File

| File | Import | Purpose |
|------|--------|---------|
| `cmd/toki/sync.go` | `charm/kv` | `kv.Repair()`, `kv.Reset()`, `kv.Wipe()` |
| `internal/charm/client.go` | `charm/client`, `charm/kv` | Core KV store operations |

## 3. Removal Strategy

### Phase 1: Replace Storage Backend

Replace Charm KV with SQLite. Create new package `internal/storage/`:

```
internal/storage/
  ├── storage.go      # Storage interface
  ├── sqlite.go       # SQLite implementation
  ├── schema.go       # Database schema
  └── storage_test.go
```

**SQLite Schema:**

```sql
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    directory_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE todos (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    done INTEGER NOT NULL DEFAULT 0,
    priority TEXT,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    due_date TIMESTAMP
);

CREATE TABLE tags (
    name TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE todo_tags (
    todo_id TEXT NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    tag_name TEXT NOT NULL REFERENCES tags(name) ON DELETE CASCADE,
    PRIMARY KEY (todo_id, tag_name)
);

CREATE INDEX idx_todos_project ON todos(project_id);
CREATE INDEX idx_todos_done ON todos(done);
CREATE INDEX idx_todo_tags_tag ON todo_tags(tag_name);
```

### Phase 2: Remove Sync Commands

- `toki sync status` - Show "sync not available"
- `toki sync now` - Remove
- `toki sync link/unlink` - Remove
- `toki sync repair/reset/wipe` - Keep for local DB maintenance

### Phase 3: Add Export Commands

```bash
toki export yaml > todos.yaml
toki export markdown > TODOS.md
toki export json --pretty > todos.json
toki export sqlite --output backup.db
```

## 4. Export Formats

### YAML Format

```yaml
exported_at: "2026-01-31T12:00:00Z"
projects:
  - id: "abc123..."
    name: "backend-api"
    path: "/home/user/projects/backend-api"
    todos:
      - id: "def456..."
        description: "Implement user auth"
        done: false
        priority: high
        tags: [urgent, backend]
        due_date: "2026-02-15T00:00:00Z"
```

### Markdown Format

```markdown
# Toki Export - 2026-01-31

## Project: backend-api

### Pending Todos

- [ ] **[HIGH]** Implement user auth
  - Due: 2026-02-15
  - Tags: urgent, backend

### Completed Todos

- [x] ~~Set up CI/CD pipeline~~
  - Completed: 2025-12-01
```

## 5. Files to Modify

### DELETE (entire charm package):
- `internal/charm/adapter.go`
- `internal/charm/client.go`
- `internal/charm/models.go`
- `internal/charm/projects.go`
- `internal/charm/tags.go`
- `internal/charm/todos.go`
- `internal/charm/*_test.go`

### CREATE:
- `internal/storage/storage.go`
- `internal/storage/sqlite.go`
- `internal/storage/schema.go`
- `internal/storage/storage_test.go`
- `cmd/toki/export.go`

### MODIFY:
- `go.mod` - Remove charmbracelet, add gopkg.in/yaml.v3
- `cmd/toki/root.go` - Replace `charm.InitClient()` with `storage.Open()`
- `cmd/toki/add.go` - Replace charm calls with storage interface
- `cmd/toki/list.go` - Replace charm calls
- `cmd/toki/done.go` - Replace charm calls
- `cmd/toki/remove.go` - Replace charm calls
- `cmd/toki/project.go` - Replace charm calls
- `cmd/toki/tag.go` - Replace charm calls
- `cmd/toki/context.go` - Replace charm calls
- `cmd/toki/sync.go` - Major rewrite
- `cmd/toki/mcp.go` - Replace charm.Client with storage
- `internal/mcp/server.go` - Replace charm.Client
- `internal/mcp/tools.go` - Replace all charm calls

## 6. Migration Path

Users with existing Charm KV data:
1. **Before upgrade**: `toki export yaml > backup.yaml`
2. **Upgrade**: Install new version
3. **After upgrade**: `toki import yaml backup.yaml`

Or automatic migration on first run.

## 7. Implementation Order

1. Create storage interface
2. Implement SQLite storage
3. Add export commands (YAML, JSON, Markdown, SQLite)
4. Create migration tool (Charm KV to SQLite)
5. Update cmd/ files
6. Update MCP server
7. Simplify sync commands
8. Remove charm package
9. Clean up go.mod
