// ABOUTME: Tests for MCP prompt definitions
// ABOUTME: Validates prompt registration and invocation

package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterPrompts(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// List all prompts
	promptsResp, err := ts.session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		t.Fatalf("Failed to list prompts: %v", err)
	}

	// Expected prompts
	expectedPrompts := map[string]bool{
		"plan-project":     false,
		"daily-review":     false,
		"sprint-planning":  false,
		"track-agent-work": false,
		"coordinate-tasks": false,
		"report-status":    false,
	}

	// Verify all expected prompts are registered
	for _, prompt := range promptsResp.Prompts {
		if _, expected := expectedPrompts[prompt.Name]; expected {
			expectedPrompts[prompt.Name] = true
		}
	}

	// Check that all prompts were found
	for name, found := range expectedPrompts {
		if !found {
			t.Errorf("Expected prompt '%s' was not registered", name)
		}
	}

	// Verify we don't have extra unexpected prompts
	if len(promptsResp.Prompts) != len(expectedPrompts) {
		t.Errorf("Expected %d prompts, got %d", len(expectedPrompts), len(promptsResp.Prompts))
	}
}

//nolint:funlen // Test needs to verify multiple aspects of prompt
func TestPlanProjectPrompt(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Test with no arguments
	result, err := ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "plan-project",
	})
	if err != nil {
		t.Fatalf("Failed to get plan-project prompt: %v", err)
	}

	if result == nil {
		t.Fatal("Expected prompt result, got nil")
	}

	if len(result.Messages) == 0 {
		t.Fatal("Expected prompt messages, got empty array")
	}

	// Check that the prompt contains expected content
	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content := textContent.Text
	if content == "" {
		t.Fatal("Expected prompt content, got empty string")
	}

	// Verify it contains key sections
	expectedSections := []string{
		"# Plan Project",
		"## Overview",
		"## When to Use",
		"## Workflow Steps",
		"## Tips and Best Practices",
		"add_project",
		"add_todo",
		"list_todos",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("Expected prompt to contain '%s'", section)
		}
	}

	// Test with project_name argument
	result, err = ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "plan-project",
		Arguments: map[string]string{
			"project_name": "test-api",
		},
	})
	if err != nil {
		t.Fatalf("Failed to get plan-project prompt with args: %v", err)
	}

	textContent, ok = result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content = textContent.Text
	if !strings.Contains(content, "test-api") {
		t.Error("Expected prompt to contain project name 'test-api'")
	}
}

func TestDailyReviewPrompt(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	result, err := ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "daily-review",
	})
	if err != nil {
		t.Fatalf("Failed to get daily-review prompt: %v", err)
	}

	if len(result.Messages) == 0 {
		t.Fatal("Expected prompt messages")
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content := textContent.Text

	// Verify key sections
	expectedSections := []string{
		"# Daily Review",
		"Check Overdue Items",
		"Review High-Priority Pending Items",
		"Identify Blockers",
		"Plan Today's Focus",
		"Clean Up",
		"toki://todos/overdue",
		"toki://todos/high-priority",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("Expected daily-review prompt to contain '%s'", section)
		}
	}
}

func TestSprintPlanningPrompt(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Test with default duration
	result, err := ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "sprint-planning",
	})
	if err != nil {
		t.Fatalf("Failed to get sprint-planning prompt: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content := textContent.Text
	if !strings.Contains(content, "2 weeks") {
		t.Error("Expected default sprint duration '2 weeks'")
	}

	// Test with custom duration
	result, err = ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "sprint-planning",
		Arguments: map[string]string{
			"sprint_duration": "1 week",
		},
	})
	if err != nil {
		t.Fatalf("Failed to get sprint-planning prompt with args: %v", err)
	}

	textContent, ok = result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content = textContent.Text
	if !strings.Contains(content, "1 week") {
		t.Error("Expected custom sprint duration '1 week'")
	}

	// Verify key sections
	expectedSections := []string{
		"# Sprint Planning",
		"Review Full Backlog",
		"Group by Priority",
		"Set Sprint Goals",
		"Commit Sprint Scope",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("Expected sprint-planning prompt to contain '%s'", section)
		}
	}
}

func TestTrackAgentWorkPrompt(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	result, err := ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "track-agent-work",
	})
	if err != nil {
		t.Fatalf("Failed to get track-agent-work prompt: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content := textContent.Text

	// Verify key guidelines for agents
	expectedSections := []string{
		"# Track Agent Work",
		"Human-Visible vs Internal Work",
		"DO Create Todos For",
		"DON'T Create Todos For",
		"Outcome-Focused Todos",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("Expected track-agent-work prompt to contain '%s'", section)
		}
	}
}

func TestCoordinateTasksPrompt(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	result, err := ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "coordinate-tasks",
	})
	if err != nil {
		t.Fatalf("Failed to get coordinate-tasks prompt: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content := textContent.Text

	// Verify multi-agent coordination sections
	expectedSections := []string{
		"# Coordinate Tasks",
		"Check for Existing Work",
		"Create or Claim a Todo",
		"Signal Work Status",
		"Handoff to Another Agent",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("Expected coordinate-tasks prompt to contain '%s'", section)
		}
	}
}

func TestReportStatusPrompt(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	ts := setupTestSession(t, database)
	defer ts.cleanup()

	ctx := context.Background()

	// Test with default time range
	result, err := ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "report-status",
	})
	if err != nil {
		t.Fatalf("Failed to get report-status prompt: %v", err)
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content := textContent.Text
	if !strings.Contains(content, "this week") {
		t.Error("Expected default time range 'this week'")
	}

	// Test with custom time range
	result, err = ts.session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "report-status",
		Arguments: map[string]string{
			"time_range": "this month",
		},
	})
	if err != nil {
		t.Fatalf("Failed to get report-status prompt with args: %v", err)
	}

	textContent, ok = result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	content = textContent.Text
	if !strings.Contains(content, "this month") {
		t.Error("Expected custom time range 'this month'")
	}

	// Verify reporting sections
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
		if !strings.Contains(content, section) {
			t.Errorf("Expected report-status prompt to contain '%s'", section)
		}
	}
}
