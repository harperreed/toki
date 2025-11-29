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
