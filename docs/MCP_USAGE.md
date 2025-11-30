# Toki MCP Server Usage Guide

## Introduction

The Toki MCP (Model Context Protocol) server enables AI agents like Claude to manage your todos and projects programmatically. This provides a powerful way for agents to track their work, coordinate with other agents, and maintain task lists that humans can review.

### What is MCP?

MCP is a standard protocol that allows AI agents to access tools, resources, and prompts. Think of it as an API that AI agents can use to interact with your todo management system.

### When to Use MCP vs CLI

- **Use MCP when:** AI agents need to track work, coordinate tasks, or manage todos programmatically
- **Use CLI when:** You're manually managing todos from the terminal

### Basic Concepts

- **Tools:** Functions agents can call to create, update, or delete todos and projects
- **Resources:** Read-only views of your data (like API endpoints that return JSON)
- **Prompts:** Workflow templates that guide agents through complex tasks

## Configuration

### Claude Desktop Setup

Add this to your Claude Desktop configuration file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`

**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

**Linux:** `~/.config/Claude/claude_desktop_config.json`

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

After adding this configuration, restart Claude Desktop. The toki MCP server will be available to Claude.

### Other MCP Clients

Any MCP-compatible client can connect to toki. The server runs in stdio mode, communicating over standard input/output.

To test the server manually:

```bash
toki serve
```

The server will wait for MCP protocol messages on stdin and respond on stdout.

### Environment Variables

Toki uses the standard XDG data directory for storage:

- **Database location:** `$XDG_DATA_HOME/toki/toki.db` (typically `~/.local/share/toki/toki.db`)

No additional environment variables are required.

## Tools Reference

Tools are the primary way agents interact with toki. Each tool performs a specific action and returns structured JSON.

### Todo Operations

#### add_todo

Create a new todo item with optional metadata.

**Parameters:**
- `description` (string, required): Brief description of the task
- `project_id` (string, optional): UUID of the project this todo belongs to
- `priority` (string, optional): Priority level - one of: `low`, `medium`, `high`
- `tags` (array of strings, optional): List of tags to categorize the todo
- `notes` (string, optional): Additional context or details about the task
- `due_date` (string, optional): Due date in ISO 8601 format (e.g., `2025-12-01T15:04:05Z`)

**Returns:** JSON object with the created todo including its UUID, all metadata, and timestamps.

**Example:**
```json
{
  "description": "Implement user authentication endpoint",
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "priority": "high",
  "tags": ["backend", "api", "auth"],
  "notes": "Use JWT tokens with 24h expiration",
  "due_date": "2025-12-05T00:00:00Z"
}
```

**Tips:**
- If `project_id` is omitted, a "default" project is automatically created and used
- Descriptions should be actionable (start with verbs like "implement", "fix", "write")
- Use tags consistently for easier filtering later

---

#### list_todos

Retrieve todos with powerful filtering capabilities.

**Parameters (all optional):**
- `project_id` (string): Filter by project UUID
- `done` (boolean): Filter by completion status (`true` = completed only, `false` = pending only)
- `priority` (string): Filter by priority level - one of: `low`, `medium`, `high`
- `tag` (string): Filter by tag name (exact match)
- `overdue` (boolean): Filter by overdue status (`true` = only overdue todos)

**Returns:** JSON object with array of todos, count, and applied filters.

**Example:**
```json
{
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "done": false,
  "priority": "high",
  "tag": "backend"
}
```

**Tips:**
- All filters can be combined for precise queries
- Omit all parameters to get all todos
- Use `overdue=true` to find tasks that need immediate attention

---

#### update_todo

Update a todo's metadata. All update fields are optional - only provide the fields you want to change.

**Parameters:**
- `todo_id` (string, required): Full UUID of the todo to update
- `description` (string, optional): New description
- `priority` (string, optional): New priority level - one of: `low`, `medium`, `high`
- `notes` (string, optional): New notes or additional context
- `due_date` (string, optional): New due date in ISO 8601 format

**Returns:** JSON object with the updated todo and all metadata.

**Example:**
```json
{
  "todo_id": "abc12345-1234-1234-1234-123456789abc",
  "priority": "high",
  "notes": "Updated requirements: must support OAuth2 and API keys"
}
```

**Tips:**
- Only include fields you want to change
- Useful for adding context as work progresses
- Can update due dates when priorities shift

---

#### mark_done

Mark a todo as complete.

**Parameters:**
- `todo_id` (string, required): Full UUID of the todo to mark as complete

**Returns:** JSON object with the updated todo showing `done: true` and completion timestamp.

**Example:**
```json
{
  "todo_id": "abc12345-1234-1234-1234-123456789abc"
}
```

**Tips:**
- Use `list_todos` first to find the UUID
- Completed todos remain in the database for historical tracking

---

#### mark_undone

Reopen a completed todo by marking it as incomplete.

**Parameters:**
- `todo_id` (string, required): Full UUID of the todo to reopen

**Returns:** JSON object with the updated todo showing `done: false` and cleared completion timestamp.

**Example:**
```json
{
  "todo_id": "abc12345-1234-1234-1234-123456789abc"
}
```

**Tips:**
- Use when a task needs to be revisited or wasn't actually finished
- Find completed todos with `list_todos(done=true)`

---

#### delete_todo

Permanently delete a todo. This action cannot be undone.

**Parameters:**
- `todo_id` (string, required): Full UUID of the todo to delete

**Returns:** JSON object with success confirmation and the deleted todo's ID.

**Example:**
```json
{
  "todo_id": "abc12345-1234-1234-1234-123456789abc"
}
```

**Tips:**
- This permanently removes the todo and all its associations
- Consider using `mark_done` instead if you want to preserve history

---

### Tag Operations

#### add_tag_to_todo

Associate a tag with a todo for categorization and filtering.

**Parameters:**
- `todo_id` (string, required): Full UUID of the todo to tag
- `tag_name` (string, required): Name of the tag to add (case-sensitive)

**Returns:** JSON object with the updated todo including all its tags.

**Example:**
```json
{
  "todo_id": "abc12345-1234-1234-1234-123456789abc",
  "tag_name": "urgent"
}
```

**Tips:**
- If the tag doesn't exist, it will be created automatically
- A todo can have multiple tags
- Use consistent tag names for better filtering

---

#### remove_tag_from_todo

Remove a tag association from a todo.

**Parameters:**
- `todo_id` (string, required): Full UUID of the todo to untag
- `tag_name` (string, required): Name of the tag to remove (case-sensitive)

**Returns:** JSON object with the updated todo showing remaining tags.

**Example:**
```json
{
  "todo_id": "abc12345-1234-1234-1234-123456789abc",
  "tag_name": "urgent"
}
```

**Tips:**
- This only removes the association - the tag itself remains in the system
- Operation succeeds silently even if the tag wasn't associated with the todo

---

### Project Operations

#### add_project

Create a new project to organize todos.

**Parameters:**
- `name` (string, required): Name of the project (should be unique and descriptive)
- `path` (string, optional): Filesystem path to git repository or project directory

**Returns:** JSON object with the created project including its UUID.

**Example:**
```json
{
  "name": "backend-api",
  "path": "/home/user/projects/backend-api"
}
```

**Tips:**
- Projects are containers for related tasks
- The `path` field links the project to a git repository
- All projects can be listed with `list_projects`

---

#### list_projects

Retrieve all projects in the system.

**Parameters:** None

**Returns:** JSON object with array of projects sorted by name, including IDs, names, paths, and creation timestamps.

**Example:** Call with no parameters

**Tips:**
- Use this to get project UUIDs before adding todos
- Helps you see all available projects before filtering

---

#### delete_project

Permanently delete a project. WARNING: This also deletes all associated todos due to database CASCADE constraints.

**Parameters:**
- `project_id` (string, required): Full UUID of the project to delete

**Returns:** JSON object with success confirmation.

**Example:**
```json
{
  "project_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Tips:**
- This action cannot be undone
- All todos in the project will be deleted
- Consider exporting or archiving todos before deleting

---

## Resources Reference

Resources provide read-only views of your data. They're faster than calling tools for common queries.

### toki://projects

Lists all projects with metadata including name, directory path, and creation time.

**When to use:** Need to see all available projects or get project UUIDs.

**Example output:**
```json
{
  "metadata": {
    "timestamp": "2025-11-30T12:00:00Z",
    "count": 3,
    "resource_uri": "toki://projects"
  },
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "backend-api",
      "directory_path": "/home/user/projects/backend-api",
      "created_at": "2025-11-25T10:30:00Z"
    }
  ]
}
```

---

### toki://todos

Lists all todos across all projects, both pending and completed.

**When to use:** Need a complete view of all tasks in the system.

**Example output:**
```json
{
  "metadata": {
    "timestamp": "2025-11-30T12:00:00Z",
    "count": 47,
    "resource_uri": "toki://todos"
  },
  "data": [
    {
      "id": "abc12345-1234-1234-1234-123456789abc",
      "project_id": "550e8400-e29b-41d4-a716-446655440000",
      "description": "Implement user authentication",
      "done": false,
      "priority": "high",
      "tags": ["backend", "auth"],
      "created_at": "2025-11-29T09:00:00Z"
    }
  ]
}
```

---

### toki://todos/pending

Lists all incomplete (not done) todos across all projects.

**When to use:** Focus on active work items, daily planning, or checking what's left to do.

---

### toki://todos/overdue

Lists all todos that are past their due date and not yet completed.

**When to use:** Daily reviews, identifying critical items requiring immediate attention.

---

### toki://todos/high-priority

Lists all todos marked with high priority, regardless of completion status.

**When to use:** Identifying important work items that need focus.

---

### toki://query

Returns all todos. In v1, use pre-built resources (toki://todos/pending, toki://todos/overdue, toki://todos/high-priority) for filtered views.

**When to use:** For now, use specific resources above or the `list_todos` tool for custom filtering. Future versions may support URI query parameters.

---

### toki://stats

Overview of todo statistics including totals, pending/completed counts, overdue items, breakdown by priority and project, and oldest pending todo.

**When to use:** Weekly reviews, generating reports, understanding overall workload.

**Example output:**
```json
{
  "metadata": {
    "timestamp": "2025-11-30T12:00:00Z",
    "resource_uri": "toki://stats"
  },
  "data": {
    "summary": {
      "total_todos": 47,
      "pending": 28,
      "completed": 19,
      "overdue": 3
    },
    "by_priority": {
      "high": 8,
      "medium": 22,
      "low": 5,
      "none": 12
    },
    "by_project": [
      {
        "project_id": "550e8400-e29b-41d4-a716-446655440000",
        "project_name": "backend-api",
        "todo_count": 23
      }
    ],
    "oldest_pending": {
      "id": "def45678-5678-5678-5678-567890abcdef",
      "description": "Refactor user service",
      "age_days": 14
    }
  }
}
```

---

## Prompts Reference

Prompts are workflow templates that guide agents through complex multi-step tasks.

### plan-project

Break down a new project into actionable tasks with phases, priorities, and tags.

**Arguments:**
- `project_name` (optional): Name of the project to plan. If not provided, you'll be prompted.

**When to use:**
- Starting work on a new initiative or feature
- Need to decompose a large effort into smaller tasks
- Want to organize work across multiple phases
- Setting up a new project with initial backlog

**Workflow:**
1. Create the project with `add_project`
2. Identify major phases (2-5 workstreams) - these become tags
3. Create high-level tasks with priorities, tags, and due dates
4. Review the breakdown with `list_todos`
5. Identify dependencies and add notes

**Example use case:** You're starting a new REST API project and need to break it into trackable tasks across setup, implementation, testing, and deployment phases.

---

### daily-review

Daily standup workflow to check overdue items, review priorities, identify blockers, and plan the day's focus.

**Arguments:** None

**When to use:**
- Start of each workday
- After completing a major task and deciding what's next
- When feeling overwhelmed and need to regain focus
- Monday morning to plan the week

**Workflow:**
1. Check overdue items (toki://todos/overdue)
2. Review high-priority pending items
3. Identify blockers (todos pending >7 days)
4. Plan today's focus (1-3 todos to complete)
5. Quick cleanup (mark completed todos as done)

**Example use case:** Morning standup where you need to quickly assess what's urgent, what's blocked, and what to focus on today.

---

### sprint-planning

Organize work into a focused iteration (sprint) by reviewing backlog, grouping by priority/tags, and setting sprint goals.

**Arguments:**
- `sprint_duration` (optional): Duration of the sprint (default: "2 weeks")

**When to use:**
- Starting a new sprint or iteration cycle
- Need to focus scattered work into clear goals
- Backlog has grown large and needs organization
- Team wants to align on shared objectives

**Workflow:**
1. Review full backlog (clean up outdated todos)
2. Group by priority (verify distribution is realistic)
3. Group by tags/themes (identify natural workstreams)
4. Set sprint goals (2-4 clear objectives)
5. Commit sprint scope (select specific todos)
6. Review scope for realism
7. Share the plan

**Example use case:** Planning a 2-week sprint where you need to commit to realistic goals and select which todos support those goals.

---

### track-agent-work

Guidelines for AI agents on when and how to use toki for tracking their work - focuses on human-visible outcomes, not internal operations.

**Arguments:** None

**When to use:**
- You're an AI agent working on tasks for a human
- Need to track findings or deliverables for human review
- Coordinating work with other agents
- Managing multi-step workflows spanning multiple sessions

**Key principle:** Create todos for human-visible work (research findings, deliverables, recommendations, blockers), NOT internal operations (search queries, file reads, reasoning steps).

**Workflow:**
1. Understand your assignment (is this multi-session? are there deliverables?)
2. Create outcome-focused todos (deliverables, not process steps)
3. Update progress (add findings to notes)
4. Mark completion (when deliverable is ready)
5. Coordinate with other agents (use tags and priorities)

**Example use case:** You're researching API rate limiting strategies and need to track the outcome (recommendation), not every search query and file you read.

---

### coordinate-tasks

Multi-agent collaboration workflow - assign work using tags/priorities, check for related work, update status for visibility.

**Arguments:** None

**When to use:**
- Multiple agents working on the same project
- Agent needs to hand off work to another agent
- Starting work and want to avoid duplicating efforts
- Need visibility into what other agents are doing

**Workflow:**
1. Check for existing work (avoid duplicates)
2. Create or claim a todo
3. Signal work status (use tags: "in-progress", "blocked", "needs-review")
4. Handoff to another agent (update notes, add handoff tag)
5. Check for handoffs to you (find work assigned to your specialty)
6. Resolve blockers (help unblock others when possible)

**Example use case:** Multiple agents collaborating on a feature where one agent implements, another tests, and another reviews security.

---

### report-status

Generate status updates and reports using toki data for different audiences and timeframes.

**Arguments:**
- `time_range` (optional): Time range for the report (default: "this week")

**When to use:**
- Daily standups or team check-ins
- Weekly status reports to manager
- Sprint reviews or retrospectives
- Monthly progress updates to stakeholders

**Workflow:**
1. Choose report type and audience
2. Gather completed work (list_todos with done=true)
3. Gather in-progress work (list_todos with done=false)
4. Identify blockers and risks
5. Generate metrics (use toki://stats resource)
6. Format for audience
7. Review and send

**Example use case:** Weekly status report to manager showing accomplishments, in-progress work, blockers, and next week's plan.

---

## Best Practices

### When to Track Work in Toki

Track work that is:
- **Human-visible:** Deliverables, findings, or recommendations humans will review
- **Multi-session:** Work spanning multiple days or sessions
- **Collaborative:** Coordinating with other agents or humans
- **Blocking:** Work that blocks other tasks or people
- **Measurable:** Clear completion criteria

Do NOT track:
- Internal agent operations (file reads, searches, parsing)
- Subtasks that complete in a single session
- Process steps within a single atomic operation
- Temporary state or reasoning steps

### How to Organize with Tags and Priorities

**Tags:**
- Use consistent naming conventions (lowercase, hyphen-separated)
- Common tag categories:
  - Work type: `bug`, `feature`, `docs`, `refactor`, `testing`
  - Domain: `backend`, `frontend`, `infrastructure`, `database`
  - Phase: `setup`, `implementation`, `review`, `deployment`
  - Status: `in-progress`, `blocked`, `needs-review`, `ready`
  - Sprint: `sprint-12`, `sprint-13`

**Priorities:**
- **High:** Blocking other work, customer-facing issues, critical deadlines
- **Medium:** Important but not urgent, can wait a few days
- **Low:** Nice to have, can be delayed if needed
- **None/Unset:** Backlog items not yet prioritized

Keep high-priority items under 30% of backlog. If everything is high priority, nothing is.

### Multi-Agent Coordination Patterns

**Pattern 1: Sequential Handoff**
- Agent A implements → tags "needs-testing"
- Agent B tests → tags "needs-review"
- Agent C reviews → tags "ready-to-deploy"
- Agent D deploys → marks done

**Pattern 2: Parallel Workstreams**
- Agent A works on backend (tag: "backend", "in-progress")
- Agent B works on frontend (tag: "frontend", "in-progress")
- Both coordinate through shared project and update notes

**Pattern 3: Blocker Resolution**
- Agent A discovers blocker → tags "blocked", adds notes
- Agent B sees blocked items → resolves blocker, removes "blocked" tag
- Agent A resumes work

### Performance Considerations

**Fast queries:**
- Use pre-built resources (`toki://todos/pending`, `toki://todos/overdue`, `toki://stats`)
- Filter at query time rather than fetching all then filtering client-side

**Slow queries:**
- Repeatedly calling `list_todos` without filters
- Fetching all todos then filtering by created_at timestamps

**Optimization tips:**
- Use resources for common views
- Use tags to group related todos (faster than searching descriptions)
- Keep total todo count reasonable (archive or delete completed todos periodically)

---

## Troubleshooting

### Server Won't Start

**Symptom:** `toki serve` command fails or hangs

**Possible causes:**
1. Database corruption
2. Permissions issue with data directory
3. Binary not in PATH

**Solutions:**
```bash
# Check if toki is in PATH
which toki

# Check database location
ls -la ~/.local/share/toki/toki.db

# Check permissions
ls -la ~/.local/share/toki/

# Reinitialize database (WARNING: deletes all data)
rm ~/.local/share/toki/toki.db
toki project add default

# Test server manually
toki serve
# Should wait for input, press Ctrl+C to exit
```

---

### Tools Not Appearing in Claude Desktop

**Symptom:** MCP server is configured but tools don't show up

**Possible causes:**
1. Configuration file has syntax errors
2. Claude Desktop wasn't restarted
3. `toki` binary not in Claude Desktop's PATH

**Solutions:**
```bash
# Validate JSON configuration
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json | jq .

# Check if toki is accessible
which toki

# Add toki to PATH in config (if needed)
{
  "mcpServers": {
    "toki": {
      "command": "/full/path/to/toki",
      "args": ["serve"]
    }
  }
}
```

Restart Claude Desktop after fixing configuration.

---

### Data Not Persisting

**Symptom:** Todos created but disappear after restart

**Possible causes:**
1. Multiple toki databases (different data directories)
2. Database not being flushed to disk
3. Using wrong data directory

**Solutions:**
```bash
# Check which database is being used
# The CLI and MCP server should use the same location

# CLI default: ~/.local/share/toki/toki.db
ls -la ~/.local/share/toki/

# Verify data is persisting
toki add "test todo"
sqlite3 ~/.local/share/toki/toki.db "SELECT * FROM todos;"
```

---

### Common Error Messages

**Error:** `invalid project_id: must be a valid UUID`

**Cause:** Project ID is malformed or not a valid UUID

**Solution:** Use `list_projects` to get the correct project UUID (full format: `550e8400-e29b-41d4-a716-446655440000`)

---

**Error:** `project not found: no project exists with ID 'xxx'`

**Cause:** Specified project doesn't exist in database

**Solution:** Use `list_projects` to see available projects, or create the project with `add_project` first

---

**Error:** `invalid priority 'urgent': must be one of 'low', 'medium', or 'high'`

**Cause:** Priority value is not one of the three allowed values

**Solution:** Use only `low`, `medium`, or `high` as priority values

---

**Error:** `invalid due_date format: must be ISO 8601 (RFC3339)`

**Cause:** Due date is not in the correct format

**Solution:** Use ISO 8601 format: `2025-12-01T15:04:05Z` or `2025-12-01T00:00:00Z` for date-only

---

**Error:** `todo not found: no todo exists with ID 'xxx'`

**Cause:** Specified todo doesn't exist or ID is incorrect

**Solution:** Use `list_todos` to find the correct todo UUID. Remember UUIDs are case-sensitive.

---

## Additional Resources

- **Toki GitHub Repository:** https://github.com/harper/toki
- **MCP Specification:** https://modelcontextprotocol.io
- **Issue Tracker:** https://github.com/harper/toki/issues

For questions or bug reports, please open an issue on GitHub.
