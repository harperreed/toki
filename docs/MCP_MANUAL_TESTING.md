# MCP Server Manual Testing Guide

This guide provides step-by-step instructions for manually testing the Toki MCP server with real MCP clients.

## Prerequisites

- Toki installed and available in PATH: `toki --version`
- An MCP client (Claude Desktop, or another MCP-compatible client)
- A terminal for running commands

## Overview

The Toki MCP server provides:
- **11 Tools** for managing todos and projects
- **7 Resources** for querying data (projects, todos, pending, overdue, high-priority, query, stats)
- **6 Prompts** for workflow guidance

## Configuration

### Claude Desktop

Add the following to your Claude Desktop MCP configuration file:

**macOS/Linux:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

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

After editing, restart Claude Desktop.

### Other MCP Clients

For other MCP clients that support stdio transport:

```bash
# Start the server manually
toki mcp
```

The server will communicate via stdin/stdout using the JSON-RPC protocol.

## Testing Scenarios

### Scenario 1: Project Management

**Objective:** Create a project, add todos, and verify data persistence.

#### Steps:

1. **Create a project:**
   - Tool: `add_project`
   - Input:
     ```json
     {
       "name": "manual-test-project",
       "path": "/tmp/manual-test"
     }
     ```
   - Expected: Returns project object with UUID `id` field

2. **List projects:**
   - Tool: `list_projects`
   - Input: `{}`
   - Expected: Returns array containing "manual-test-project"

3. **Read projects resource:**
   - Resource: `toki://projects`
   - Expected: JSON with metadata, data array, and links
   - Verify: "manual-test-project" appears in data array

4. **Save the project ID** for next steps (e.g., `abc-123-def...`)

### Scenario 2: Todo Lifecycle

**Objective:** Create, update, complete, and delete a todo.

#### Steps:

1. **Create a todo:**
   - Tool: `add_todo`
   - Input:
     ```json
     {
       "description": "Test the MCP server integration",
       "project_id": "<project-id-from-scenario-1>",
       "priority": "high",
       "tags": ["testing", "integration"]
     }
     ```
   - Expected: Returns todo object with `id`, `done: false`, `priority: "high"`

2. **List todos (via tool):**
   - Tool: `list_todos`
   - Input:
     ```json
     {
       "project_id": "<project-id-from-scenario-1>",
       "done": false
     }
     ```
   - Expected: Returns todos array with count=1, containing our test todo

3. **Read all todos resource:**
   - Resource: `toki://todos`
   - Expected: Contains our test todo in the data array

4. **Read pending todos resource:**
   - Resource: `toki://todos/pending`
   - Expected: Contains our test todo in the data array

5. **Update the todo:**
   - Tool: `update_todo`
   - Input:
     ```json
     {
       "todo_id": "<todo-id-from-step-1>",
       "notes": "Testing update functionality",
       "priority": "medium"
     }
     ```
   - Expected: Returns updated todo with new notes and priority

6. **Mark todo as done:**
   - Tool: `mark_done`
   - Input:
     ```json
     {
       "todo_id": "<todo-id-from-step-1>"
     }
     ```
   - Expected: Returns todo with `done: true`

7. **Read all todos (should show done status):**
   - Resource: `toki://todos`
   - Expected: Contains our completed todo with `done: true`

8. **Mark todo as undone:**
   - Tool: `mark_undone`
   - Input:
     ```json
     {
       "todo_id": "<todo-id-from-step-1>"
     }
     ```
   - Expected: Returns todo with `done: false`

9. **Delete the todo:**
   - Tool: `delete_todo`
   - Input:
     ```json
     {
       "todo_id": "<todo-id-from-step-1>"
     }
     ```
   - Expected: Returns `{"success": true}`

10. **Verify deletion:**
   - Tool: `list_todos`
   - Input: `{}`
   - Expected: Todo no longer appears in results

### Scenario 3: Tags Management

**Objective:** Add and remove tags from todos.

#### Steps:

1. **Create a todo:**
   - Tool: `add_todo`
   - Input:
     ```json
     {
       "description": "Test tag functionality",
       "project_id": "<project-id>"
     }
     ```

2. **Add a tag:**
   - Tool: `add_tag_to_todo`
   - Input:
     ```json
     {
       "todo_id": "<todo-id>",
       "tag_name": "urgent"
     }
     ```
   - Expected: Returns todo with `tags: ["urgent"]`

3. **Add another tag:**
   - Tool: `add_tag_to_todo`
   - Input:
     ```json
     {
       "todo_id": "<todo-id>",
       "tag_name": "bug"
     }
     ```
   - Expected: Returns todo with `tags: ["urgent", "bug"]`

4. **Filter by tag:**
   - Tool: `list_todos`
   - Input:
     ```json
     {
       "tag": "urgent"
     }
     ```
   - Expected: Returns our tagged todo

5. **Remove a tag:**
   - Tool: `remove_tag_from_todo`
   - Input:
     ```json
     {
       "todo_id": "<todo-id>",
       "tag_name": "urgent"
     }
     ```
   - Expected: Returns todo with `tags: ["bug"]`

### Scenario 4: Overdue Todos

**Objective:** Test overdue todo tracking.

#### Steps:

1. **Create an overdue todo:**
   - Tool: `add_todo`
   - Input:
     ```json
     {
       "description": "Overdue task",
       "project_id": "<project-id>",
       "due_date": "2024-01-01T00:00:00Z"
     }
     ```
   - Expected: Returns todo with past due_date

2. **Read overdue resource:**
   - Resource: `toki://todos/overdue`
   - Expected: Contains our overdue todo

3. **Filter overdue via tool:**
   - Tool: `list_todos`
   - Input:
     ```json
     {
       "overdue": true
     }
     ```
   - Expected: Returns our overdue todo

### Scenario 5: Statistics

**Objective:** Verify stats resource calculation.

#### Steps:

1. **Create test data:**
   - Create 3 pending todos
   - Create 2 done todos (mark them done)
   - Create 1 overdue todo

2. **Read stats resource:**
   - Resource: `toki://stats`
   - Expected output structure:
     ```json
     {
       "metadata": {
         "resource_uri": "toki://stats",
         "count": 1
       },
       "data": {
         "total_todos": 6,
         "pending_todos": 4,
         "done_todos": 2,
         "overdue_todos": 1,
         "total_projects": 1
       }
     }
     ```

### Scenario 6: Prompts

**Objective:** Test workflow prompts.

#### Steps:

1. **Get plan-project prompt:**
   - Prompt: `plan-project`
   - Expected: Returns markdown with project planning workflow

2. **Get daily-review prompt:**
   - Prompt: `daily-review`
   - Expected: Returns markdown with daily review workflow

3. **Get sprint-planning prompt:**
   - Prompt: `sprint-planning`
   - Expected: Returns markdown with sprint planning workflow

4. **Get track-agent-work prompt:**
   - Prompt: `track-agent-work`
   - Expected: Returns markdown with agent work tracking workflow

5. **Get coordinate-tasks prompt:**
   - Prompt: `coordinate-tasks`
   - Expected: Returns markdown with task coordination workflow

6. **Get report-status prompt:**
   - Prompt: `report-status`
   - Expected: Returns markdown with status reporting workflow

### Scenario 7: Data Persistence

**Objective:** Verify data persists across server restarts.

#### Steps:

1. **Create project and todo** (via MCP tools)

2. **Stop the MCP server** (restart Claude Desktop or kill server process)

3. **Restart the MCP server**

4. **List todos:**
   - Tool: `list_todos`
   - Expected: Previously created todos still exist

5. **Verify via CLI:**
   ```bash
   toki list
   ```
   - Expected: Todos created via MCP appear in CLI output

6. **Create todo via CLI:**
   ```bash
   toki add "CLI-created todo" --project <project-id>
   ```

7. **Read via MCP resource:**
   - Resource: `toki://todos/all`
   - Expected: CLI-created todo appears in MCP results

### Scenario 8: Error Handling

**Objective:** Test proper error responses.

#### Steps:

1. **Invalid UUID:**
   - Tool: `mark_done`
   - Input: `{"todo_id": "not-a-uuid"}`
   - Expected: Error response with message about invalid UUID

2. **Non-existent todo:**
   - Tool: `mark_done`
   - Input: `{"todo_id": "00000000-0000-0000-0000-000000000000"}`
   - Expected: Error response with "not found" message

3. **Invalid priority:**
   - Tool: `add_todo`
   - Input:
     ```json
     {
       "description": "test",
       "priority": "super-urgent"
     }
     ```
   - Expected: Validation error (schema rejection)

4. **Missing required field:**
   - Tool: `add_todo`
   - Input: `{}`
   - Expected: Validation error about missing description

5. **Non-existent project:**
   - Tool: `add_todo`
   - Input:
     ```json
     {
       "description": "test",
       "project_id": "00000000-0000-0000-0000-000000000000"
     }
     ```
   - Expected: Error response about project not found

## Expected Results Summary

### Tool Calls

All successful tool calls should:
- Return JSON with expected structure
- Not throw protocol errors
- Persist data to database
- Return meaningful error messages for invalid input

### Resource Reads

All resource reads should:
- Return consistent JSON structure with `metadata`, `data`, and `links`
- Include accurate counts in metadata
- Return current data from database
- Update when data changes

### Prompts

All prompts should:
- Return markdown-formatted text
- Include workflow steps and guidance
- Be usable by AI agents

## Troubleshooting

### Server won't start

```bash
# Check if toki is in PATH
which toki

# Check version
toki --version

# Try running manually
toki mcp

# Check for error messages in Claude Desktop logs (if using Claude)
# macOS: ~/Library/Logs/Claude/mcp*.log
```

### Changes not persisting

```bash
# Check database location
toki config

# Verify database exists
ls -la ~/.config/toki/

# Check database directly
sqlite3 ~/.config/toki/toki.db "SELECT * FROM projects;"
```

### Tools not appearing

1. Verify MCP config is valid JSON
2. Restart MCP client completely
3. Check client logs for connection errors
4. Try running `toki mcp` manually to see startup errors

## Testing Checklist

Use this checklist to track your manual testing:

- [ ] Server starts without errors
- [ ] All 11 tools are discoverable
- [ ] All 7 resources are discoverable
- [ ] All 6 prompts are discoverable
- [ ] Can create project via `add_project`
- [ ] Can list projects via `list_projects`
- [ ] Can read `toki://projects` resource
- [ ] Can create todo via `add_todo`
- [ ] Can list todos via `list_todos`
- [ ] Can read `toki://todos` resource
- [ ] Can mark todo done via `mark_done`
- [ ] Can mark todo undone via `mark_undone`
- [ ] Can update todo via `update_todo`
- [ ] Can delete todo via `delete_todo`
- [ ] Can add tags via `add_tag_to_todo`
- [ ] Can remove tags via `remove_tag_from_todo`
- [ ] Can filter todos by tag
- [ ] Can filter todos by priority
- [ ] Can filter todos by project
- [ ] Can filter todos by done status
- [ ] Overdue todos appear in `toki://todos/overdue`
- [ ] Stats resource shows accurate counts
- [ ] All prompts return workflow text
- [ ] Data persists across server restarts
- [ ] CLI and MCP can access same data
- [ ] Invalid input returns proper errors
- [ ] Non-existent resources return errors

## Additional Notes

- Default database location: `~/.config/toki/toki.db`
- Server uses stdio transport (stdin/stdout)
- All timestamps are in RFC3339 format
- All IDs are UUIDs
- Tags are created automatically when first used

## Success Criteria

The MCP server is working correctly if:

1. All capabilities are discoverable
2. Tools create/modify data that persists
3. Resources reflect current database state
4. Prompts provide workflow guidance
5. CLI and MCP server can access the same database
6. Error handling returns clear messages
7. No protocol errors or crashes occur

## Reporting Issues

If you encounter problems:

1. Note which scenario/step failed
2. Capture the exact input used
3. Record the error message or unexpected behavior
4. Check server logs if available
5. Try reproducing with `toki mcp` running manually
6. Verify database state with CLI commands

Report issues with this information to help debugging.
