// ABOUTME: MCP tool definitions and handlers
// ABOUTME: Provides CRUD operations and workflow tools for todos/projects

package mcp

import (
	"context"
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

func (s *Server) registerTools() {
	// Register add_todo tool
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
