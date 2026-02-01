// ABOUTME: Tests for MCP server initialization
// ABOUTME: Covers server creation and prompt handlers

package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//nolint:funlen
func TestHandlePlanProject(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns prompt without arguments", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "plan-project",
			},
		}

		result, err := server.handlePlanProject(ctx, req)
		if err != nil {
			t.Fatalf("handlePlanProject failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Messages) == 0 {
			t.Fatal("expected messages")
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "[project name]") {
			t.Error("expected default project name placeholder")
		}
	})

	t.Run("returns prompt with project name argument", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name:      "plan-project",
				Arguments: map[string]string{"project_name": "my-api"},
			},
		}

		result, err := server.handlePlanProject(ctx, req)
		if err != nil {
			t.Fatalf("handlePlanProject failed: %v", err)
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "my-api") {
			t.Error("expected project name in prompt")
		}
	})

	t.Run("contains expected sections", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "plan-project",
			},
		}

		result, err := server.handlePlanProject(ctx, req)
		if err != nil {
			t.Fatalf("handlePlanProject failed: %v", err)
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		expectedSections := []string{
			"# Plan Project",
			"## Overview",
			"## When to Use",
			"## Workflow Steps",
			"add_project",
			"add_todo",
		}

		for _, section := range expectedSections {
			if !strings.Contains(textContent.Text, section) {
				t.Errorf("expected prompt to contain %q", section)
			}
		}
	})
}

func TestHandleDailyReview(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns daily review prompt", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "daily-review",
			},
		}

		result, err := server.handleDailyReview(ctx, req)
		if err != nil {
			t.Fatalf("handleDailyReview failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		expectedSections := []string{
			"# Daily Review",
			"Check Overdue Items",
			"Review High-Priority Pending Items",
			"Identify Blockers",
			"toki://todos/overdue",
		}

		for _, section := range expectedSections {
			if !strings.Contains(textContent.Text, section) {
				t.Errorf("expected prompt to contain %q", section)
			}
		}
	})
}

//nolint:funlen
func TestHandleSprintPlanning(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns prompt with default duration", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "sprint-planning",
			},
		}

		result, err := server.handleSprintPlanning(ctx, req)
		if err != nil {
			t.Fatalf("handleSprintPlanning failed: %v", err)
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "2 weeks") {
			t.Error("expected default sprint duration '2 weeks'")
		}
	})

	t.Run("returns prompt with custom duration", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name:      "sprint-planning",
				Arguments: map[string]string{"sprint_duration": "1 week"},
			},
		}

		result, err := server.handleSprintPlanning(ctx, req)
		if err != nil {
			t.Fatalf("handleSprintPlanning failed: %v", err)
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "1 week") {
			t.Error("expected custom sprint duration '1 week'")
		}
	})

	t.Run("contains expected sections", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "sprint-planning",
			},
		}

		result, err := server.handleSprintPlanning(ctx, req)
		if err != nil {
			t.Fatalf("handleSprintPlanning failed: %v", err)
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		expectedSections := []string{
			"# Sprint Planning",
			"Review Full Backlog",
			"Group by Priority",
			"Set Sprint Goals",
			"Commit Sprint Scope",
		}

		for _, section := range expectedSections {
			if !strings.Contains(textContent.Text, section) {
				t.Errorf("expected prompt to contain %q", section)
			}
		}
	})
}

func TestHandleTrackAgentWork(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns track agent work prompt", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "track-agent-work",
			},
		}

		result, err := server.handleTrackAgentWork(ctx, req)
		if err != nil {
			t.Fatalf("handleTrackAgentWork failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		expectedSections := []string{
			"# Track Agent Work",
			"Human-Visible vs Internal Work",
			"DO Create Todos For",
			"DON'T Create Todos For",
			"Outcome-Focused Todos",
		}

		for _, section := range expectedSections {
			if !strings.Contains(textContent.Text, section) {
				t.Errorf("expected prompt to contain %q", section)
			}
		}
	})
}

func TestHandleCoordinateTasks(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns coordinate tasks prompt", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "coordinate-tasks",
			},
		}

		result, err := server.handleCoordinateTasks(ctx, req)
		if err != nil {
			t.Fatalf("handleCoordinateTasks failed: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		expectedSections := []string{
			"# Coordinate Tasks",
			"Check for Existing Work",
			"Create or Claim a Todo",
			"Signal Work Status",
			"Handoff to Another Agent",
		}

		for _, section := range expectedSections {
			if !strings.Contains(textContent.Text, section) {
				t.Errorf("expected prompt to contain %q", section)
			}
		}
	})
}

//nolint:funlen
func TestHandleReportStatus(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns prompt with default time range", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "report-status",
			},
		}

		result, err := server.handleReportStatus(ctx, req)
		if err != nil {
			t.Fatalf("handleReportStatus failed: %v", err)
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "this week") {
			t.Error("expected default time range 'this week'")
		}
	})

	t.Run("returns prompt with custom time range", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name:      "report-status",
				Arguments: map[string]string{"time_range": "this month"},
			},
		}

		result, err := server.handleReportStatus(ctx, req)
		if err != nil {
			t.Fatalf("handleReportStatus failed: %v", err)
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "this month") {
			t.Error("expected custom time range 'this month'")
		}
	})

	t.Run("contains expected sections", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "report-status",
			},
		}

		result, err := server.handleReportStatus(ctx, req)
		if err != nil {
			t.Fatalf("handleReportStatus failed: %v", err)
		}

		textContent := result.Messages[0].Content.(*mcp.TextContent)
		expectedSections := []string{
			"# Report Status",
			"Choose Report Type",
			"Gather Completed Work",
			"Gather In-Progress Work",
			"Identify Blockers and Risks",
			"Generate Metrics",
			"toki://stats",
		}

		for _, section := range expectedSections {
			if !strings.Contains(textContent.Text, section) {
				t.Errorf("expected prompt to contain %q", section)
			}
		}
	})
}
