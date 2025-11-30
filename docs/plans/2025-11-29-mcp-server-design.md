# Toki MCP Server Design

**Date:** 2025-11-29
**Status:** Approved for implementation

## Overview

Add Model Context Protocol (MCP) server capabilities to toki, making it a general-purpose task management API for AI agents. Agents can manage project work, coordinate tasks, and integrate toki into their workflows while keeping internal agent-specific tracking separate.

## Goals

- Enable AI agents to use toki for visible/shared project work
- Provide both simple CRUD operations and intelligent workflow tools
- Make tools self-documenting with compelling definitions that guide proper usage
- Maintain toki's "super simple" philosophy - one binary, minimal dependencies

## Architecture

### Command Structure

Add `toki serve` command to existing CLI binary:

```bash
toki serve [--stdio]  # stdio is default and only mode
```

### Package Organization

```
cmd/
  toki/
    serve.go       # New MCP server command

internal/
  mcp/
    server.go      # MCP server initialization and config
    tools.go       # Tool definitions and handlers
    resources.go   # Resource providers
    prompts.go     # Prompt templates
```

### Integration Points

- MCP server uses same `internal/db/` package as CLI
- Shares `internal/models/` for data structures
- Single SQLite database serves both CLI and MCP operations
- No code duplication, just different interface layer

## Tools

### Core CRUD Tools (Simple Operations)

**Todo Operations:**
- `add_todo` - Create new todo with full metadata
  - Parameters: description (required), project_id, priority, tags, notes, due_date
  - Returns: Created todo with UUID

- `mark_done` - Mark todo complete by ID/prefix
  - Parameters: todo_id (supports UUID prefix matching)
  - Returns: Updated todo

- `mark_undone` - Reopen completed todo
  - Parameters: todo_id
  - Returns: Updated todo

- `delete_todo` - Remove todo permanently
  - Parameters: todo_id
  - Returns: Success confirmation

- `update_todo` - Generic update for advanced fields
  - Parameters: todo_id, priority, notes, due_date, description
  - Returns: Updated todo

- `list_todos` - Rich filtering
  - Parameters: project_id, done, priority, tag, overdue
  - Supports combining filters
  - Returns: Array of todos with full metadata

**Project Operations:**
- `add_project` - Create project with optional git path
  - Parameters: name (required), path (optional)
  - Returns: Created project with UUID

- `list_projects` - Get all projects
  - Returns: Array of projects

- `delete_project` - Remove project (cascades to todos)
  - Parameters: project_id
  - Returns: Success confirmation

**Tag Operations:**
- `add_tag_to_todo` - Associate tag with todo
  - Parameters: todo_id, tag_name
  - Returns: Updated todo with tags

- `remove_tag_from_todo` - Remove tag association
  - Parameters: todo_id, tag_name
  - Returns: Updated todo with tags

### Workflow Tools (Higher-Level Operations)

**Planning & Breakdown:**
- `breakdown_task` - Suggest subtasks for large task
  - Input: todo_id or task description
  - Output: Array of suggested subtasks with recommended priorities/tags
  - Agent reviews suggestions before creating todos

**Analysis & Reporting:**
- `analyze_workload` - Analyze current todo state
  - Output: Summary stats (total pending, overdue count, by priority, by project)
  - Identifies bottlenecks and suggests focus areas

- `generate_status` - Create status report from todos
  - Input: project_id (optional), time_range
  - Output: Markdown report of completed/pending work

**Smart Queries:**
- `find_related_tasks` - Keyword-based search
  - Input: query keywords
  - Searches descriptions, notes, tags
  - Returns ranked results by relevance
  - Implementation: Simple keyword matching (extensible to embeddings later)

- `suggest_priorities` - Recommend priority adjustments
  - Considers due dates (overdue → high)
  - Analyzes tag patterns (bugs → high, docs → medium)
  - Returns suggested changes (not automatic updates)

### Tool Quality Standards

Each tool definition includes:

1. **Rich Description** (2-3 sentences)
   - What it does
   - When to use it
   - Example use cases

2. **Strong Typing**
   - Strict JSONSchema for parameters
   - Enum values for choices
   - Clear required vs optional

3. **Usage Hints**
   - Parameter descriptions include examples
   - "Example: 'implement user authentication endpoint'"
   - Related tools mentioned

4. **Error Guidance**
   - Actionable error messages
   - Not just "invalid input"
   - Suggest corrections

5. **Success Context**
   - Return values documented
   - Next steps suggested
   - "Use this UUID to mark done later"

## Resources

Resources provide read-only data access for agents to browse state.

### Pre-built Resources (Common Views)

- `toki://projects` - List all projects with metadata
- `toki://todos` - All todos (paginated if large)
- `toki://todos/pending` - Only incomplete todos
- `toki://todos/overdue` - Todos past due date
- `toki://todos/high-priority` - High priority todos
- `toki://projects/{project-id}/todos` - Todos for specific project
- `toki://stats` - Summary statistics

### Query Resource (Custom Filters)

- `toki://query?project={id}&priority={level}&done={bool}&tag={name}`
- Supports combining multiple filters
- URL parameters map to same filters as `list_todos` tool
- Returns same structured format as pre-built resources

### Resource Format

All resources return JSON with:
```json
{
  "metadata": {
    "timestamp": "2025-11-29T10:00:00Z",
    "count": 42,
    "filters": {"priority": "high"}
  },
  "data": [
    // array of todos or projects
  ],
  "links": {
    // related resources where applicable
  }
}
```

### Why Both Pre-built and Query?

- Pre-built resources = fast access to common patterns
- Query resource = flexibility for custom combinations
- Agents can browse common views, drill down with queries

## Prompts

Prompts are workflow templates that guide agents on HOW to use toki effectively.

### Task Management Prompts

1. **`plan-project`** - Breaking down new projects
   - Understanding scope, identifying phases, creating initial tasks
   - Suggests using tags for phases, priorities for criticality
   - Template for organizing large efforts

2. **`daily-review`** - Daily standup workflow
   - Check overdue items, review high-priority pending
   - Identify blockers (long-pending todos)
   - Suggest focus for the day

3. **`sprint-planning`** - Organizing iterations
   - Review backlog of pending todos
   - Group by priority and tags
   - Create sprint goals from high-priority items

### Agent Integration Prompts

4. **`track-agent-work`** - How agents use toki for their tasks
   - **Key guideline:** Use toki for work visible to humans or other agents
   - Internal agent tracking stays internal (not in toki)
   - Example: Agent doing research creates todos for findings, not for each search query
   - Prevents toki from becoming cluttered with agent internals

5. **`coordinate-tasks`** - Multi-agent collaboration
   - How to assign work using tags/priorities
   - Checking for related work before starting
   - Updating status for visibility

6. **`report-status`** - Generating updates
   - Using `generate_status` tool effectively
   - Formatting reports for different audiences
   - Frequency guidance (daily vs weekly)

## Implementation Notes

### Technology Stack

- Go MCP SDK: https://github.com/modelcontextprotocol/go-sdk
- Existing toki database layer (no changes needed)
- Existing toki models (reuse as-is)

### Database Considerations

- MCP server and CLI share same SQLite database
- Concurrent access handled by SQLite's locking
- Read-heavy operations (resources) are safe
- Write operations (tools) use same transactions as CLI

### Testing Strategy

- Unit tests for each tool handler
- Integration tests for tool chains (add → list → mark done)
- Example MCP client interactions documented
- Prompt templates tested with sample scenarios
- Test that agents can discover and use tools from descriptions alone

### Error Handling

- All tools return structured errors with:
  - Error code (not_found, invalid_input, conflict)
  - Human-readable message
  - Suggested action
- Example: "Todo 'abc' not found. Use list_todos to see available todos."

### Future Extensions

- Embedding-based semantic search (upgrade from keyword search)
- WebSocket transport (in addition to stdio)
- Multi-user support (currently single-user SQLite)
- Webhooks for todo changes
- Integration with calendar systems for due dates

## Success Criteria

1. **Discoverability** - Agents can understand tools from descriptions alone
2. **Usability** - Common operations require minimal tool calls
3. **Flexibility** - Advanced workflows are possible
4. **Performance** - Resource queries return in <100ms for typical databases
5. **Reliability** - No data corruption from concurrent CLI/MCP access

## Non-Goals

- Replace agent-specific internal todo tracking
- Become a full project management system
- Support real-time collaboration (WebSocket, etc.) in v1
- Multi-user authentication/authorization
