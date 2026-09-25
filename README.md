# Toki

A super simple git-aware CLI todo manager.

## Features

- **Git-aware context detection** - Automatically associates todos with projects based on your current directory
- **Rich metadata** - Priority, tags, notes, and due dates
- **UUID-based identifiers** - Stable IDs with short prefix matching
- **Clean CLI** - Intuitive commands with short aliases
- **Flexible storage** - SQLite or Markdown backends (user-configurable)
- **Export & import** - YAML, JSON, Markdown, and SQLite backup formats
- **Backend migration** - Move data between storage backends
- **MCP server** - Model Context Protocol integration for AI agents

## Installation

```bash
go install github.com/harper/toki/cmd/toki@latest
```

Or build from source:

```bash
git clone https://github.com/harperreed/toki
cd toki
make install
```

## Quick Start

```bash
# Create a project
toki project add myproject --path ~/code/myproject

# Add a todo (from within the project directory)
cd ~/code/myproject
toki add "implement feature" --priority high --tags backend,api

# List todos
toki list

# Mark done (use first 6+ chars of UUID)
toki done a3f2b9

# Add tags
toki tag add a3f2b9 urgent

# Remove todo
toki remove a3f2b9
```

## Commands

### Todos

```bash
toki add <description> [flags]             # Create todo (alias: a)
  --project, -p <name>                     # Specify project
  --priority <low|medium|high>             # Set priority
  --tags <tag1,tag2>                       # Add tags
  --notes <text>                           # Add notes
  --due <YYYY-MM-DD>                       # Set due date

toki list [flags]                          # List todos (aliases: ls, l)
  --project, -p <name>                     # Filter by project
  --tag, -t <tag>                          # Filter by tag
  --done                                   # Show completed todos
  --pending                                # Show pending todos only
  --priority <level>                       # Filter by priority

toki done <uuid-prefix> [uuid-prefix...]   # Mark complete (alias: d)
toki undone <uuid-prefix> [uuid-prefix...] # Mark incomplete (alias: ud)
toki remove <uuid-prefix>                  # Delete todo (alias: rm)
```

`toki list` shows pending todos by default. Pass `--done` to see completed
items or `--pending` to be explicit. Multiple UUID prefixes may be supplied to
`toki done` and `toki undone` to update several todos at once.

### Projects

```bash
toki project add <name> [--path <dir>]     # Create project (alias: a)
toki project list                          # List projects (aliases: ls, l)
toki project set-path <name> <path>        # Link directory (alias: sp)
toki project remove <name>                 # Delete project (aliases: rm, r)
toki project cleanup                       # Remove duplicate projects
```

`toki project` itself has the alias `p`. Removing a project also removes its
todos. `toki project cleanup` keeps the oldest project for each name and
reassigns todos from duplicates to the kept project.

### Tags

```bash
toki tag add <uuid-prefix> <tag>           # Add tag to todo (alias: a)
toki tag remove <uuid-prefix> <tag>        # Remove tag (aliases: rm, r)
toki tag list                              # Show all tags
```

`toki tag add` and `toki tag remove` normalize tag names to lowercase. Tags
supplied via `toki add --tags` or `toki import` are stored verbatim, so use
lowercase tag names there to keep filtering with `toki list --tag` consistent.

### Export

Export all projects and their todos. Output is written to stdout, so redirect
to a file.

```bash
toki export yaml                           # YAML (human-readable)
toki export json [--pretty]                # JSON (machine-readable)
toki export markdown                       # Markdown checklist (alias: md)
toki export sqlite [--output, -o <path>]   # Copy the SQLite database file
```

Examples:

```bash
toki export yaml > todos.yaml
toki export json --pretty > todos.json
toki export markdown > TODOS.md
toki export sqlite --output backup.db
```

### Import

Restore data from a YAML export produced by `toki export yaml`. The import is
idempotent: projects and todos whose IDs already exist are skipped.

```bash
toki import <file> [--dry-run]
```

Example:

```bash
toki import backup.yaml
toki import --dry-run backup.yaml
```

### Utilities

```bash
toki setup                                 # Interactive storage configuration wizard
toki migrate --to <sqlite|markdown> [flags] # Migrate between storage backends
  --to <sqlite|markdown>                   # Target backend (required)
  --data-dir <dir>                         # Target data directory
  --force                                  # Allow a non-empty target directory
toki version                               # Display version information
toki install-skill [--yes, -y]             # Install the Claude Code skill
toki mcp                                   # Start the MCP server (stdio mode)
```

`toki migrate` reads from the currently configured backend and writes to the
target; it does not update the config file. Verify the migration, then update
`config.json` (or rerun `toki setup`).

## Git-Aware Context

When you run `toki add` or `toki list` from within a git repository:

1. Toki finds the repository root
2. Looks up the associated project
3. If no project exists, offers to create one
4. Uses that project as the default context

This means you can just run `toki add "task"` without specifying `--project` when you're in the right directory. Outside a git repository, todos fall back to a `default` project.

## MCP Server

Toki includes a Model Context Protocol (MCP) server that enables AI agents like Claude to manage your todos and projects programmatically.

### What is MCP?

MCP (Model Context Protocol) is a standard that allows AI agents to access tools, resources, and workflows. With the toki MCP server, agents can create, update, and manage todos, track their work, and coordinate with other agents - all while keeping you informed through a centralized task list.

### Quick Setup

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "toki": {
      "command": "toki",
      "args": ["mcp"]
    }
  }
}
```

Restart Claude Desktop and the toki MCP server will be available. The `toki mcp` command runs in stdio mode until interrupted.

### Capabilities

**11 Tools** - Full CRUD operations for todos and projects:
- Create, list, update, and delete todos
- Mark todos done/undone
- Add/remove tags
- Create, list, and delete projects

**7 Resources** - Read-only views of your data:
- `toki://todos` - All todos
- `toki://todos/pending` - Incomplete todos
- `toki://todos/overdue` - Past-due todos
- `toki://todos/high-priority` - High-priority items
- `toki://projects` - All projects
- `toki://stats` - Summary statistics
- `toki://query` - Custom queries

**6 Prompts** - Workflow templates for effective task management:
- `plan-project` - Break down new projects into actionable tasks
- `daily-review` - Daily standup and planning workflow
- `sprint-planning` - Organize work into focused iterations
- `track-agent-work` - Guidelines for agents tracking their work
- `coordinate-tasks` - Multi-agent collaboration workflow
- `report-status` - Generate status updates and reports

### Example Usage

```javascript
// Agent creates a todo
add_todo({
  description: "Implement user authentication",
  priority: "high",
  tags: ["backend", "auth"],
  due_date: "2025-12-05T00:00:00Z"
})

// Agent checks what's overdue
// Access resource: toki://todos/overdue

// Agent reviews daily work
// Use prompt: daily-review
```

### Claude Code Skill

Run `toki install-skill` to install a Claude Code skill that teaches Claude how to
use toki for task management. Pass `--yes` (`-y`) to skip the confirmation prompt.

### Documentation

See [docs/MCP_USAGE.md](docs/MCP_USAGE.md) for complete documentation including:
- Configuration guide
- Tool reference with examples
- Resource reference
- Prompt workflows
- Best practices for agent integration
- Troubleshooting

## Development

```bash
# Run tests
make test

# Run tests with the race detector
make test-race

# Build binary
make build

# Install locally
make install
```

## Data Storage

Toki supports two storage backends, configurable via `toki setup`:

- **SQLite** (default for existing users): `~/.local/share/toki/toki.db`
- **Markdown** (default for new users): `~/.local/share/toki/` directory with `.md` files

Both backends follow the XDG standard (`$XDG_DATA_HOME/toki/`). Use
`toki migrate --to <backend>` to move data between them, and `toki export` /
`toki import` to back up and restore.

## Design

See `docs/plans/2025-11-29-toki-todo-manager-design.md` for the full design document.

## License

MIT
