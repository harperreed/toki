// ABOUTME: Tests for UI formatting functions
// ABOUTME: Covers todo and project formatting with all variations

package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/harper/toki/internal/models"
)

func TestFormatTodo_ShowsDoneStatus(t *testing.T) {
	project := models.NewProject("test", nil)

	// Test pending todo
	pendingTodo := models.NewTodo(project.ID, "pending task")
	pendingOutput := FormatTodo(pendingTodo, nil)

	if strings.Contains(pendingOutput, "✓") {
		t.Error("Pending todo should not show checkmark")
	}

	// Test completed todo
	doneTodo := models.NewTodo(project.ID, "completed task")
	doneTodo.MarkDone()
	doneOutput := FormatTodo(doneTodo, nil)

	if !strings.Contains(doneOutput, "✓") {
		t.Error("Completed todo should show checkmark")
	}

	// Ensure they're visually different
	if pendingOutput == doneOutput {
		t.Error("Pending and completed todos should look different")
	}
}

func TestFormatTodo_DoneWithPriority(t *testing.T) {
	project := models.NewProject("test", nil)
	priority := "high"

	todo := models.NewTodo(project.ID, "important completed task")
	todo.Priority = &priority
	todo.MarkDone()

	output := FormatTodo(todo, nil)

	if !strings.Contains(output, "✓") {
		t.Error("Completed todo with priority should show checkmark")
	}
	if !strings.Contains(output, "HIGH") {
		t.Error("Completed todo should still show priority")
	}
}

func TestFormatTodo_DoneWithTags(t *testing.T) {
	project := models.NewProject("test", nil)

	todo := models.NewTodo(project.ID, "tagged completed task")
	todo.MarkDone()

	tags := []*models.Tag{
		{ID: 1, Name: "bug"},
		{ID: 2, Name: "urgent"},
	}

	output := FormatTodo(todo, tags)

	if !strings.Contains(output, "✓") {
		t.Error("Completed todo with tags should show checkmark")
	}
	if !strings.Contains(output, "bug") || !strings.Contains(output, "urgent") {
		t.Error("Completed todo should still show tags")
	}
}

func TestFormatTodo_AllPriorities(t *testing.T) {
	project := models.NewProject("test", nil)

	tests := []struct {
		name     string
		priority string
		expected string
	}{
		{"high priority", "high", "HIGH"},
		{"medium priority", "medium", "MEDIUM"},
		{"low priority", "low", "LOW"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			todo := models.NewTodo(project.ID, "task with priority")
			todo.Priority = &tt.priority

			output := FormatTodo(todo, nil)

			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected output to contain %q", tt.expected)
			}
		})
	}
}

func TestFormatTodo_NoPriority(t *testing.T) {
	project := models.NewProject("test", nil)
	todo := models.NewTodo(project.ID, "task without priority")

	output := FormatTodo(todo, nil)

	// Should not contain any priority markers
	if strings.Contains(output, "[HIGH]") || strings.Contains(output, "[MEDIUM]") || strings.Contains(output, "[LOW]") {
		t.Error("todo without priority should not show priority marker")
	}
}

func TestFormatTodo_WithDueDate(t *testing.T) {
	project := models.NewProject("test", nil)

	t.Run("future due date", func(t *testing.T) {
		todo := models.NewTodo(project.ID, "future task")
		futureDate := time.Now().Add(48 * time.Hour)
		todo.DueDate = &futureDate

		output := FormatTodo(todo, nil)

		if !strings.Contains(output, "Due:") {
			t.Error("expected output to contain 'Due:'")
		}
		// Should not show overdue for future dates
		if strings.Contains(output, "(overdue)") {
			t.Error("future task should not show overdue")
		}
	})

	t.Run("overdue due date", func(t *testing.T) {
		todo := models.NewTodo(project.ID, "overdue task")
		pastDate := time.Now().Add(-48 * time.Hour)
		todo.DueDate = &pastDate

		output := FormatTodo(todo, nil)

		if !strings.Contains(output, "Due:") {
			t.Error("expected output to contain 'Due:'")
		}
		if !strings.Contains(output, "(overdue)") {
			t.Error("past due task should show overdue")
		}
	})
}

func TestFormatTodo_WithMultipleTags(t *testing.T) {
	project := models.NewProject("test", nil)
	todo := models.NewTodo(project.ID, "multi-tag task")

	tags := []*models.Tag{
		{ID: 1, Name: "backend"},
		{ID: 2, Name: "api"},
		{ID: 3, Name: "urgent"},
	}

	output := FormatTodo(todo, tags)

	if !strings.Contains(output, "Tags:") {
		t.Error("expected output to contain 'Tags:'")
	}
	if !strings.Contains(output, "backend") {
		t.Error("expected output to contain 'backend'")
	}
	if !strings.Contains(output, "api") {
		t.Error("expected output to contain 'api'")
	}
	if !strings.Contains(output, "urgent") {
		t.Error("expected output to contain 'urgent'")
	}
}

func TestFormatTodo_WithDueDateAndTags(t *testing.T) {
	project := models.NewProject("test", nil)
	todo := models.NewTodo(project.ID, "complete task")
	futureDate := time.Now().Add(24 * time.Hour)
	todo.DueDate = &futureDate

	tags := []*models.Tag{
		{ID: 1, Name: "test"},
	}

	output := FormatTodo(todo, tags)

	// Should have both due date and tags
	if !strings.Contains(output, "Due:") {
		t.Error("expected output to contain 'Due:'")
	}
	if !strings.Contains(output, "Tags:") {
		t.Error("expected output to contain 'Tags:'")
	}
	// Metadata should be separated by pipe
	if !strings.Contains(output, "|") {
		t.Error("expected metadata to be pipe-separated")
	}
}

func TestFormatTodo_ContainsDescription(t *testing.T) {
	project := models.NewProject("test", nil)
	todo := models.NewTodo(project.ID, "my unique description")

	output := FormatTodo(todo, nil)

	if !strings.Contains(output, "my unique description") {
		t.Error("expected output to contain the todo description")
	}
}

func TestFormatTodo_ContainsShortID(t *testing.T) {
	project := models.NewProject("test", nil)
	todo := models.NewTodo(project.ID, "task")

	output := FormatTodo(todo, nil)

	// The output should contain the first 6 characters of the UUID
	shortID := todo.ID.String()[:6]
	if !strings.Contains(output, shortID) {
		t.Errorf("expected output to contain short ID %q", shortID)
	}
}

func TestFormatProjectHeader(t *testing.T) {
	t.Run("without path", func(t *testing.T) {
		project := models.NewProject("test-project", nil)
		output := FormatProjectHeader(project)

		if !strings.Contains(output, "PROJECT:") {
			t.Error("expected output to contain 'PROJECT:'")
		}
		if !strings.Contains(output, "test-project") {
			t.Error("expected output to contain project name")
		}
	})

	t.Run("with path", func(t *testing.T) {
		path := "/path/to/project"
		project := models.NewProject("test-project", &path)
		output := FormatProjectHeader(project)

		if !strings.Contains(output, "PROJECT:") {
			t.Error("expected output to contain 'PROJECT:'")
		}
		if !strings.Contains(output, "test-project") {
			t.Error("expected output to contain project name")
		}
		if !strings.Contains(output, "/path/to/project") {
			t.Error("expected output to contain project path")
		}
	})
}

func TestFormatSeparator(t *testing.T) {
	output := FormatSeparator()

	// Should be non-empty
	if len(output) == 0 {
		t.Error("separator should not be empty")
	}

	// Should contain the horizontal line character
	if !strings.Contains(output, "─") {
		t.Error("separator should contain horizontal line character")
	}
}
