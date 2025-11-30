// ABOUTME: MCP resource providers
// ABOUTME: Exposes read-only views of projects, todos, and stats

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harper/toki/internal/db"
	"github.com/harper/toki/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResourceData is the standard response format for all resources.
type ResourceData struct {
	Metadata ResourceMetadata  `json:"metadata"`
	Data     interface{}       `json:"data"`
	Links    map[string]string `json:"links"`
}

// ResourceMetadata contains metadata about the resource response.
type ResourceMetadata struct {
	Timestamp   time.Time      `json:"timestamp"`
	Count       int            `json:"count"`
	ResourceURI string         `json:"resource_uri"`
	Filters     map[string]any `json:"filters,omitempty"`
}

func (s *Server) registerResources() {
	// Pre-built resources (common views)
	s.registerProjectsResource()
	s.registerTodosResource()
	s.registerTodosPendingResource()
	s.registerTodosOverdueResource()
	s.registerTodosHighPriorityResource()

	// Query resource (custom filters)
	s.registerQueryResource()

	// Statistics and analytics
	s.registerStatsResource()

	// TODO(v2): Project-specific todos resource (toki://projects/{project-id}/todos)
	// The MCP Go SDK v1.1.0 doesn't support URI templates for resources with path parameters.
	// This would require matching URIs like toki://projects/abc123.../todos and extracting
	// the project-id parameter. For v1, use the list_todos tool with project_id parameter,
	// or filter all todos client-side. Future SDK versions may add URI template support.
}

func (s *Server) registerProjectsResource() {
	s.mcp.AddResource(&mcp.Resource{
		URI:         "toki://projects",
		Name:        "All Projects",
		Description: "List all projects with metadata including name, directory path, and creation time",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		projects, err := db.ListProjects(s.db)
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}

		// Convert to output format
		projectOutputs := make([]map[string]interface{}, 0, len(projects))
		for _, proj := range projects {
			output := map[string]interface{}{
				"id":         proj.ID.String(),
				"name":       proj.Name,
				"created_at": proj.CreatedAt,
			}
			if proj.DirectoryPath != nil {
				output["directory_path"] = *proj.DirectoryPath
			}
			projectOutputs = append(projectOutputs, output)
		}

		resourceData := ResourceData{
			Metadata: ResourceMetadata{
				Timestamp:   time.Now(),
				Count:       len(projectOutputs),
				ResourceURI: "toki://projects",
			},
			Data: projectOutputs,
			Links: map[string]string{
				"todos": "toki://todos",
			},
		}

		jsonBytes, err := json.MarshalIndent(resourceData, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource data: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonBytes),
				},
			},
		}, nil
	})
}

func (s *Server) registerTodosResource() {
	s.mcp.AddResource(&mcp.Resource{
		URI:         "toki://todos",
		Name:        "All Todos",
		Description: "List all todos across all projects, both pending and completed. Use filtered views like toki://todos/pending for specific subsets.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return s.handleTodoResource(ctx, req, nil, nil, nil, nil, false)
	})
}

func (s *Server) registerTodosPendingResource() {
	s.mcp.AddResource(&mcp.Resource{
		URI:         "toki://todos/pending",
		Name:        "Pending Todos",
		Description: "List all incomplete (not done) todos across all projects. Useful for seeing active work items.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		done := false
		return s.handleTodoResource(ctx, req, nil, &done, nil, nil, false)
	})
}

func (s *Server) registerTodosOverdueResource() {
	s.mcp.AddResource(&mcp.Resource{
		URI:         "toki://todos/overdue",
		Name:        "Overdue Todos",
		Description: "List all todos that are past their due date and not yet completed. Critical items requiring immediate attention.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return s.handleTodoResource(ctx, req, nil, nil, nil, nil, true)
	})
}

func (s *Server) registerTodosHighPriorityResource() {
	s.mcp.AddResource(&mcp.Resource{
		URI:         "toki://todos/high-priority",
		Name:        "High Priority Todos",
		Description: "List all todos marked with high priority, regardless of completion status. Important work items that need focus.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		priority := "high" //nolint:goconst // Priority values are data, not constants
		return s.handleTodoResource(ctx, req, nil, nil, &priority, nil, false)
	})
}

func (s *Server) registerQueryResource() {
	// TODO(v2): The MCP Go SDK v1.1.0 doesn't support URI templates for resources.
	// This means we can't register `toki://query{?params}` as a single resource
	// that matches query://query?foo=bar URLs.
	//
	// For v1, we register the base toki://query resource which returns all todos.
	// Future versions could:
	// 1. Use a custom resource matcher (if SDK adds support)
	// 2. Register multiple specific query combinations
	// 3. Move complex queries to Tools instead of Resources
	//
	// For now, use pre-built resources (pending, overdue, high-priority) for
	// common filtered views, and use the list_todos Tool for custom queries.
	s.mcp.AddResource(&mcp.Resource{
		URI:         "toki://query",
		Name:        "All Todos (Query Base)",
		Description: "Returns all todos. In v1, use pre-built resources (toki://todos/pending, toki://todos/overdue, toki://todos/high-priority) for filtered views, or use the list_todos tool for custom filtering.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		// v1: Just return all todos
		// For filtered queries, clients should use pre-built resources or the list_todos tool
		return s.handleTodoResource(ctx, req, nil, nil, nil, nil, false)
	})
}

//nolint:funlen // Resource handler combines data fetching, transformation, and formatting
func (s *Server) handleTodoResource(
	_ context.Context, //nolint:unparam // ctx reserved for future use
	req *mcp.ReadResourceRequest,
	projectID *uuid.UUID, //nolint:unparam // v1: always nil, v2 will use for project filtering
	done *bool,
	priority *string,
	tag *string, //nolint:unparam // v1: always nil, v2 will use for tag filtering
	overdue bool,
) (*mcp.ReadResourceResult, error) {
	// Fetch and filter todos
	todos, err := s.fetchAndFilterTodos(projectID, done, priority, tag, overdue)
	if err != nil {
		return nil, err
	}

	// Convert to output format
	todoOutputs, err := s.buildTodoOutputs(todos)
	if err != nil {
		return nil, err
	}

	// Build response metadata
	filters := buildFiltersMetadata(projectID, done, priority, tag, overdue)
	links := s.buildTodoResourceLinks(projectID, done, priority, tag, overdue)

	resourceData := ResourceData{
		Metadata: ResourceMetadata{
			Timestamp:   time.Now(),
			Count:       len(todoOutputs),
			ResourceURI: req.Params.URI,
			Filters:     filters,
		},
		Data:  todoOutputs,
		Links: links,
	}

	jsonBytes, err := json.MarshalIndent(resourceData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource data: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonBytes),
			},
		},
	}, nil
}

// fetchAndFilterTodos retrieves todos from database and applies filters.
func (s *Server) fetchAndFilterTodos(
	projectID *uuid.UUID,
	done *bool,
	priority *string,
	tag *string,
	overdue bool,
) ([]*models.Todo, error) {
	todos, err := db.ListTodos(s.db, projectID, done, priority, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos: %w", err)
	}

	if overdue {
		todos = filterOverdueTodos(todos)
	}

	return todos, nil
}

// buildTodoOutputs converts todos to JSON-serializable format.
func (s *Server) buildTodoOutputs(todos []*models.Todo) ([]map[string]interface{}, error) {
	todoOutputs := make([]map[string]interface{}, 0, len(todos))

	for _, todo := range todos {
		tags, err := db.GetTodoTags(s.db, todo.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags: %w", err)
		}

		tagNames := make([]string, 0, len(tags))
		for _, t := range tags {
			tagNames = append(tagNames, t.Name)
		}

		output := map[string]interface{}{
			"id":          todo.ID.String(),
			"project_id":  todo.ProjectID.String(),
			"description": todo.Description,
			"done":        todo.Done,
			"created_at":  todo.CreatedAt,
			"tags":        tagNames,
		}

		if todo.Priority != nil {
			output["priority"] = *todo.Priority
		}
		if todo.Notes != nil {
			output["notes"] = *todo.Notes
		}
		if todo.CompletedAt != nil {
			output["completed_at"] = *todo.CompletedAt
		}
		if todo.DueDate != nil {
			output["due_date"] = *todo.DueDate
		}

		todoOutputs = append(todoOutputs, output)
	}

	return todoOutputs, nil
}

// buildFiltersMetadata creates the filters map for resource metadata.
func buildFiltersMetadata(
	projectID *uuid.UUID,
	done *bool,
	priority *string,
	tag *string,
	overdue bool,
) map[string]any {
	filters := make(map[string]any)

	if projectID != nil {
		filters["project_id"] = projectID.String()
	}
	if done != nil {
		filters["done"] = *done
	}
	if priority != nil {
		filters["priority"] = *priority
	}
	if tag != nil {
		filters["tag"] = *tag
	}
	if overdue {
		filters["overdue"] = true
	}

	return filters
}

// buildTodoResourceLinks constructs relevant links for todo resources.
func (s *Server) buildTodoResourceLinks(
	projectID *uuid.UUID,
	done *bool,
	priority *string,
	tag *string,
	overdue bool,
) map[string]string {
	links := map[string]string{
		"all_todos": "toki://todos",
	}

	// Build query URL with same filters
	var queryParts []string
	if projectID != nil {
		queryParts = append(queryParts, fmt.Sprintf("project_id=%s", projectID.String()))
	}
	if done != nil {
		queryParts = append(queryParts, fmt.Sprintf("done=%t", *done))
	}
	if priority != nil {
		queryParts = append(queryParts, fmt.Sprintf("priority=%s", *priority))
	}
	if tag != nil {
		queryParts = append(queryParts, fmt.Sprintf("tag=%s", *tag))
	}
	if overdue {
		queryParts = append(queryParts, "overdue=true")
	}

	if len(queryParts) > 0 {
		links["query"] = "toki://query?" + strings.Join(queryParts, "&")
	}

	// Add links to related pre-built views
	if done == nil || !*done {
		links["pending"] = "toki://todos/pending"
	}
	if priority == nil {
		links["high_priority"] = "toki://todos/high-priority"
	}
	if !overdue {
		links["overdue"] = "toki://todos/overdue"
	}

	return links
}

func (s *Server) registerStatsResource() {
	s.mcp.AddResource(&mcp.Resource{
		URI:         "toki://stats",
		Name:        "Summary Statistics",
		Description: "Overview of todo statistics including totals, pending/completed counts, overdue items, breakdown by priority and project, and oldest pending todo",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		stats, err := s.calculateStats()
		if err != nil {
			return nil, fmt.Errorf("failed to calculate stats: %w", err)
		}

		resourceData := ResourceData{
			Metadata: ResourceMetadata{
				Timestamp:   time.Now(),
				Count:       0, // Stats don't have a count
				ResourceURI: "toki://stats",
			},
			Data: stats,
			Links: map[string]string{
				"all_todos": "toki://todos",
				"pending":   "toki://todos/pending",
				"overdue":   "toki://todos/overdue",
				"projects":  "toki://projects",
			},
		}

		jsonBytes, err := json.MarshalIndent(resourceData, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource data: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonBytes),
				},
			},
		}, nil
	})
}

// StatsData represents the statistics summary.
type StatsData struct {
	Summary       StatsSummary       `json:"summary"`
	ByPriority    map[string]int     `json:"by_priority"`
	ByProject     []ProjectStats     `json:"by_project"`
	OldestPending *OldestPendingTodo `json:"oldest_pending,omitempty"`
}

// StatsSummary contains overall todo counts.
type StatsSummary struct {
	TotalTodos int `json:"total_todos"`
	Pending    int `json:"pending"`
	Completed  int `json:"completed"`
	Overdue    int `json:"overdue"`
}

// ProjectStats contains per-project todo counts.
type ProjectStats struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	TodoCount   int    `json:"todo_count"`
}

// OldestPendingTodo represents the oldest incomplete todo.
type OldestPendingTodo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	AgeDays     int    `json:"age_days"`
}

//nolint:funlen // Stats calculation aggregates multiple data sources in a single pass
func (s *Server) calculateStats() (*StatsData, error) {
	// Fetch all todos
	allTodos, err := db.ListTodos(s.db, nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos: %w", err)
	}

	// Fetch all projects
	projects, err := db.ListProjects(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Calculate summary stats
	summary := StatsSummary{
		TotalTodos: len(allTodos),
	}

	priorityCounts := make(map[string]int)
	projectCounts := make(map[uuid.UUID]int)
	var oldestPending *models.Todo
	now := time.Now()

	for _, todo := range allTodos {
		// Count by completion status
		if todo.Done {
			summary.Completed++
		} else {
			summary.Pending++

			// Track oldest pending
			if oldestPending == nil || todo.CreatedAt.Before(oldestPending.CreatedAt) {
				oldestPending = todo
			}
		}

		// Count overdue (not done and past due date)
		if !todo.Done && todo.DueDate != nil && todo.DueDate.Before(now) {
			summary.Overdue++
		}

		// Count by priority
		priority := "none"
		if todo.Priority != nil {
			priority = *todo.Priority
		}
		priorityCounts[priority]++

		// Count by project
		projectCounts[todo.ProjectID]++
	}

	// Build by_priority map (only include priorities that exist)
	byPriority := make(map[string]int)
	for priority, count := range priorityCounts {
		byPriority[priority] = count
	}

	// Build by_project array (sorted by count, descending)
	projectStatsMap := make(map[uuid.UUID]*ProjectStats)
	for _, proj := range projects {
		if count, ok := projectCounts[proj.ID]; ok {
			projectStatsMap[proj.ID] = &ProjectStats{
				ProjectID:   proj.ID.String(),
				ProjectName: proj.Name,
				TodoCount:   count,
			}
		}
	}

	// Convert to sorted slice
	byProject := make([]ProjectStats, 0, len(projectStatsMap))
	for _, stats := range projectStatsMap {
		byProject = append(byProject, *stats)
	}

	// Sort by todo count (descending)
	for i := 0; i < len(byProject)-1; i++ {
		for j := i + 1; j < len(byProject); j++ {
			if byProject[j].TodoCount > byProject[i].TodoCount {
				byProject[i], byProject[j] = byProject[j], byProject[i]
			}
		}
	}

	// Build oldest pending info
	var oldestPendingData *OldestPendingTodo
	if oldestPending != nil {
		ageDays := int(now.Sub(oldestPending.CreatedAt).Hours() / 24)
		oldestPendingData = &OldestPendingTodo{
			ID:          oldestPending.ID.String(),
			Description: oldestPending.Description,
			AgeDays:     ageDays,
		}
	}

	return &StatsData{
		Summary:       summary,
		ByPriority:    byPriority,
		ByProject:     byProject,
		OldestPending: oldestPendingData,
	}, nil
}
