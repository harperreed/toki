// ABOUTME: MCP tool definitions and handlers
// ABOUTME: Provides CRUD operations and workflow tools for todos/projects

package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AddTodoInput defines the input parameters for the add_todo tool.
type AddTodoInput struct {
	Description string   `json:"description"`
	ProjectID   *string  `json:"project_id,omitempty"`
	Priority    *string  `json:"priority,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
	DueDate     *string  `json:"due_date,omitempty"`
}

// AddTodoOutput defines the output structure for the add_todo tool.
type AddTodoOutput struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Description string     `json:"description"`
	Done        bool       `json:"done"`
	Priority    *string    `json:"priority,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// ListTodosInput defines the input parameters for the list_todos tool.
type ListTodosInput struct {
	ProjectID *string `json:"project_id,omitempty"`
	Done      *bool   `json:"done,omitempty"`
	Priority  *string `json:"priority,omitempty"`
	Tag       *string `json:"tag,omitempty"`
	Overdue   *bool   `json:"overdue,omitempty"`
}

// TodoOutput represents a single todo in list output.
type TodoOutput struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Description string     `json:"description"`
	Done        bool       `json:"done"`
	Priority    *string    `json:"priority,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// ListTodosOutput defines the output structure for the list_todos tool.
type ListTodosOutput struct {
	Todos   []TodoOutput   `json:"todos"`
	Count   int            `json:"count"`
	Filters map[string]any `json:"filters"`
}

func (s *Server) registerTools() {
	s.registerAddTodoTool()
	s.registerListTodosTool()
	s.registerMarkDoneTool()
	s.registerMarkUndoneTool()
	s.registerDeleteTodoTool()
	s.registerUpdateTodoTool()
	s.registerAddTagToTodoTool()
	s.registerRemoveTagFromTodoTool()
	s.registerAddProjectTool()
	s.registerListProjectsTool()
	s.registerDeleteProjectTool()
}

func (s *Server) registerAddTodoTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "add_todo",
		Description: `Create a new todo item with optional metadata like priority, tags, and due date. Use this when you need to track a new task or action item. This tool handles everything from quick one-line tasks to complex todos with full context, deadlines, and categorization. Returns the created todo with a UUID that you can use to update, tag, or mark it done later.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Brief description of the task. Example: 'implement user authentication endpoint'",
				},
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "UUID of the project this todo belongs to. If not provided, uses the default project. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"low", "medium", "high"},
					"description": "Priority level of the task. Must be one of: low, medium, high. Example: 'high'",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "List of tags to categorize the todo. Example: ['bug', 'urgent', 'backend']",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Additional context, details, or notes about the task. Example: 'User reported login failure on mobile app'",
				},
				"due_date": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "Due date in ISO 8601 format. Example: '2025-12-01T15:04:05Z'",
				},
			},
			"required": []string{"description"},
		},
	}, s.handleAddTodo)
}

func (s *Server) registerListTodosTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "list_todos",
		Description: `Retrieve todos with powerful filtering capabilities. Filter by project, completion status, priority, tags, or due date. All filters are optional and can be combined for precise queries. Use this to view your task list, find specific todos, or generate reports. Returns an array of todos with full metadata, count, and applied filters for context.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Filter by project UUID. Only todos from this project will be returned. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
				"done": map[string]interface{}{
					"type":        "boolean",
					"description": "Filter by completion status. true = completed todos only, false = incomplete todos only. Omit to see all todos. Example: false",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"low", "medium", "high"},
					"description": "Filter by priority level. Must be one of: low, medium, high. Example: 'high'",
				},
				"tag": map[string]interface{}{
					"type":        "string",
					"description": "Filter by tag name. Only todos with this exact tag will be returned. Example: 'bug'",
				},
				"overdue": map[string]interface{}{
					"type":        "boolean",
					"description": "Filter by overdue status. true = only overdue todos (due date in the past), false = only non-overdue todos. Example: true",
				},
			},
		},
	}, s.handleListTodos)
}

func (s *Server) handleAddTodo(_ context.Context, req *mcp.CallToolRequest, input AddTodoInput) (*mcp.CallToolResult, AddTodoOutput, error) {
	projectID, err := s.resolveProjectID(input.ProjectID)
	if err != nil {
		return nil, AddTodoOutput{}, err
	}

	if err := validatePriority(input.Priority); err != nil {
		return nil, AddTodoOutput{}, err
	}

	dueDate, err := parseDueDate(input.DueDate)
	if err != nil {
		return nil, AddTodoOutput{}, err
	}

	todo, err := s.createTodoWithTags(projectID, input, dueDate)
	if err != nil {
		return nil, AddTodoOutput{}, err
	}

	return buildAddTodoResult(todo, input.Tags, dueDate)
}

func (s *Server) resolveProjectID(projectIDStr *string) (uuid.UUID, error) {
	if projectIDStr != nil && *projectIDStr != "" {
		return s.parseAndVerifyProjectID(*projectIDStr)
	}
	return s.getOrCreateDefaultProject()
}

func (s *Server) parseAndVerifyProjectID(projectIDStr string) (uuid.UUID, error) {
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid project_id: must be a valid UUID. Use the project's full UUID. Error: %w", err)
	}

	_, err = db.GetProjectByID(s.db, projectID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("project not found: no project exists with ID '%s'. Create the project first or omit project_id to use default", projectID)
	}

	return projectID, nil
}

func (s *Server) getOrCreateDefaultProject() (uuid.UUID, error) {
	project, err := db.GetProjectByName(s.db, "default")
	if err == nil {
		return project.ID, nil
	}

	project = models.NewProject("default", nil)
	if err := db.CreateProject(s.db, project); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create default project: %w", err)
	}
	return project.ID, nil
}

func validatePriority(priority *string) error {
	if priority == nil {
		return nil
	}
	validPriorities := map[string]bool{"low": true, "medium": true, "high": true}
	if !validPriorities[*priority] {
		return fmt.Errorf("invalid priority '%s': must be one of 'low', 'medium', or 'high'", *priority)
	}
	return nil
}

func parseDueDate(dueDateStr *string) (*time.Time, error) {
	if dueDateStr == nil || *dueDateStr == "" {
		return nil, nil //nolint:nilnil // nil pointer is valid for optional due_date
	}
	parsed, err := time.Parse(time.RFC3339, *dueDateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid due_date format: must be ISO 8601 (RFC3339). Example: '2025-12-01T15:04:05Z'. Error: %w", err)
	}
	return &parsed, nil
}

func (s *Server) createTodoWithTags(projectID uuid.UUID, input AddTodoInput, dueDate *time.Time) (*models.Todo, error) {
	todo := models.NewTodo(projectID, input.Description)
	todo.Priority = input.Priority
	todo.Notes = input.Notes
	todo.DueDate = dueDate

	if err := db.CreateTodo(s.db, todo); err != nil {
		return nil, fmt.Errorf("failed to create todo: %w", err)
	}

	for _, tag := range input.Tags {
		if err := db.AddTagToTodo(s.db, todo.ID, tag); err != nil {
			return nil, fmt.Errorf("failed to add tag '%s': %w", tag, err)
		}
	}

	return todo, nil
}

func buildAddTodoResult(todo *models.Todo, tags []string, dueDate *time.Time) (*mcp.CallToolResult, AddTodoOutput, error) {
	output := AddTodoOutput{
		ID:          todo.ID.String(),
		ProjectID:   todo.ProjectID.String(),
		Description: todo.Description,
		Done:        todo.Done,
		Priority:    todo.Priority,
		Notes:       todo.Notes,
		Tags:        tags,
		CreatedAt:   todo.CreatedAt,
		DueDate:     dueDate,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, output, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
	}, output, nil
}

func (s *Server) handleListTodos(_ context.Context, req *mcp.CallToolRequest, input ListTodosInput) (*mcp.CallToolResult, ListTodosOutput, error) {
	projectID, err := s.resolveOptionalProjectID(input.ProjectID)
	if err != nil {
		return nil, ListTodosOutput{}, err
	}

	if err := validatePriority(input.Priority); err != nil {
		return nil, ListTodosOutput{}, err
	}

	todos, err := s.fetchFilteredTodos(projectID, input)
	if err != nil {
		return nil, ListTodosOutput{}, err
	}

	return buildListTodosResult(s.db, todos, input)
}

func (s *Server) resolveOptionalProjectID(projectIDStr *string) (*uuid.UUID, error) {
	if projectIDStr == nil || *projectIDStr == "" {
		return nil, nil //nolint:nilnil // nil means no filter
	}

	projectID, err := uuid.Parse(*projectIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid project_id: must be a valid UUID. Error: %w", err)
	}

	return &projectID, nil
}

func (s *Server) fetchFilteredTodos(projectID *uuid.UUID, input ListTodosInput) ([]*models.Todo, error) {
	todos, err := db.ListTodos(s.db, projectID, input.Done, input.Priority, input.Tag)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos: %w", err)
	}

	if input.Overdue != nil && *input.Overdue {
		todos = filterOverdueTodos(todos)
	} else if input.Overdue != nil && !*input.Overdue {
		todos = filterNonOverdueTodos(todos)
	}

	return todos, nil
}

func filterOverdueTodos(todos []*models.Todo) []*models.Todo {
	now := time.Now()
	var filtered []*models.Todo
	for _, todo := range todos {
		if todo.DueDate != nil && todo.DueDate.Before(now) && !todo.Done {
			filtered = append(filtered, todo)
		}
	}
	return filtered
}

func filterNonOverdueTodos(todos []*models.Todo) []*models.Todo {
	now := time.Now()
	var filtered []*models.Todo
	for _, todo := range todos {
		if todo.DueDate == nil || !todo.DueDate.Before(now) || todo.Done {
			filtered = append(filtered, todo)
		}
	}
	return filtered
}

func buildListTodosResult(database *sql.DB, todos []*models.Todo, input ListTodosInput) (*mcp.CallToolResult, ListTodosOutput, error) {
	todoOutputs := make([]TodoOutput, 0, len(todos))

	for _, todo := range todos {
		tags, err := db.GetTodoTags(database, todo.ID)
		if err != nil {
			return nil, ListTodosOutput{}, fmt.Errorf("failed to get tags for todo %s: %w", todo.ID, err)
		}

		tagNames := make([]string, len(tags))
		for i, tag := range tags {
			tagNames[i] = tag.Name
		}

		todoOutputs = append(todoOutputs, TodoOutput{
			ID:          todo.ID.String(),
			ProjectID:   todo.ProjectID.String(),
			Description: todo.Description,
			Done:        todo.Done,
			Priority:    todo.Priority,
			Notes:       todo.Notes,
			Tags:        tagNames,
			CreatedAt:   todo.CreatedAt,
			UpdatedAt:   todo.UpdatedAt,
			DueDate:     todo.DueDate,
		})
	}

	appliedFilters := buildAppliedFilters(input)

	output := ListTodosOutput{
		Todos:   todoOutputs,
		Count:   len(todoOutputs),
		Filters: appliedFilters,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, output, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
	}, output, nil
}

func buildAppliedFilters(input ListTodosInput) map[string]any {
	filters := make(map[string]any)

	if input.ProjectID != nil && *input.ProjectID != "" {
		filters["project_id"] = *input.ProjectID
	}
	if input.Done != nil {
		filters["done"] = *input.Done
	}
	if input.Priority != nil && *input.Priority != "" {
		filters["priority"] = *input.Priority
	}
	if input.Tag != nil && *input.Tag != "" {
		filters["tag"] = *input.Tag
	}
	if input.Overdue != nil {
		filters["overdue"] = *input.Overdue
	}

	return filters
}

// MarkDoneInput defines the input parameters for the mark_done tool.
type MarkDoneInput struct {
	TodoID string `json:"todo_id"`
}

func (s *Server) registerMarkDoneTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "mark_done",
		Description: `Mark a todo as complete by its UUID. Use this when a task is finished to track completion and update status. The todo will be marked as done with a completion timestamp. Returns the updated todo with all metadata so you can verify the change. To find the UUID of a todo, use list_todos first.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"todo_id": map[string]interface{}{
					"type":        "string",
					"description": "Full UUID of the todo to mark as complete. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
			},
			"required": []string{"todo_id"},
		},
	}, s.handleMarkDone)
}

func (s *Server) handleMarkDone(_ context.Context, req *mcp.CallToolRequest, input MarkDoneInput) (*mcp.CallToolResult, TodoOutput, error) {
	todoID, err := uuid.Parse(input.TodoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("invalid todo_id: must be a valid UUID. Error: %w", err)
	}

	todo, err := db.GetTodoByID(s.db, todoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("todo not found: no todo exists with ID '%s'. Use list_todos to see available todos", input.TodoID)
	}

	todo.MarkDone()
	if err := db.UpdateTodo(s.db, todo); err != nil {
		return nil, TodoOutput{}, fmt.Errorf("failed to update todo: %w", err)
	}

	return buildTodoResult(s.db, todo)
}

// MarkUndoneInput defines the input parameters for the mark_undone tool.
type MarkUndoneInput struct {
	TodoID string `json:"todo_id"`
}

func (s *Server) registerMarkUndoneTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "mark_undone",
		Description: `Reopen a completed todo by marking it as incomplete. Use this when a task needs to be revisited or wasn't actually finished. The todo will be marked as not done and the completion timestamp will be cleared. Returns the updated todo with all metadata. To find the UUID of a todo, use list_todos with done=true to see completed todos.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"todo_id": map[string]interface{}{
					"type":        "string",
					"description": "Full UUID of the todo to reopen. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
			},
			"required": []string{"todo_id"},
		},
	}, s.handleMarkUndone)
}

func (s *Server) handleMarkUndone(_ context.Context, req *mcp.CallToolRequest, input MarkUndoneInput) (*mcp.CallToolResult, TodoOutput, error) {
	todoID, err := uuid.Parse(input.TodoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("invalid todo_id: must be a valid UUID. Error: %w", err)
	}

	todo, err := db.GetTodoByID(s.db, todoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("todo not found: no todo exists with ID '%s'. Use list_todos to see available todos", input.TodoID)
	}

	todo.MarkUndone()
	if err := db.UpdateTodo(s.db, todo); err != nil {
		return nil, TodoOutput{}, fmt.Errorf("failed to update todo: %w", err)
	}

	return buildTodoResult(s.db, todo)
}

// buildTodoResult builds a TodoOutput from a todo model.
func buildTodoResult(database *sql.DB, todo *models.Todo) (*mcp.CallToolResult, TodoOutput, error) {
	tags, err := db.GetTodoTags(database, todo.ID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("failed to get tags: %w", err)
	}

	tagNames := make([]string, len(tags))
	for i, tag := range tags {
		tagNames[i] = tag.Name
	}

	output := TodoOutput{
		ID:          todo.ID.String(),
		ProjectID:   todo.ProjectID.String(),
		Description: todo.Description,
		Done:        todo.Done,
		Priority:    todo.Priority,
		Notes:       todo.Notes,
		Tags:        tagNames,
		CreatedAt:   todo.CreatedAt,
		UpdatedAt:   todo.UpdatedAt,
		DueDate:     todo.DueDate,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, output, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
	}, output, nil
}

// DeleteTodoInput defines the input parameters for the delete_todo tool.
type DeleteTodoInput struct {
	TodoID string `json:"todo_id"`
}

// DeleteTodoOutput defines the output structure for the delete_todo tool.
type DeleteTodoOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	TodoID  string `json:"todo_id"`
}

func (s *Server) registerDeleteTodoTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "delete_todo",
		Description: `Permanently delete a todo by its UUID. Use this when a task is no longer relevant or was created by mistake. This action cannot be undone. The todo and all its associations (tags, etc.) will be removed from the database. Returns success confirmation with the deleted todo's ID. To find the UUID of a todo, use list_todos first.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"todo_id": map[string]interface{}{
					"type":        "string",
					"description": "Full UUID of the todo to delete permanently. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
			},
			"required": []string{"todo_id"},
		},
	}, s.handleDeleteTodo)
}

func (s *Server) handleDeleteTodo(_ context.Context, req *mcp.CallToolRequest, input DeleteTodoInput) (*mcp.CallToolResult, DeleteTodoOutput, error) {
	todoID, err := uuid.Parse(input.TodoID)
	if err != nil {
		return nil, DeleteTodoOutput{}, fmt.Errorf("invalid todo_id: must be a valid UUID. Error: %w", err)
	}

	// Check if todo exists first
	_, err = db.GetTodoByID(s.db, todoID)
	if err != nil {
		return nil, DeleteTodoOutput{}, fmt.Errorf("todo not found: no todo exists with ID '%s'. Use list_todos to see available todos", input.TodoID)
	}

	if err := db.DeleteTodo(s.db, todoID); err != nil {
		return nil, DeleteTodoOutput{}, fmt.Errorf("failed to delete todo: %w", err)
	}

	output := DeleteTodoOutput{
		Success: true,
		Message: fmt.Sprintf("Todo '%s' successfully deleted", input.TodoID),
		TodoID:  input.TodoID,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, output, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
	}, output, nil
}

// UpdateTodoInput defines the input parameters for the update_todo tool.
type UpdateTodoInput struct {
	TodoID      string  `json:"todo_id"`
	Description *string `json:"description,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
}

func (s *Server) registerUpdateTodoTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "update_todo",
		Description: `Update a todo's metadata including description, priority, notes, and due date. All update fields are optional - only provide the fields you want to change. Use this for modifying existing todos without recreating them. Returns the updated todo with all metadata. To find the UUID of a todo, use list_todos first.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"todo_id": map[string]interface{}{
					"type":        "string",
					"description": "Full UUID of the todo to update. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "New description for the todo. Example: 'implement user authentication with OAuth'",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"low", "medium", "high"},
					"description": "New priority level. Must be one of: low, medium, high. Example: 'high'",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "New notes or additional context. Example: 'Reviewed with team, needs to support Google and GitHub'",
				},
				"due_date": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "New due date in ISO 8601 format. Example: '2025-12-15T15:04:05Z'",
				},
			},
			"required": []string{"todo_id"},
		},
	}, s.handleUpdateTodo)
}

func (s *Server) handleUpdateTodo(_ context.Context, req *mcp.CallToolRequest, input UpdateTodoInput) (*mcp.CallToolResult, TodoOutput, error) {
	todoID, err := uuid.Parse(input.TodoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("invalid todo_id: must be a valid UUID. Error: %w", err)
	}

	todo, err := db.GetTodoByID(s.db, todoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("todo not found: no todo exists with ID '%s'. Use list_todos to see available todos", input.TodoID)
	}

	// Validate priority if provided
	if err := validatePriority(input.Priority); err != nil {
		return nil, TodoOutput{}, err
	}

	// Parse due date if provided
	var dueDate *time.Time
	if input.DueDate != nil {
		dueDate, err = parseDueDate(input.DueDate)
		if err != nil {
			return nil, TodoOutput{}, err
		}
	}

	// Update fields if provided
	if input.Description != nil {
		todo.Description = *input.Description
	}
	if input.Priority != nil {
		todo.Priority = input.Priority
	}
	if input.Notes != nil {
		todo.Notes = input.Notes
	}
	if input.DueDate != nil {
		todo.DueDate = dueDate
	}

	// Update the timestamp
	todo.UpdatedAt = time.Now()

	if err := db.UpdateTodo(s.db, todo); err != nil {
		return nil, TodoOutput{}, fmt.Errorf("failed to update todo: %w", err)
	}

	return buildTodoResult(s.db, todo)
}

// AddTagToTodoInput defines the input parameters for the add_tag_to_todo tool.
type AddTagToTodoInput struct {
	TodoID  string `json:"todo_id"`
	TagName string `json:"tag_name"`
}

func (s *Server) registerAddTagToTodoTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "add_tag_to_todo",
		Description: `Associate a tag with a todo for categorization and filtering. Tags are labels that help organize and find related todos. If the tag doesn't exist, it will be created automatically. A todo can have multiple tags. Returns the updated todo with all its tags. Use list_todos with tag filter to find todos by tag.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"todo_id": map[string]interface{}{
					"type":        "string",
					"description": "Full UUID of the todo to tag. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
				"tag_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the tag to add. Tags are case-sensitive. Example: 'bug', 'urgent', 'backend'",
				},
			},
			"required": []string{"todo_id", "tag_name"},
		},
	}, s.handleAddTagToTodo)
}

func (s *Server) handleAddTagToTodo(_ context.Context, req *mcp.CallToolRequest, input AddTagToTodoInput) (*mcp.CallToolResult, TodoOutput, error) {
	todoID, err := uuid.Parse(input.TodoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("invalid todo_id: must be a valid UUID. Error: %w", err)
	}

	// Check if todo exists
	todo, err := db.GetTodoByID(s.db, todoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("todo not found: no todo exists with ID '%s'. Use list_todos to see available todos", input.TodoID)
	}

	if err := db.AddTagToTodo(s.db, todoID, input.TagName); err != nil {
		return nil, TodoOutput{}, fmt.Errorf("failed to add tag: %w", err)
	}

	return buildTodoResult(s.db, todo)
}

// RemoveTagFromTodoInput defines the input parameters for the remove_tag_from_todo tool.
type RemoveTagFromTodoInput struct {
	TodoID  string `json:"todo_id"`
	TagName string `json:"tag_name"`
}

func (s *Server) registerRemoveTagFromTodoTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "remove_tag_from_todo",
		Description: `Remove a tag association from a todo. Use this when a tag no longer applies to a task. This only removes the association between the tag and the todo - the tag itself remains in the system for use with other todos. Returns the updated todo with remaining tags. If the tag wasn't associated with the todo, the operation succeeds silently.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"todo_id": map[string]interface{}{
					"type":        "string",
					"description": "Full UUID of the todo to untag. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
				"tag_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the tag to remove. Tags are case-sensitive. Example: 'bug', 'urgent'",
				},
			},
			"required": []string{"todo_id", "tag_name"},
		},
	}, s.handleRemoveTagFromTodo)
}

func (s *Server) handleRemoveTagFromTodo(_ context.Context, req *mcp.CallToolRequest, input RemoveTagFromTodoInput) (*mcp.CallToolResult, TodoOutput, error) {
	todoID, err := uuid.Parse(input.TodoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("invalid todo_id: must be a valid UUID. Error: %w", err)
	}

	// Check if todo exists
	todo, err := db.GetTodoByID(s.db, todoID)
	if err != nil {
		return nil, TodoOutput{}, fmt.Errorf("todo not found: no todo exists with ID '%s'. Use list_todos to see available todos", input.TodoID)
	}

	if err := db.RemoveTagFromTodo(s.db, todoID, input.TagName); err != nil {
		return nil, TodoOutput{}, fmt.Errorf("failed to remove tag: %w", err)
	}

	return buildTodoResult(s.db, todo)
}

// AddProjectInput defines the input parameters for the add_project tool.
type AddProjectInput struct {
	Name string  `json:"name"`
	Path *string `json:"path,omitempty"`
}

// ProjectOutput defines the output structure for project operations.
type ProjectOutput struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      *string   `json:"path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) registerAddProjectTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "add_project",
		Description: `Create a new project to organize todos. Projects are containers for related tasks and can optionally be associated with a git repository path. Use this to separate work for different codebases, clients, or areas of focus. Returns the created project with a UUID that you can use when adding todos. All projects can be listed with list_projects.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the project. Should be unique and descriptive. Example: 'backend-api', 'mobile-app', 'client-acme'",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Optional filesystem path to git repository or project directory. Example: '/home/user/projects/backend-api'",
				},
			},
			"required": []string{"name"},
		},
	}, s.handleAddProject)
}

func (s *Server) handleAddProject(_ context.Context, req *mcp.CallToolRequest, input AddProjectInput) (*mcp.CallToolResult, ProjectOutput, error) {
	project := models.NewProject(input.Name, input.Path)
	if err := db.CreateProject(s.db, project); err != nil {
		return nil, ProjectOutput{}, fmt.Errorf("failed to create project: %w", err)
	}

	output := ProjectOutput{
		ID:        project.ID.String(),
		Name:      project.Name,
		Path:      project.DirectoryPath,
		CreatedAt: project.CreatedAt,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, output, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
	}, output, nil
}

// ListProjectsInput defines the input parameters for the list_projects tool (empty).
type ListProjectsInput struct{}

// ListProjectsOutput defines the output structure for the list_projects tool.
type ListProjectsOutput struct {
	Projects []ProjectOutput `json:"projects"`
	Count    int             `json:"count"`
}

func (s *Server) registerListProjectsTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "list_projects",
		Description: `Retrieve all projects in the system. Projects organize todos into logical groups and can be associated with git repositories. Use this to see available projects before adding todos or to get project UUIDs for filtering. Returns an array of projects sorted by name with full metadata including IDs, names, paths, and creation timestamps.`,
		InputSchema: map[string]interface{}{
			"type": "object",
		},
	}, s.handleListProjects)
}

func (s *Server) handleListProjects(_ context.Context, req *mcp.CallToolRequest, input ListProjectsInput) (*mcp.CallToolResult, ListProjectsOutput, error) {
	projects, err := db.ListProjects(s.db)
	if err != nil {
		return nil, ListProjectsOutput{}, fmt.Errorf("failed to list projects: %w", err)
	}

	projectOutputs := make([]ProjectOutput, 0, len(projects))
	for _, project := range projects {
		projectOutputs = append(projectOutputs, ProjectOutput{
			ID:        project.ID.String(),
			Name:      project.Name,
			Path:      project.DirectoryPath,
			CreatedAt: project.CreatedAt,
		})
	}

	output := ListProjectsOutput{
		Projects: projectOutputs,
		Count:    len(projectOutputs),
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, output, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
	}, output, nil
}

// DeleteProjectInput defines the input parameters for the delete_project tool.
type DeleteProjectInput struct {
	ProjectID string `json:"project_id"`
}

// DeleteProjectOutput defines the output structure for the delete_project tool.
type DeleteProjectOutput struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ProjectID string `json:"project_id"`
}

func (s *Server) registerDeleteProjectTool() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "delete_project",
		Description: `Permanently delete a project by its UUID. This action cannot be undone. WARNING: Deleting a project will also delete all associated todos due to database CASCADE constraints. Use this when a project is no longer needed and you want to clean up all related tasks. Returns success confirmation. To find the UUID of a project, use list_projects first.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Full UUID of the project to delete. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
			},
			"required": []string{"project_id"},
		},
	}, s.handleDeleteProject)
}

func (s *Server) handleDeleteProject(_ context.Context, req *mcp.CallToolRequest, input DeleteProjectInput) (*mcp.CallToolResult, DeleteProjectOutput, error) {
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, DeleteProjectOutput{}, fmt.Errorf("invalid project_id: must be a valid UUID. Error: %w", err)
	}

	// Check if project exists first
	_, err = db.GetProjectByID(s.db, projectID)
	if err != nil {
		return nil, DeleteProjectOutput{}, fmt.Errorf("project not found: no project exists with ID '%s'. Use list_projects to see available projects", input.ProjectID)
	}

	if err := db.DeleteProject(s.db, projectID); err != nil {
		return nil, DeleteProjectOutput{}, fmt.Errorf("failed to delete project: %w", err)
	}

	output := DeleteProjectOutput{
		Success:   true,
		Message:   fmt.Sprintf("Project '%s' and all associated todos successfully deleted", input.ProjectID),
		ProjectID: input.ProjectID,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, output, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
	}, output, nil
}
