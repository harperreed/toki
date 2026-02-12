---
name: toki
description: Task and todo management - add todos, manage projects, track completion. Use when the user wants to create, list, or manage tasks and todos.
---

# toki - Task Management

Personal todo and task tracking with projects, tags, priorities, and due dates.

## When to use toki

- User wants to add a task or todo
- User asks about their tasks or what's pending
- User wants to mark something as done
- User mentions deadlines, priorities, or project organization

## Available MCP tools

| Tool | Purpose |
|------|---------|
| `mcp__toki__add_todo` | Create a new todo |
| `mcp__toki__list_todos` | List todos with filters |
| `mcp__toki__mark_done` | Complete a todo |
| `mcp__toki__mark_undone` | Reopen a todo |
| `mcp__toki__update_todo` | Modify todo details |
| `mcp__toki__delete_todo` | Remove a todo |
| `mcp__toki__add_project` | Create a project |
| `mcp__toki__list_projects` | List all projects |
| `mcp__toki__add_tag_to_todo` | Tag a todo |
| `mcp__toki__remove_tag_from_todo` | Untag a todo |
| `mcp__toki__delete_project` | Delete a project and its todos |

## Common patterns

### Add a task
```
mcp__toki__add_todo(description="Fix the login bug", priority="high", tags=["bugs", "urgent"])
```

### List pending tasks
```
mcp__toki__list_todos(done=false)
```

### List by project
```
mcp__toki__list_todos(project_id="uuid-here", done=false)
```

### Complete a task
```
mcp__toki__mark_done(todo_id="uuid-here")
```

### Add with due date
```
mcp__toki__add_todo(description="Submit report", due_date="2026-02-15T17:00:00Z")
```

## CLI commands (if MCP unavailable)

```bash
toki add "Task description" --priority high --tags urgent
toki list                    # All pending
toki list --done            # Completed
toki list --project myproj  # By project
toki done <id>              # Mark complete
toki export markdown        # Export to markdown
```

## Data location

`~/.local/share/toki/toki.db` (SQLite, respects XDG_DATA_HOME)
