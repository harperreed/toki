# Toki Todo Manager - Design Document

**Date:** 2025-11-29
**Status:** Approved

## Overview

Toki is a super simple CLI-based todo manager with git-aware project detection. It organizes todos under projects, supports rich metadata (priority, tags, notes, due dates), and automatically detects project context from git repositories.

## Core Data Model

### Database Schema (SQLite)

**projects table:**
- `id` - UUID, primary key
- `name` - text, unique
- `directory_path` - text, nullable (linked git repo or directory)
- `created_at` - timestamp

**todos table:**
- `id` - UUID, primary key
- `project_id` - UUID, foreign key
- `description` - text
- `done` - boolean
- `priority` - text: "low", "medium", "high", nullable
- `notes` - text, nullable
- `created_at` - timestamp
- `completed_at` - timestamp, nullable
- `due_date` - timestamp, nullable

**tags table:**
- `id` - integer, primary key
- `name` - text, unique

**todo_tags junction table:**
- `todo_id` - UUID
- `tag_id` - integer

### Key Design Decisions

- UUIDs for todos/projects ensure stability across operations
- Tags are normalized for reusability
- Project directory path enables git-aware context detection
- Priority is optional (nullable)
- Timestamps track full lifecycle

## CLI Command Structure

### Project Management

```bash
toki project add <name> [--path <dir>]    # alias: toki pa
toki project list                          # alias: toki pl
toki project set-path <name> <dir>         # alias: toki psp
toki project remove <name>                 # alias: toki pr
```

### Todo Management

```bash
toki add <description> [flags]             # alias: toki a
  --project, -p <name>
  --priority <low|medium|high>
  --due <date>
  --tags <tag1,tag2>
  --notes <text>

toki list [filters]                        # alias: toki ls
  --project, -p <name>
  --tag, -t <tag>
  --done / --pending
  --priority <level>

toki done <uuid-prefix>                    # alias: toki d
toki undone <uuid-prefix>                  # alias: toki ud
toki edit <uuid-prefix>                    # alias: toki e
toki remove <uuid-prefix>                  # alias: toki rm
```

### Tag Management

```bash
toki tag add <todo-uuid> <tag>             # alias: toki ta
toki tag remove <todo-uuid> <tag>          # alias: toki tr
toki tags                                  # Show all tags
```

### Smart Behaviors

- If in git repo directory, auto-detect project for `add` and `list`
- `list` shows all pending todos by default, grouped by project
- UUID prefix matching accepts 6-8 character minimum
- Interactive prompts when required arguments missing

## Git-Aware Context Detection

### How It Works

When running commands like `toki add` or `toki list`:

1. Check current directory - walk up tree looking for `.git/`
2. Match against projects - query projects table for matching `directory_path`
3. Auto-context - if match found, use that project as default
4. Manual override - `--project` flag always overrides auto-detection

### Project-Path Linking

```bash
# Explicit linking
cd ~/code/myapp
toki project add myapp --path .

# Auto-create from git repo
cd ~/code/another-project
toki add "implement feature"  # prompts: "Create project 'another-project'?"

# Later usage
cd ~/code/myapp
toki add "fix bug"  # automatically added to myapp project
toki ls             # shows only myapp todos
```

### Implementation Details

- Store absolute paths, resolve symlinks for matching
- Fallback: if not in known project, require `--project` flag or prompt

## Output Formatting

### List View

```
PROJECT: myapp (~/code/myapp)
─────────────────────────────────────────────
  a3f2b9  [HIGH] Fix authentication bug
          Due: 2025-12-01 | Tags: security, backend

  b4c8d2  Implement user settings page
          Tags: frontend, ui

  c5e9f1  [MEDIUM] Update documentation
          Due: 2025-11-30 | Tags: docs

PROJECT: personal
─────────────────────────────────────────────
  d6a0e3  Buy groceries
          Tags: errands

────────────────────────────────────────────
3 pending todos across 2 projects
```

### Display Features

- Short UUID prefix shown (6 characters)
- Priority shown in brackets if set
- Metadata (due date, tags) on second line
- Notes hidden in list view
- Completed todos hidden by default (use `--done` to show)
- Color coding: priorities, overdue dates, project headers

## Architecture

### Project Structure

```
toki/
├── cmd/
│   └── toki/
│       └── main.go           # CLI entry point
├── internal/
│   ├── db/
│   │   ├── db.go            # SQLite connection & migrations
│   │   ├── projects.go      # Project CRUD
│   │   ├── todos.go         # Todo CRUD
│   │   └── tags.go          # Tag operations
│   ├── git/
│   │   └── detect.go        # Git repo detection & path matching
│   ├── ui/
│   │   └── format.go        # Output formatting & colors
│   └── models/
│       └── models.go        # Go structs for Project, Todo, Tag
├── go.mod
└── README.md
```

### Dependencies

- `github.com/spf13/cobra` - CLI framework (commands, flags, aliases)
- `github.com/mattn/go-sqlite3` or `modernc.org/sqlite` - SQLite driver
- `github.com/fatih/color` - Terminal colors
- `github.com/google/uuid` - UUID generation

### Data Storage

- Location: `~/.local/share/toki/toki.db` (XDG standard)
- Initialization: Auto-create DB and run migrations on first use

## Error Handling

### Common Scenarios

1. **UUID prefix ambiguity** - "Multiple todos match 'a3f', did you mean: a3f2b9, a3f8c1?"
2. **No project context** - "Not in a project directory. Use --project or run 'toki project list'"
3. **Database lock** - Retry with backoff
4. **Invalid date formats** - Accept ISO dates (2025-12-01)
5. **Missing project** - "Project 'foo' not found. Run 'toki project list' to see available projects"
6. **Empty list** - "No pending todos. Nice work! Run 'toki add' to create one."

### Validation Rules

- Project names: unique, non-empty
- Todo descriptions: required, minimum 3 characters
- Priority: must be low/medium/high if provided
- Tags: lowercase, alphanumeric + hyphens only

### Exit Codes

- 0 = success
- 1 = user error (bad input, validation failure)
- 2 = system error (DB issues, file permissions)

## Testing Strategy

### Test Coverage

1. **Unit tests** - DB operations (CRUD for projects/todos/tags)
2. **Integration tests** - Git detection logic with temp git repos
3. **CLI tests** - Command execution with test database
4. **Test fixtures** - Sample data for consistent testing

### Out of Scope (YAGNI for v1)

- Sync across machines
- Recurring todos
- Subtasks / checklist items
- Time tracking
- Natural language date parsing
- Config file
- Export/import

### Nice-to-Haves for Later

- `toki init` to create project from current directory
- Color scheme customization
- Bash/zsh completion
- Filter by date ranges
- Bulk operations

## Summary

Toki provides a simple, git-aware todo management system with:

- Flat project list with tag-based additional organization
- Rich metadata (priority, tags, notes, due dates)
- Smart context detection from git repositories
- Clean CLI with verb-noun commands and short aliases
- UUID-based stable identifiers with prefix matching
- SQLite storage for reliable querying
- Thoughtful output formatting and error handling
