# Toki

A super simple git-aware CLI todo manager.

## Features

- **Git-aware context detection** - Automatically associates todos with projects based on your current directory
- **Rich metadata** - Priority, tags, notes, and due dates
- **UUID-based identifiers** - Stable IDs with short prefix matching
- **Clean CLI** - Intuitive commands with short aliases
- **SQLite storage** - Fast, reliable, single-file database

## Installation

```bash
go install github.com/harper/toki/cmd/toki@latest
```

Or build from source:

```bash
git clone https://github.com/harper/toki
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

### Projects

```bash
toki project add <name> [--path <dir>]    # Create project
toki project list                          # List projects
toki project set-path <name> <path>        # Link directory
toki project remove <name>                 # Delete project
```

### Todos

```bash
toki add <description> [flags]             # Create todo
  --project, -p <name>                     # Specify project
  --priority <low|medium|high>             # Set priority
  --tags <tag1,tag2>                       # Add tags
  --notes <text>                           # Add notes
  --due <YYYY-MM-DD>                       # Set due date

toki list [flags]                          # List todos
  --project, -p <name>                     # Filter by project
  --tag, -t <tag>                          # Filter by tag
  --done / --pending                       # Filter by status
  --priority <level>                       # Filter by priority

toki done <uuid-prefix>                    # Mark complete
toki undone <uuid-prefix>                  # Mark incomplete
toki remove <uuid-prefix>                  # Delete todo
```

### Tags

```bash
toki tag add <uuid-prefix> <tag>           # Add tag to todo
toki tag remove <uuid-prefix> <tag>        # Remove tag
toki tag list                              # Show all tags
```

## Git-Aware Context

When you run `toki add` or `toki list` from within a git repository:

1. Toki finds the repository root
2. Looks up the associated project
3. If no project exists, offers to create one
4. Uses that project as the default context

This means you can just run `toki add "task"` without specifying `--project` when you're in the right directory.

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
      "args": ["serve"]
    }
  }
}
```

Restart Claude Desktop and the toki MCP server will be available.

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

# Build binary
make build

# Install locally
make install
```

## Data Storage

Toki stores all data in `~/.local/share/toki/toki.db` (XDG standard).

## Design

See `docs/plans/2025-11-29-toki-todo-manager-design.md` for the full design document.

## License

MIT
